package user

import (
	"context"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/domain/user"
)

type UserCreatedHandler struct {
}

func NewUserCreatedHandler() *UserCreatedHandler {
	return &UserCreatedHandler{}
}

func (s UserCreatedHandler) Listening() []mediator.EventKind {
	return []mediator.EventKind{user.EKUserCreated}
}

func (s UserCreatedHandler) Handle(ctx context.Context, ev mediator.Event) {
	select {
	case <-ctx.Done():
	default:
	}
}
