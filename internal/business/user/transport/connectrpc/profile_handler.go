package connectrpc

import (
	"context"

	connect "connectrpc.com/connect"
	userpublicv1 "github.com/go-jimu/template/gen/user/public/v1"
	"github.com/go-jimu/template/gen/user/public/v1/userpublicv1connect"
	"github.com/go-jimu/template/internal/business/user/application/query"
	"github.com/samber/oops"
)

// ProfileHandler implements the public ProfileService contract for web and
// app clients. It reads through the same query side as the private
// UserService and maps the result to the public view, which leaves out the
// email.
type ProfileHandler struct {
	users *query.Users
}

var _ userpublicv1connect.ProfileServiceHandler = (*ProfileHandler)(nil)

// NewProfileHandler creates the public User ConnectRPC Transport adapter.
func NewProfileHandler(users *query.Users) userpublicv1connect.ProfileServiceHandler {
	return &ProfileHandler{users: users}
}

// GetProfile maps the protocol request to the Users.Get query.
func (h *ProfileHandler) GetProfile(
	ctx context.Context,
	request *connect.Request[userpublicv1.GetProfileRequest],
) (*connect.Response[userpublicv1.GetProfileResponse], error) {
	if request.Msg.GetId() == "" {
		return nil, connect.NewError(
			connect.CodeInvalidArgument,
			oops.With("field", "id").Errorf("id is required"),
		)
	}
	result, err := h.users.Get(ctx, request.Msg.GetId())
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&userpublicv1.GetProfileResponse{
		Profile: readUserToProfile(result),
	}), nil
}
