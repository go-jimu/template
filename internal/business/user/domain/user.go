package domain

import (
	"strings"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/samber/oops"
	"golang.org/x/crypto/bcrypt"
)

const (
	minimumPasswordLength = 8
	maximumPasswordLength = 72
)

var validate = validator.New()

// User is the Aggregate Root for identity, profile, and credential rules.
type User struct {
	ID             string `validate:"required,uuid"`
	Name           string `validate:"required,max=24"`
	Email          string `validate:"required,email,max=48"`
	HashedPassword []byte `validate:"required"`
	Version        int
	Events         event.Collection
}

// NewUser creates a valid User and records the UserCreated Domain Event.
func NewUser(name, password, email string) (*User, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	passwordLength := len([]byte(password))
	if passwordLength < minimumPasswordLength || passwordLength > maximumPasswordLength {
		return nil, ErrInvalidPassword
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, oops.Wrap(err)
	}

	now := time.Now().UTC()
	user := &User{
		ID:             uuid.Must(uuid.NewV7()).String(),
		Name:           name,
		Email:          email,
		HashedPassword: hashedPassword,
		Events:         event.NewCollection(),
	}
	if err = user.Validate(); err != nil {
		return nil, err
	}
	user.Events.Add(UserCreated{
		UserID:     user.ID,
		Name:       user.Name,
		Email:      user.Email,
		OccurredAt: now,
	})
	return user, nil
}

// ChangePassword verifies the current password before replacing it.
func (u *User) ChangePassword(currentPassword, newPassword string) error {
	if err := bcrypt.CompareHashAndPassword(u.HashedPassword, []byte(currentPassword)); err != nil {
		return ErrPasswordMismatch
	}
	passwordLength := len([]byte(newPassword))
	if passwordLength < minimumPasswordLength || passwordLength > maximumPasswordLength {
		return ErrInvalidPassword
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return oops.Wrap(err)
	}
	u.HashedPassword = hashedPassword
	return nil
}

// Validate checks the Aggregate's business data.
func (u *User) Validate() error {
	err := validate.Struct(u)
	if err != nil {
		return oops.With("validation_error", err.Error()).
			Wrap(ErrInvalidUser)
	}
	return nil
}
