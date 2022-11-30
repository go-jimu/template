package eventbus

import (
	"github.com/go-jimu/components/mediator"
)

var eventbus mediator.Mediator

func Subscribe(eh mediator.EventHandler) {
	eventbus.Subscribe(eh)
}

func Dispatch(event mediator.Event) {
	eventbus.Dispatch(event)
}

// Default return the default event bus instance
func Default() mediator.Mediator {
	return eventbus
}

func SetDefault(ev mediator.Mediator) {
	if ev == nil {
		panic("bad EventBus")
	}
	eventbus = ev
}
