package pkg

import (
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"go.uber.org/fx"
)

// Lifecycle orders the process's entry points. Shutdown drains only them, so
// that every running execution reaches its end: committed or rolled back,
// acknowledged or left for redelivery. List it after every bounded-context
// module, so the resources it uses are already registered. Fx stops hooks in
// reverse, so shutdown runs:
//
//  1. Entry points stop taking work and finish what is running: the ConnectRPC
//     server reports not ready, keeps serving for the shutdown delay, then
//     drains its requests; the Kafka consumer stops fetching and lets its
//     running handlers finish and commit. A task worker or scheduler added
//     later belongs to this layer: entry points do not hand work to each
//     other in-process, since messages go to the broker and tasks to their
//     queue.
//  2. The Domain Event dispatcher drains: every handler above may raise
//     Domain Events, so it closes last, handling their follow-ups until idle.
//     It is the only step that loses work when it runs out of time; the
//     others fall back on retries, redelivery, or a requeue.
//  3. The process exits. The Kafka client closes after the dispatcher (its
//     constructor's hook), because handlers publish through it; connection
//     pools such as MySQL are not closed.
var Lifecycle = fx.Module(
	"internal.lifecycle",
	fx.Invoke(func(
		lifecycle fx.Lifecycle,
		dispatcher event.Dispatcher,
		consume messagebus.ConsumeHook,
		serve connectrpc.ServeHook,
	) {
		lifecycle.Append(fx.StopHook(dispatcher.Close))
		lifecycle.Append(consume.Hook)
		lifecycle.Append(serve.Hook)
	}),
)
