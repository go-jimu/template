package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/samber/oops"
	"go.uber.org/fx"
	"xorm.io/xorm"
)

type Option struct {
	Host         string `json:"host" yaml:"host" toml:"host"`
	Port         int    `json:"port,string" yaml:"port" toml:"port"`
	User         string `json:"user" yaml:"user" toml:"user"`
	Password     string `json:"password" yaml:"password" toml:"password"`
	Database     string `json:"database" yaml:"database" toml:"database"`
	MaxOpenConns int    `json:"max_open_conns" yaml:"max_open_conns" toml:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns" yaml:"max_idle_conns" toml:"max_idle_conns"`
	MaxIdleTime  string `json:"max_idle_time" yaml:"max_idle_time" toml:"max_idle_time"`
	MaxLifetime  string `json:"max_lifetime" yaml:"max_lifetime" toml:"max_lifetime"`
}

func NewMySQLDriver(lc fx.Lifecycle, opt Option, logger *slog.Logger) (*xorm.Engine, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Local", opt.User, opt.Password, opt.Host, opt.Port, opt.Database)
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		return nil, err
	}
	engine.SetLogger(NewXormSlog(logger))
	engine.ShowSQL(false)
	engine.SetMaxIdleConns(opt.MaxIdleConns)
	engine.SetMaxOpenConns(opt.MaxOpenConns)
	if duration, err := time.ParseDuration(opt.MaxIdleTime); err == nil {
		engine.SetConnMaxIdleTime(duration)
	}
	if duration, err := time.ParseDuration(opt.MaxLifetime); err == nil {
		engine.SetConnMaxLifetime(duration)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := engine.PingContext(ctx); err != nil {
				return oops.With("dsn", dsn).Wrap(err)
			}
			logger.InfoContext(ctx, "connected to database", slog.Any("option", opt))
			return nil
		},
	})
	return engine, nil
}
