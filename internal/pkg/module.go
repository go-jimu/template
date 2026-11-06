// Package pkg assembles shared runtime infrastructure.
package pkg

import (
	"github.com/go-jimu/components/ddd/event/outbox"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"github.com/go-jimu/template/internal/pkg/telemetry"
	"go.uber.org/fx"
)

// Module provides the shared runtime adapters used by bounded contexts.
// Lifecycle starts and stops the ones that carry traffic.
var Module = fx.Module(
	"internal.pkg",
	fx.Provide(telemetry.NewTracerProvider),
	fx.Provide(connectrpc.NewServer),
	fx.Provide(database.NewMySQLDriver),
	fx.Provide(database.NewDB),
	fx.Provide(database.NewTransactor),
	fx.Provide(eventbus.NewBestEffort),
	fx.Provide(outbox.NewRegistry),
	fx.Provide(eventbus.NewOutbox),
	fx.Provide(messagebus.NewKafka),
)
