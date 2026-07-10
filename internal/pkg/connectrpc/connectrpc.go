// Package connectrpc owns the shared Connect, gRPC, and gRPC-Web HTTP server.
package connectrpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/go-chi/chi/v5"
	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Option contains ConnectRPC server runtime settings.
type Option struct {
	Addr string `json:"addr" yaml:"addr" toml:"addr"`
}

// Validate checks the ConnectRPC component's own configuration.
func (o Option) Validate() error {
	if strings.TrimSpace(o.Addr) == "" {
		return errors.New("connectrpc address is required")
	}
	if _, err := net.ResolveTCPAddr("tcp", o.Addr); err != nil {
		return oops.With("address", o.Addr).
			With("operation", "connectrpc.address.resolve").
			Wrap(oops.Wrap(err))
	}
	return nil
}

// Server registers generated Connect handlers and owns HTTP lifecycle.
type Server interface {
	GetGlobalInterceptors() []connect.Interceptor
	Register(string, http.Handler)
	Address() string
}

type server struct {
	option       Option
	logger       *slog.Logger
	shutdowner   fx.Shutdowner
	interceptors []connect.Interceptor
	router       *chi.Mux
	httpServer   *http.Server
	listener     net.Listener
}

var _ Server = (*server)(nil)

// NewServer creates the shared ConnectRPC server.
func NewServer(
	lifecycle fx.Lifecycle,
	shutdowner fx.Shutdowner,
	option Option,
	logger *slog.Logger,
) (Server, error) {
	if err := option.Validate(); err != nil {
		return nil, err
	}
	router := chi.NewRouter()
	router.Get("/healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	})
	connectServer := &server{
		option:       option,
		logger:       logger,
		shutdowner:   shutdowner,
		interceptors: []connect.Interceptor{NewCarrier(logger).Intercept()},
		router:       router,
	}
	connectServer.httpServer = &http.Server{
		Addr:              option.Addr,
		Handler:           h2c.NewHandler(router, &http2.Server{}),
		ReadHeaderTimeout: 3 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			listener, err := net.Listen("tcp", option.Addr)
			if err != nil {
				return oops.With("address", option.Addr).
					With("operation", "connectrpc.listen").
					Wrap(oops.Wrap(err))
			}
			connectServer.listener = listener
			connectServer.logger.Info(
				"ConnectRPC server started",
				slog.String("address", listener.Addr().String()),
			)
			go connectServer.serve()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return oops.Wrap(connectServer.httpServer.Shutdown(ctx))
		},
	})
	return connectServer, nil
}

// GetGlobalInterceptors returns an isolated copy of shared interceptors.
func (s *server) GetGlobalInterceptors() []connect.Interceptor {
	return append([]connect.Interceptor(nil), s.interceptors...)
}

// Register mounts one generated Connect handler without changing its URL path.
func (s *server) Register(pattern string, handler http.Handler) {
	pattern = strings.TrimSuffix(pattern, "/")
	s.router.Handle(pattern+"/*", handler)
	s.logger.Info("ConnectRPC handler registered", slog.String("pattern", pattern))
}

// Address returns the bound address after startup or the configured address before startup.
func (s *server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.option.Addr
}

func (s *server) serve() {
	err := s.httpServer.Serve(s.listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
	err = oops.Wrap(err)
	s.logger.Error("ConnectRPC server stopped unexpectedly", sloghelper.Error(err))
	if shutdownErr := s.shutdowner.Shutdown(fx.ExitCode(1)); shutdownErr != nil {
		s.logger.Error("failed to request process shutdown", sloghelper.Error(oops.Wrap(shutdownErr)))
	}
}
