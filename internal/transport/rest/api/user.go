package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/pkg/context"
)

type UserController struct {
	app *user.UserApplication
}

func NewUserController() *UserController {
	return &UserController{}
}

func (uc *UserController) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	ctx, _ := context.GenDefaultContext()
	user, err := uc.app.Get(ctx, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(user)
}
