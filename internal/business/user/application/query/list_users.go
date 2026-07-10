package query

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
)

const (
	defaultPageSize = 20
	maximumPageSize = 100
)

// ListUsers describes the untrusted pagination input from a Transport adapter.
type ListUsers struct {
	NamePrefix string
	Page       int
	PageSize   int
}

// ListUsersHandler normalizes pagination and serves the User list read model.
type ListUsersHandler struct {
	repository Repository
}

// NewListUsersHandler creates the ListUsers query handler.
func NewListUsersHandler(repository Repository) *ListUsersHandler {
	return &ListUsersHandler{repository: repository}
}

// Handle returns a normalized page of Users.
func (h *ListUsersHandler) Handle(
	ctx context.Context,
	input ListUsers,
) (UserPage, error) {
	startedAt := time.Now()
	logger := sloghelper.FromContext(ctx)

	if input.Page < 1 {
		input.Page = 1
	}
	if input.PageSize < 1 {
		input.PageSize = defaultPageSize
	}
	if input.PageSize > maximumPageSize {
		input.PageSize = maximumPageSize
	}
	result, err := h.repository.List(ctx, ListFilter{
		NamePrefix: input.NamePrefix,
		Page:       input.Page,
		PageSize:   input.PageSize,
	})
	if err != nil {
		err = oops.With("operation", "user.list").
			With("page", input.Page).
			With("page_size", input.PageSize).
			With("name_prefix", input.NamePrefix).
			Wrap(err)
		logger.InfoContext(
			ctx,
			"query completed",
			slog.String("operation", "user.list"),
			slog.String("outcome", "failed"),
			slog.Int("page", input.Page),
			slog.Int("page_size", input.PageSize),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return UserPage{}, err
	}
	logger.InfoContext(
		ctx,
		"query completed",
		slog.String("operation", "user.list"),
		slog.String("outcome", "success"),
		slog.Int("page", input.Page),
		slog.Int("page_size", input.PageSize),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return result, nil
}
