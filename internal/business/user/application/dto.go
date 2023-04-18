package application

type (
	User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	CommandChangePassword struct {
		ID          string `json:"_"`
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	QueryFindUserListRequest struct {
		Name     string `form:"name"`
		Page     int    `form:"page"`
		PageSize int    `form:"page_size"`
	}

	QueryFindUserListResponse struct {
		Total int     `json:"total"`
		Users []*User `json:"users"`
	}
)
