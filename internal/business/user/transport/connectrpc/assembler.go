package connectrpc

import (
	"github.com/go-jimu/template/internal/business/user/application/command"
	"github.com/go-jimu/template/internal/business/user/application/query"
	userv1 "github.com/go-jimu/template/pkg/gen/user/v1"
)

func createdUserToProto(user command.CreatedUser) *userv1.User {
	return &userv1.User{Id: user.ID, Name: user.Name, Email: user.Email}
}

func queryUserToProto(user query.User) *userv1.User {
	return &userv1.User{Id: user.ID, Name: user.Name, Email: user.Email}
}
