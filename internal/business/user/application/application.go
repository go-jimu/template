package application

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/domain"
	userv1 "github.com/go-jimu/template/pkg/gen/user/v1"
	"github.com/go-jimu/template/pkg/gen/user/v1/userv1connect"
)

// Queries groups all read-side use cases.
type Queries struct {
	FindUserList *FindUserListHandler
}

// Commands groups all write-side use cases.
type Commands struct {
	ChangePassword *CommandChangePasswordHandler
}

// Application is the entry point for the user module's application layer.
// It exposes commands and queries to the interface layer.
type Application struct {
	repo     domain.Repository
	Queries  *Queries
	Commands *Commands
	handlers []event.Handler
}

// NewApplication creates a new Application instance.
// It initializes command and query handlers and subscribes to domain events.
func NewApplication(sub event.Subscriber, dispatcher event.Dispatcher, repo domain.Repository, read QueryRepository) userv1connect.UserAPIHandler {
	app := &Application{
		repo: repo,
		Queries: &Queries{
			FindUserList: NewFindUserListHandler(read),
		},
		Commands: &Commands{
			ChangePassword: NewCommandChangePasswordHandler(repo, dispatcher),
		},
		handlers: []event.Handler{
			NewUserCreatedHandler(),
		},
	}
	for _, hdl := range app.handlers {
		sub.Subscribe(hdl)
	}
	return app
}

func (app *Application) Get(ctx context.Context, req *connect.Request[userv1.GetRequest]) (*connect.Response[userv1.GetResponse], error) {
	logger := sloghelper.FromContext(ctx).With(slog.String("user_id", req.Msg.GetId()))
	logger.Info("invoke Get method")
	entity, err := app.repo.Get(ctx, req.Msg.GetId())
	if err != nil {
		logger.Error("failed to get user", sloghelper.Error(err))
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err = entity.Validate(); err != nil {
		logger.Error("bad user entity", sloghelper.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(assembleDomainUser(entity)), nil
}
