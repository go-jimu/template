package mysql

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/fx"
	"golang.org/x/exp/slog"
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

func NewMySQLDriver(lc fx.Lifecycle, opt Option, logger *slog.Logger) (*sqlx.DB, error) {
	db, err := sqlx.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true", opt.User, opt.Password, opt.Host, opt.Port, opt.Database))
	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.InfoCtx(ctx, "initiating connection to the MySQL server.", slog.Any("option", opt))
			if err := db.PingContext(ctx); err != nil {
				return err
			}

			db.SetMaxOpenConns(opt.MaxOpenConns)
			duration, err := time.ParseDuration(opt.MaxIdleTime)
			if err != nil {
				return err
			}
			db.SetConnMaxIdleTime(duration)
			return nil
		},
	})
	return db, nil
}
