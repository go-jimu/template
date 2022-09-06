package user

import (
	"context"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/application/assembler"
	"github.com/go-jimu/template/internal/application/dto"
	"github.com/go-jimu/template/internal/domain/user"
)

type Queries struct {
	FindUserList *FindUserListHandler
}

type Commands struct {
	ChangePassword *CommandChangePasswordHandler
}

type userApplication struct {
	log      *logger.Helper
	repo     user.UserRepository
	Queries  *Queries
	Commands *Commands
	handlers []mediator.EventHandler
}

func NewUserApplication(log logger.Logger, ev mediator.Mediator, repo user.UserRepository, read QueryUserRepository) *userApplication {
	app := &userApplication{
		log:  logger.NewHelper(log),
		repo: repo,
		Queries: &Queries{
			FindUserList: NewFindUserListHandler(log, read),
		},
		Commands: &Commands{
			ChangePassword: NewCommandChangePasswordHandler(log, repo),
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

func (app *userApplication) Get(ctx context.Context, uid string) (*dto.User, error) {
	log := app.log.WithContext(ctx)
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
	dto, _ := assembler.AssembleDomainUser(entity)
	return dto, nil
}
