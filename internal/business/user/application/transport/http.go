package transport

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-jimu/template/internal/bootstrap/httpsrv"
	"github.com/go-jimu/template/internal/bootstrap/httpsrv/binding"
	"github.com/go-jimu/template/internal/business/user/application"
	"github.com/go-jimu/template/internal/pkg/bytesconv"
	"github.com/go-jimu/template/internal/pkg/log"
	"golang.org/x/exp/slog"
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
		{Method: http.MethodGet, Pattern: "/details/{userID}", Func: uc.GetUserByID},
		{Method: http.MethodPatch, Pattern: "/details/{userID}", Func: uc.ChangePassword},
		{Method: http.MethodPost, Pattern: "/users", Func: uc.FindUsers},
	}
}

func (uc *controller) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	logger := log.FromContext(r.Context())
	logger = logger.With(slog.String("user_id", userID))
	user, err := uc.app.Get(r.Context(), logger, userID)
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
	if err := binding.Default(r).Bind(r, command); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	command.ID = chi.URLParam(r, "userID")

	if err := uc.app.Commands.ChangePassword.Handle(r.Context(), log.FromContext(r.Context()), command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(bytesconv.StringToBytes("{}"))
}

func (uc *controller) FindUsers(w http.ResponseWriter, r *http.Request) {
	query := new(application.QueryFindUserListRequest)
	if err := binding.Default(r).Bind(r, query); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := uc.app.Queries.FindUserList.Handle(r.Context(), log.FromContext(r.Context()), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
