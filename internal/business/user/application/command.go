package application

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/components/sloghelper"
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

func (h *CommandChangePasswordHandler) Handle(ctx context.Context, logger *slog.Logger, command *CommandChangePassword) error {
	entity, err := h.repo.Get(ctx, command.ID)
	if err != nil {
		logger.ErrorContext(ctx, "failed to get user password", sloghelper.Error(err))
		return err
	}
	if err = entity.ChangePassword(command.OldPassword, command.NewPassword); err != nil {
		logger.ErrorContext(ctx, "failed to change password", sloghelper.Error(err))
		return err
	}
	if err = h.repo.Save(ctx, entity); err != nil {
		logger.ErrorContext(ctx, "failed to save new password", sloghelper.Error(err))
		return err
	}
	logger.InfoContext(ctx, "password is changed")
	entity.Events.Raise(mediator.Default())
	return nil
}
