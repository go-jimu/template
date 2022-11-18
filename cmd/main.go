package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/driver/persistence"
	"github.com/go-jimu/template/internal/driver/rest"
	"github.com/go-jimu/template/internal/eventbus"
	"github.com/go-jimu/template/internal/pkg/context"
	"github.com/go-jimu/template/internal/pkg/log"
	"github.com/go-jimu/template/internal/pkg/option"
	"github.com/go-jimu/template/internal/user"
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

	// eventbus layer
	eb := mediator.NewInMemMediator(10)
	eventbus.Set(eb)

	// infra layer
	dbOpt := new(persistence.Option)
	if err := conf.Value("mysql").Scan(dbOpt); err != nil {
		panic(err)
	}
	conn := persistence.NewMySQLDriver(*dbOpt)
	log.Infof("init infra layer, option=%v", *dbOpt)

	// transport layer
	httpOpt := new(rest.Option)
	if err := conf.Value("http-server").Scan(httpOpt); err != nil {
		panic(err)
	}
	cg := rest.NewControllerGroup(*httpOpt, log)
	log.Infof("init transport layer, option=%v", *httpOpt)

	// each business layer
	user.Init(eb, conn, cg)

	// graceful shutdown
	srv := cg.Server()
	errChan := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Errorf("an unknown error occurred in http server: %s", err.Error())
			errChan <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.RootContext(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer stop()

	select {
	case <-ctx.Done():
	case <-errChan:
	}

	log.Warnf("caught quit signal, try to shutdown http server")
	srvCtx, srvCancel := context.GenDefaultContext()
	defer srvCancel()
	if err := srv.Shutdown(srvCtx); err != nil {
		log.Errorf("failed to shutdown http server: %s", err.Error())
	}

	log.Warnf("kill all available contexts in %s", ctxOpt.ShutdownTimeout)
	context.KillContextAfterTimeout()
	log.Infof("bye")
	os.Exit(0)
}
