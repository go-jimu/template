package eventbus_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
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
				Workers:        1,
				HandlerTimeout: "1s",
			},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		),
		fx.Provide(eventbus.NewDispatcher),
		fx.Invoke(func(lifecycle fx.Lifecycle, dispatcher event.Dispatcher) {
			lifecycle.Append(fx.StopHook(dispatcher.Close))
		}),
		fx.Populate(&dispatcher),
	)
	app.RequireStart()
	app.RequireStop()

	if err := dispatcher.Dispatch(context.Background(), testEvent{}); !errors.Is(err, event.ErrDispatcherClosed) {
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
				Workers:        1,
				HandlerTimeout: "1s",
			},
			slog.New(slog.NewTextHandler(io.Discard, nil)),
		),
		fx.Provide(eventbus.NewDispatcher),
		fx.Invoke(func(lifecycle fx.Lifecycle, dispatcher event.Dispatcher) {
			lifecycle.Append(fx.StopHook(dispatcher.Close))
		}),
		fx.Populate(&dispatcher, &subscriber),
	)
	handler := &testHandler{handled: make(chan struct{}, 1)}
	subscriber.Subscribe(handler)
	app.RequireStart()
	if err := dispatcher.Dispatch(context.Background(), testEvent{}); err != nil {
		t.Fatal(err)
	}
	app.RequireStop()

	select {
	case <-handler.handled:
	default:
		t.Fatal("accepted event was not drained during shutdown")
	}
}

type loggingHandler struct {
	handled chan struct{}
}

func (*loggingHandler) Listening() []event.Kind { return []event.Kind{"test.event"} }

// Handle logs the way the template's handlers do: through the ctx logger,
// naming the event itself.
func (h *loggingHandler) Handle(ctx context.Context, ev event.Event) {
	sloghelper.FromContext(ctx).InfoContext(ctx, "handled", slog.String("event_kind", string(ev.Kind())))
	h.handled <- struct{}{}
}

// Intent: the handler's ctx logger carries the eventbus component, and a
// handler that names its event logs event_kind once, not twice.
func TestHandlerContextLoggerHasComponentOnly(t *testing.T) {
	var logs bytes.Buffer
	dispatcher, subscriber, err := eventbus.NewDispatcher(
		eventbus.Config{BufferSize: 1, Workers: 1, HandlerTimeout: "1s"},
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer dispatcher.Close(context.Background())
	handler := &loggingHandler{handled: make(chan struct{}, 1)}
	subscriber.Subscribe(handler)
	if err = dispatcher.Dispatch(context.Background(), testEvent{}); err != nil {
		t.Fatal(err)
	}
	<-handler.handled

	line := logs.String()
	if got := strings.Count(line, `"event_kind"`); got != 1 {
		t.Fatalf("event_kind logged %d times: %s", got, line)
	}
	if !strings.Contains(line, `"component":"eventbus"`) {
		t.Fatalf("log %s has no eventbus component", line)
	}
}

type traceHandler struct {
	traced chan trace.SpanContext
}

func (*traceHandler) Listening() []event.Kind { return []event.Kind{"test.event"} }

func (h *traceHandler) Handle(ctx context.Context, _ event.Event) {
	h.traced <- trace.SpanContextFromContext(ctx)
}

// Intent: a Domain Event handler continues the trace of the request that
// dispatched the event, so what it does and publishes stays on that trace.
func TestHandlerContinuesTheDispatchersTrace(t *testing.T) {
	dispatcher, subscriber, err := eventbus.NewDispatcher(
		eventbus.Config{BufferSize: 1, Workers: 1, HandlerTimeout: "1s"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer dispatcher.Close(context.Background())
	handler := &traceHandler{traced: make(chan trace.SpanContext, 1)}
	subscriber.Subscribe(handler)

	requestCtx, span := sdktrace.NewTracerProvider().Tracer("test").Start(context.Background(), "request")
	defer span.End()
	if err = dispatcher.Dispatch(requestCtx, testEvent{}); err != nil {
		t.Fatal(err)
	}

	got := <-handler.traced
	if got.TraceID() != span.SpanContext().TraceID() {
		t.Fatalf("handler trace = %s, want the dispatcher's %s", got.TraceID(), span.SpanContext().TraceID())
	}
}
