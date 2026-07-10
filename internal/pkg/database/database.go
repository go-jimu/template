package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/samber/oops"
	"go.uber.org/fx"
	"xorm.io/xorm"
)

// Option contains MySQL connection-pool settings.
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

// Validate checks the MySQL component's own configuration.
func (o Option) Validate() error {
	if o.Host == "" {
		return errors.New("mysql host is required")
	}
	if o.Port < 1 || o.Port > 65535 {
		return fmt.Errorf("mysql port %d is outside 1..65535", o.Port)
	}
	if o.User == "" {
		return errors.New("mysql user is required")
	}
	if o.Database == "" {
		return errors.New("mysql database is required")
	}
	if o.MaxOpenConns < 0 || o.MaxIdleConns < 0 {
		return errors.New("mysql connection limits cannot be negative")
	}
	if o.MaxOpenConns > 0 && o.MaxIdleConns > o.MaxOpenConns {
		return errors.New("mysql max_idle_conns cannot exceed max_open_conns")
	}
	if _, err := time.ParseDuration(o.MaxIdleTime); err != nil {
		return oops.With("option", "max_idle_time").
			With("value", o.MaxIdleTime).
			Wrap(oops.Wrap(err))
	}
	if _, err := time.ParseDuration(o.MaxLifetime); err != nil {
		return oops.With("option", "max_lifetime").
			With("value", o.MaxLifetime).
			Wrap(oops.Wrap(err))
	}
	return nil
}

// NewMySQLDriver creates and lifecycle-manages the shared xorm engine.
func NewMySQLDriver(
	lifecycle fx.Lifecycle,
	option Option,
	logger *slog.Logger,
) (*xorm.Engine, error) {
	if err := option.Validate(); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=UTC",
		option.User,
		option.Password,
		option.Host,
		option.Port,
		option.Database,
	)
	engine, err := xorm.NewEngine("mysql", dsn)
	if err != nil {
		return nil, oops.With("host", option.Host).
			With("database", option.Database).
			Wrap(oops.Wrap(err))
	}
	engine.SetLogger(NewXormSlog(logger))
	engine.ShowSQL(false)
	engine.SetMaxIdleConns(option.MaxIdleConns)
	engine.SetMaxOpenConns(option.MaxOpenConns)

	maxIdleTime, err := time.ParseDuration(option.MaxIdleTime)
	if err != nil {
		_ = engine.Close()
		return nil, oops.With("option", "max_idle_time").
			With("value", option.MaxIdleTime).
			Wrap(oops.Wrap(err))
	}
	maxLifetime, err := time.ParseDuration(option.MaxLifetime)
	if err != nil {
		_ = engine.Close()
		return nil, oops.With("option", "max_lifetime").
			With("value", option.MaxLifetime).
			Wrap(oops.Wrap(err))
	}
	engine.SetConnMaxIdleTime(maxIdleTime)
	engine.SetConnMaxLifetime(maxLifetime)

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			pingContext, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if err := engine.PingContext(pingContext); err != nil {
				_ = engine.Close()
				return oops.With("host", option.Host).
					With("port", option.Port).
					With("database", option.Database).
					Wrap(oops.Wrap(err))
			}
			logger.InfoContext(
				ctx,
				"connected to database",
				slog.String("host", option.Host),
				slog.Int("port", option.Port),
				slog.String("database", option.Database),
			)
			return nil
		},
		OnStop: func(context.Context) error {
			return oops.Wrap(engine.Close())
		},
	})
	return engine, nil
}
