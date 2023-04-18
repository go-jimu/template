package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/bootstrap/httpsrv"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/pkg/bytesconv"
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
	log := logger.With(logger.FromContext(r.Context()), "user_id", userID)
	user, err := uc.app.Get(r.Context(), log, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(bytesconv.StringToBytes(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(user)
}

func (uc *controller) ChangePassword(w http.ResponseWriter, r *http.Request) {
	command := new(application.CommandChangePassword)
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(command); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := uc.app.Commands.ChangePassword.Handle(r.Context(), logger.FromContext(r.Context()), command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(bytesconv.StringToBytes("{}"))
}
