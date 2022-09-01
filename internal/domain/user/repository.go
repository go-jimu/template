package user

import "context"

type UserRepository interface {
	Get(context.Context, string) (*User, error)
	Save(context.Context, *User) error
}
