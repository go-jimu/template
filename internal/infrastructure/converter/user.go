package converter

import (
	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/application/dto"
	"github.com/go-jimu/template/internal/domain/user"
	"github.com/go-jimu/template/internal/infrastructure/do"
	"github.com/jinzhu/copier"
)

func ConvertUserToDO(entity *user.User) (*do.User, error) {
	do := new(do.User)
	if err := copier.Copy(do, entity); err != nil {
		return nil, err
	}
	return do, nil
}

func ConvertDoUser(do *do.User) (*user.User, error) {
	entity := new(user.User)
	if err := copier.Copy(entity, do); err != nil {
		return nil, err
	}
	entity.Events = mediator.NewEventCollection()
	return entity, nil
}

func ConvertDoUserToDTO(do *do.User) (*dto.User, error) {
	dto := new(dto.User)
	if err := copier.Copy(dto, do); err != nil {
		return nil, err
	}
	return dto, nil
}
