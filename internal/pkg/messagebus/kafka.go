// Package messagebus provides the shared Integration Message Kafka runtime.
// The constructor owns the client; pkg.Lifecycle starts and stops consumption
// through ConsumeHook, so the consumer stops before the client closes.
package messagebus

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	jimukafka "github.com/go-jimu/contrib/message/kafka"
	"github.com/samber/oops"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/fx"
)

// Option configures the shared Kafka publisher and consumer.
type Option struct {
	Brokers       []string `json:"brokers" toml:"brokers" yaml:"brokers"`
	ClientID      string   `json:"client-id" toml:"client-id" yaml:"client-id"`
	ConsumerGroup string   `json:"consumer-group" toml:"consumer-group" yaml:"consumer-group"`
	Topics        []string `json:"topics" toml:"topics" yaml:"topics"`
	// FailurePolicy is drop or dlq. drop commits a failed message and moves
	// on; dlq publishes it to <source-topic>.dlq first.
	FailurePolicy string `json:"failure-policy" toml:"failure-policy" yaml:"failure-policy"`
}

const (
	failurePolicyDrop = "drop"
	failurePolicyDLQ  = "dlq"
)

// Validate checks Kafka configuration before opening the client.
func (o Option) Validate() error {
	if len(o.Brokers) == 0 {
		return errors.New("kafka brokers are required")
	}
	for _, broker := range o.Brokers {
		if strings.TrimSpace(broker) == "" {
			return errors.New("kafka broker cannot be empty")
		}
	}
	if strings.TrimSpace(o.ClientID) == "" {
		return errors.New("kafka client-id is required")
	}
	if strings.TrimSpace(o.ConsumerGroup) == "" {
		return errors.New("kafka consumer-group is required")
	}
	if len(o.Topics) == 0 {
		return errors.New("kafka topics are required")
	}
	for _, topic := range o.Topics {
		if strings.TrimSpace(topic) == "" {
			return errors.New("kafka topic cannot be empty")
		}
	}
	switch o.FailurePolicy {
	case failurePolicyDrop, failurePolicyDLQ:
	default:
		return fmt.Errorf("kafka failure-policy %q must be drop or dlq", o.FailurePolicy)
	}
	return nil
}

// KafkaOut exposes provider-neutral message ports, the payload registry, and
// the consumer hook for pkg.Lifecycle.
type KafkaOut struct {
	fx.Out

	Publisher       message.Publisher
	Subscriber      message.Subscriber
	PayloadRegistry *message.PayloadRegistry
	ConsumeHook     ConsumeHook
}

// ConsumeHook starts and stops Kafka consumption.
type ConsumeHook struct {
	fx.Hook
}

type consumerRuntime struct {
	consumer      *jimukafka.Consumer
	logger        *slog.Logger
	shutdowner    fx.Shutdowner
	consumerGroup string
	cancel        context.CancelFunc
	done          chan struct{}
}

// NewKafka creates the go-jimu Kafka publisher and consumer. It registers the
// client ping and close; consumption starts through ConsumeHook.
func NewKafka(
	lifecycle fx.Lifecycle,
	shutdowner fx.Shutdowner,
	option Option,
	logger *slog.Logger,
) (KafkaOut, error) {
	if err := option.Validate(); err != nil {
		return KafkaOut{}, err
	}

	client, err := jimukafka.NewClient(
		kgo.SeedBrokers(option.Brokers...),
		kgo.ClientID(option.ClientID),
		kgo.ConsumerGroup(option.ConsumerGroup),
		kgo.ConsumeTopics(option.Topics...),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return KafkaOut{}, oops.With("operation", "kafka.client.create").Wrap(err)
	}

	kafkaLogger := logger.With(slog.String("component", "messagebus.kafka"))
	registry := message.NewPayloadRegistry()
	policy, policyOptions := failurePolicy(option.FailurePolicy)
	consumer, err := jimukafka.NewConsumer(
		client,
		policy,
		append(policyOptions,
			jimukafka.WithPayloadResolver(registry),
			jimukafka.WithErrorObserver(observeError(kafkaLogger)),
		)...,
	)
	if err != nil {
		client.Close()
		return KafkaOut{}, oops.With("operation", "kafka.consumer.create").Wrap(err)
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := client.Ping(ctx); err != nil {
				return oops.With("operation", "kafka.brokers.ping").Wrap(err)
			}
			return nil
		},
		OnStop: func(context.Context) error {
			client.Close()
			return nil
		},
	})

	runtime := &consumerRuntime{
		consumer:      consumer,
		logger:        kafkaLogger,
		shutdowner:    shutdowner,
		consumerGroup: option.ConsumerGroup,
	}
	return KafkaOut{
		Publisher:       jimukafka.NewPublisher(client),
		Subscriber:      completionLogging{subscriber: consumer},
		PayloadRegistry: registry,
		ConsumeHook:     ConsumeHook{fx.Hook{OnStart: runtime.start, OnStop: runtime.stop}},
	}, nil
}

func failurePolicy(name string) (jimukafka.FailurePolicy, []jimukafka.Option) {
	if name == failurePolicyDLQ {
		return jimukafka.DLQPolicy(), []jimukafka.Option{jimukafka.WithDefaultFailureTopics()}
	}
	return jimukafka.DropAndCommitPolicy(), nil
}

func observeError(logger *slog.Logger) jimukafka.ErrorObserver {
	return func(ctx context.Context, failure jimukafka.Error) {
		observedErr := failure.Err
		if _, ok := oops.AsOops(observedErr); !ok {
			observedErr = oops.Wrap(observedErr)
		}
		attributes := []slog.Attr{
			slog.String("stage", string(failure.Stage)),
			sloghelper.Error(observedErr),
		}
		if failure.Record != nil {
			attributes = append(
				attributes,
				slog.String("topic", failure.Record.Topic),
				slog.Int("partition", int(failure.Record.Partition)),
				slog.Int64("offset", failure.Record.Offset),
			)
		}
		logger.LogAttrs(ctx, slog.LevelError, "kafka message processing failed", attributes...)
	}
}

// completionLogging wraps every subscribed handler so the runtime, not each
// handler, logs one line per handled message. Failures are logged by the
// consumer's error observer with their stage.
type completionLogging struct {
	subscriber message.Subscriber
}

func (s completionLogging) Subscribe(handler message.Handler) error {
	return s.subscriber.Subscribe(loggedHandler{Handler: handler})
}

type loggedHandler struct {
	message.Handler
}

func (h loggedHandler) Handle(ctx context.Context, integrationMessage message.Message) error {
	startedAt := time.Now()
	if err := h.Handler.Handle(ctx, integrationMessage); err != nil {
		return err
	}
	sloghelper.FromContext(ctx).InfoContext(
		ctx,
		"integration message handled",
		slog.String("message_id", integrationMessage.ID()),
		slog.String("message_kind", string(integrationMessage.Kind())),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return nil
}

func (r *consumerRuntime) start(context.Context) error {
	runCtx, cancel := context.WithCancel(
		sloghelper.NewContext(context.Background(), r.logger),
	)
	r.cancel = cancel
	r.done = make(chan struct{})
	go func() {
		defer close(r.done)
		err := r.consumer.Run(runCtx)
		if err == nil {
			return
		}
		r.logger.ErrorContext(
			runCtx,
			"kafka consumer stopped unexpectedly",
			sloghelper.Error(oops.With("operation", "kafka.consumer.run").
				With("consumer_group", r.consumerGroup).
				Wrap(err)),
		)
		if shutdownErr := r.shutdowner.Shutdown(fx.ExitCode(1)); shutdownErr != nil {
			r.logger.Error("failed to request process shutdown", sloghelper.Error(oops.Wrap(shutdownErr)))
		}
	}()
	r.logger.Info("kafka consumer started", slog.String("consumer_group", r.consumerGroup))
	return nil
}

func (r *consumerRuntime) stop(ctx context.Context) error {
	if r.cancel == nil {
		return nil
	}
	r.cancel()
	select {
	case <-r.done:
	case <-ctx.Done():
		return oops.With("operation", "kafka.consumer.stop").Wrap(ctx.Err())
	}
	r.logger.Info("kafka consumer stopped")
	return nil
}
