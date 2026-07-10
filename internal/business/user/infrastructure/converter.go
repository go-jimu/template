package infrastructure

import (
	"github.com/go-jimu/components/ddd/event"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/samber/oops"
)

func userToDO(user *domain.User) *userDO {
	return &userDO{
		ID:       user.ID,
		Name:     user.Name,
		Password: user.HashedPassword,
		Email:    user.Email,
		Version:  user.Version,
	}
}

func userFromDO(data *userDO) (*domain.User, error) {
	if data.Version < 1 {
		return nil, oops.With("version", data.Version).
			With("source", "persistence").
			Wrap(domain.ErrInvalidUser)
	}
	user := &domain.User{
		ID:             data.ID,
		Name:           data.Name,
		Email:          data.Email,
		HashedPassword: data.Password,
		Version:        data.Version,
		Events:         event.NewCollection(),
	}
	if err := user.Validate(); err != nil {
		return nil, err
	}
	return user, nil
}

func userReadModelFromDO(data *userDO) query.User {
	return query.User{ID: data.ID, Name: data.Name, Email: data.Email}
}
