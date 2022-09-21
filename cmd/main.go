package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/eventbus"
	"github.com/go-jimu/template/internal/infrastructure/persistence"
	"github.com/go-jimu/template/internal/pkg/context"
	"github.com/go-jimu/template/internal/pkg/log"
	"github.com/go-jimu/template/internal/pkg/option"
	"github.com/go-jimu/template/internal/transport/rest"
)

func main() {
	conf := option.Load()

	loggerOpt := new(log.Option)
	if err := conf.Value("logger").Scan(loggerOpt); err != nil {
		panic(err)
	}
	log := log.NewLog(*loggerOpt).(*logger.Helper)
	log.Infof("init global logger, option=%v", *loggerOpt)

	// pkg layer
	ctxOpt := new(context.Option)
	if err := conf.Value("context").Scan(ctxOpt); err != nil {
		panic(err)
	}
	context.New(*ctxOpt)
	log.Infof("init context, option=%v", *ctxOpt)

	// domain layer
	eb := mediator.NewInMemMediator(10)
	eventbus.Set(eb)

	// infra layer
	dbOpt := new(persistence.Option)
	if err := conf.Value("mysql").Scan(dbOpt); err != nil {
		panic(err)
	}
	repos := persistence.NewRepositories(*dbOpt, log)
	log.Infof("init infra layer, option=%v", *dbOpt)

	// application layer
	app := user.NewUserApplication(log, eb, repos.User, repos.QueryUser)

	// transport layer
	httpOpt := new(rest.Option)
	if err := conf.Value("http-server").Scan(httpOpt); err != nil {
		panic(err)
	}
	srv := rest.NewServer(*httpOpt, log, app)
	log.Infof("init transport layer, option=%v", *httpOpt)

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

		log.Warnf("caught quit signal, try to shutdown server in %s: %s", ctxOpt.Timeout, sig.String())
		ctx, cancel := context.GenDefaultContext()
		defer cancel()
		if anotherErr := srv.Shutdown(ctx); anotherErr != nil {
			err = fmt.Errorf("%w | failed to shutdown http server: %s", err, anotherErr.Error())
		}
		errChan <- err
	}()

	err := <-errChan
	log.Warnf("start to shutdown server, %s", err.Error())

	ctx, cancel := context.GenShutdownContext()
	defer cancel()
	log.Warnf("kill all available contexts in %s", ctxOpt.ShutdownTimeout)
	<-ctx.Done()
	context.KillContextsImmediately()
	log.Infof("bye")
	os.Exit(0)
}
