package domain

import "github.com/go-jimu/components/ddd/event"

type EventUserCreated struct {
	ID    string
	Name  string
	Email string
}

const EKUserCreated = event.Kind("user.created")

func (uc EventUserCreated) Kind() event.Kind {
	return EKUserCreated
}
