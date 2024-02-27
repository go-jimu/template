package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
	"github.com/uptrace/bun/extra/bunslog"
	"go.uber.org/fx"
)

type Option struct {
	Host         string `json:"host" yaml:"host" toml:"host"`
	Port         int    `json:"port,string" yaml:"port" toml:"port"`
	User         string `json:"user" yaml:"user" toml:"user"`
	Password     string `json:"password" yaml:"password" toml:"password"`
	Database     string `json:"database" yaml:"database" toml:"database"`
	MaxOpenConns int    `json:"max_open_conns,string" yaml:"max_open_conns" toml:"max_open_conns"`
	MaxIdleTime  string `json:"max_idle_time" yaml:"max_idle_time" toml:"max_idle_time"`
}

func NewMySQLDriver(lc fx.Lifecycle, opt Option, logger *slog.Logger) (*bun.DB, error) {
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true", opt.User, opt.Password, opt.Host, opt.Port, opt.Database))
	if err != nil {
		return nil, err
	}

	database := bun.NewDB(db, mysqldialect.New())
	database.SetMaxOpenConns(opt.MaxOpenConns)
	duration, err := time.ParseDuration(opt.MaxIdleTime)
	if err != nil {
		return nil, err
	}
	database.SetConnMaxIdleTime(duration)

	hook := bunslog.NewQueryHook(
		bunslog.WithQueryLogLevel(slog.LevelInfo),
		bunslog.WithSlowQueryLogLevel(slog.LevelWarn),
		bunslog.WithSlowQueryThreshold(3*time.Second),
	)
	database.AddQueryHook(hook)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.InfoContext(ctx, "initiating connection to the MySQL server.", slog.Any("option", opt))
			return database.PingContext(ctx)
		},
	})
	return database, nil
}
