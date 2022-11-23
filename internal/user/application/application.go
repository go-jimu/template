package application

import (
	"context"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/user/domain"
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

func (app *Application) Get(ctx context.Context, uid string) (*User, error) {
	log := logger.NewHelper(logger.FromContext(ctx)).WithContext(ctx)
	log.Infof("start to get user by id: %s", uid)

	entity, err := app.repo.Get(ctx, uid)
	if err != nil {
		log.Errorf("failed to get user by id: %s, %s", uid, err.Error())
		return nil, err
	}
	if err := entity.Validate(); err != nil {
		log.Errorf("bad user entity: %s", err.Error())
		return nil, err
	}
	dto, _ := assembleDomainUser(entity)
	return dto, nil
}
