package command

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
)

// ChangePassword is the protocol-independent input for changing credentials.
type ChangePassword struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

// ChangePasswordHandler orchestrates a password change on one User Aggregate.
type ChangePasswordHandler struct {
	repository domain.Repository
}

// NewChangePasswordHandler creates the ChangePassword command handler.
func NewChangePasswordHandler(repository domain.Repository) *ChangePasswordHandler {
	return &ChangePasswordHandler{repository: repository}
}

// Handle loads one Aggregate, applies Domain behavior, and saves it.
func (h *ChangePasswordHandler) Handle(
	ctx context.Context,
	cmd ChangePassword,
) error {
	startedAt := time.Now()
	logger := sloghelper.FromContext(ctx)

	user, err := h.repository.Get(ctx, cmd.UserID)
	if err != nil {
		err = oops.With("operation", "user.change_password").
			With("user_id", cmd.UserID).
			Wrap(err)
		logger.InfoContext(
			ctx,
			"command completed",
			slog.String("operation", "user.change_password"),
			slog.String("outcome", "failed"),
			slog.String("user_id", cmd.UserID),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return err
	}
	if err = user.ChangePassword(cmd.CurrentPassword, cmd.NewPassword); err != nil {
		err = oops.With("operation", "user.change_password").
			With("user_id", cmd.UserID).
			Wrap(err)
		logger.InfoContext(
			ctx,
			"command completed",
			slog.String("operation", "user.change_password"),
			slog.String("outcome", "failed"),
			slog.String("user_id", cmd.UserID),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return err
	}
	if err = h.repository.Save(ctx, user); err != nil {
		err = oops.With("operation", "user.change_password").
			With("user_id", cmd.UserID).
			Wrap(err)
		logger.InfoContext(
			ctx,
			"command completed",
			slog.String("operation", "user.change_password"),
			slog.String("outcome", "failed"),
			slog.String("user_id", cmd.UserID),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return err
	}
	logger.InfoContext(
		ctx,
		"command completed",
		slog.String("operation", "user.change_password"),
		slog.String("outcome", "success"),
		slog.String("user_id", cmd.UserID),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return nil
}
