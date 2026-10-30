// Package eventbus builds the in-process Domain Event dispatcher. The
// dispatcher is drained by pkg.Lifecycle, which orders it after the servers
// and consumers that feed it.
package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
)

// Config contains Domain Event dispatcher runtime settings.
type Config struct {
	BufferSize     int    `json:"buffer-size" toml:"buffer-size" yaml:"buffer-size"`
	Workers        int    `json:"workers" toml:"workers" yaml:"workers"`
	DelayClose     string `json:"delay-close" toml:"delay-close" yaml:"delay-close"`
	HandlerTimeout string `json:"handler-timeout" toml:"handler-timeout" yaml:"handler-timeout"`
}

// Validate checks the Domain Event dispatcher component's own configuration.
func (c Config) Validate() error {
	if c.BufferSize < 1 {
		return fmt.Errorf("eventbus buffer-size must be positive")
	}
	if c.Workers < 1 {
		return fmt.Errorf("eventbus workers must be positive")
	}
	delayClose, err := time.ParseDuration(c.DelayClose)
	if err != nil {
		return oops.With("option", "delay-close").
			With("value", c.DelayClose).
			Wrap(err)
	}
	if delayClose < 0 {
		return fmt.Errorf("eventbus delay-close cannot be negative")
	}
	handlerTimeout, err := time.ParseDuration(c.HandlerTimeout)
	if err != nil {
		return oops.With("option", "handler-timeout").
			With("value", c.HandlerTimeout).
			Wrap(err)
	}
	if handlerTimeout <= 0 {
		return fmt.Errorf("eventbus handler-timeout must be positive")
	}
	return nil
}

// NewDispatcher creates the one shared dispatcher.
func NewDispatcher(
	config Config,
	logger *slog.Logger,
) (event.Dispatcher, event.Subscriber, error) {
	if err := config.Validate(); err != nil {
		return nil, nil, err
	}
	delayClose, err := time.ParseDuration(config.DelayClose)
	if err != nil {
		return nil, nil, oops.With("option", "delay-close").
			With("value", config.DelayClose).
			Wrap(err)
	}
	handlerTimeout, err := time.ParseDuration(config.HandlerTimeout)
	if err != nil {
		return nil, nil, oops.With("option", "handler-timeout").
			With("value", config.HandlerTimeout).
			Wrap(err)
	}

	eventLogger := logger.With(slog.String("component", "eventbus"))
	dispatcher := event.NewDispatcher(
		event.WithLogger(eventLogger),
		event.WithBufferSize(config.BufferSize),
		event.WithWorkers(config.Workers),
		event.WithDelayClose(delayClose),
		event.WithHandlerTimeout(handlerTimeout),
		event.WithContextFactory(func(ctx context.Context, domainEvent event.Event) context.Context {
			return sloghelper.NewContext(
				ctx,
				eventLogger.With(slog.String("event_kind", string(domainEvent.Kind()))),
			)
		}),
		event.WithUnhandledEventHandler(func(unhandled event.UnhandledContext) {
			eventLogger.Warn(
				"domain event has no handler",
				slog.String("event_kind", string(unhandled.Event.Kind())),
			)
		}),
	)
	return dispatcher, dispatcher, nil
}
