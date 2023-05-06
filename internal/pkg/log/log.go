package log

import (
	"os"
	"strings"

	"golang.org/x/exp/slog"
)

type Option struct {
	Level string `json:"level" toml:"level" yaml:"level"`
}

var levelDescriptions = map[string]slog.Leveler{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func NewLog(opt Option) *slog.Logger {
	opts := slog.HandlerOptions{
		AddSource: true,
		Level:     levelDescriptions[opt.Level],
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				subs := strings.Split(a.Value.String(), "/")
				if len(subs) >= 2 {
					a.Value = slog.StringValue(strings.Join(subs[len(subs)-2:], "/"))
				}
			}
			return a
		},
	}
	handler := newCustomHandler(opts.NewJSONHandler(os.Stdout))
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

func Error(err error) slog.Attr {
	return slog.String("error", err.Error())
}
