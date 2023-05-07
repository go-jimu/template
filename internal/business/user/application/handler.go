package application

import (
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type UserCreatedHandler struct {
}

func NewUserCreatedHandler() *UserCreatedHandler {
	return &UserCreatedHandler{}
}

func (s UserCreatedHandler) Listening() []mediator.EventKind {
	return []mediator.EventKind{domain.EKUserCreated}
}

func (s UserCreatedHandler) Handle(ev mediator.Event) {
}
