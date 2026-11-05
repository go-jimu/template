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
	// FailurePolicy is retry or drop. retry handles a failed message again in
	// place, after a backoff and before the rest of its partition, and drops
	// it after 3 attempts; drop commits it and moves on at once.
	FailurePolicy string `json:"failure-policy" toml:"failure-policy" yaml:"failure-policy"`
	// HandlerTimeout bounds one handling of one message, such as "5s".
	HandlerTimeout string `json:"handler-timeout" toml:"handler-timeout" yaml:"handler-timeout"`
	// MaxPollRecords caps the records of one batch. A batch must finish
	// within the rebalance timeout, so MaxPollRecords × HandlerTimeout stays
	// below it.
	MaxPollRecords int `json:"max-poll-records" toml:"max-poll-records" yaml:"max-poll-records"`
}

const (
	failurePolicyRetry = "retry"
	failurePolicyDrop  = "drop"

	// rebalanceTimeout is set on the client: the consumer holds a rebalance
	// until its batch ends.
	rebalanceTimeout = 60 * time.Second
)

// Validate checks Kafka configuration before opening the client.
func (o Option) Validate() error {
	_, err := o.handlerTimeout()
	return err
}

// handlerTimeout validates o and returns the parsed handler timeout.
func (o Option) handlerTimeout() (time.Duration, error) {
	if len(o.Brokers) == 0 {
		return 0, errors.New("kafka brokers are required")
	}
	for _, broker := range o.Brokers {
		if strings.TrimSpace(broker) == "" {
			return 0, errors.New("kafka broker cannot be empty")
		}
	}
	if strings.TrimSpace(o.ClientID) == "" {
		return 0, errors.New("kafka client-id is required")
	}
	if strings.TrimSpace(o.ConsumerGroup) == "" {
		return 0, errors.New("kafka consumer-group is required")
	}
	if len(o.Topics) == 0 {
		return 0, errors.New("kafka topics are required")
	}
	for _, topic := range o.Topics {
		if strings.TrimSpace(topic) == "" {
			return 0, errors.New("kafka topic cannot be empty")
		}
	}
	switch o.FailurePolicy {
	case failurePolicyRetry, failurePolicyDrop:
	default:
		return 0, fmt.Errorf("kafka failure-policy %q must be retry or drop", o.FailurePolicy)
	}
	timeout, err := time.ParseDuration(o.HandlerTimeout)
	if err != nil {
		return 0, oops.With("option", "handler-timeout").With("value", o.HandlerTimeout).Wrap(err)
	}
	if timeout <= 0 {
		return 0, fmt.Errorf("kafka handler-timeout %q must be positive", o.HandlerTimeout)
	}
	if o.MaxPollRecords <= 0 {
		return 0, errors.New("kafka max-poll-records must be positive")
	}
	if time.Duration(o.MaxPollRecords)*timeout >= rebalanceTimeout {
		return 0, fmt.Errorf("kafka max-poll-records %d × handler-timeout %s must stay below the %s rebalance timeout",
			o.MaxPollRecords, timeout, rebalanceTimeout)
	}
	return timeout, nil
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
	consumer   *jimukafka.Consumer
	logger     *slog.Logger
	shutdowner fx.Shutdowner
	cancel     context.CancelFunc
	done       chan struct{}
}

// NewKafka creates the go-jimu Kafka publisher and consumer. It registers the
// client ping and close; consumption starts through ConsumeHook.
func NewKafka(
	lifecycle fx.Lifecycle,
	shutdowner fx.Shutdowner,
	option Option,
	logger *slog.Logger,
) (KafkaOut, error) {
	handlerTimeout, err := option.handlerTimeout()
	if err != nil {
		return KafkaOut{}, err
	}

	kafkaLogger := logger.With(slog.String("component", "messagebus.kafka"))
	client, err := jimukafka.NewClient(
		kgo.SeedBrokers(option.Brokers...),
		kgo.ClientID(option.ClientID),
		kgo.ConsumerGroup(option.ConsumerGroup),
		kgo.ConsumeTopics(option.Topics...),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
		kgo.RebalanceTimeout(rebalanceTimeout),
		kgo.WithLogger(jimukafka.KgoLogger(kafkaLogger, kgo.LogLevelInfo)),
	)
	if err != nil {
		return KafkaOut{}, oops.With("operation", "kafka.client.create").Wrap(err)
	}

	registry := message.NewPayloadRegistry()
	consumer, err := jimukafka.NewConsumer(
		client,
		failurePolicy(option.FailurePolicy),
		jimukafka.WithPayloadResolver(registry),
		jimukafka.WithHandlerTimeout(handlerTimeout),
		jimukafka.WithMaxPollRecords(option.MaxPollRecords),
		jimukafka.WithLogger(kafkaLogger),
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
		consumer:   consumer,
		logger:     kafkaLogger,
		shutdowner: shutdowner,
	}
	return KafkaOut{
		Publisher:       jimukafka.NewPublisher(client),
		Subscriber:      loggingSubscriber{subscriber: consumer, logger: kafkaLogger},
		PayloadRegistry: registry,
		ConsumeHook:     ConsumeHook{fx.Hook{OnStart: runtime.start, OnStop: runtime.stop}},
	}, nil
}

func failurePolicy(name string) jimukafka.FailurePolicy {
	if name == failurePolicyRetry {
		return jimukafka.RetryPolicy(jimukafka.RetryConfig{})
	}
	return jimukafka.DropAndCommitPolicy()
}

// loggingSubscriber wraps every subscribed handler in message.Logging, so the
// runtime, not each handler, records each handled message at Debug. The
// consumer logs a failure at the level its failure policy implies: Warn for
// a retry, Error for a dropped message.
type loggingSubscriber struct {
	subscriber message.Subscriber
	logger     *slog.Logger
}

func (s loggingSubscriber) Subscribe(handler message.Handler) error {
	return s.subscriber.Subscribe(message.Chain(handler, message.Logging(s.logger)))
}

func (r *consumerRuntime) start(context.Context) error {
	runCtx, cancel := context.WithCancel(
		sloghelper.NewContext(context.Background(), r.logger),
	)
	r.cancel = cancel
	r.done = make(chan struct{})
	go func() {
		defer close(r.done)
		// The consumer logs its start, its stop, and the error that stopped it.
		if err := r.consumer.Run(runCtx); err == nil {
			return
		}
		r.logger.ErrorContext(runCtx, "shutting down: kafka consumer cannot continue")
		if shutdownErr := r.shutdowner.Shutdown(fx.ExitCode(1)); shutdownErr != nil {
			r.logger.Error("failed to request process shutdown", sloghelper.Error(oops.Wrap(shutdownErr)))
		}
	}()
	return nil
}

func (r *consumerRuntime) stop(ctx context.Context) error {
	if r.cancel == nil {
		return nil
	}
	r.cancel()
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return oops.With("operation", "kafka.consumer.stop").Wrap(ctx.Err())
	}
}
