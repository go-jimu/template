package domain

import "context"

type Repository interface {
	Get(context.Context, string) (*User, error)
	Save(context.Context, *User) error
}
