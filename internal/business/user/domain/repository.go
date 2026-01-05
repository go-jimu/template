package domain

import "context"

// Repository defines the interface for user persistence.
// It abstracts the underlying storage mechanism.
type Repository interface {
	Get(context.Context, string) (*User, error)
	Save(context.Context, *User) error
}
