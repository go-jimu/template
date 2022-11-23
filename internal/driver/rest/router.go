package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-jimu/components/logger"
	internalCtx "github.com/go-jimu/template/internal/pkg/context"
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

	MiddlewareScope int

	Middleware struct {
		Middleware func(http.Handler) http.Handler
		Scope      MiddlewareScope
	}

	Controller interface {
		Slug() string
		Middlewares() []Middleware
		APIs() []API
	}

	HTTPServer interface {
		With(Controller)
		Service(context.Context) error
	}

	group struct {
		router      *chi.Mux
		option      Option
		logger      *logger.Helper
		root        Controller
		controllers []Controller
	}
)

const (
	ScopeController MiddlewareScope = iota // controler 层面
	ScopeGlobal                            // 全局中间件
)

var readTimeout = 3 * time.Second

func NewHTTPServer(opt Option, log logger.Logger, cs ...Controller) HTTPServer {
	g := &group{
		router:      chi.NewRouter(),
		option:      opt,
		logger:      logger.NewHelper(log),
		root:        &rootController{logger: log},
		controllers: make([]Controller, 0),
	}

	for _, controller := range cs {
		g.With(controller)
	}
	return g
}

func (g *group) With(c Controller) {
	g.controllers = append(g.controllers, c)
}

// chi: all middlewares must be defined before routes on a mux
func (g *group) lazyLoad() {
	// apply global middlewares
	if g.root != nil {
		for _, middleware := range g.root.Middlewares() {
			g.router.Use(middleware.Middleware)
		}
	}

	for _, controller := range g.controllers {
		for _, middleware := range controller.Middlewares() {
			if middleware.Scope == ScopeGlobal {
				g.router.Use(middleware.Middleware)
			}
		}
	}

	for _, api := range g.root.APIs() {
		g.router.Method(api.Method, api.Pattern, api.Func)
	}

	// each child controller
	for _, controller := range g.controllers {
		g.router.Route(controller.Slug(), func(r chi.Router) {
			for _, middleware := range controller.Middlewares() {
				if middleware.Scope != ScopeGlobal {
					r.Use(middleware.Middleware)
				}
			}

			for _, api := range controller.APIs() {
				r.Method(api.Method, api.Pattern, api.Func)
			}
		})
	}
}

func (g *group) Service(ctx context.Context) error {
	g.lazyLoad()

	srv := &http.Server{
		Addr:              g.option.Addr,
		Handler:           g.router,
		ReadHeaderTimeout: readTimeout, // https://cwe.mitre.org/data/definitions/400.html
	}
	internalErr := make(chan error, 1)
	defer close(internalErr)

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			internalErr <- err
		}
	}()

	var err error
	select {
	case <-ctx.Done():
		g.logger.Warnf("caught quit signal")
	case err = <-internalErr:
		g.logger.Errorf("an unknown error occurred in http server: %s", err.Error())
	}

	ctx, cancel := internalCtx.GenDefaultContext()
	defer cancel()
	g.logger.Warnf("try to shutdown http server")
	return srv.Shutdown(ctx)
}
