package application

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type CommandChangePasswordHandler struct {
	repo       domain.Repository
	dispatcher event.Dispatcher
}

func NewCommandChangePasswordHandler(repo domain.Repository, dispatcher event.Dispatcher) *CommandChangePasswordHandler {
	return &CommandChangePasswordHandler{
		repo:       repo,
		dispatcher: dispatcher,
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
	if err = h.dispatcher.DispatchAll(entity.Events.Drain()); err != nil {
		logger.WarnContext(ctx, "failed to dispatch domain events", sloghelper.Error(err))
	}
	return nil
}
