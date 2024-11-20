package infrastructure

import (
	"time"

	"github.com/go-jimu/components/mediator"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/domain"
	"github.com/go-jimu/template/internal/pkg/database"
	"github.com/jinzhu/copier"
	"github.com/samber/oops"
)

func convertUserToDO(entity *domain.User) (*UserDO, error) {
	do := new(UserDO)
	if err := copier.Copy(do, entity); err != nil {
		return nil, oops.Wrap(err)
	}
	do.UpdatedAt = database.NewTimestamp(time.Now())
	if entity.Deleted {
		do.DeletedAt = database.NewTimestamp(time.Now())
	} else {
		do.DeletedAt = database.NewTimestamp(database.UnixEpoch)
	}
	return do, nil
}

func convertUserDO(do *UserDO) (*domain.User, error) {
	entity := new(domain.User)
	if err := copier.Copy(entity, do); err != nil {
		return nil, oops.Wrap(err)
	}
	entity.Events = mediator.NewEventCollection()
	entity.CreatedAt = do.CreatedAt.Time
	entity.UpdatedAt = do.UpdatedAt.Time
	if !do.DeletedAt.Time.IsZero() {
		entity.Deleted = true
	}
	return entity, nil
}

func convertUserDOToDTO(do *UserDO) (*application.User, error) {
	dto := new(application.User)
	if err := copier.Copy(dto, do); err != nil {
		return nil, oops.Wrap(err)
	}
	return dto, nil
}
