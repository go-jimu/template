package domain

import "context"

// Repository is the write-side collection for User Aggregate Roots.
type Repository interface {
	Get(context.Context, string) (*User, error)
	Save(context.Context, *User) error
}
