package application

import (
	"context"

	"github.com/go-jimu/components/logger"
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

func (h *FindUserListHandler) Handle(ctx context.Context, log logger.Logger, req *FindUserListRequest) (*FindUserListResponse, error) {
	helper := logger.NewHelper(log).WithContext(ctx)

	helper.Info("start to handle FindUserList: name=%s, page=%d, page_size=%d", req.Name, req.Page, req.PageSize)
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.Page == 0 {
		req.Page = 1
	}

	total, err := h.readModel.CountUserNumber(ctx, req.Name)
	if err != nil {
		helper.Error("failed to count users", "error", err.Error())
		return nil, err
	}
	users, err := h.readModel.FindUserList(ctx, req.Name, req.PageSize, req.Page*(req.PageSize-1))
	if err != nil {
		helper.Error("failed to find user", "error", err.Error())
		return nil, err
	}
	return &FindUserListResponse{Total: total, Users: users}, nil
}
