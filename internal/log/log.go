package log

import (
	"os"

	"github.com/go-jimu/components/logger"
)

type Option struct {
	Level      string
	MessageKey string
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

func NewLogger(opt Option) logger.Logger {
	log = logger.NewStdLogger(os.Stdout)
	log = logger.With(log, "caller", Caller())
	log = logger.NewHelper(log,
		logger.WithLevel(levelDescriptions[opt.Level]),
		logger.WithMessageKey(opt.MessageKey),
	)
	return log
}

func Caller() logger.Valuer {
	return logger.Caller(5)
}
