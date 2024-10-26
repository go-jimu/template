package infrastructure

import (
	"context"
	"database/sql"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
	"xorm.io/xorm"
)

type (
	userRepository struct {
		engine   *xorm.Engine
		mediator mediator.Mediator
	}

	queryUserRepository struct {
		engine *xorm.Engine
	}
)

var (
	_ domain.Repository           = (*userRepository)(nil)
	_ application.QueryRepository = (*queryUserRepository)(nil)
)

func NewRepository(engine *xorm.Engine, mediator mediator.Mediator) domain.Repository {
	return &userRepository{engine: engine, mediator: mediator}
}

func (ur *userRepository) Get(ctx context.Context, uid string) (*domain.User, error) {
	do := new(UserDO)
	has, err := ur.engine.Context(ctx).Where("id = ? AND deleted_at is null", uid).Get(do)
	if err != nil {
		return nil, oops.With("user_id", uid).Wrap(err)
	}
	if !has {
		return nil, oops.With("user_id", uid).Wrap(sql.ErrNoRows)
	}
	entity, err := convertUserDO(do)
	if err != nil {
		return nil, oops.With("user_id", uid).Wrap(err)
	}
	return entity, nil
}

func (ur *userRepository) Save(ctx context.Context, user *domain.User) error {
	data, err := convertUserToDO(user)
	if err != nil {
		return err
	}

	if user.Version == 0 {
		affected, err := ur.engine.Context(ctx).Insert(data)
		if err != nil {
			return oops.With("user_id", user.ID).Wrap(err)
		}
		if affected != 1 {
			return oops.With("user_id", user.ID).Wrap(sql.ErrNoRows)
		}
		return nil
	}

	affected, err := ur.engine.Context(ctx).Cols("name", "password", "email").Where("id = ?", user.ID).Where("deleted_at IS NULL").Update(data)
	if err != nil {
		return oops.With("user_id", user.ID).Wrap(err)
	}
	if affected == 0 {
		return oops.With("user_id", user.ID).With("version", user.Version).Errorf("failed to save user")
	}
	return nil
}

func NewQueryRepository(engine *xorm.Engine) application.QueryRepository {
	return &queryUserRepository{engine: engine}
}

func (q *queryUserRepository) CountUserNumber(ctx context.Context, name string) (int, error) {
	db := new(UserDO)
	count, err := q.engine.Context(ctx).Where("name like ? and deleted_at IS NULL", "%"+name+"%").Count(db)
	if err != nil {
		return 0, oops.With("name", name).Wrap(err)
	}
	return int(count), nil
}

func (q *queryUserRepository) FindUserList(ctx context.Context, name string, limit, offset int) ([]*application.User, error) {
	users := make([]*UserDO, 0)
	err := q.engine.Context(ctx).Where("name like ? and deleted_at IS NULL", "%"+name+"%").Limit(limit, offset).Find(&users)
	if err != nil {
		return nil, oops.With("name", name).Wrap(err)
	}

	dtos := make([]*application.User, len(users))
	for index, u := range users {
		d, err := convertUserDOToDTO(u)
		if err != nil {
			return nil, oops.With("name", name).Wrap(err)
		}
		dtos[index] = d
	}
	return dtos, nil
}
