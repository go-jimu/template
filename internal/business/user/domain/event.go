package domain

import (
	"time"

	"github.com/go-jimu/components/ddd/event"
)

// EventKindUserCreated identifies the UserCreated Domain Event inside the User context.
const EventKindUserCreated event.Kind = "user.created"

// UserCreated records the fact that a new User Aggregate was created.
type UserCreated struct {
	UserID     string
	Name       string
	Email      string
	OccurredAt time.Time
}

// Kind implements event.Event.
func (UserCreated) Kind() event.Kind { return EventKindUserCreated }

// Key implements event.Keyed, so the events of one User are handled in order.
func (e UserCreated) Key() string { return e.UserID }
