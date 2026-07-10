package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/go-jimu/components/config/loader"
	"github.com/go-jimu/components/sloghelper"
	"github.com/go-jimu/template/internal/business/notification"
	"github.com/go-jimu/template/internal/business/user"
	"github.com/go-jimu/template/internal/pkg"
	"github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/go-jimu/template/internal/pkg/eventbus"
	"github.com/go-jimu/template/internal/pkg/messagebus"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// Option aggregates component-owned runtime options for Fx injection.
type Option struct {
	fx.Out
	Logger   sloghelper.Options `json:"logger" toml:"logger" yaml:"logger"`
	MySQL    database.Option    `json:"mysql" toml:"mysql" yaml:"mysql"`
	EventBus eventbus.Config    `json:"eventbus" toml:"eventbus" yaml:"eventbus"`
	Kafka    messagebus.Option  `json:"kafka" toml:"kafka" yaml:"kafka"`
	Connect  connectrpc.Option  `json:"connect" toml:"connect" yaml:"connect"`
}

func main() {
	var option Option
	if err := loader.Load(
		&option,
		loader.WithConfigurationDirectory("./configs", "defaults"),
	); err != nil {
		slog.Error("failed to load configuration", sloghelper.Error(oops.Wrap(err)))
		os.Exit(1)
	}
	app := fx.New(
		fx.Supply(option),
		fx.Provide(sloghelper.NewLog),
		fx.Provide(eventbus.NewDispatcher),
		pkg.Module,
		user.Module,
		notification.Module,
		fx.StartTimeout(15*time.Second),
		fx.StopTimeout(30*time.Second),
	)
	slog.Info("starting application")
	app.Run()
}
