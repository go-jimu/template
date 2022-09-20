package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/eventbus"
	"github.com/go-jimu/template/internal/infrastructure/persistence"
	"github.com/go-jimu/template/internal/log"
	"github.com/go-jimu/template/internal/pkg/context"
	"github.com/go-jimu/template/internal/transport/rest"
)

func main() {
	log := log.NewLog(log.Option{Level: "info", MessageKey: "msg"}).(*logger.Helper)
	log.Infof("inited global logger")

	// pkg layer
	context.New(context.Option{Timeout: 3 * time.Second, ShutdownTimeout: 5 * time.Second})

	// domain layer
	eb := mediator.NewInMemMediator(10)
	eventbus.Set(eb)

	// infra layer
	repos := persistence.NewRepositories(persistence.Option{Host: "localhost", Port: 3306, User: "root", Password: "root", Database: "jimu"}, log)

	// application layer
	app := user.NewUserApplication(log, eb, repos.User, repos.QueryUser)

	// transport layer
	srv := rest.NewServer(rest.Option{Addr: ":9090"}, log, app)

	// graceful shutdown
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
		sig := <-sigs
		err := fmt.Errorf("quit signal: %s", sig.String())
		log.Warnf("caught quit signal, try to shutdown service in 5 seconds: %s", sig.String())
		errChan <- err
	}()

	err := <-errChan
	log.Warnf("start to shutdown server: %s", err.Error())
	ctx, cancel := context.GenShutdownContext()
	defer cancel()
	log.Warnf("kill all available contexts in %s", (1 * time.Second).String())
	<-ctx.Done()
	context.KillContextsImmediately()
	os.Exit(0)
}
