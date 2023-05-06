package log

import (
	"go.uber.org/fx/fxevent"
	"golang.org/x/exp/slog"
)

type FXLogger struct{}

var _ fxevent.Logger = (*FXLogger)(nil)

func (fl *FXLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		slog.Info("OnStartExecuting", slog.Any("event", e))

	case *fxevent.OnStartExecuted:
		slog.Info("OnStartExecuted", slog.Any("event", e))
	}
}
