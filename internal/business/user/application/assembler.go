package application

import (
	userv1 "github.com/go-jimu/template/gen/user/v1"
	"github.com/go-jimu/template/internal/business/user/domain"
)

func assembleDomainUser(entity *domain.User) *userv1.GetResponse {
	return &userv1.GetResponse{
		Id:    entity.ID,
		Name:  entity.Name,
		Email: entity.Email,
	}
}
