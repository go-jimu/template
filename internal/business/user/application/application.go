package application

import (
	"context"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/go-jimu/template/internal/pkg/log"
	"golang.org/x/exp/slog"
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
		logger.ErrorCtx(ctx, "failed to get user", log.Error(err))
		return nil, err
	}
	if err := entity.Validate(); err != nil {
		logger.ErrorCtx(ctx, "bad user entity", log.Error(err))
		return nil, err
	}
	dto, _ := assembleDomainUser(entity)
	return dto, nil
}
