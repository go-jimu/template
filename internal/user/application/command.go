package application

import (
	"context"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/user/domain"
)

type CommandChangePasswordHandler struct {
	repo domain.Repository
	log  *logger.Helper
}

func NewCommandChangePasswordHandler(log logger.Logger, repo domain.Repository) *CommandChangePasswordHandler {
	return &CommandChangePasswordHandler{
		log:  logger.NewHelper(log),
		repo: repo,
	}
}

func (h *CommandChangePasswordHandler) Handle(ctx context.Context, command *CommandChangePassword) error {
	log := h.log.WithContext(ctx)

	log.Infof("start to change user password: %s", command.ID)
	entity, err := h.repo.Get(ctx, command.ID)
	if err != nil {
		log.Errorf("failed to change password: %s", err.Error())
		return err
	}
	if err = entity.ChangePassword(command.OldPassword, command.NewPassword); err != nil {
		log.Errorf("failed to change password: %s", err.Error())
		return err
	}
	if err = h.repo.Save(ctx, entity); err != nil {
		log.Errorf("failed to change password: %s", err.Error())
		return err
	}
	log.Infof("user password is changed: %s", command.ID)
	return nil
}
