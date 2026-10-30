// Package pkg assembles shared runtime infrastructure.
package pkg

import (
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"go.uber.org/fx"
)

// Module provides the shared runtime adapters used by bounded contexts.
// Lifecycle starts and stops the ones that carry traffic.
var Module = fx.Module(
	"internal.pkg",
	fx.Provide(connectrpc.NewServer),
	fx.Provide(database.NewMySQLDriver),
	fx.Provide(eventbus.NewDispatcher),
	fx.Provide(messagebus.NewKafka),
)
