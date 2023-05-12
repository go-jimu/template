package bootstrap

import (
	"github.com/go-jimu/template/internal/bootstrap/httpsrv"
	"github.com/go-jimu/template/internal/bootstrap/mysql"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"bootstrap",
	fx.Provide(httpsrv.NewHTTPServer),
	fx.Provide(mysql.NewMySQLDriver),
)
