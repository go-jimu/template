package pkg

import (
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/go-jimu/template/internal/pkg/grpcsrv"
	"github.com/go-jimu/template/internal/pkg/httpsrv"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"internal.pkg",
	fx.Provide(httpsrv.NewHTTPServer),
	fx.Provide(database.NewMySQLDriver),
	fx.Provide(grpcsrv.NewGRPCServ),
)
