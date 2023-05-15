package infrastructure

import (
	"context"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/jmoiron/sqlx"
)

type (
	userRepository struct {
		db       *sqlx.DB
		mediator mediator.Mediator
	}

	queryUserRepository struct {
		db *sqlx.DB
	}
)

func NewRepository(db *sqlx.DB, mediator mediator.Mediator) domain.Repository {
	return &userRepository{db: db, mediator: mediator}
}

func (ur *userRepository) Get(ctx context.Context, uid string) (*domain.User, error) {
	do := new(User)
	err := ur.db.GetContext(ctx, do, "select * from user where id=? and deleted=0 limit 1", uid)
	if err != nil {
		return nil, err
	}
	entity, err := convertDoUser(do)
	if err != nil {
		return nil, err
	}
	return entity, nil
}

func (ur *userRepository) Save(ctx context.Context, user *domain.User) error {
	data, err := convertUserToDO(user)
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
	user.Events.Raise(ur.mediator)
	return nil
}

func NewQueryRepository(db *sqlx.DB) application.QueryRepository {
	return &queryUserRepository{db: db}
}

func (q *queryUserRepository) CountUserNumber(ctx context.Context, name string) (int, error) {
	ret := make([]int, 1)
	err := q.db.SelectContext(ctx, &ret, "select count(1) from user where name like ? and deleted=0 ;", "%"+name+"%")
	if err != nil {
		return 0, err
	}
	return ret[0], nil
}

func (q *queryUserRepository) FindUserList(ctx context.Context, name string, limit, offset int) ([]*application.User, error) {
	ret := make([]*User, 0)
	err := q.db.SelectContext(ctx, &ret, "select * from user where name like ? and deleted=0 order by ctime limit ? offset ?", "%"+name+"%", limit, offset)
	if err != nil {
		return nil, err
	}
	dtos := make([]*application.User, len(ret))
	for index, u := range ret {
		d, err := convertDoUserToDTO(u)
		if err != nil {
			return nil, err
		}
		dtos[index] = d
	}
	return dtos, nil
}
