package app

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"sync"
	"testing"

	"connectrpc.com/connect/v2"
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"xorm.io/xorm"
)

// recorder records the start and stop hooks of the fake infrastructure in the
// order Fx runs them.
type recorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *recorder) record(call string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, call)
}

func (r *recorder) hook(name string) fx.Hook {
	return fx.Hook{
		OnStart: func(context.Context) error { r.record(name + ".start"); return nil },
		OnStop:  func(context.Context) error { r.record(name + ".stop"); return nil },
	}
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

// fakeInfrastructure provides what pkg.Module provides, with every hook
// recorded and no I/O. Like the real constructors, the resource fakes
// register their own hooks; the runtime fakes leave theirs to pkg.Lifecycle.
func fakeInfrastructure(r *recorder) fx.Option {
	return fx.Provide(
		func() connectrpc.ServeOut {
			return connectrpc.ServeOut{
				Server:    fakeServer{registry: connect.NewServer()},
				ServeHook: connectrpc.ServeHook{Hook: r.hook("server")},
			}
		},
		func(lifecycle fx.Lifecycle) (*xorm.Engine, error) {
			lifecycle.Append(fx.StartHook(func() { r.record("database.start") }))
			// NewEngine does not connect.
			return xorm.NewEngine("mysql", "user@tcp(127.0.0.1:1)/db")
		},
		func() (event.Dispatcher, event.Subscriber) {
			dispatcher := event.NewDispatcher(event.WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
			return recordingDispatcher{InMemoryDispatcher: dispatcher, recorder: r}, dispatcher
		},
		func(lifecycle fx.Lifecycle) messagebus.KafkaOut {
			lifecycle.Append(r.hook("kafka.client"))
			return messagebus.KafkaOut{
				Publisher:       fakePublisher{},
				Subscriber:      fakeSubscriber{},
				PayloadRegistry: message.NewPayloadRegistry(),
				ConsumeHook:     messagebus.ConsumeHook{Hook: r.hook("consumer")},
			}
		},
	)
}

type fakeServer struct{ registry *connect.Server }

func (s fakeServer) Registry() *connect.Server { return s.registry }
func (fakeServer) Address() string             { return ":0" }

type recordingDispatcher struct {
	*event.InMemoryDispatcher
	recorder *recorder
}

func (d recordingDispatcher) Close(ctx context.Context) error {
	d.recorder.record("eventbus.stop")
	return d.InMemoryDispatcher.Close(ctx)
}

type fakePublisher struct{}

func (fakePublisher) Publish(context.Context, message.Message) error { return nil }

type fakeSubscriber struct{}

func (fakeSubscriber) Subscribe(message.Handler) error { return nil }

// Intent: the whole application starts its resources before the entry points
// and stops in the order docs/ddd/design.md states: the entry points (server,
// then consumer), then the Domain Event dispatcher, then the Kafka client. A
// pkg.Lifecycle left out of application or listed before the modules breaks
// this order silently, and fx.ValidateApp does not catch it.
func TestApplicationStartsAndStopsInOrder(t *testing.T) {
	r := &recorder{}
	app := fxtest.New(t, application(Option{}, fakeInfrastructure(r)), fx.NopLogger)
	app.RequireStart()
	started := r.snapshot()
	app.RequireStop()
	stopped := r.snapshot()[len(started):]

	index := func(call string) int {
		i := slices.Index(started, call)
		if i < 0 {
			t.Fatalf("start hooks = %v, missing %s", started, call)
		}
		return i
	}
	if !(index("database.start") < index("consumer.start") &&
		index("kafka.client.start") < index("consumer.start") &&
		index("consumer.start") < index("server.start")) {
		t.Fatalf("start hooks = %v, want database and kafka.client, then consumer, then server", started)
	}
	want := []string{"server.stop", "consumer.stop", "eventbus.stop", "kafka.client.stop"}
	if !slices.Equal(stopped, want) {
		t.Fatalf("stop hooks = %v, want %v", stopped, want)
	}
}
