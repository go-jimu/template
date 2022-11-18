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
		log       *logger.Helper
		readModel QueryRepository
	}
)

func NewFindUserListHandler(log logger.Logger, read QueryRepository) *FindUserListHandler {
	return &FindUserListHandler{
		log:       logger.NewHelper(log),
		readModel: read,
	}
}

func (h *FindUserListHandler) Handle(ctx context.Context, req *FindUserListRequest) (*FindUserListResponse, error) {
	log := h.log.WithContext(ctx)

	log.Infof("start to handle FindUserList: name=%s, page=%d, page_size=%d", req.Name, req.Page, req.PageSize)
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	if req.Page == 0 {
		req.Page = 1
	}

	total, err := h.readModel.CountUserNumber(ctx, req.Name)
	if err != nil {
		log.Errorf("failed to count users: %s", err.Error())
		return nil, err
	}
	users, err := h.readModel.FindUserList(ctx, req.Name, req.PageSize, req.Page*(req.PageSize-1))
	if err != nil {
		log.Errorf("failed to find user: %s", err.Error())
		return nil, err
	}
	return &FindUserListResponse{Total: total, Users: users}, nil
}
