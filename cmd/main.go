package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/bootstrap/httpsrv"
	"github.com/go-jimu/template/internal/bootstrap/mysql"
	"github.com/go-jimu/template/internal/business/user"
	"github.com/go-jimu/template/internal/pkg/context"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"github.com/go-jimu/template/internal/pkg/log"
	"github.com/go-jimu/template/internal/pkg/option"
)

type Option struct {
	Logger     log.Option     `json:"logger" toml:"logger" yaml:"logger"`
	Context    context.Option `json:"context" toml:"context" yaml:"context"`
	MySQL      mysql.Option   `json:"mysql" toml:"mysql" yaml:"mysql"`
	HTTPServer httpsrv.Option `json:"http-server" toml:"http-server" yaml:"http-server"`
}

func main() {
	opt := new(Option)
	conf := option.Load()
	if err := conf.Scan(opt); err != nil {
		panic(err)
	}

	// pkg layer
	log := log.NewLog(opt.Logger).(*logger.Helper)
	log.Info("loaded configurations", "option", *opt)

	context.New(opt.Context)

	// eventbus layer
	eb := mediator.NewInMemMediator(10)
	eventbus.SetDefault(eb)

	// driver layer
	conn := mysql.NewMySQLDriver(opt.MySQL)
	cg := httpsrv.NewHTTPServer(opt.HTTPServer, log)

	// each business layer
	user.Init(eb, conn, cg)

	// graceful shutdown
	ctx, stop := signal.NotifyContext(context.RootContext(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer stop()

	if err := cg.Serve(ctx); err != nil {
		log.Error("failed to shutdown http server", "error", err.Error())
	}
	log.Warnf("kill all available contexts in %s", opt.Context.ShutdownTimeout)
	context.KillContextAfterTimeout()
	log.Info("bye")
	os.Exit(0)
}
