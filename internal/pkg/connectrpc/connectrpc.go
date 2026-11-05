// Package connectrpc owns the shared Connect, gRPC, and gRPC-Web HTTP servers:
// one listener for the deployment-private contracts (proto/<context>/private)
// and one for the externally supported contracts (proto/<context>/public),
// so exposing the public port to web and app clients never exposes a private
// contract. pkg.Lifecycle starts and stops both through ServeHook, so they
// stop taking requests before the runtimes behind them shut down.
package connectrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Option contains ConnectRPC server runtime settings.
type Option struct {
	// Addr serves the deployment-private contracts. Keep it inside the
	// deployment's network.
	Addr string `json:"addr" yaml:"addr" toml:"addr"`
	// PublicAddr serves the externally supported contracts for web and app
	// clients. Expose only this address.
	PublicAddr string `json:"public-addr" yaml:"public-addr" toml:"public-addr"`
	// ShutdownDelay is how long both servers keep serving with /readyz
	// unavailable before they close their listeners, so load balancers stop
	// routing first, such as "5s". Empty means no delay. It counts against
	// the process stop timeout.
	ShutdownDelay string `json:"shutdown-delay" yaml:"shutdown-delay" toml:"shutdown-delay"`
}

// Validate checks the ConnectRPC component's own configuration: both
// addresses resolve, they do not share a port, and the shutdown delay is a
// duration that is not negative.
func (o Option) Validate() error {
	_, err := o.shutdownDelay()
	return err
}

// shutdownDelay validates o and returns its parsed shutdown delay.
func (o Option) shutdownDelay() (time.Duration, error) {
	if err := o.validateAddresses(); err != nil {
		return 0, err
	}
	if o.ShutdownDelay == "" {
		return 0, nil
	}
	delay, err := time.ParseDuration(o.ShutdownDelay)
	if err != nil {
		return 0, oops.With("option", "shutdown-delay").With("value", o.ShutdownDelay).Wrap(err)
	}
	if delay < 0 {
		return 0, fmt.Errorf("connectrpc shutdown-delay %q cannot be negative", o.ShutdownDelay)
	}
	return delay, nil
}

func (o Option) validateAddresses() error {
	private, err := resolve("addr", o.Addr)
	if err != nil {
		return err
	}
	public, err := resolve("public-addr", o.PublicAddr)
	if err != nil {
		return err
	}
	if private.Port != 0 && private.Port == public.Port &&
		(private.IP == nil || public.IP == nil || private.IP.Equal(public.IP)) {
		return fmt.Errorf("connectrpc addr %q and public-addr %q share a port: the private contracts would be public",
			o.Addr, o.PublicAddr)
	}
	return nil
}

func resolve(name, address string) (*net.TCPAddr, error) {
	if strings.TrimSpace(address) == "" {
		return nil, fmt.Errorf("connectrpc %s is required", name)
	}
	resolved, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, oops.With("option", name).
			With("address", address).
			With("operation", "connectrpc.address.resolve").
			Wrap(err)
	}
	return resolved, nil
}

// Server registers generated Connect handlers on one listener.
type Server interface {
	GetGlobalInterceptors() []connect.Interceptor
	Register(string, http.Handler)
	Address() string
}

// PrivateServer serves proto/<context>/private contracts. Register only
// handlers generated from gen/<context>/private.
type PrivateServer interface{ Server }

// PublicServer serves proto/<context>/public contracts to web and app
// clients. Register only handlers generated from gen/<context>/public.
type PublicServer interface{ Server }

// ServeOut exposes both servers to bounded contexts and their ServeHook to
// pkg.Lifecycle.
type ServeOut struct {
	fx.Out

	Private   PrivateServer
	Public    PublicServer
	ServeHook ServeHook
}

// ServeHook starts and stops serving requests on both listeners.
type ServeHook struct {
	fx.Hook
}

// servers owns the two listeners and the readiness they share.
type servers struct {
	logger        *slog.Logger
	shutdowner    fx.Shutdowner
	shutdownDelay time.Duration
	ready         atomic.Bool
	private       *server
	public        *server
}

// server is one listener with its own routes.
type server struct {
	surface      string
	address      string
	logger       *slog.Logger
	interceptors []connect.Interceptor
	mux          *http.ServeMux
	httpServer   *http.Server
	listener     net.Listener
}

var _ Server = (*server)(nil)

// NewServer creates the private and public ConnectRPC servers. On both ports
// /healthz reports that the process is alive, and /readyz whether it is
// serving; /readyz turns unavailable once shutdown begins so load balancers
// stop routing to it.
func NewServer(
	shutdowner fx.Shutdowner,
	option Option,
	logger *slog.Logger,
) (ServeOut, error) {
	shutdownDelay, err := option.shutdownDelay()
	if err != nil {
		return ServeOut{}, err
	}
	interceptors := []connect.Interceptor{NewCarrier(logger).Intercept()}
	s := &servers{logger: logger, shutdowner: shutdowner, shutdownDelay: shutdownDelay}
	s.private = s.newServer("private", option.Addr, interceptors)
	s.public = s.newServer("public", option.PublicAddr, interceptors)
	return ServeOut{
		Private:   s.private,
		Public:    s.public,
		ServeHook: ServeHook{fx.Hook{OnStart: s.start, OnStop: s.stop}},
	}, nil
}

func (s *servers) newServer(surface, address string, interceptors []connect.Interceptor) *server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /readyz", func(response http.ResponseWriter, _ *http.Request) {
		if !s.ready.Load() {
			response.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		response.WriteHeader(http.StatusOK)
	})
	return &server{
		surface:      surface,
		address:      address,
		logger:       s.logger.With(slog.String("surface", surface)),
		interceptors: interceptors,
		mux:          mux,
		httpServer: &http.Server{
			Addr:              address,
			Handler:           h2c.NewHandler(mux, &http2.Server{}),
			ReadHeaderTimeout: 3 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    16 * 1024,
		},
	}
}

// start binds both listeners before serving either, so a port conflict fails
// startup without leaving one surface half up.
func (s *servers) start(context.Context) error {
	for _, srv := range []*server{s.private, s.public} {
		listener, err := net.Listen("tcp", srv.address)
		if err != nil {
			s.closeListeners()
			return oops.With("surface", srv.surface).
				With("address", srv.address).
				With("operation", "connectrpc.listen").
				Wrap(err)
		}
		srv.listener = listener
	}
	for _, srv := range []*server{s.private, s.public} {
		srv.logger.Info("ConnectRPC server started", slog.String("address", srv.listener.Addr().String()))
		go s.serve(srv)
	}
	s.ready.Store(true)
	return nil
}

func (s *servers) closeListeners() {
	for _, srv := range []*server{s.private, s.public} {
		if srv.listener != nil {
			_ = srv.listener.Close()
			srv.listener = nil
		}
	}
}

// stop turns /readyz unavailable, keeps serving for the shutdown delay so
// load balancers stop routing, then closes both listeners and drains their
// requests concurrently. ctx bounds the whole stop, the delay included.
func (s *servers) stop(ctx context.Context) error {
	s.ready.Store(false)
	if s.shutdownDelay > 0 {
		s.logger.InfoContext(ctx, "ConnectRPC servers not ready, closing after delay",
			slog.Int64("shutdown_delay_ms", s.shutdownDelay.Milliseconds()))
		timer := time.NewTimer(s.shutdownDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
	}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for index, srv := range []*server{s.private, s.public} {
		wg.Go(func() {
			if err := srv.httpServer.Shutdown(ctx); err != nil {
				errs[index] = oops.With("surface", srv.surface).Wrap(err)
			}
		})
	}
	wg.Wait()
	return errors.Join(errs...)
}

func (s *servers) serve(srv *server) {
	err := srv.httpServer.Serve(srv.listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
	srv.logger.Error("ConnectRPC server stopped unexpectedly", sloghelper.Error(oops.Wrap(err)))
	if shutdownErr := s.shutdowner.Shutdown(fx.ExitCode(1)); shutdownErr != nil {
		s.logger.Error("failed to request process shutdown", sloghelper.Error(oops.Wrap(shutdownErr)))
	}
}

// GetGlobalInterceptors returns an isolated copy of shared interceptors.
func (s *server) GetGlobalInterceptors() []connect.Interceptor {
	return append([]connect.Interceptor(nil), s.interceptors...)
}

// Register mounts one generated Connect handler on its service path, such as
// "/user.private.v1.UserService/", which serves every procedure under it.
func (s *server) Register(pattern string, handler http.Handler) {
	pattern = strings.TrimSuffix(pattern, "/") + "/"
	s.mux.Handle(pattern, handler)
	s.logger.Info("ConnectRPC handler registered", slog.String("pattern", pattern))
}

// Address returns the bound address after startup or the configured address
// before startup.
func (s *server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.address
}
