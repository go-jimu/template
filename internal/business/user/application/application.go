package application

import (
	"context"
	"log/slog"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/components/sloghelper"
	userv1 "github.com/go-jimu/template/gen/user/v1"
	"github.com/go-jimu/template/gen/user/v1/userv1connect"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type Queries struct {
	FindUserList *FindUserListHandler
}

type Commands struct {
	ChangePassword *CommandChangePasswordHandler
}

type Application struct {
	repo     domain.Repository
	Queries  *Queries
	Commands *Commands
	handlers []mediator.EventHandler
}

func NewApplication(ev mediator.Mediator, repo domain.Repository, read QueryRepository) userv1connect.UserAPIHandler {
	app := &Application{
		repo: repo,
		Queries: &Queries{
			FindUserList: NewFindUserListHandler(read),
		},
		Commands: &Commands{
			ChangePassword: NewCommandChangePasswordHandler(repo),
		},
		handlers: []mediator.EventHandler{
			NewUserCreatedHandler(),
		},
	}
	for _, hdl := range app.handlers {
		ev.Subscribe(hdl)
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
