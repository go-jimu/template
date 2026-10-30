package application

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
)

// CreateUser is the protocol-neutral input for registering a User.
type CreateUser struct {
	Name     string
	Password string
	Email    string
}

// CreateUser registers a User, persists it, then dispatches its Domain Events.
// Event dispatch is best effort: registration succeeds once the User is saved,
// and a dispatch failure is logged here, where it is suppressed.
func (a *Application) CreateUser(ctx context.Context, cmd CreateUser) (*User, error) {
	user, err := domain.NewUser(cmd.Name, cmd.Password, cmd.Email)
	if err != nil {
		return nil, err
	}
	if err = a.repository.Save(ctx, user); err != nil {
		return nil, err
	}
	if err = a.dispatcher.DispatchAll(ctx, user.Events.Drain()); err != nil {
		sloghelper.FromContext(ctx).WarnContext(
			ctx,
			"user created; domain event dispatch failed",
			slog.String("operation", "user.events.dispatch"),
			slog.String("user_id", user.ID),
			sloghelper.Error(oops.With("user_id", user.ID).Wrap(err)),
		)
	}
	return AssembleUserEntity(user), nil
}
