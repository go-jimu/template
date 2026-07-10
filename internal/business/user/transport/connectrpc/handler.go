// Package connectrpc is the User context's inbound protocol transport.
package connectrpc

import (
	"context"

	connect "connectrpc.com/connect"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/command"
	"github.com/go-jimu/template/internal/business/user/application/query"
	userv1 "github.com/go-jimu/template/pkg/gen/user/v1"
	"github.com/go-jimu/template/pkg/gen/user/v1/userv1connect"
	"github.com/samber/oops"
)

// Handler implements the generated UserService transport contract.
type Handler struct {
	application *application.Application
}

var _ userv1connect.UserServiceHandler = (*Handler)(nil)

// NewHandler creates the User ConnectRPC Transport adapter.
func NewHandler(app *application.Application) userv1connect.UserServiceHandler {
	return &Handler{application: app}
}

// Create maps the protocol request to the CreateUser command.
func (h *Handler) Create(
	ctx context.Context,
	request *connect.Request[userv1.CreateRequest],
) (*connect.Response[userv1.CreateResponse], error) {
	result, err := h.application.Commands.Create.Handle(ctx, command.CreateUser{
		Name:     request.Msg.GetName(),
		Password: request.Msg.GetPassword(),
		Email:    request.Msg.GetEmail(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&userv1.CreateResponse{
		User: createdUserToProto(result),
	}), nil
}

// Get maps the protocol request to the GetUser query.
func (h *Handler) Get(
	ctx context.Context,
	request *connect.Request[userv1.GetRequest],
) (*connect.Response[userv1.GetResponse], error) {
	if request.Msg.GetId() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			oops.With("field", "id").Errorf("id is required"),
		)
	}
	result, err := h.application.Queries.Get.Handle(ctx, request.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&userv1.GetResponse{
		User: queryUserToProto(result),
	}), nil
}

// ChangePassword maps the protocol request to the ChangePassword command.
func (h *Handler) ChangePassword(
	ctx context.Context,
	request *connect.Request[userv1.ChangePasswordRequest],
) (*connect.Response[userv1.ChangePasswordResponse], error) {
	if request.Msg.GetId() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			oops.With("field", "id").Errorf("id is required"),
		)
	}
	err := h.application.Commands.ChangePassword.Handle(ctx, command.ChangePassword{
		UserID:          request.Msg.GetId(),
		CurrentPassword: request.Msg.GetCurrentPassword(),
		NewPassword:     request.Msg.GetNewPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&userv1.ChangePasswordResponse{}), nil
}

// List maps the protocol request to the ListUsers query.
func (h *Handler) List(
	ctx context.Context,
	request *connect.Request[userv1.ListRequest],
) (*connect.Response[userv1.ListResponse], error) {
	if request.Msg.GetPage() < 0 || request.Msg.GetPageSize() < 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			oops.With("page", request.Msg.GetPage()).
				With("page_size", request.Msg.GetPageSize()).
				Errorf("page and page_size cannot be negative"),
		)
	}
	result, err := h.application.Queries.List.Handle(ctx, query.ListUsers{
		NamePrefix: request.Msg.GetNamePrefix(),
		Page:       int(request.Msg.GetPage()),
		PageSize:   int(request.Msg.GetPageSize()),
	})
	if err != nil {
		return nil, mapError(err)
	}
	users := make([]*userv1.User, len(result.Users))
	for index := range result.Users {
		users[index] = queryUserToProto(result.Users[index])
	}
	return connect.NewResponse(&userv1.ListResponse{
		Users:    users,
		Total:    int64(result.Total),
		Page:     int32(result.Page),
		PageSize: int32(result.PageSize),
	}), nil
}
