package application

import (
	"context"

	"github.com/go-jimu/template/internal/business/user/application/query"
)

const (
	defaultPageSize = 20
	maximumPageSize = 100
)

// ListUsers is the untrusted list request from a Transport adapter.
type ListUsers struct {
	NamePrefix string
	PageSize   int
	// After is the cursor returned as UserPage.NextCursor; empty starts from
	// the newest User.
	After string
}

// GetUser returns one User read model without loading the write Aggregate.
func (a *Application) GetUser(ctx context.Context, userID string) (query.User, error) {
	return a.queries.Get(ctx, userID)
}

// ListUsers normalizes the page size and returns one page, newest User first.
func (a *Application) ListUsers(ctx context.Context, input ListUsers) (query.UserPage, error) {
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maximumPageSize {
		pageSize = maximumPageSize
	}
	return a.queries.List(ctx, query.ListFilter{
		NamePrefix: input.NamePrefix,
		Limit:      pageSize,
		After:      input.After,
	})
}
