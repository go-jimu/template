// Package app composes the server: it loads the aggregate configuration and
// lists the modules Fx runs, in start order. cmd/server only calls it, so the
// composition is tested here.
package app

import (
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

// configDirectory is relative to the repository root, where the server runs.
const configDirectory = "./configs/server"

// LoadOption loads configs/server/defaults.yml, with its environment
// placeholders resolved. Each component validates its own part when Fx
// constructs it.
func LoadOption() (Option, error) {
	var option Option
	err := loader.Load(
		&option,
		loader.WithConfigurationDirectory(configDirectory, "defaults"),
	)
	return option, err
}

// New returns the server application for fx.New.
func New(option Option) fx.Option {
	return application(option, pkg.Module)
}

// application lists the modules in start order. infrastructure provides the
// shared runtime adapters, pkg.Module in production. pkg.Lifecycle comes last
// so it can order the runtimes the bounded contexts registered with.
func application(option Option, infrastructure fx.Option) fx.Option {
	return fx.Options(
		fx.Supply(option),
		fx.Provide(sloghelper.NewLog),
		infrastructure,
		user.Module,
		notification.Module,
		pkg.Lifecycle,
		fx.StartTimeout(15*time.Second),
		fx.StopTimeout(30*time.Second),
	)
}
