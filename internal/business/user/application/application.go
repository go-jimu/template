package application

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/components/sloghelper"
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

func NewApplication(ev mediator.Mediator, repo domain.Repository, read QueryRepository) *Application {
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

func (app *Application) Get(ctx context.Context, logger *slog.Logger, uid string) (*User, error) {
	entity, err := app.repo.Get(ctx, uid)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get user", sloghelper.Error(err))
		return nil, err
	}
	if err := entity.Validate(); err != nil {
		logger.ErrorContext(ctx, "bad user entity", sloghelper.Error(err))
		return nil, err
	}
	dto, _ := assembleDomainUser(entity)
	return dto, nil
}
