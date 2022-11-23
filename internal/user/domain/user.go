package domain

import (
	"sync/atomic"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/pkg/validator"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             string `validate:"required"`
	Name           string `validate:"required"`
	Email          string `validate:"required,email"`
	HashedPassword []byte `copier:"Password"`
	Events         mediator.EventCollection
	Version        int
	Dirty          int32
}

func NewUser(name, password, email string) (*User, error) {
	user := &User{
		ID:     uuid.NewString(),
		Name:   name,
		Email:  email,
		Events: mediator.NewEventCollection(),
	}
	if err := user.genPassword(password); err != nil {
		return nil, err
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	user.Events.Add(
		EventUserCreated{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email},
	)
	return user, nil
}

func (u *User) genPassword(password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.HashedPassword = hashed
	return nil
}

func (u *User) ChangePassword(o, n string) error {
	if err := bcrypt.CompareHashAndPassword(u.HashedPassword, []byte(o)); err != nil {
		return err
	}
	if err := u.genPassword(n); err != nil {
		return err
	}
	atomic.CompareAndSwapInt32(&u.Dirty, 0, 1)
	return nil
}

func (u *User) Validate() error {
	return validator.Validate(u)
}
