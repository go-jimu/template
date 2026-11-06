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

// CreateUser registers a User: it saves the User and dispatches its Domain
// Events in one transaction, so the events are recorded exactly when the User
// is. A dispatch error means the events cannot be recorded, so it rolls the
// registration back.
func (a *Application) CreateUser(ctx context.Context, cmd CreateUser) (*User, error) {
	user, err := domain.NewUser(cmd.Name, cmd.Password, cmd.Email)
	if err != nil {
		return nil, err
	}
	err = a.transactor.Within(ctx, func(ctx context.Context) error {
		if err := a.repository.Save(ctx, user); err != nil {
			return err
		}
		return a.dispatcher.DispatchAll(ctx, user.Events.Drain())
	})
	if err != nil {
		return nil, err
	}
	return AssembleUserEntity(user), nil
}
