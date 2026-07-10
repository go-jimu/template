// Package messagesubscriber adapts Integration Messages into Notification commands.
package messagesubscriber

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/notification/application/command"
	userintegrationv1 "github.com/go-jimu/template/pkg/gen/user/integration/v1"
	"github.com/samber/oops"
)

// UserRegisteredSubscriber demonstrates a receiving-context Integration Message boundary.
type UserRegisteredSubscriber struct {
	handler *command.SendWelcomeNotificationHandler
}

var _ message.Handler = (*UserRegisteredSubscriber)(nil)

// NewUserRegisteredSubscriber creates the Notification Integration Message subscriber.
func NewUserRegisteredSubscriber(
	handler *command.SendWelcomeNotificationHandler,
) *UserRegisteredSubscriber {
	return &UserRegisteredSubscriber{handler: handler}
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
	startedAt := time.Now()
	logger := sloghelper.FromContext(ctx)

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
			Wrap(oops.Wrap(err))
	}
	if err := s.handler.Handle(ctx, command.SendWelcomeNotification{
		UserID: payload.GetUserId(),
		Name:   payload.GetName(),
		Email:  payload.GetEmail(),
	}); err != nil {
		return oops.With("message_id", integrationMessage.ID()).
			With("message_kind", string(integrationMessage.Kind())).
			With("user_id", payload.GetUserId()).
			Wrap(err)
	}
	logger.InfoContext(
		ctx,
		"integration message processing completed",
		slog.String("operation", "notification.user_registered"),
		slog.String("outcome", "success"),
		slog.String("message_id", integrationMessage.ID()),
		slog.String("message_kind", string(integrationMessage.Kind())),
		slog.String("user_id", payload.GetUserId()),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return nil
}
