package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/go-jimu/components/config/loader"
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/user"
	"github.com/go-jimu/template/internal/pkg"
	"github.com/go-jimu/template/internal/pkg/httpsrv"
	"github.com/go-jimu/template/internal/pkg/mysql"
	"go.uber.org/fx"
)

type Option struct {
	fx.Out
	Logger     sloghelper.Options `json:"logger" toml:"logger" yaml:"logger"`
	MySQL      mysql.Option       `json:"mysql" toml:"mysql" yaml:"mysql"`
	HTTPServer httpsrv.Option     `json:"http-server" toml:"http-server" yaml:"http-server"`
	Eventbus   mediator.Options   `json:"eventbus" toml:"eventbus" yaml:"eventbus"`
}

func parseOption() (Option, error) {
	opt := new(Option)
	err := loader.Load(opt)
	return *opt, err
}

func main() {
	app := fx.New(
		fx.Provide(parseOption),
		fx.Provide(sloghelper.NewLog),
		fx.Provide(mediator.NewInMemMediator),
		fx.Invoke(mediator.SetDefault),
		pkg.Module,
		user.Module,
		fx.NopLogger,
	)
	startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := app.Start(startCtx); err != nil {
		slog.ErrorContext(startCtx, "failed to start application", sloghelper.Error(err))
		os.Exit(1)
	}

	<-app.Done()
	slog.Warn("caught quit signal")

	stopCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.Stop(stopCtx); err != nil {
		slog.Error("failed to stop application", sloghelper.Error(err))
		os.Exit(1)
	}

	slog.Info("bye")
	os.Exit(0)
}
