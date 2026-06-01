package eventbus_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

type testEvent struct{}

func (testEvent) Kind() event.Kind {
	return "test.event"
}

// Eventbus must remain open after fx stops so shutdown paths do not lose late domain events.
func TestDispatcherStaysOpenAfterFxStop(t *testing.T) {
	var dispatcher event.Dispatcher
	app := fxtest.New(
		t,
		fx.Supply(
			eventbus.Config{
				BufferSize:     1,
				DelayClose:     "0s",
				HandlerTimeout: "0s",
			},
			slog.Default(),
		),
		fx.Provide(eventbus.NewDispatcher),
		fx.Populate(&dispatcher),
	)

	app.RequireStart()
	t.Cleanup(func() {
		_ = dispatcher.Close(context.Background())
	})
	app.RequireStop()

	if err := dispatcher.Dispatch(testEvent{}); errors.Is(err, event.ErrDispatcherClosed) {
		t.Fatal(err)
	} else if err != nil {
		t.Fatal(err)
	}
}
