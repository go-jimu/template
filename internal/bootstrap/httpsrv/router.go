package httpsrv

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-jimu/components/sloghelper"
	"go.uber.org/fx"
	"golang.org/x/exp/slog"
)

type (
	Option struct {
		Addr     string `json:"addr" yaml:"addr" toml:"addr"`
		CertFile string `json:"cert_file" toml:"cert_file" yaml:"cert_file"`
		KeyFile  string `json:"key_file" toml:"key_file" yaml:"key_file"`
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
		Serve() error
	}

	router struct {
		logger      *slog.Logger
		router      *chi.Mux
		option      Option
		root        Controller
		controllers []Controller
		server      *http.Server
	}
)

const (
	ScopeController MiddlewareScope = iota // controller 层面
	ScopeGlobal                            // 全局中间件
)

var readTimeout = 3 * time.Second

func NewHTTPServer(lc fx.Lifecycle, opt Option, logger *slog.Logger, cs ...Controller) HTTPServer {
	slog.Info("create a new HTTP server", slog.Any("option", opt))
	g := &router{
		logger:      logger,
		router:      chi.NewRouter(),
		option:      opt,
		root:        newRootController(),
		controllers: make([]Controller, 0),
	}

	for _, controller := range cs {
		g.With(controller)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return g.Serve()
		},

		OnStop: func(ctx context.Context) error {
			return g.server.Shutdown(ctx)
		},
	})

	return g
}

func (g *router) With(c Controller) {
	g.controllers = append(g.controllers, c)
	g.logger.Info("a new controller is append to HTTP server", slog.String("slug", c.Slug()))
}

// chi: all middlewares must be defined before routes on a mux
func (g *router) lazyLoad() {
	// apply global middlewares
	if g.root != nil {
		for _, middleware := range g.root.Middlewares() {
			g.router.Use(middleware.Middleware)
		}
	}

	// chi: all middlewares must be defined before routes on a mux
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
				if middleware.Scope == ScopeController {
					r.Use(middleware.Middleware)
				}
			}

			for _, api := range controller.APIs() {
				r.Method(api.Method, api.Pattern, api.Func)
			}
		})
	}
}

func (g *router) Serve() error {
	g.lazyLoad()

	ln, err := net.Listen("tcp", g.option.Addr)
	if err != nil {
		return err
	}

	g.logger.Info("HTTP server is running", slog.String("address", g.option.Addr))

	g.server = &http.Server{
		Handler:           g.router,
		ReadHeaderTimeout: readTimeout, // https://cwe.mitre.org/data/definitions/400.html
	}

	go func() {
		err := g.server.Serve(ln)
		if errors.Is(err, http.ErrServerClosed) {
			g.logger.Warn("HTTP Server was shutdown")
			return
		}
		g.logger.Error("an error occurred durinng runtime of the HTTP server", sloghelper.Error(err))
	}()

	return nil
}
