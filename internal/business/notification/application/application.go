// Package application implements the Notification context's
// protocol-neutral use cases.
package application

// Application is the Notification context's entry point for Transport
// adapters.
type Application struct{}

// NewApplication creates the Notification use cases.
func NewApplication() *Application {
	return &Application{}
}
