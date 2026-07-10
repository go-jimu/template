package query

import "context"

// Repository serves the User context's product read-model family.
type Repository interface {
	Get(context.Context, string) (User, error)
	List(context.Context, ListFilter) (UserPage, error)
}

// ListFilter describes the normalized User list view.
type ListFilter struct {
	NamePrefix string
	Page       int
	PageSize   int
}
