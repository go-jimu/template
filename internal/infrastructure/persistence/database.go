package persistence

import (
	"fmt"

	"github.com/go-jimu/components/logger"
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
		db       *sqlx.DB
		log      *logger.Helper
		builders []builder
		User     user.UserRepository
	}

	builder func(*repositoryFactory)
)

var factory = new(repositoryFactory)

func BuildRepositories(opt Option, log *logger.Helper) *repositoryFactory {
	db, err := sqlx.Connect("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8", opt.User, opt.Password, opt.Host, opt.Port, opt.Database))
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(opt.MaxOpenConns)

	factory.db = db
	factory.log = log

	for _, builder := range factory.builders {
		builder(factory)
	}

	return factory
}

func init() {
	factory = &repositoryFactory{
		builders: []builder{
			userRepositoryBuilder,
		},
	}
}
