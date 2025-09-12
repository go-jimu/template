package application

import (
	"github.com/go-jimu/template/internal/business/user/domain"
	userv1 "github.com/go-jimu/template/pkg/gen/user/v1"
)

func assembleDomainUser(entity *domain.User) *userv1.GetResponse {
	return &userv1.GetResponse{
		Id:    entity.ID,
		Name:  entity.Name,
		Email: entity.Email,
	}
}
