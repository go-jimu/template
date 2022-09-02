package main

import (
	"context"
	"os"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/infrastructure/persistence"
	"github.com/go-jimu/template/internal/log"
)

func main() {
	std := logger.NewStdLogger(os.Stdout)
	std = logger.With(std, "request_id", log.LogRequestID("request-id"), "caller", log.Caller())
	log := logger.NewHelper(std)
	log.Info("msg", "hello guys")

	repos := persistence.BuildRepositories(persistence.Option{Host: "localhost", Port: 3306, User: "root"}, log)

	app := user.NewUserApplication(log, repos.User)
	ctx := context.WithValue(context.Background(), "request-id", 1)
	app.Get(ctx, "abc")
}
