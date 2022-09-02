package persistence

import (
	"context"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/domain/user"
	"github.com/go-jimu/template/internal/eventbus"
	"github.com/go-jimu/template/internal/infrastructure/converter"
	"github.com/go-jimu/template/internal/infrastructure/do"
	"github.com/jmoiron/sqlx"
)

type userRepository struct {
	log *logger.Helper
	db  *sqlx.DB
}

var _ user.UserRepository = (*userRepository)(nil)

func userRepositoryBuilder(conn *sqlx.DB, log *logger.Helper, repos *Repositories) {
	repo := newUserRepository(conn, log)
	repos.User = repo
}

func newUserRepository(db *sqlx.DB, log *logger.Helper) user.UserRepository {
	return &userRepository{db: db, log: log}
}

func (ur *userRepository) Get(ctx context.Context, uid string) (*user.User, error) {
	do := new(do.User)
	err := ur.db.GetContext(ctx, do, "select * from user where id=? and deleted=0 limit 1", uid)
	if err != nil {
		return nil, err
	}
	entity, err := converter.ConvertDoUser(do)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (ur *userRepository) Save(ctx context.Context, user *user.User) error {
	data, err := converter.ConvertEntityUser(user)
	if err != nil {
		return err
	}

	if user.Version == 0 {
		if _, err := ur.db.NamedExecContext(ctx,
			"INSERT INTO user (id, name, password, email, version) VALUES (:id, :name, :password, :email, 1)", data); err != nil {
			return err
		}
	} else {
		if _, err := ur.db.NamedExecContext(ctx,
			"UPDATE user SET name=:name, password=:password, email=:email, version=version+1 where id=:id and deleted=0 and version=:version", data); err != nil {
			return err
		}
	}
	user.Events.Raise(ctx, eventbus.Default())
	return nil
}
