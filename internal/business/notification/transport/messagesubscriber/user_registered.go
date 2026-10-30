// Package messagesubscriber adapts Integration Messages into Notification commands.
package messagesubscriber

import (
	"context"

	"github.com/go-jimu/components/ddd/message"
	userintegrationv1 "github.com/go-jimu/template/gen/user/integration/v1"
	"github.com/go-jimu/template/internal/business/notification/application"
	"github.com/samber/oops"
)

// UserRegisteredSubscriber demonstrates a receiving-context Integration Message boundary.
type UserRegisteredSubscriber struct {
	application *application.Application
}

var _ message.Handler = (*UserRegisteredSubscriber)(nil)

// NewUserRegisteredSubscriber creates the Notification Integration Message subscriber.
func NewUserRegisteredSubscriber(
	application *application.Application,
) *UserRegisteredSubscriber {
	return &UserRegisteredSubscriber{application: application}
}

// Listening returns the stable published contract consumed by Notification.
func (*UserRegisteredSubscriber) Listening() []message.Kind {
	return []message.Kind{message.KindOf(&userintegrationv1.UserRegisteredV1{})}
}

// Handle validates the published payload before translating it into local work.
func (s *UserRegisteredSubscriber) Handle(
	ctx context.Context,
	integrationMessage message.Message,
) error {
	payload, ok := integrationMessage.Payload().(*userintegrationv1.UserRegisteredV1)
	if !ok {
		return oops.With("message_id", integrationMessage.ID()).
			With("message_kind", string(integrationMessage.Kind())).
			Errorf("unexpected integration message payload %T", integrationMessage.Payload())
	}
	if payload.GetUserId() == "" || payload.GetName() == "" || payload.GetEmail() == "" {
		return oops.With("message_id", integrationMessage.ID()).
			With("message_kind", string(integrationMessage.Kind())).
			With("user_id", payload.GetUserId()).
			Errorf("invalid user registered payload")
	}
	if payload.GetOccurredAt() == nil {
		return oops.With("message_id", integrationMessage.ID()).
			With("message_kind", string(integrationMessage.Kind())).
			With("user_id", payload.GetUserId()).
			Errorf("user registered occurrence time is required")
	}
	if err := payload.GetOccurredAt().CheckValid(); err != nil {
		return oops.With("message_id", integrationMessage.ID()).
			With("message_kind", string(integrationMessage.Kind())).
			With("user_id", payload.GetUserId()).
			Wrap(err)
	}
	if err := s.application.SendWelcomeNotification(ctx, application.SendWelcomeNotification{
		UserID: payload.GetUserId(),
		Name:   payload.GetName(),
		Email:  payload.GetEmail(),
	}); err != nil {
		return oops.With("message_id", integrationMessage.ID()).
			With("message_kind", string(integrationMessage.Kind())).
			With("user_id", payload.GetUserId()).
			Wrap(err)
	}
	return nil
}
