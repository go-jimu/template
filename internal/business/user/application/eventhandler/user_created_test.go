package eventhandler_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user/application/eventhandler"
	"github.com/go-jimu/template/internal/business/user/domain"
	userintegrationv1 "github.com/go-jimu/template/pkg/gen/user/integration/v1"
)

type publisherSpy struct {
	message message.Message
	err     error
}

func (p *publisherSpy) Publish(_ context.Context, integrationMessage message.Message) error {
	p.message = integrationMessage
	return p.err
}

func TestUserCreatedHandlerPublishesStableIntegrationMessage(t *testing.T) {
	publisher := new(publisherSpy)
	handler := eventhandler.NewUserCreatedHandler(publisher)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)
	occurredAt := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)

	handler.Handle(ctx, domain.UserCreated{
		UserID:     "user-1",
		Name:       "Alice",
		Email:      "alice@example.com",
		OccurredAt: occurredAt,
	})

	wantKind := message.KindOf(&userintegrationv1.UserRegisteredV1{})
	if publisher.message.Kind() != wantKind || publisher.message.Key() != "user-1" {
		t.Fatalf(
			"message kind/key = %q/%q, want %q/user-1",
			publisher.message.Kind(),
			publisher.message.Key(),
			wantKind,
		)
	}
	payload, ok := publisher.message.Payload().(*userintegrationv1.UserRegisteredV1)
	if !ok {
		t.Fatalf("payload type = %T", publisher.message.Payload())
	}
	if payload.GetUserId() != "user-1" || payload.GetOccurredAt().AsTime() != occurredAt {
		t.Fatalf("payload = %#v", payload)
	}
	for _, field := range []string{
		`"operation":"user.event.user_created"`,
		`"outcome":"success"`,
		`"event_kind":"user.created"`,
		`"duration_ms":`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}
}

func TestUserCreatedHandlerLogsPublishFailure(t *testing.T) {
	publisher := &publisherSpy{err: errors.New("publisher unavailable")}
	handler := eventhandler.NewUserCreatedHandler(publisher)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	handler.Handle(ctx, domain.UserCreated{
		UserID:     "user-1",
		Name:       "Alice",
		Email:      "alice@example.com",
		OccurredAt: time.Now(),
	})

	for _, field := range []string{
		`"operation":"user.event.user_created"`,
		`"outcome":"failed"`,
		`"duration_ms":`,
		`"user_id":"user-1"`,
		`"error":`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}
}
