package eventbus

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/components/sloghelper"
)

func NewMediator(opt mediator.Options, logger *slog.Logger) mediator.Mediator {
	logger = logger.With("name", "eventbus")

	m := mediator.NewInMemMediator(opt).(*mediator.InMemMediator)
	m.WithOrphanEventHandler(func(event mediator.Event) {
		logger.Warn("find orphan event", slog.String("event_kind", string(event.Kind())))
	})
	m.WithGenContext(func(ctx context.Context, event mediator.Event) context.Context {
		eventLogger := logger.With(slog.String("event_kind", string(event.Kind())))
		ctx = sloghelper.NewContext(ctx, eventLogger)
		return ctx
	})

	mediator.SetDefault(m)
	return m
}

func Dispatch(event mediator.Event) {
	mediator.Dispatch(event)
}

func Subscribe(handler mediator.EventHandler) {
	mediator.Subscribe(handler)
}

func Default() mediator.Mediator {
	return mediator.Default()
}
