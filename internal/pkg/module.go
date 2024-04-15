package pkg

import (
	"github.com/go-jimu/template/internal/pkg/httpsrv"
	"github.com/go-jimu/template/internal/pkg/mysql"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"internal/pkg",
	fx.Provide(httpsrv.NewHTTPServer),
	fx.Provide(mysql.NewMySQLDriver),
)
