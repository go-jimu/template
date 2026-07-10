// Package command contains Notification write-side use cases.
package command

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/sloghelper"
)

// SendWelcomeNotification is the protocol-independent welcome notification input.
type SendWelcomeNotification struct {
	UserID string
	Name   string
	Email  string
}

// SendWelcomeNotificationHandler demonstrates the Notification use case.
type SendWelcomeNotificationHandler struct{}

// NewSendWelcomeNotificationHandler creates the welcome notification handler.
func NewSendWelcomeNotificationHandler() *SendWelcomeNotificationHandler {
	return &SendWelcomeNotificationHandler{}
}

// Handle performs the template's observable welcome notification behavior.
func (*SendWelcomeNotificationHandler) Handle(
	ctx context.Context,
	cmd SendWelcomeNotification,
) error {
	startedAt := time.Now()
	sloghelper.FromContext(ctx).InfoContext(
		ctx,
		"welcome notification sent",
		slog.String("operation", "notification.send_welcome"),
		slog.String("outcome", "success"),
		slog.String("user_id", cmd.UserID),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return nil
}
