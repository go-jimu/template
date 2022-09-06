package persistence

import (
	"fmt"
	"time"

	"github.com/go-jimu/components/logger"
	uapp "github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/domain/user"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type (
	Option struct {
		Host         string
		Port         int
		User         string
		Password     string
		Database     string
		MaxOpenConns int
	}

	Repositories struct {
		User      user.UserRepository
		QueryUser uapp.QueryUserRepository
	}
)

func NewRepositories(opt Option, log logger.Logger) *Repositories {
	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true", opt.User, opt.Password, opt.Host, opt.Port, opt.Database))
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(opt.MaxOpenConns)
	db.SetConnMaxIdleTime(60 * time.Second)

	repos := &Repositories{
		User:      newUserRepository(db, log),
		QueryUser: newQueryUserRepository(db, log),
	}
	return repos
}
