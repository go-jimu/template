package eventbus

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"go.uber.org/fx"
)

type Option struct {
	BufferSize     int    `json:"buffer-size" toml:"buffer-size" yaml:"buffer-size"`
	DelayClose     string `json:"delay-close" toml:"delay-close" yaml:"delay-close"`
	HandlerTimeout string `json:"handler-timeout" toml:"handler-timeout" yaml:"handler-timeout"`
}

func NewDispatcher(lc fx.Lifecycle, opt Option, logger *slog.Logger) (event.Dispatcher, event.Subscriber) {
	logger = logger.With("name", "eventbus")

	opts := []event.Option{
		event.WithLogger(logger),
		event.WithBufferSize(opt.BufferSize),
		event.WithContextFactory(func(ctx context.Context, ev event.Event) context.Context {
			eventLogger := logger.With(slog.String("event_kind", string(ev.Kind())))
			ctx = sloghelper.NewContext(ctx, eventLogger)
			return ctx
		}),
	}
	if delayClose, err := time.ParseDuration(opt.DelayClose); err == nil {
		opts = append(opts, event.WithDelayClose(delayClose))
	}
	if handlerTimeout, err := time.ParseDuration(opt.HandlerTimeout); err == nil {
		opts = append(opts, event.WithHandlerTimeout(handlerTimeout))
	}

	dispatcher := event.NewDispatcher(opts...)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return dispatcher.Close(ctx)
		},
	})
	return dispatcher, dispatcher
}
