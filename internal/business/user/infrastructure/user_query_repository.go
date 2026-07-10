package infrastructure

import (
	"context"

	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
	"xorm.io/xorm"
)

type userQueryRepository struct {
	engine *xorm.Engine
}

var _ query.Repository = (*userQueryRepository)(nil)

// NewUserQueryRepository creates the User product read-model Repository.
func NewUserQueryRepository(engine *xorm.Engine) query.Repository {
	return &userQueryRepository{engine: engine}
}

// Get returns one non-deleted User read model.
func (r *userQueryRepository) Get(ctx context.Context, userID string) (query.User, error) {
	data := new(userDO)
	found, err := r.engine.Context(ctx).
		Cols("id", "name", "email").
		Where("id = ? AND deleted_at = 0", userID).
		Get(data)
	if err != nil {
		return query.User{}, oops.With("operation", "user.query.get").
			With("user_id", userID).
			Wrap(oops.Wrap(err))
	}
	if !found {
		return query.User{}, oops.With("operation", "user.query.get").
			With("user_id", userID).
			Wrap(domain.ErrUserNotFound)
	}
	return userReadModelFromDO(data), nil
}

// List returns a stable, normalized page of non-deleted User read models.
func (r *userQueryRepository) List(ctx context.Context, filter query.ListFilter) (query.UserPage, error) {
	countSession := r.engine.Context(ctx).Where("deleted_at = 0")
	if filter.NamePrefix != "" {
		countSession = countSession.Where("name LIKE ?", filter.NamePrefix+"%")
	}
	total, err := countSession.Count(new(userDO))
	if err != nil {
		return query.UserPage{}, oops.With("operation", "user.query.count").
			With("name_prefix", filter.NamePrefix).
			With("page", filter.Page).
			With("page_size", filter.PageSize).
			Wrap(oops.Wrap(err))
	}

	rows := make([]userDO, 0, filter.PageSize)
	listSession := r.engine.Context(ctx).
		Cols("id", "name", "email").
		Where("deleted_at = 0")
	if filter.NamePrefix != "" {
		listSession = listSession.Where("name LIKE ?", filter.NamePrefix+"%")
	}
	offset := (filter.Page - 1) * filter.PageSize
	err = listSession.
		Desc("created_at", "id").
		Limit(filter.PageSize, offset).
		Find(&rows)
	if err != nil {
		return query.UserPage{}, oops.With("operation", "user.query.list").
			With("name_prefix", filter.NamePrefix).
			With("page", filter.Page).
			With("page_size", filter.PageSize).
			Wrap(oops.Wrap(err))
	}

	users := make([]query.User, len(rows))
	for index := range rows {
		users[index] = userReadModelFromDO(&rows[index])
	}
	return query.UserPage{
		Users:    users,
		Total:    int(total),
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}
