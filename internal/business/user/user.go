package user

import (
	"connectrpc.com/connect"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/infrastructure"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/pkg/gen/user/v1/userv1connect"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"domain.user",
	fx.Provide(infrastructure.NewQueryRepository),
	fx.Provide(application.NewApplication),
	fx.Provide(infrastructure.NewRepository),
	fx.Invoke(func(srv userv1connect.UserAPIHandler, c connectrpc.ConnectServer) {
		c.Register(userv1connect.NewUserAPIHandler(
			srv,
			connect.WithInterceptors(c.GetGlobalInterceptors()...)))
	}),
)
