package messagesubscriber_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/notification/application/command"
	"github.com/go-jimu/template/internal/business/notification/transport/messagesubscriber"
	userintegrationv1 "github.com/go-jimu/template/pkg/gen/user/integration/v1"
	"github.com/samber/oops"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		command.NewSendWelcomeNotificationHandler(),
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
		`"operation":"notification.user_registered"`,
		`"outcome":"success"`,
		`"duration_ms":`,
		`"user_id":"user-1"`,
		`"message_id":`,
		`"operation":"notification.send_welcome"`,
	} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
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
		command.NewSendWelcomeNotificationHandler(),
	)
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)
	if err = subscriber.Handle(ctx, integrationMessage); err == nil {
		t.Fatal("Handle() error = nil, want invalid payload error")
	}
	enhanced, ok := oops.AsOops(err)
	if !ok || enhanced.Context()["message_id"] != integrationMessage.ID() {
		t.Fatalf("error context = %#v, want message_id %q", enhanced.Context(), integrationMessage.ID())
	}
	if logs.Len() != 0 {
		t.Fatalf("message subscriber logged an inner error: %q", logs.String())
	}
}
