package user

import (
	"context"

	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/application/dto"
)

type (
	QueryUserRepository interface {
		FindUserList(ctx context.Context, name string, limit, offset int) ([]*dto.User, error)
		CountUserNumber(context.Context, string) (int, error)
	}

	FindUserListRequest struct {
		Name     string `json:"name"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}

	FindUserListResponse struct {
		Total int         `json:"total"`
		Users []*dto.User `json:"users"`
	}

	FindUserListHandler struct {
		log       *logger.Helper
		readModel QueryUserRepository
	}
)

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
