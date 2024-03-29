package infrastructure

import (
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/jinzhu/copier"
)

func convertUserToDO(entity *domain.User) (*User, error) {
	do := new(User)
	if err := copier.Copy(do, entity); err != nil {
		return nil, err
	}
	return do, nil
}

func convertDoUser(do *User) (*domain.User, error) {
	entity := new(domain.User)
	if err := copier.Copy(entity, do); err != nil {
		return nil, err
	}
	entity.Events = mediator.NewEventCollection()
	return entity, nil
}

func convertDoUserToDTO(do *User) (*application.User, error) {
	dto := new(application.User)
	if err := copier.Copy(dto, do); err != nil {
		return nil, err
	}
	return dto, nil
}
