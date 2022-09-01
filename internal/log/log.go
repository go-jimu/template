package log

import (
	"context"

	"github.com/go-jimu/components/logger"
)

func LogRequestID(key string) logger.Valuer {
	return func(ctx context.Context) interface{} {
		if ctx == nil {
			return "none"
		}
		value := ctx.Value(key)
		return value
	}
}

func Caller() logger.Valuer {
	return logger.Caller(5)
}
