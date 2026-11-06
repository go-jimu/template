// Package eventhandler reacts to Domain Events owned by the User context.
package eventhandler

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/ddd/event/outbox"
	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	userintegrationv1 "github.com/go-jimu/template/gen/user/integration/v1"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserCreatedHandler translates UserCreated into the UserRegisteredV1
// Integration Message and publishes it through the provider-neutral port. It
// is a must-run handler: the Outbox records UserCreated with the User, and the
// Relay runs this handler until the message is published.
type UserCreatedHandler struct {
	publisher message.Publisher
}

var _ outbox.Handler = (*UserCreatedHandler)(nil)

// NewUserCreatedHandler creates the UserCreated Domain Event Handler.
func NewUserCreatedHandler(
	publisher message.Publisher,
) *UserCreatedHandler {
	return &UserCreatedHandler{publisher: publisher}
}

// Name keys this handler's outbox records; it stays the same across
// versions.
func (*UserCreatedHandler) Name() string { return "user.publish-user-registered" }

// Listening returns the one Domain Event kind owned by this handler.
func (*UserCreatedHandler) Listening() []event.Kind {
	return []event.Kind{domain.EventKindUserCreated}
}

// Handle publishes the UserRegisteredV1 message for one UserCreated. Delivery
// is at least once, so the message ID is the Delivery key, the same on every
// retry: consumers deduplicate by it. A returned error makes the Relay retry,
// and the Relay logs it with the record and attempt.
func (h *UserCreatedHandler) Handle(ctx context.Context, domainEvent event.Event) error {
	startedAt := time.Now()
	created, ok := domainEvent.(domain.UserCreated)
	if !ok {
		return oops.With("event_kind", string(domainEvent.Kind())).
			Wrapf(outbox.ErrPermanent, "unexpected domain event payload %T", domainEvent)
	}
	delivery, ok := outbox.DeliveryFrom(ctx)
	if !ok {
		return oops.With("event_kind", string(created.Kind())).
			Wrapf(outbox.ErrPermanent, "handler run outside the outbox relay")
	}

	payload := &userintegrationv1.UserRegisteredV1{
		UserId:     created.UserID,
		Name:       created.Name,
		Email:      created.Email,
		OccurredAt: timestamppb.New(created.OccurredAt),
	}
	integrationMessage, err := message.New(
		message.KindOf(payload),
		payload,
		message.WithID(delivery.Key()),
		message.WithKey(created.UserID),
		message.WithOccurredAt(created.OccurredAt),
	)
	if err != nil {
		return oops.With("stage", "build").
			With("user_id", created.UserID).
			Wrap(err)
	}
	if err := h.publisher.Publish(ctx, integrationMessage); err != nil {
		return oops.With("stage", "publish").
			With("user_id", created.UserID).
			With("message_id", integrationMessage.ID()).
			Wrap(err)
	}
	sloghelper.FromContext(ctx).DebugContext(
		ctx,
		"integration message published",
		slog.String("operation", "user.event.user_created"),
		slog.String("outcome", "success"),
		slog.String("event_kind", string(created.Kind())),
		slog.String("user_id", created.UserID),
		slog.String("message_id", integrationMessage.ID()),
		slog.String("message_kind", string(integrationMessage.Kind())),
		slog.Int64("elapsed_ms", time.Since(startedAt).Milliseconds()),
	)
	return nil
}
