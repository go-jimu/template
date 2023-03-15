package application

import (
	"context"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type CommandChangePasswordHandler struct {
	repo domain.Repository
}

func NewCommandChangePasswordHandler(repo domain.Repository) *CommandChangePasswordHandler {
	return &CommandChangePasswordHandler{
		repo: repo,
	}
}

func (h *CommandChangePasswordHandler) Handle(ctx context.Context, log logger.Logger, command *CommandChangePassword) error {
	helper := logger.NewHelper(log).WithContext(ctx)

	entity, err := h.repo.Get(ctx, command.ID)
	if err != nil {
		helper.Error("failed to get user password", "error", err.Error())
		return err
	}
	if err = entity.ChangePassword(command.OldPassword, command.NewPassword); err != nil {
		helper.Error("failed to change password", "error", err.Error())
		return err
	}
	if err = h.repo.Save(ctx, entity); err != nil {
		helper.Error("failed to save new password", "error", err.Error())
		return err
	}
	helper.Info("password is changed")
	return nil
}
