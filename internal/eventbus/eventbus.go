package eventbus

import (
	"context"

	"github.com/go-jimu/components/mediator"
)

var eventbus = mediator.NewInMemMediator(10)

func Subscribe(eh mediator.EventHandler) {
	eventbus.Subscribe(eh)
}

func Dispatch(ctx context.Context, event mediator.Event) {
	eventbus.Dispatch(ctx, event)
}

// Default return the default event bus instance
func Default() mediator.Mediator {
	return eventbus
}

func Set(ev mediator.Mediator) {
	if ev == nil {
		panic("bad EventBus")
	}
	eventbus = ev
}
