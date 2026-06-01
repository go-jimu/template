package application

import (
	"context"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type UserCreatedHandler struct {
}

func NewUserCreatedHandler() *UserCreatedHandler {
	return &UserCreatedHandler{}
}

func (s UserCreatedHandler) Listening() []event.Kind {
	return []event.Kind{domain.EKUserCreated}
}

func (s UserCreatedHandler) Handle(ctx context.Context, ev event.Event) {
}
