package user

import (
	"context"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/application/assembler"
	"github.com/go-jimu/template/internal/domain/user"
	"github.com/go-jimu/template/internal/transport/dto"
)

type userApplication struct {
	log      *logger.Helper
	repo     user.UserRepository
	Commands any
	Queries  any
	Handlers []mediator.EventHandler
}

func NewUserApplication(log *logger.Helper, repo user.UserRepository) *userApplication {
	return &userApplication{log: log, repo: repo}
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
