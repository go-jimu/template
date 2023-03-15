package mysql

import (
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
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

func NewMySQLDriver(opt Option) *sqlx.DB {
	db, err := sqlx.Connect(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true", opt.User, opt.Password, opt.Host, opt.Port, opt.Database))
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(opt.MaxOpenConns)
	duration, err := time.ParseDuration(opt.MaxIdleTime)
	if err != nil {
		panic(err)
	}
	db.SetConnMaxIdleTime(duration)
	return db
}
