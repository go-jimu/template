// Package connectrpc is the User context's inbound protocol transport.
package connectrpc

import (
	"context"

	"connectrpc.com/connect/v2"
	userv1 "github.com/go-jimu/template/gen/user/private/v1"
	"github.com/go-jimu/template/gen/user/private/v1/userv1connect"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/samber/oops"
)

// Handler implements the generated UserService transport contract.
type Handler struct {
	application *application.Application
	users       *query.Users
}

var _ userv1connect.UserServiceHandler = (*Handler)(nil)

// NewHandler creates the User ConnectRPC Transport adapter. Writes go to the
// Application and reads to the query side.
func NewHandler(app *application.Application, users *query.Users) userv1connect.UserServiceHandler {
	return &Handler{application: app, users: users}
}

// Create maps the protocol request to the CreateUser command.
func (h *Handler) Create(
	ctx context.Context,
	request *userv1.CreateRequest,
) (*userv1.CreateResponse, error) {
	result, err := h.application.CreateUser(ctx, application.CreateUser{
		Name:     request.GetName(),
		Password: request.GetPassword(),
		Email:    request.GetEmail(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.CreateResponse{
		User: userToProto(result),
	}, nil
}

// Get maps the protocol request to the Users.Get query.
func (h *Handler) Get(
	ctx context.Context,
	request *userv1.GetRequest,
) (*userv1.GetResponse, error) {
	if request.GetId() == "" {
		return nil, publicError(connect.CodeInvalidArgument, oops.With("field", "id").Errorf("id is required"))
	}
	result, err := h.users.Get(ctx, request.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.GetResponse{
		User: readUserToProto(result),
	}, nil
}

// ChangePassword maps the protocol request to the ChangePassword command.
func (h *Handler) ChangePassword(
	ctx context.Context,
	request *userv1.ChangePasswordRequest,
) (*userv1.ChangePasswordResponse, error) {
	if request.GetId() == "" {
		return nil, publicError(connect.CodeInvalidArgument, oops.With("field", "id").Errorf("id is required"))
	}
	err := h.application.ChangePassword(ctx, application.ChangePassword{
		UserID:          request.GetId(),
		CurrentPassword: request.GetCurrentPassword(),
		NewPassword:     request.GetNewPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &userv1.ChangePasswordResponse{}, nil
}

// List maps the protocol request to the ListUsers query.
func (h *Handler) List(
	ctx context.Context,
	request *userv1.ListRequest,
) (*userv1.ListResponse, error) {
	if request.GetPageSize() < 0 {
		return nil, publicError(connect.CodeInvalidArgument, oops.With("page_size", request.GetPageSize()).
			Errorf("page_size cannot be negative"))
	}
	after, err := decodePageToken(request.GetPageToken())
	if err != nil {
		return nil, publicError(connect.CodeInvalidArgument, oops.With("field", "page_token").Errorf("page_token is malformed"))
	}
	result, err := h.users.List(ctx, query.ListUsers{
		NamePrefix: request.GetNamePrefix(),
		PageSize:   int(request.GetPageSize()),
		After:      after,
	})
	if err != nil {
		return nil, mapError(err)
	}
	users := make([]*userv1.User, len(result.Users))
	for index := range result.Users {
		users[index] = readUserToProto(result.Users[index])
	}
	return &userv1.ListResponse{
		Users:         users,
		NextPageToken: encodePageToken(result.NextCursor),
	}, nil
}
