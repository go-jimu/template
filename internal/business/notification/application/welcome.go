package application

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/sloghelper"
)

// SendWelcomeNotification is the protocol-neutral welcome notification input.
type SendWelcomeNotification struct {
	UserID string
	Name   string
	Email  string
}

// SendWelcomeNotification stands in for a real delivery: the template's
// observable effect is this log line.
func (*Application) SendWelcomeNotification(ctx context.Context, cmd SendWelcomeNotification) error {
	sloghelper.FromContext(ctx).InfoContext(
		ctx,
		"welcome notification sent",
		slog.String("operation", "notification.send_welcome"),
		slog.String("user_id", cmd.UserID),
	)
	return nil
}
