package domain

import (
	"time"

	"github.com/go-jimu/components/ddd/event"
)

// EventKindUserCreated identifies the UserCreated Domain Event inside the User context.
const EventKindUserCreated event.Kind = "user.created"

// UserCreated records the fact that a new User Aggregate was created.
//
// Its JSON form is stored in outbox records until the Relay runs them, so the
// tags are a contract: add fields, and keep existing names decodable.
type UserCreated struct {
	UserID     string    `json:"user_id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	OccurredAt time.Time `json:"occurred_at"`
}

// Kind implements event.Event.
func (UserCreated) Kind() event.Kind { return EventKindUserCreated }

// Key implements event.Keyed, so the events of one User are handled in order.
func (e UserCreated) Key() string { return e.UserID }
