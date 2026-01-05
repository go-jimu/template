package pkg

import (
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/go-jimu/template/internal/pkg/grpcsrv"
	"github.com/go-jimu/template/internal/pkg/httpsrv"
	"go.uber.org/fx"
)

// Module is the fx module for the internal package.
// It provides common infrastructure components like HTTP server, database driver, etc.
var Module = fx.Module(
	"internal.pkg",
	fx.Provide(httpsrv.NewHTTPServer),
	fx.Provide(database.NewMySQLDriver),
	fx.Provide(grpcsrv.NewGRPCServ),
	fx.Provide(connectrpc.NewConnectRPCServer),
)
