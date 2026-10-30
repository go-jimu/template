// Package application implements the User context's write-side use cases.
// Reads live in package query.
package application

import (
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/business/user/domain"
)

// Application is the User context's entry point for Transport adapters.
type Application struct {
	repository domain.Repository
	dispatcher event.Dispatcher
}

// NewApplication assigns the User use-case dependencies.
func NewApplication(
	repository domain.Repository,
	dispatcher event.Dispatcher,
) *Application {
	return &Application{
		repository: repository,
		dispatcher: dispatcher,
	}
}
