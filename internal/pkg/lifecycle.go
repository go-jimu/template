package pkg

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/ddd/event/outbox"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"go.uber.org/fx"
)

// Lifecycle orders the process's entry points and the Outbox's Relay.
// Shutdown drains them so that every running execution reaches its end:
// committed or rolled back, acknowledged or left for redelivery. List it
// after every bounded-context module, so the handlers they subscribe are
// already registered. Fx starts hooks in order and stops them in reverse:
//
//  1. The Relay starts first, so the records of the first requests are
//     handed to it at commit.
//  2. Entry points stop taking work and finish what is running: the ConnectRPC
//     server reports not ready, keeps serving for the shutdown delay, then
//     drains its requests; the Kafka consumer stops fetching and lets its
//     running handlers finish and commit. A task worker or scheduler added
//     later belongs to this layer.
//  3. The Outbox closes: it rejects further dispatches, lets the Relay finish
//     its running records, and closes the best-effort dispatcher, which
//     handles the follow-ups of the events already committed. A record the
//     Relay does not finish stays in the database for the next instance; only
//     the best-effort dispatcher loses work when it runs out of time.
//  4. The process exits. The Kafka client closes after the Outbox (its
//     constructor's hook), because must-run handlers publish through it;
//     connection pools such as MySQL are not closed.
var Lifecycle = fx.Module(
	"internal.lifecycle",
	fx.Invoke(func(
		lifecycle fx.Lifecycle,
		relay *outbox.Outbox,
		logger *slog.Logger,
		consume messagebus.ConsumeHook,
		serve connectrpc.ServeHook,
	) {
		lifecycle.Append(fx.Hook{
			OnStart: func(context.Context) error {
				// Run's ctx carries values only: Close stops the Relay.
				ctx := sloghelper.NewContext(context.Background(), logger.With(slog.String("component", "outbox")))
				go func() {
					if err := relay.Run(ctx); err != nil {
						logger.ErrorContext(ctx, "outbox relay stopped", sloghelper.Error(err))
					}
				}()
				return nil
			},
			OnStop: relay.Close,
		})
		lifecycle.Append(consume.Hook)
		lifecycle.Append(serve.Hook)
	}),
)
