package log

import (
	"context"
	"os"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/components/logger"
)

type Option struct {
	Level      string `json:"level" toml:"level" yaml:"level"`
	MessageKey string `json:"message_key" toml:"message_key" yaml:"message_key"`
}

var (
	levelDescriptions = map[string]logger.Level{
		"debug": logger.Debug,
		"info":  logger.Info,
		"warn":  logger.Warn,
		"error": logger.Error,
		"panic": logger.Panic,
		"fatal": logger.Fatal,
	}

	log logger.Logger
)

func NewLog(opt Option) logger.Logger {
	log = logger.NewStdLogger(os.Stdout)
	log = logger.With(log,
		"caller", Caller(),
		"request-id", Carry(middleware.RequestIDKey),
	)
	log = logger.NewHelper(log,
		logger.WithLevel(levelDescriptions[opt.Level]),
		logger.WithMessageKey(opt.MessageKey),
	)
	return log
}

func Caller() logger.Valuer {
	return logger.Caller(5)
}

func Carry(key any) logger.Valuer {
	return func(ctx context.Context) interface{} {
		if ctx == nil {
			return ""
		}
		return ctx.Value(key)
	}
}
