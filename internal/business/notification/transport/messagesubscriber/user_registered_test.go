package messagesubscriber_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	userintegrationv1 "github.com/go-jimu/template/gen/user/integration/v1"
	"github.com/go-jimu/template/internal/business/notification/application"
	"github.com/go-jimu/template/internal/business/notification/transport/messagesubscriber"
	"github.com/samber/oops"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUserRegisteredSubscriberAcceptsValidPublishedContract(t *testing.T) {
	payload := &userintegrationv1.UserRegisteredV1{
		UserId:     "user-1",
		Name:       "Alice",
		Email:      "alice@example.com",
		OccurredAt: timestamppb.Now(),
	}
	integrationMessage, err := message.New(message.KindOf(payload), payload)
	if err != nil {
		t.Fatal(err)
	}
	subscriber := messagesubscriber.NewUserRegisteredSubscriber(
		application.NewApplication(),
	)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)
	if err = subscriber.Handle(ctx, integrationMessage); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{
		`"operation":"notification.send_welcome"`,
		`"reference":"` + integrationMessage.ID() + `"`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("notification log %q does not contain %s", logs.String(), field)
		}
	}
}

func TestUserRegisteredSubscriberRejectsIncompleteContract(t *testing.T) {
	payload := &userintegrationv1.UserRegisteredV1{UserId: "user-1"}
	integrationMessage, err := message.New(message.KindOf(payload), payload)
	if err != nil {
		t.Fatal(err)
	}
	subscriber := messagesubscriber.NewUserRegisteredSubscriber(
		application.NewApplication(),
	)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)
	if err = subscriber.Handle(ctx, integrationMessage); err == nil {
		t.Fatal("Handle() error = nil, want invalid payload error")
	}
	if !errors.Is(err, message.ErrPermanent) {
		t.Fatalf("Handle() error = %v, want it to match message.ErrPermanent", err)
	}
	enhanced, ok := oops.AsOops(err)
	if !ok || enhanced.Context()["message_id"] != integrationMessage.ID() {
		t.Fatalf("error context = %#v, want message_id %q", enhanced.Context(), integrationMessage.ID())
	}
	if logs.Len() != 0 {
		t.Fatalf("message subscriber logged an inner error: %q", logs.String())
	}
}

// Intent: every way a payload breaks the published contract fails the same on
// redelivery, so the error matches message.ErrPermanent and the consumer does
// not retry it.
func TestUserRegisteredSubscriberMarksContractViolationsPermanent(t *testing.T) {
	valid := func() *userintegrationv1.UserRegisteredV1 {
		return &userintegrationv1.UserRegisteredV1{
			UserId:     "user-1",
			Name:       "Alice",
			Email:      "alice@example.com",
			OccurredAt: timestamppb.Now(),
		}
	}
	missingOccurredAt := valid()
	missingOccurredAt.OccurredAt = nil
	invalidOccurredAt := valid()
	invalidOccurredAt.OccurredAt = &timestamppb.Timestamp{Nanos: -1}

	for _, tc := range []struct {
		name    string
		payload proto.Message
	}{
		{"unexpected payload type", wrapperspb.String("user-1")},
		{"missing occurrence time", missingOccurredAt},
		{"invalid occurrence time", invalidOccurredAt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			integrationMessage, err := message.New(message.KindOf(tc.payload), tc.payload)
			if err != nil {
				t.Fatal(err)
			}
			subscriber := messagesubscriber.NewUserRegisteredSubscriber(application.NewApplication())

			err = subscriber.Handle(context.Background(), integrationMessage)
			if !errors.Is(err, message.ErrPermanent) {
				t.Fatalf("Handle() error = %v, want it to match message.ErrPermanent", err)
			}
		})
	}
}
