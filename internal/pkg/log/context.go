package log

import (
	"context"

	"golang.org/x/exp/slog"
)

var ctxKey = &struct{}{}

func FromContext(ctx context.Context) *slog.Logger {
	log, ok := ctx.Value(ctxKey).(*slog.Logger)
	if !ok {
		return slog.Default()
	}
	return log
}

func InContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey, log)
}
