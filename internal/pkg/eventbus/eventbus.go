// Package eventbus builds the two Domain Event dispatchers. Use cases dispatch
// to the Outbox, inside their transaction; it records the events for the
// must-run handlers and hands every committed event to the best-effort
// dispatcher. pkg.Lifecycle runs the Outbox's Relay and closes both.
package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/pkg/telemetry"
	"github.com/samber/oops"
)

// Config contains Domain Event dispatcher runtime settings.
type Config struct {
	BufferSize     int          `json:"buffer-size" toml:"buffer-size" yaml:"buffer-size"`
	Workers        int          `json:"workers" toml:"workers" yaml:"workers"`
	HandlerTimeout string       `json:"handler-timeout" toml:"handler-timeout" yaml:"handler-timeout"`
	Outbox         OutboxConfig `json:"outbox" toml:"outbox" yaml:"outbox"`
}

// Validate checks the Domain Event dispatchers' own configuration.
func (c Config) Validate() error {
	if c.BufferSize < 1 {
		return fmt.Errorf("eventbus buffer-size must be positive")
	}
	if c.Workers < 1 {
		return fmt.Errorf("eventbus workers must be positive")
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
	return c.Outbox.Validate()
}

// NewBestEffort creates the in-memory dispatcher of best-effort handlers:
// handlers whose effects may be lost, such as cache invalidation. It receives
// every Domain Event of a committed transaction from the Outbox.
func NewBestEffort(
	config Config,
	logger *slog.Logger,
) (*event.InMemoryDispatcher, event.Subscriber, error) {
	if err := config.Validate(); err != nil {
		return nil, nil, err
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
		event.WithHandlerTimeout(handlerTimeout),
		// Handlers log through this logger and name the event themselves, and
		// continue the trace of the Dispatch caller, so a handler, and what it
		// publishes, stays on the trace of the request that raised the event.
		event.WithContextFactory(func(ctx context.Context, _ event.Event) context.Context {
			if source, ok := event.SourceContext(ctx); ok {
				ctx = telemetry.ContinueTrace(ctx, source)
			}
			return sloghelper.NewContext(ctx, eventLogger)
		}),
	)
	return dispatcher, dispatcher, nil
}
