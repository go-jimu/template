package user

import (
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/infrastructure"
	"github.com/go-jimu/template/internal/business/user/transport"
	"github.com/go-jimu/template/internal/pkg/httpsrv"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
)

var Module = fx.Module(
	"domain.user",
	fx.Provide(infrastructure.NewQueryRepository),
	fx.Provide(transport.NewController),
	fx.Provide(transport.NewGreetServer),
	fx.Provide(application.NewApplication),
	fx.Provide(infrastructure.NewRepository),
	fx.Invoke(func(srv httpsrv.HTTPServer, controller httpsrv.Controller) {
		srv.With(controller)
	}),
	fx.Invoke(func(g grpc.ServiceRegistrar, impl helloworld.GreeterServer) {
		helloworld.RegisterGreeterServer(g, impl)
	}),
)
