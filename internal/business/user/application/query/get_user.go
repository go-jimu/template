package query

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
)

// GetUserHandler serves one User read model without loading the write Aggregate.
type GetUserHandler struct {
	repository Repository
}

// NewGetUserHandler creates the GetUser query handler.
func NewGetUserHandler(repository Repository) *GetUserHandler {
	return &GetUserHandler{repository: repository}
}

// Handle returns one User read model.
func (h *GetUserHandler) Handle(ctx context.Context, userID string) (User, error) {
	startedAt := time.Now()
	logger := sloghelper.FromContext(ctx)

	result, err := h.repository.Get(ctx, userID)
	if err != nil {
		err = oops.With("operation", "user.get").
			With("user_id", userID).
			Wrap(err)
		logger.InfoContext(
			ctx,
			"query completed",
			slog.String("operation", "user.get"),
			slog.String("outcome", "failed"),
			slog.String("user_id", userID),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
		)
		return User{}, err
	}
	logger.InfoContext(
		ctx,
		"query completed",
		slog.String("operation", "user.get"),
		slog.String("outcome", "success"),
		slog.String("user_id", userID),
		slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
	)
	return result, nil
}
