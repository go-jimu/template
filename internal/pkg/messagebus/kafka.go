// Package messagebus provides the shared Integration Message Kafka runtime.
package messagebus

import (
	"context"
	"errors"
	"log/slog"
	"strings"

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
}

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
	return nil
}

// KafkaOut exposes provider-neutral message ports and the payload registry.
type KafkaOut struct {
	fx.Out

	Publisher       message.Publisher
	Subscriber      message.Subscriber
	PayloadRegistry *message.PayloadRegistry
}

type kafkaRuntime struct {
	client        *kgo.Client
	consumer      *jimukafka.Consumer
	logger        *slog.Logger
	shutdowner    fx.Shutdowner
	consumerGroup string
	cancel        context.CancelFunc
	done          chan error
}

// NewKafka creates the go-jimu Kafka publisher, consumer, and lifecycle.
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
		return KafkaOut{}, oops.With("operation", "kafka.client.create").
			Wrap(oops.Wrap(err))
	}

	kafkaLogger := logger.With(slog.String("component", "messagebus.kafka"))
	registry := message.NewPayloadRegistry()
	consumer, err := jimukafka.NewConsumer(
		client,
		jimukafka.DropAndCommitPolicy(),
		jimukafka.WithPayloadResolver(registry),
		jimukafka.WithErrorObserver(func(ctx context.Context, failure jimukafka.Error) {
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
			kafkaLogger.LogAttrs(
				ctx,
				slog.LevelError,
				"kafka message processing failed",
				attributes...,
			)
		}),
	)
	if err != nil {
		client.Close()
		return KafkaOut{}, oops.With("operation", "kafka.consumer.create").
			Wrap(oops.Wrap(err))
	}

	runtime := &kafkaRuntime{
		client:        client,
		consumer:      consumer,
		logger:        kafkaLogger,
		shutdowner:    shutdowner,
		consumerGroup: option.ConsumerGroup,
	}
	lifecycle.Append(fx.Hook{
		OnStart: runtime.start,
		OnStop:  runtime.stop,
	})

	return KafkaOut{
		Publisher:       jimukafka.NewPublisher(client),
		Subscriber:      consumer,
		PayloadRegistry: registry,
	}, nil
}

func (r *kafkaRuntime) start(ctx context.Context) error {
	if err := r.client.Ping(ctx); err != nil {
		r.client.Close()
		return oops.With("operation", "kafka.brokers.ping").
			Wrap(oops.Wrap(err))
	}

	runCtx, cancel := context.WithCancel(
		sloghelper.NewContext(context.Background(), r.logger),
	)
	r.cancel = cancel
	r.done = make(chan error, 1)
	go func() {
		err := r.consumer.Run(runCtx)
		r.done <- err
		close(r.done)
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		err = oops.With("operation", "kafka.consumer.run").
			With("consumer_group", r.consumerGroup).
			Wrap(oops.Wrap(err))
		r.logger.ErrorContext(
			runCtx,
			"kafka consumer stopped unexpectedly",
			sloghelper.Error(err),
		)
		if shutdownErr := r.shutdowner.Shutdown(); shutdownErr != nil {
			r.logger.Error(
				"failed to request process shutdown",
				sloghelper.Error(oops.Wrap(shutdownErr)),
			)
		}
	}()
	r.logger.Info(
		"kafka message runtime started",
		slog.String("consumer_group", r.consumerGroup),
	)
	return nil
}

func (r *kafkaRuntime) stop(ctx context.Context) error {
	if r.cancel != nil {
		r.cancel()
	}
	if r.done != nil {
		select {
		case <-r.done:
		case <-ctx.Done():
			r.client.Close()
			return oops.With("operation", "kafka.consumer.stop").
				Wrap(ctx.Err())
		}
	}
	r.client.Close()
	r.logger.Info("kafka message runtime stopped")
	return nil
}
