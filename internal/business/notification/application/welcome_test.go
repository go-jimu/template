package application_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/notification/application"
)

func TestSendWelcomeNotificationLogsObservableBehavior(t *testing.T) {
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(
		context.Background(),
		slog.New(slog.NewJSONHandler(&logs, nil)),
	)

	err := application.NewApplication().SendWelcomeNotification(ctx, application.SendWelcomeNotification{
		UserID: "user-1",
		Name:   "Alice",
		Email:  "alice@example.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"operation":"notification.send_welcome"`, `"user_id":"user-1"`} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("notification log %q does not contain %s", logs.String(), field)
		}
	}
}
