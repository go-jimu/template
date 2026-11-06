// Package eventbus builds the in-process Domain Event dispatcher. The
// dispatcher is drained by pkg.Lifecycle, which closes it after the servers
// and consumers that feed it have finished their running work.
package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/pkg/requestid"
	"github.com/samber/oops"
)

// Config contains Domain Event dispatcher runtime settings.
type Config struct {
	BufferSize     int    `json:"buffer-size" toml:"buffer-size" yaml:"buffer-size"`
	Workers        int    `json:"workers" toml:"workers" yaml:"workers"`
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
		// Handlers log through this logger and name the event themselves. The
		// request ID of the Dispatch caller is copied, so a handler's logs name
		// the request that raised the event.
		event.WithContextFactory(func(ctx context.Context, _ event.Event) context.Context {
			if source, ok := event.SourceContext(ctx); ok {
				if id, ok := requestid.FromContext(source); ok {
					ctx = requestid.NewContext(ctx, id)
				}
			}
			return sloghelper.NewContext(ctx, eventLogger)
		}),
	)
	return dispatcher, dispatcher, nil
}
