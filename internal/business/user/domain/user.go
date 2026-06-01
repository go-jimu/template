package domain

import (
	"sync/atomic"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/pkg/validator"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// User represents the user aggregate root.
type User struct {
	ID             string `validate:"required"`
	Name           string `validate:"required"`
	Email          string `validate:"required,email"`
	HashedPassword []byte `copier:"Password"`
	Events         event.Collection
	Version        int
	Dirty          int32
	Deleted        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewUser creates a new User entity.
// It initializes the ID, validates the input, and generates a domain event.
func NewUser(name, password, email string) (*User, error) {
	user := &User{
		ID:     uuid.Must(uuid.NewV7()).String(),
		Name:   name,
		Email:  email,
		Events: event.NewCollection(),
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

// ChangePassword changes the user's password.
// It verifies the old password before setting the new one.
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

// Validate validates the User entity.
func (u *User) Validate() error {
	return validator.Validate(u)
}
