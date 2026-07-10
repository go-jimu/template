// Package user assembles the User bounded context.
package user

import (
	connect "connectrpc.com/connect"
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/command"
	"github.com/go-jimu/template/internal/business/user/application/eventhandler"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/infrastructure"
	userconnectrpc "github.com/go-jimu/template/internal/business/user/transport/connectrpc"
	sharedconnectrpc "github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/pkg/gen/user/v1/userv1connect"
	"go.uber.org/fx"
)

// Module wires User Domain, Application, Transport, and Infrastructure objects.
var Module = fx.Module(
	"domain.user",
	fx.Provide(
		infrastructure.NewUserRepository,
		infrastructure.NewUserQueryRepository,
		command.NewCreateUserHandler,
		command.NewChangePasswordHandler,
		query.NewGetUserHandler,
		query.NewListUsersHandler,
		eventhandler.NewUserCreatedHandler,
		application.NewApplication,
		userconnectrpc.NewHandler,
	),
	fx.Invoke(func(
		subscriber event.Subscriber,
		handler *eventhandler.UserCreatedHandler,
	) {
		subscriber.Subscribe(handler)
	}),
	fx.Invoke(func(
		handler userv1connect.UserServiceHandler,
		server sharedconnectrpc.Server,
	) {
		server.Register(userv1connect.NewUserServiceHandler(
			handler,
			connect.WithInterceptors(server.GetGlobalInterceptors()...),
		))
	}),
)
