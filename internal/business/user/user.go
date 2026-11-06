// Package user assembles the User bounded context.
package user

import (
	"github.com/go-jimu/components/ddd/event/outbox"
	"github.com/go-jimu/template/gen/user/private/v1/userv1connect"
	"github.com/go-jimu/template/gen/user/public/v1/userpublicv1connect"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/eventhandler"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
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
	// Register every Domain Event type a must-run handler listens to, then
	// subscribe the must-run handlers to the Outbox.
	fx.Invoke(func(
		registry *outbox.Registry,
		relay *outbox.Outbox,
		handler *eventhandler.UserCreatedHandler,
	) error {
		if err := outbox.Register[domain.UserCreated](registry); err != nil {
			return err
		}
		return relay.Subscribe(handler)
	}),
	fx.Invoke(func(
		handler userv1connect.UserServiceHandler,
		profiles userpublicv1connect.ProfileServiceHandler,
		server sharedconnectrpc.Server,
	) {
		userv1connect.RegisterUserServiceHandler(server.Registry(), handler)
		userpublicv1connect.RegisterProfileServiceHandler(server.Registry(), profiles)
	}),
)
