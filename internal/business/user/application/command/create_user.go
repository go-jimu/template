package command

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
)

// CreateUser is the protocol-independent input for creating a User.
type CreateUser struct {
	Name     string
	Password string
	Email    string
}

// CreatedUser is the command result returned to Transport adapters.
type CreatedUser struct {
	ID    string
	Name  string
	Email string
}

// CreateUserHandler orchestrates creation of one User Aggregate.
type CreateUserHandler struct {
	repository domain.Repository
	dispatcher event.Dispatcher
}

// NewCreateUserHandler creates the CreateUser command handler.
func NewCreateUserHandler(
	repository domain.Repository,
	dispatcher event.Dispatcher,
) *CreateUserHandler {
	return &CreateUserHandler{
		repository: repository,
		dispatcher: dispatcher,
	}
}

// Handle creates and persists a User before dispatching its Domain Events.
func (h *CreateUserHandler) Handle(
	ctx context.Context,
	cmd CreateUser,
) (CreatedUser, error) {
	startedAt := time.Now()
	logger := sloghelper.FromContext(ctx)

	user, err := domain.NewUser(cmd.Name, cmd.Password, cmd.Email)
	if err != nil {
		err = oops.With("operation", "user.create").Wrap(err)
		logger.InfoContext(
			ctx,
			"command completed",
			slog.String("operation", "user.create"),
			slog.String("outcome", "failed"),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return CreatedUser{}, err
	}
	if err = h.repository.Save(ctx, user); err != nil {
		err = oops.With("operation", "user.create").
			With("user_id", user.ID).
			Wrap(err)
		logger.InfoContext(
			ctx,
			"command completed",
			slog.String("operation", "user.create"),
			slog.String("outcome", "failed"),
			slog.String("user_id", user.ID),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return CreatedUser{}, err
	}

	result := CreatedUser{ID: user.ID, Name: user.Name, Email: user.Email}
	if dispatchErr := h.dispatcher.DispatchAll(user.Events.Drain()); dispatchErr != nil {
		dispatchErr = oops.With("operation", "user.events.dispatch").
			With("user_id", user.ID).
			Wrap(dispatchErr)
		logger.WarnContext(
			ctx,
			"command completed",
			slog.String("operation", "user.create"),
			slog.String("outcome", "success"),
			slog.String("event_dispatch_outcome", "failed"),
			slog.String("user_id", user.ID),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			sloghelper.Error(dispatchErr),
		)
		return result, nil
	}
	logger.InfoContext(
		ctx,
		"command completed",
		slog.String("operation", "user.create"),
		slog.String("outcome", "success"),
		slog.String("user_id", user.ID),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return result, nil
}
