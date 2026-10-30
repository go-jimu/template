// Package application implements the Notification context's
// protocol-neutral use cases.
package application

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/sloghelper"
)

// Application is the Notification context's entry point for Transport
// adapters.
type Application struct{}

// NewApplication creates the Notification use cases.
func NewApplication() *Application {
	return &Application{}
}

// Recipient is whoever a notification is addressed to, in Notification's
// own language rather than that of the context that triggered it.
type Recipient struct {
	Address     string
	DisplayName string
}

// SendWelcomeNotification is the command to welcome a new Recipient.
type SendWelcomeNotification struct {
	Recipient Recipient
	// Reference identifies the upstream fact that triggered the notification,
	// for tracing and, once delivery is real, for idempotency.
	Reference string
}

// SendWelcomeNotification stands in for a real delivery: the template's
// observable effect is this log line. The address is personal data and stays
// out of the log.
func (*Application) SendWelcomeNotification(ctx context.Context, cmd SendWelcomeNotification) error {
	sloghelper.FromContext(ctx).InfoContext(
		ctx,
		"welcome notification sent",
		slog.String("operation", "notification.send_welcome"),
		slog.String("reference", cmd.Reference),
	)
	return nil
}
