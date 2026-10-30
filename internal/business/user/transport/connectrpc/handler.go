// Package connectrpc is the User context's inbound protocol transport.
package connectrpc

import (
	"context"

	connect "connectrpc.com/connect"
	userv1 "github.com/go-jimu/template/gen/user/private/v1"
	"github.com/go-jimu/template/gen/user/private/v1/userv1connect"
	"github.com/go-jimu/template/internal/business/user/application"
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
	result, err := h.application.CreateUser(ctx, application.CreateUser{
		Name:     request.Msg.GetName(),
		Password: request.Msg.GetPassword(),
		Email:    request.Msg.GetEmail(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&userv1.CreateResponse{
		User: userToProto(result),
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
	result, err := h.application.GetUser(ctx, request.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&userv1.GetResponse{
		User: readUserToProto(result),
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
	err := h.application.ChangePassword(ctx, application.ChangePassword{
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
	if request.Msg.GetPageSize() < 0 {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			oops.With("page_size", request.Msg.GetPageSize()).
				Errorf("page_size cannot be negative"),
		)
	}
	after, err := decodePageToken(request.Msg.GetPageToken())
	if err != nil {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			oops.With("field", "page_token").Errorf("page_token is malformed"),
		)
	}
	result, err := h.application.ListUsers(ctx, application.ListUsers{
		NamePrefix: request.Msg.GetNamePrefix(),
		PageSize:   int(request.Msg.GetPageSize()),
		After:      after,
	})
	if err != nil {
		return nil, mapError(err)
	}
	users := make([]*userv1.User, len(result.Users))
	for index := range result.Users {
		users[index] = readUserToProto(result.Users[index])
	}
	return connect.NewResponse(&userv1.ListResponse{
		Users:         users,
		NextPageToken: encodePageToken(result.NextCursor),
	}), nil
}
