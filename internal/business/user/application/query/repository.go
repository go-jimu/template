package query

import "context"

// Repository serves the User context's product read models.
type Repository interface {
	Get(context.Context, string) (User, error)
	List(context.Context, ListFilter) (UserPage, error)
}

// ListFilter describes one normalized keyset page of the User list.
type ListFilter struct {
	NamePrefix string
	Limit      int
	// After is the ID of the last User on the previous page. User IDs are
	// UUIDv7, so ID order is creation order.
	After string
}
