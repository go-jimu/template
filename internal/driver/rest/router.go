package rest

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-jimu/components/logger"
)

type (
	Option struct {
		Addr string `json:"addr" yaml:"addr" toml:"addr"`
	}

	API struct {
		Pattern string
		Method  string
		Func    http.HandlerFunc
	}

	Controller interface {
		Slug() string
		Middlewares() chi.Middlewares
		APIs() []API
	}

	group struct {
		router *chi.Mux
		option Option
		logger logger.Logger
	}

	ControllerGroup interface {
		With(Controller)
		Server() *http.Server
	}
)

func NewControllerGroup(opt Option, log logger.Logger, cgs ...Controller) ControllerGroup {
	g := &group{
		router: chi.NewRouter(),
		option: opt,
		logger: log,
	}

	g.router.Use(
		InjectContext,
		middleware.RequestID,
		middleware.RealIP,
		RequestLog(log),
		middleware.Recoverer,
	)
	g.router.Use(middleware.Heartbeat("/ping"))

	for _, controller := range cgs {
		g.With(controller)
	}
	return g
}

func (g *group) With(c Controller) {
	g.router.Route(c.Slug(), func(r chi.Router) {
		for _, mid := range c.Middlewares() {
			r.Use(mid)
		}

		for _, api := range c.APIs() {
			r.Method(api.Method, api.Pattern, api.Func)
		}
	})
}

func (g *group) Server() *http.Server {
	return &http.Server{
		Addr:    g.option.Addr,
		Handler: g.router,
	}
}
