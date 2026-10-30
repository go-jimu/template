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
			Wrap(err)
	}
	if !found {
		return query.User{}, oops.With("operation", "user.query.get").
			With("user_id", userID).
			Wrap(domain.ErrUserNotFound)
	}
	return userReadModelFromDO(data), nil
}

// List returns one keyset page of non-deleted User read models, newest first.
// UUIDv7 IDs sort in creation order, so the ID alone is a stable cursor.
func (r *userQueryRepository) List(ctx context.Context, filter query.ListFilter) (query.UserPage, error) {
	session := r.engine.Context(ctx).
		Cols("id", "name", "email").
		Where("deleted_at = 0")
	if filter.NamePrefix != "" {
		session = session.Where("name LIKE ?", filter.NamePrefix+"%")
	}
	if filter.After != "" {
		session = session.Where("id < ?", filter.After)
	}
	rows := make([]userDO, 0, filter.Limit+1)
	if err := session.Desc("id").Limit(filter.Limit + 1).Find(&rows); err != nil {
		return query.UserPage{}, oops.With("operation", "user.query.list").
			With("name_prefix", filter.NamePrefix).
			With("after", filter.After).
			With("limit", filter.Limit).
			Wrap(err)
	}

	page := query.UserPage{}
	if len(rows) > filter.Limit {
		rows = rows[:filter.Limit]
		page.NextCursor = rows[len(rows)-1].ID
	}
	page.Users = make([]query.User, len(rows))
	for index := range rows {
		page.Users[index] = userReadModelFromDO(&rows[index])
	}
	return page, nil
}
