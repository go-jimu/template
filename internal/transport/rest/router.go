package rest

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/template/internal/application/user"
	"github.com/go-jimu/template/internal/transport/rest/api"
)

type Option struct {
	Addr string
}

func NewRouter(app *user.UserApplication) http.Handler {
	router := chi.NewRouter()

	router.Use(
		InjectContext,
		middleware.RequestID,
		middleware.RealIP,
		middleware.Recoverer,
		middleware.Timeout(3*time.Second),
	)

	router.Use(middleware.Heartbeat("/ping"))

	{
		u := api.NewUserController(app)
		router.Get("/api/user/{userID}", u.GetUserByID)
	}

	return router
}
