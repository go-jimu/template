package domain

import "github.com/go-jimu/components/mediator"

type EventUserCreated struct {
	ID    string
	Name  string
	Email string
}

const EKUserCreated = mediator.EventKind("user.created")

func (uc EventUserCreated) Kind() mediator.EventKind {
	return EKUserCreated
}
