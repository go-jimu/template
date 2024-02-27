package application

import (
	"context"
	"log/slog"

	"github.com/go-jimu/components/sloghelper"
)

type (
	QueryRepository interface {
		FindUserList(ctx context.Context, name string, limit, offset int) ([]*User, error)
		CountUserNumber(context.Context, string) (int, error)
	}

	FindUserListHandler struct {
		readModel QueryRepository
	}
)

func NewFindUserListHandler(read QueryRepository) *FindUserListHandler {
	return &FindUserListHandler{
		readModel: read,
	}
}

func (h *FindUserListHandler) Handle(ctx context.Context, logger *slog.Logger, req *QueryFindUserListRequest) (*QueryFindUserListResponse, error) {
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}
	if req.Page == 0 {
		req.Page = 1
	}

	total, err := h.readModel.CountUserNumber(ctx, req.Name)
	if err != nil {
		logger.ErrorContext(ctx, "failed to count users", sloghelper.Error(err))
		return nil, err
	}
	users, err := h.readModel.FindUserList(ctx, req.Name, req.PageSize, req.Page*(req.PageSize-1))
	if err != nil {
		logger.ErrorContext(ctx, "failed to find user", sloghelper.Error(err))
		return nil, err
	}
	return &QueryFindUserListResponse{Total: total, Users: users}, nil
}
