package eventbus

import (
	"github.com/go-jimu/components/mediator"
	"golang.org/x/exp/slog"
)

type Option struct {
	Concurrent int `json:"concurrent" yaml:"concurrent" toml:"concurrent"`
}

var eventbus mediator.Mediator

func New(opt Option) mediator.Mediator {
	slog.Info("create a new eventbus", slog.Any("option", opt))
	eb := mediator.NewInMemMediator(opt.Concurrent)
	SetDefault(eb)
	return eb
}

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
