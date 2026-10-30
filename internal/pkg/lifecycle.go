package pkg

import (
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"go.uber.org/fx"
)

// Lifecycle orders the runtimes that move work between each other. List it
// after every bounded-context module, so the resources it uses are already
// registered. Fx stops hooks in reverse, so shutdown runs:
//
//  1. the ConnectRPC server stops taking requests and drains them;
//  2. the Kafka consumer stops fetching and finishes its current message;
//  3. the Domain Event dispatcher handles every accepted event;
//  4. the Kafka client and the database close (their constructors' hooks).
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
