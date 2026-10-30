package query

import "context"

const (
	defaultPageSize = 20
	maximumPageSize = 100
)

// ListUsers is the untrusted list query from a Transport adapter.
type ListUsers struct {
	NamePrefix string
	PageSize   int
	// After is the cursor returned as UserPage.NextCursor; empty starts from
	// the newest User.
	After string
}

// Users serves User reads straight from the read model. Reads have no
// invariant, transaction, or Domain Event, so they skip the Aggregate and
// the write-side Application.
type Users struct {
	repository Repository
}

// NewUsers creates the User read side.
func NewUsers(repository Repository) *Users {
	return &Users{repository: repository}
}

// Get returns one User read model.
func (q *Users) Get(ctx context.Context, userID string) (User, error) {
	return q.repository.Get(ctx, userID)
}

// List normalizes the page size and returns one page, newest User first.
func (q *Users) List(ctx context.Context, input ListUsers) (UserPage, error) {
	pageSize := input.PageSize
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maximumPageSize {
		pageSize = maximumPageSize
	}
	return q.repository.List(ctx, ListFilter{
		NamePrefix: input.NamePrefix,
		Limit:      pageSize,
		After:      input.After,
	})
}
