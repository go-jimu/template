package log

import (
	"io"
	"os"
	"strings"

	"github.com/go-jimu/components/sloghelper"
	"golang.org/x/exp/slog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Option struct {
	Level      string `json:"level" toml:"level" yaml:"level"`
	Output     string `json:"output" toml:"output" yaml:"output"` // 输出位置，支持console或者文件路径
	MaxSize    int    `json:"max_size" toml:"max_size" yaml:"max_size"`
	MaxAge     int    `json:"max_age" toml:"max_age" yaml:"max_age"`
	MaxBackups int    `json:"max_backups" toml:"max_backups" yaml:"max_backups"`
	LocalTime  bool   `json:"local_time" toml:"local_time" yaml:"local_time"`
	Compress   bool   `json:"compress" toml:"compress" yaml:"compress"`
}

var levelDescriptions = map[string]slog.Leveler{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func NewLog(opt Option) *slog.Logger {
	opts := &slog.HandlerOptions{
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
	var output io.Writer
	if strings.ToLower(opt.Output) == "console" {
		output = os.Stdout
	} else {
		output = &lumberjack.Logger{
			Filename:   opt.Output,
			MaxSize:    opt.MaxAge,
			MaxBackups: opt.MaxBackups,
			MaxAge:     opt.MaxAge,
			LocalTime:  opt.LocalTime,
			Compress:   opt.Compress,
		}
	}

	handler := sloghelper.NewHandler(slog.NewJSONHandler(output, opts))
	logger := slog.New(handler)
	slog.SetDefault(logger)
	logger.Info("the log module has been initialized successfully.", slog.Any("option", opt))
	return logger
}
