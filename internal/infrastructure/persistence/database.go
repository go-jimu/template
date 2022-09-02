package persistence

import (
	"fmt"

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

	repositoryFactory struct {
		builders []builder
	}

	builder func(*sqlx.DB, *logger.Helper, *Repositories)

	Repositories struct {
		User      user.UserRepository
		QueryUser uapp.QueryUserRepository
	}
)

var factory = new(repositoryFactory)

func BuildRepositories(opt Option, log *logger.Helper) *Repositories {
	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true", opt.User, opt.Password, opt.Host, opt.Port, opt.Database))
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(opt.MaxOpenConns)

	repos := new(Repositories)
	for _, builder := range factory.builders {
		builder(db, log, repos)
	}
	return repos
}

func init() {
	factory = &repositoryFactory{
		builders: []builder{
			userRepositoryBuilder,
			queryUserRepositoryBuilder,
		},
	}
}
