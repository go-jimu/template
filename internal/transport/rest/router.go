package rest

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/components/logger"
	"github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/transport/rest/api"
)

type Option struct {
	Addr string
}

func NewServer(opt Option, log logger.Logger, app *user.UserApplication) *http.Server {
	router := chi.NewRouter()

	router.Use(
		InjectContext,
		middleware.RequestID,
		middleware.RealIP,
		RequestLog(log),
		middleware.Recoverer,
		middleware.Timeout(3*time.Second),
	)
	router.Use(middleware.Heartbeat("/ping"))

	{
		u := api.NewUserController(app)
		router.Get("/api/user/{userID}", u.GetUserByID)
	}

	return &http.Server{
		Addr:    opt.Addr,
		Handler: router,
	}
}
