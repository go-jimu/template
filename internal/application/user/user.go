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
	log.Info("msg", "start to get user", "user_id", uid)

	entity, err := app.repo.Get(ctx, uid)
	if err != nil {
		log.Error("msg", "failed to get user", "user_id", uid, "error", err)
		return nil, err
	}
	if err := entity.Validate(); err != nil {
		log.Error("msg", "bad user entity", "error", err)
		return nil, err
	}
	dto, _ := assembler.AssembleDomainUser(entity)
	return dto, nil
}
