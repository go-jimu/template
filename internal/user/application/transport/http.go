package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-jimu/template/internal/bootstrap/httpsrv"
	"github.com/go-jimu/template/internal/user/application"
)

type controller struct {
	app *application.Application
}

func NewController(app *application.Application) httpsrv.Controller {
	return &controller{app: app}
}

func (uc *controller) Slug() string {
	return "/api/v1/user"
}

func (un *controller) Middlewares() []httpsrv.Middleware {
	return []httpsrv.Middleware{}
}

func (uc *controller) APIs() []httpsrv.API {
	return []httpsrv.API{
		{Method: http.MethodGet, Pattern: "/{userID}", Func: uc.GetUserByID},
	}
}

func (uc *controller) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	user, err := uc.app.Get(r.Context(), userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(user)
}
