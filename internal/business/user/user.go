package user

import (
	"github.com/go-jimu/template/internal/bootstrap/httpsrv"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/transport"
	"github.com/go-jimu/template/internal/business/user/infrastructure"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"domain.user",
	fx.Provide(infrastructure.NewQueryRepository),
	fx.Provide(transport.NewController),
	fx.Provide(application.NewApplication),
	fx.Provide(infrastructure.NewRepository),
	fx.Invoke(func(srv httpsrv.HTTPServer, controller httpsrv.Controller) {
		srv.With(controller)
	}),
)
