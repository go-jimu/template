package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	jimuxorm "github.com/go-jimu/contrib/transaction/xorm"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/samber/oops"
)

type userRepository struct {
	db *jimuxorm.DB
}

var _ domain.Repository = (*userRepository)(nil)

// NewUserRepository creates the xorm-backed User Aggregate Repository. Its
// statements run in the use case's transaction when there is one.
func NewUserRepository(db *jimuxorm.DB) domain.Repository {
	return &userRepository{db: db}
}

// Get loads one non-deleted User Aggregate.
func (r *userRepository) Get(ctx context.Context, userID string) (*domain.User, error) {
	session, err := r.db.Session(ctx)
	if err != nil {
		return nil, oops.With("operation", "user.get").With("user_id", userID).Wrap(err)
	}
	data := new(userDO)
	found, err := session.
		Cols("id", "name", "password", "email", "version", "created_at", "updated_at", "deleted_at").
		Where("id = ? AND deleted_at = 0", userID).
		Get(data)
	if err != nil {
		return nil, oops.With("operation", "user.get").
			With("user_id", userID).
			Wrap(err)
	}
	if !found {
		return nil, oops.With("operation", "user.get").
			With("user_id", userID).
			Wrap(domain.ErrUserNotFound)
	}
	user, err := userFromDO(data)
	if err != nil {
		return nil, oops.With("operation", "user.rehydrate").
			With("user_id", userID).
			Wrap(err)
	}
	return user, nil
}

// Save inserts a new User or updates an existing User with optimistic locking.
func (r *userRepository) Save(ctx context.Context, user *domain.User) error {
	if user == nil {
		return oops.With("operation", "user.save").Wrap(errors.New("nil aggregate"))
	}
	session, err := r.db.Session(ctx)
	if err != nil {
		return oops.With("operation", "user.save").With("user_id", user.ID).Wrap(err)
	}
	data := userToDO(user)
	now := time.Now().UTC()

	if user.Version == 0 {
		data.Version = 1
		data.CreatedAt = database.NewTimestamp(now)
		data.UpdatedAt = database.NewTimestamp(now)
		data.DeletedAt = database.NewTimestamp(database.UnixEpoch)
		affected, err := session.Insert(data)
		if err != nil {
			return oops.With("operation", "user.insert").
				With("user_id", user.ID).
				Wrap(err)
		}
		if affected != 1 {
			return oops.With("operation", "user.insert").
				With("user_id", user.ID).
				Wrap(sql.ErrNoRows)
		}
		return nil
	}

	data.UpdatedAt = database.NewTimestamp(now)
	affected, err := session.
		Cols("name", "password", "email", "updated_at").
		Incr("version").
		Where("id = ? AND version = ? AND deleted_at = 0", user.ID, user.Version).
		Update(data)
	if err != nil {
		return oops.With("operation", "user.update").
			With("user_id", user.ID).
			With("version", user.Version).
			Wrap(err)
	}
	if affected != 1 {
		return oops.With("operation", "user.update").
			With("user_id", user.ID).
			With("version", user.Version).
			Wrap(domain.ErrConcurrentModification)
	}
	return nil
}
