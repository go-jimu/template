// Package application implements the User context's protocol-neutral use
// cases.
package application

import (
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
)

// Application is the User context's entry point for Transport adapters.
type Application struct {
	repository domain.Repository
	queries    query.Repository
	dispatcher event.Dispatcher
}

// NewApplication assigns the User use-case dependencies.
func NewApplication(
	repository domain.Repository,
	queries query.Repository,
	dispatcher event.Dispatcher,
) *Application {
	return &Application{
		repository: repository,
		queries:    queries,
		dispatcher: dispatcher,
	}
}
