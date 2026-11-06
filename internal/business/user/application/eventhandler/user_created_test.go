package eventhandler_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/go-jimu/components/ddd/event/outbox"
	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	userintegrationv1 "github.com/go-jimu/template/gen/user/integration/v1"
	"github.com/go-jimu/template/internal/business/user/application/eventhandler"
	"github.com/go-jimu/template/internal/business/user/domain"
)

type publisherSpy struct {
	messages []message.Message
	err      error
}

func (p *publisherSpy) Publish(_ context.Context, integrationMessage message.Message) error {
	p.messages = append(p.messages, integrationMessage)
	return p.err
}

func userCreated(occurredAt time.Time) domain.UserCreated {
	return domain.UserCreated{
		UserID:     "user-1",
		Name:       "Alice",
		Email:      "alice@example.com",
		OccurredAt: occurredAt,
	}
}

func deliveryContext(ctx context.Context, attempt int) context.Context {
	return outbox.ContextWithDelivery(ctx, outbox.Delivery{RecordID: "record-1", Index: 0, Attempt: attempt})
}

func TestUserCreatedHandlerPublishesStableIntegrationMessage(t *testing.T) {
	publisher := new(publisherSpy)
	handler := eventhandler.NewUserCreatedHandler(publisher)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	)
	occurredAt := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)

	if err := handler.Handle(deliveryContext(ctx, 1), userCreated(occurredAt)); err != nil {
		t.Fatal(err)
	}

	published := publisher.messages[0]
	wantKind := message.KindOf(&userintegrationv1.UserRegisteredV1{})
	if published.Kind() != wantKind || published.Key() != "user-1" {
		t.Fatalf("message kind/key = %q/%q, want %q/user-1", published.Kind(), published.Key(), wantKind)
	}
	payload, ok := published.Payload().(*userintegrationv1.UserRegisteredV1)
	if !ok {
		t.Fatalf("payload type = %T", published.Payload())
	}
	if payload.GetUserId() != "user-1" || payload.GetOccurredAt().AsTime() != occurredAt {
		t.Fatalf("payload = %#v", payload)
	}
	for _, field := range []string{
		`"operation":"user.event.user_created"`,
		`"level":"DEBUG"`,
		`"outcome":"success"`,
		`"event_kind":"user.created"`,
		`"elapsed_ms":`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}
}

// Intent: the Relay reruns a delivery whose success it did not store, so every
// attempt publishes the same message ID and consumers deduplicate by it.
func TestUserCreatedHandlerRepublishesTheSameMessageID(t *testing.T) {
	publisher := new(publisherSpy)
	handler := eventhandler.NewUserCreatedHandler(publisher)
	created := userCreated(time.Now())

	for attempt := 1; attempt <= 2; attempt++ {
		if err := handler.Handle(deliveryContext(context.Background(), attempt), created); err != nil {
			t.Fatal(err)
		}
	}

	first, second := publisher.messages[0].ID(), publisher.messages[1].ID()
	if first != "record-1/0" || second != first {
		t.Fatalf("message IDs = %q, %q, want record-1/0 twice", first, second)
	}
}

// Intent: a publish failure goes back to the Relay, which retries the record.
func TestUserCreatedHandlerReturnsPublishFailure(t *testing.T) {
	publishErr := errors.New("publisher unavailable")
	handler := eventhandler.NewUserCreatedHandler(&publisherSpy{err: publishErr})

	err := handler.Handle(deliveryContext(context.Background(), 1), userCreated(time.Now()))
	if !errors.Is(err, publishErr) {
		t.Fatalf("Handle() error = %v, want %v", err, publishErr)
	}
	if errors.Is(err, outbox.ErrPermanent) {
		t.Fatal("publish failure is permanent; the Relay would stop retrying it")
	}
}

// Intent: an event the handler cannot run fails its record instead of
// retrying forever.
func TestUserCreatedHandlerRejectsPermanently(t *testing.T) {
	handler := eventhandler.NewUserCreatedHandler(new(publisherSpy))

	if err := handler.Handle(context.Background(), userCreated(time.Now())); !errors.Is(err, outbox.ErrPermanent) {
		t.Fatalf("Handle() without a Delivery error = %v, want %v", err, outbox.ErrPermanent)
	}
	if err := handler.Handle(deliveryContext(context.Background(), 1), &domain.UserCreated{}); !errors.Is(err, outbox.ErrPermanent) {
		t.Fatalf("Handle() of *UserCreated error = %v, want %v", err, outbox.ErrPermanent)
	}
}
