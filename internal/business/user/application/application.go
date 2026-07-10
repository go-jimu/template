package application

import (
	"github.com/go-jimu/template/internal/business/user/application/command"
	"github.com/go-jimu/template/internal/business/user/application/query"
)

// Commands groups the write-side User use cases exposed to Transport adapters.
type Commands struct {
	Create         *command.CreateUserHandler
	ChangePassword *command.ChangePasswordHandler
}

// Queries groups the read-side User use cases exposed to Transport adapters.
type Queries struct {
	Get  *query.GetUserHandler
	List *query.ListUsersHandler
}

// Application is the protocol-independent entry point for the User context.
type Application struct {
	Commands Commands
	Queries  Queries
}

// NewApplication assembles the User context's commands and queries.
func NewApplication(
	create *command.CreateUserHandler,
	changePassword *command.ChangePasswordHandler,
	get *query.GetUserHandler,
	list *query.ListUsersHandler,
) *Application {
	return &Application{
		Commands: Commands{
			Create:         create,
			ChangePassword: changePassword,
		},
		Queries: Queries{
			Get:  get,
			List: list,
		},
	}
}
