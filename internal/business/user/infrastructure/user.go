package infrastructure

import (
	"context"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/uptrace/bun"
)

type (
	userRepository struct {
		db       *bun.DB
		mediator mediator.Mediator
	}

	queryUserRepository struct {
		db *bun.DB
	}
)

var _ domain.Repository = (*userRepository)(nil)
var _ application.QueryRepository = (*queryUserRepository)(nil)

func NewRepository(db *bun.DB, mediator mediator.Mediator) domain.Repository {
	return &userRepository{db: db, mediator: mediator}
}

func (ur *userRepository) Get(ctx context.Context, uid string) (*domain.User, error) {
	do := new(User)
	err := ur.db.NewSelect().Model(do).Where("id = ? AND deleted = 0", uid).Limit(1).Scan(ctx)
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
		_, err := ur.db.NewInsert().Model(data).Exec(ctx)
		if err != nil {
			return err
		}
	} else {
		_, err := ur.db.NewUpdate().Model(data).Where("id = ? AND deleted = 0", data.ID).
			Column("name", "password", "email", "version").Set("version=version+1").Exec(ctx)
		if err != nil {
			return err
		}
	}
	user.Events.Raise(ur.mediator)
	return nil
}

func NewQueryRepository(db *bun.DB) application.QueryRepository {
	return &queryUserRepository{db: db}
}

func (q *queryUserRepository) CountUserNumber(ctx context.Context, name string) (int, error) {
	ret := make([]int, 1)
	err := q.db.NewRaw("select count(1) FROM user WHERE name like ? and deleted = 0", "%"+name+"%").Scan(ctx, &ret)
	if err != nil {
		return 0, err
	}
	return ret[0], nil
}

func (q *queryUserRepository) FindUserList(ctx context.Context, name string, limit, offset int) ([]*application.User, error) {
	ret := make([]*User, 0)
	err := q.db.NewRaw("select * from user where name like ? and deleted=0 order by ctime limit ? offset ?", "%"+name+"%", limit, offset).Scan(ctx, &ret)
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
