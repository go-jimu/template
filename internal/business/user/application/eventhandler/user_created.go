// Package eventhandler reacts to Domain Events owned by the User context.
package eventhandler

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/domain"
	userintegrationv1 "github.com/go-jimu/template/pkg/gen/user/integration/v1"
	"github.com/samber/oops"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserCreatedHandler translates UserCreated into the UserRegisteredV1
// Integration Message and publishes it through the provider-neutral port.
type UserCreatedHandler struct {
	publisher message.Publisher
}

var _ event.Handler = (*UserCreatedHandler)(nil)

// NewUserCreatedHandler creates the UserCreated Domain Event Handler.
func NewUserCreatedHandler(
	publisher message.Publisher,
) *UserCreatedHandler {
	return &UserCreatedHandler{publisher: publisher}
}

// Listening returns the one Domain Event kind owned by this handler.
func (*UserCreatedHandler) Listening() []event.Kind {
	return []event.Kind{domain.EventKindUserCreated}
}

// Handle translates and publishes one UserCreated Domain Event.
func (h *UserCreatedHandler) Handle(ctx context.Context, domainEvent event.Event) {
	startedAt := time.Now()
	logger := sloghelper.FromContext(ctx)
	created, ok := domainEvent.(domain.UserCreated)
	if !ok {
		err := oops.With("event_kind", string(domainEvent.Kind())).
			Errorf("unexpected domain event payload %T", domainEvent)
		logger.ErrorContext(
			ctx,
			"unexpected domain event payload",
			slog.String("operation", "user.event.user_created"),
			slog.String("outcome", "failed"),
			slog.String("event_kind", string(domainEvent.Kind())),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			sloghelper.Error(err),
		)
		return
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
		message.WithKey(created.UserID),
		message.WithOccurredAt(created.OccurredAt),
	)
	stage := "build"
	if err == nil {
		stage = "publish"
		err = h.publisher.Publish(ctx, integrationMessage)
	}
	if err != nil {
		err = oops.With("stage", stage).
			With("user_id", created.UserID).
			With("message_kind", string(message.KindOf(payload))).
			Wrap(oops.Wrap(err))
		logger.ErrorContext(
			ctx,
			"integration message publication failed",
			slog.String("operation", "user.event.user_created"),
			slog.String("outcome", "failed"),
			slog.String("event_kind", string(created.Kind())),
			slog.String("message_kind", string(message.KindOf(payload))),
			slog.String("user_id", created.UserID),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			sloghelper.Error(err),
		)
		return
	}
	logger.InfoContext(
		ctx,
		"integration message published",
		slog.String("operation", "user.event.user_created"),
		slog.String("outcome", "success"),
		slog.String("event_kind", string(created.Kind())),
		slog.String("user_id", created.UserID),
		slog.String("message_id", integrationMessage.ID()),
		slog.String("message_kind", string(integrationMessage.Kind())),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
}
