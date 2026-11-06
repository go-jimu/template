// Package user assembles the User bounded context.
package user

import (
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/gen/user/private/v1/userv1connect"
	"github.com/go-jimu/template/gen/user/public/v1/userpublicv1connect"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/eventhandler"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/infrastructure"
	userconnectrpc "github.com/go-jimu/template/internal/business/user/transport/connectrpc"
	sharedconnectrpc "github.com/go-jimu/template/internal/pkg/connectrpc"
	"go.uber.org/fx"
)

// Module wires User Domain, Application, Transport, and Infrastructure objects.
var Module = fx.Module(
	"business.user",
	fx.Provide(
		infrastructure.NewUserRepository,
		infrastructure.NewUserQueryRepository,
		eventhandler.NewUserCreatedHandler,
		application.NewApplication,
		query.NewUsers,
		userconnectrpc.NewHandler,
		userconnectrpc.NewProfileHandler,
	),
	fx.Invoke(func(
		subscriber event.Subscriber,
		handler *eventhandler.UserCreatedHandler,
	) {
		subscriber.Subscribe(handler)
	}),
	fx.Invoke(func(
		handler userv1connect.UserServiceHandler,
		profiles userpublicv1connect.ProfileServiceHandler,
		private sharedconnectrpc.PrivateServer,
		public sharedconnectrpc.PublicServer,
	) {
		// private/v1 serves this deployment; public/v1 serves web and app
		// clients on its own listener, the only one to expose.
		userv1connect.RegisterUserServiceHandler(private.Registry(), handler)
		userpublicv1connect.RegisterProfileServiceHandler(public.Registry(), profiles)
	}),
)
