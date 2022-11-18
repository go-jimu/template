package application

type (
	User struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	CommandChangePassword struct {
		ID          string `json:"id"`
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	FindUserListRequest struct {
		Name     string `json:"name"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}

	FindUserListResponse struct {
		Total int     `json:"total"`
		Users []*User `json:"users"`
	}
)
