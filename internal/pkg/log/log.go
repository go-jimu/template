package log

import (
	"context"

	"github.com/go-jimu/components/logger"
	zl "github.com/go-jimu/contrib/logger/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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

func newZap() (*zap.Logger, error) {
	conf := zap.NewProductionConfig()
	conf.Sampling = nil
	conf.DisableCaller = true
	conf.EncoderConfig.TimeKey = "@timestamp"
	conf.EncoderConfig.MessageKey = zapcore.OmitKey
	conf.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return conf.Build()
}

func NewLog(opt Option) logger.Logger {
	zapLog, err := newZap()
	if err != nil {
		panic(err)
	}
	log = zl.NewLog(zapLog)
	log = logger.NewHelper(log,
		logger.WithLevel(levelDescriptions[opt.Level]),
		logger.WithMessageKey(opt.MessageKey),
	)
	logger.SetDefault(Default)
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

func Default() logger.Logger {
	return log
}
