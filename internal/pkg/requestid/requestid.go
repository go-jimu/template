// Package requestid carries the ID of the request a piece of work serves, so
// every log written with that work's ctx names the request. See "API logging"
// in docs/ddd/design.md.
//
// The ConnectRPC interceptor stores the ID with NewContext; the Domain Event
// dispatcher copies it onto its handlers' ctx from event.SourceContext.
// Logger adds the ID from ctx to every record of a logger, so code only logs
// with the ctx of its work.
package requestid

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/sloghelper"
)

// Key is the log attribute that carries the request ID.
const Key = "request_id"

type contextKey struct{}

// NewContext returns a copy of ctx that carries id. An empty id leaves ctx
// unchanged.
func NewContext(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, id)
}

// FromContext returns the request ID ctx carries, or false when it carries
// none.
func FromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	id, ok := ctx.Value(contextKey{}).(string)
	return id, ok
}

// Logger returns a logger that writes through logger's handler and adds the
// request ID of the logging ctx to every record, under Key. It also makes the
// result the slog default, since sloghelper.FromContext falls back to it.
// Loggers derived with With share the behavior.
func Logger(logger *slog.Logger) *slog.Logger {
	wrapped := slog.New(sloghelper.NewHandler(logger.Handler(), sloghelper.WithHandleFunc(addAttr)))
	slog.SetDefault(wrapped)
	return wrapped
}

func addAttr(ctx context.Context, record *slog.Record) {
	if id, ok := FromContext(ctx); ok {
		record.AddAttrs(slog.String(Key, id))
	}
}
