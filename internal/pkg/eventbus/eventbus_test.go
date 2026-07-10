package eventbus_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type testEvent struct{}

func (testEvent) Kind() event.Kind { return "test.event" }

type testHandler struct {
	handled chan struct{}
}

func (*testHandler) Listening() []event.Kind { return []event.Kind{"test.event"} }

func (h *testHandler) Handle(context.Context, event.Event) {
	h.handled <- struct{}{}
}

func TestDispatcherRejectsEventsAfterFxStop(t *testing.T) {
	var dispatcher event.Dispatcher
	app := fxtest.New(
		t,
		fx.Supply(
			eventbus.Config{
				BufferSize:     1,
				DelayClose:     "0s",
				HandlerTimeout: "1s",
			},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		),
		fx.Provide(eventbus.NewDispatcher),
		fx.Populate(&dispatcher),
	)
	app.RequireStart()
	app.RequireStop()

	if err := dispatcher.Dispatch(testEvent{}); !errors.Is(err, event.ErrDispatcherClosed) {
		t.Fatalf("Dispatch() error = %v, want %v", err, event.ErrDispatcherClosed)
	}
}

func TestDispatcherDrainsAcceptedEventsDuringFxStop(t *testing.T) {
	var dispatcher event.Dispatcher
	var subscriber event.Subscriber
	app := fxtest.New(
		t,
		fx.Supply(
			eventbus.Config{
				BufferSize:     1,
				DelayClose:     "0s",
				HandlerTimeout: "1s",
			},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		),
		fx.Provide(eventbus.NewDispatcher),
		fx.Populate(&dispatcher, &subscriber),
	)
	handler := &testHandler{handled: make(chan struct{}, 1)}
	subscriber.Subscribe(handler)
	app.RequireStart()
	if err := dispatcher.Dispatch(testEvent{}); err != nil {
		t.Fatal(err)
	}
	app.RequireStop()

	select {
	case <-handler.handled:
	default:
		t.Fatal("accepted event was not drained during shutdown")
	}
}
