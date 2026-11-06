package application

import (
	"context"

	"github.com/go-jimu/template/internal/business/user/domain"
)

// CreateUser is the protocol-neutral input for registering a User.
type CreateUser struct {
	Name     string
	Password string
	Email    string
}

// CreateUser registers a User, persists it, then dispatches its Domain Events.
// Registration succeeds once the User is saved: a dispatch error means only
// that the follow-ups will not run, and the dispatcher has already logged each
// rejected event, so the error is ignored here.
func (a *Application) CreateUser(ctx context.Context, cmd CreateUser) (*User, error) {
	user, err := domain.NewUser(cmd.Name, cmd.Password, cmd.Email)
	if err != nil {
		return nil, err
	}
	if err = a.repository.Save(ctx, user); err != nil {
		return nil, err
	}
	_ = a.dispatcher.DispatchAll(ctx, user.Events.Drain())
	return AssembleUserEntity(user), nil
}
