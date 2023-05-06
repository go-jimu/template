package log

import (
	"context"
	"runtime/debug"

	"golang.org/x/exp/slog"
)

type customHandler struct {
	handler slog.Handler
}

var _ slog.Handler = (*customHandler)(nil)

func newCustomHandler(hdl slog.Handler) *customHandler {
	return &customHandler{handler: hdl}
}

func (ch *customHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return ch.handler.Enabled(ctx, level)
}

func (ch *customHandler) Handle(ctx context.Context, r slog.Record) error {
	if ch.Enabled(ctx, r.Level) && r.Level == slog.LevelError {
		r.AddAttrs(slog.String("runtime_stack", string(debug.Stack())))
	}
	return ch.handler.Handle(ctx, r)
}

func (ch *customHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hdl := ch.handler.WithAttrs(attrs)
	return newCustomHandler(hdl)
}

func (ch *customHandler) WithGroup(name string) slog.Handler {
	hdl := ch.handler.WithGroup(name)
	return newCustomHandler(hdl)
}
