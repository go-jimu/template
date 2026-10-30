// Package connectrpc owns the shared Connect, gRPC, and gRPC-Web HTTP server.
// pkg.Lifecycle starts and stops serving through ServeHook, so the server stops
// taking requests before the runtimes behind it shut down.
package connectrpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
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
			Wrap(err)
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
	ready        atomic.Bool
}

// ServeOut exposes the Server to bounded contexts and its ServeHook to
// pkg.Lifecycle.
type ServeOut struct {
	fx.Out

	Server    Server
	ServeHook ServeHook
}

// ServeHook starts and stops serving requests.
type ServeHook struct {
	fx.Hook
}

var _ Server = (*server)(nil)

// NewServer creates the shared ConnectRPC server. /healthz reports that the
// process is alive; /readyz reports whether it is serving, and turns
// unavailable once shutdown begins so load balancers stop routing to it.
func NewServer(
	shutdowner fx.Shutdowner,
	option Option,
	logger *slog.Logger,
) (ServeOut, error) {
	if err := option.Validate(); err != nil {
		return ServeOut{}, err
	}
	router := chi.NewRouter()
	connectServer := &server{
		option:       option,
		logger:       logger,
		shutdowner:   shutdowner,
		interceptors: []connect.Interceptor{NewCarrier(logger).Intercept()},
		router:       router,
	}
	router.Get("/healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	})
	router.Get("/readyz", func(response http.ResponseWriter, _ *http.Request) {
		if !connectServer.ready.Load() {
			response.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		response.WriteHeader(http.StatusOK)
	})
	connectServer.httpServer = &http.Server{
		Addr:              option.Addr,
		Handler:           h2c.NewHandler(router, &http2.Server{}),
		ReadHeaderTimeout: 3 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}
	return ServeOut{
		Server:    connectServer,
		ServeHook: ServeHook{fx.Hook{OnStart: connectServer.start, OnStop: connectServer.stop}},
	}, nil
}

func (s *server) start(context.Context) error {
	listener, err := net.Listen("tcp", s.option.Addr)
	if err != nil {
		return oops.With("address", s.option.Addr).
			With("operation", "connectrpc.listen").
			Wrap(err)
	}
	s.listener = listener
	s.logger.Info("ConnectRPC server started", slog.String("address", listener.Addr().String()))
	go s.serve()
	s.ready.Store(true)
	return nil
}

func (s *server) stop(ctx context.Context) error {
	s.ready.Store(false)
	return oops.Wrap(s.httpServer.Shutdown(ctx))
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
