package main

import (
	"context"
	"os"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/log"
)

func main() {
	std := logger.NewStdLogger(os.Stdout)
	std = logger.With(std, "request_id", log.LogRequestID("request-id"), "caller", log.Caller())
	log := logger.NewHelper(std)
	log.Info("msg", "hello guys")

	app := user.NewUserApplication(log, nil)
	ctx := context.WithValue(context.Background(), "request-id", 1)
	app.Get(ctx, "abc")
}
