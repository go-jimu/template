package mysql

// bunslog provides logging functionalities for Bun using slog.
// This package allows SQL queries issued by Bun to be displayed using slog.

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	"github.com/go-jimu/components/sloghelper"
	"github.com/uptrace/bun"
)

// Option is a function that configures a QueryHook.
type HookOption func(*QueryHook)

// WithQueryLogLevel sets the log level for general queries.
func WithQueryLogLevel(level slog.Level) HookOption {
	return func(h *QueryHook) {
		h.queryLogLevel = level
	}
}

// WithSlowQueryLogLevel sets the log level for slow queries.
func WithSlowQueryLogLevel(level slog.Level) HookOption {
	return func(h *QueryHook) {
		h.slowQueryLogLevel = level
	}
}

// WithErrorQueryLogLevel sets the log level for queries that result in an error.
func WithErrorQueryLogLevel(level slog.Level) HookOption {
	return func(h *QueryHook) {
		h.errorLogLevel = level
	}
}

// WithSlowQueryThreshold sets the duration threshold for identifying slow queries.
func WithSlowQueryThreshold(threshold time.Duration) HookOption {
	return func(h *QueryHook) {
		h.slowQueryThreshold = threshold
	}
}

// WithLogFormat sets the custom format for slog output.
func WithLogFormat(f logFormat) HookOption {
	return func(h *QueryHook) {
		h.logFormat = f
	}
}

type logFormat func(event *bun.QueryEvent) []slog.Attr

// QueryHook is a hook for Bun that enables logging with slog.
// It implements bun.QueryHook interface.
type QueryHook struct {
	queryLogLevel      slog.Level
	slowQueryLogLevel  slog.Level
	errorLogLevel      slog.Level
	slowQueryThreshold time.Duration
	logFormat          func(event *bun.QueryEvent) []slog.Attr
	now                func() time.Time
}

// NewQueryHook initializes a new QueryHook with the given options.
func NewQueryHook(opts ...HookOption) *QueryHook {
	h := &QueryHook{
		queryLogLevel:     slog.LevelDebug,
		slowQueryLogLevel: slog.LevelWarn,
		errorLogLevel:     slog.LevelError,
		now:               time.Now,
	}

	for _, opt := range opts {
		opt(h)
	}

	// use default format
	if h.logFormat == nil {
		h.logFormat = func(event *bun.QueryEvent) []slog.Attr {
			duration := h.now().Sub(event.StartTime)

			if event.Err != nil && !errors.Is(event.Err, sql.ErrNoRows) {
				return []slog.Attr{
					slog.String("operation", event.Operation()),
					slog.String("query", event.Query),
					slog.String("duration", duration.String()),
					sloghelper.Error(event.Err),
				}
			}
			return []slog.Attr{
				slog.String("operation", event.Operation()),
				slog.String("query", event.Query),
				slog.String("duration", duration.String()),
			}
		}
	}

	return h
}

// BeforeQuery is called before a query is executed.
func (h *QueryHook) BeforeQuery(ctx context.Context, event *bun.QueryEvent) context.Context {
	return ctx
}

// AfterQuery is called after a query is executed.
// It logs the query based on its duration and whether it resulted in an error.
func (h *QueryHook) AfterQuery(ctx context.Context, event *bun.QueryEvent) {
	logger := sloghelper.FromContext(ctx)

	level := h.queryLogLevel
	duration := h.now().Sub(event.StartTime)
	if h.slowQueryThreshold > 0 && h.slowQueryThreshold <= duration {
		level = h.slowQueryLogLevel
	}

	if event.Err != nil && !errors.Is(event.Err, sql.ErrNoRows) {
		level = h.errorLogLevel
	}

	attrs := h.logFormat(event)
	logger.LogAttrs(ctx, level, "", attrs...)
}

var (
	_ bun.QueryHook = (*QueryHook)(nil)
)
