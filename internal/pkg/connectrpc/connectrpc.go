// Package connectrpc owns the process's ConnectRPC HTTP server: one listener
// that serves every registered contract over Connect, gRPC, and gRPC-Web, plus
// gRPC's standard health-checking service (see NewServer).
//
// A contract is public or private by its proto package (proto/<context>/public
// or proto/<context>/private), not by listener. Which procedures leave the
// deployment is decided by the gateway's routes; see "Inbound protocol
// boundary" in docs/ddd/design.md.
//
// pkg.Lifecycle starts and stops the server through ServeHook, so it stops
// taking requests before the runtimes behind it shut down.
package connectrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"connectrpc.com/grpchealth/v2"
	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// LivenessService is the health-check service name that reports whether the
// process is alive: SERVING for as long as it runs. The empty service name
// reports readiness.
const LivenessService = "liveness"

// Option contains ConnectRPC server runtime settings.
type Option struct {
	// Addr is the listen address, such as ":8080".
	Addr string `json:"addr" yaml:"addr" toml:"addr"`
	// ShutdownDelay is how long the server keeps serving while it reports
	// NOT_SERVING for readiness before it closes its listener, so load balancers stop
	// routing first, such as "5s". Empty means no delay. It counts against
	// the process stop timeout.
	ShutdownDelay string `json:"shutdown-delay" yaml:"shutdown-delay" toml:"shutdown-delay"`
}

// Validate checks the ConnectRPC component's own configuration: the address
// resolves and the shutdown delay is a duration that is not negative.
func (o Option) Validate() error {
	_, err := o.shutdownDelay()
	return err
}

// shutdownDelay validates o and returns its parsed shutdown delay.
func (o Option) shutdownDelay() (time.Duration, error) {
	if err := o.validateAddress(); err != nil {
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

func (o Option) validateAddress() error {
	if strings.TrimSpace(o.Addr) == "" {
		return errors.New("connectrpc addr is required")
	}
	if _, err := net.ResolveTCPAddr("tcp", o.Addr); err != nil {
		return oops.With("option", "addr").
			With("address", o.Addr).
			With("operation", "connectrpc.address.resolve").
			Wrap(err)
	}
	return nil
}

// Server is the listener and the connect.Server that dispatches its RPCs.
// Every RPC runs through the shared interceptor (see NewCarrier).
type Server interface {
	// Registry is the connect.Server that generated code registers handlers
	// on, such as userv1connect.RegisterUserServiceHandler(s.Registry(), h),
	// for private and public contracts alike. Register before the Fx start
	// hook runs: the listener mounts routes for the services registered by
	// then, and a service registered later is never routed.
	Registry() *connect.Server
	// Address returns the bound address after startup or the configured
	// address before startup.
	Address() string
}

// ServeOut exposes the server to bounded contexts and its ServeHook to
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

// server is the listener, its routes, and its connect.Server.
type server struct {
	logger        *slog.Logger
	shutdowner    fx.Shutdowner
	shutdownDelay time.Duration
	health        *grpchealth.StaticChecker
	address       string
	registry      *connect.Server
	mux           *http.ServeMux
	httpServer    *http.Server
	listener      net.Listener
}

var _ Server = (*server)(nil)

// NewServer creates the ConnectRPC server. It serves HTTP/1.1 and unencrypted
// HTTP/2 (h2c), accepts the Connect, gRPC, and gRPC-Web protocols, and keeps
// connect-go's default 4 MiB limit per received message.
//
// It registers grpc.health.v1.Health, which Kubernetes gRPC probes and
// gateway health checks call: the empty service name reports readiness,
// SERVING from start until shutdown begins, so load balancers stop routing
// first; LivenessService reports that the process is alive.
func NewServer(
	shutdowner fx.Shutdowner,
	option Option,
	logger *slog.Logger,
) (ServeOut, error) {
	shutdownDelay, err := option.shutdownDelay()
	if err != nil {
		return ServeOut{}, err
	}
	health := grpchealth.NewStaticChecker(LivenessService)
	health.SetStatus("", grpchealth.StatusNotServing)
	registry := connect.NewServer(NewCarrier(logger).Intercept())
	grpchealth.Register(registry, health)

	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	mux := http.NewServeMux()
	s := &server{
		logger:        logger,
		shutdowner:    shutdowner,
		shutdownDelay: shutdownDelay,
		health:        health,
		address:       option.Addr,
		registry:      registry,
		mux:           mux,
		httpServer: &http.Server{
			Addr:              option.Addr,
			Handler:           mux,
			Protocols:         &protocols,
			ReadHeaderTimeout: 3 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    16 * 1024,
		},
	}
	return ServeOut{
		Server:    s,
		ServeHook: ServeHook{fx.Hook{OnStart: s.start, OnStop: s.stop}},
	}, nil
}

// start mounts the registered procedures, binds the listener, and serves.
func (s *server) start(context.Context) error {
	s.mount()
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return oops.With("address", s.address).
			With("operation", "connectrpc.listen").
			Wrap(err)
	}
	s.listener = listener
	s.logger.Info("ConnectRPC server started", slog.String("address", listener.Addr().String()))
	go s.serve()
	s.health.SetStatus("", grpchealth.StatusServing)
	return nil
}

// stop reports NOT_SERVING for readiness, keeps serving for the shutdown delay so
// load balancers stop routing, then closes the listener and drains requests.
// ctx bounds the whole stop, the delay included.
func (s *server) stop(ctx context.Context) error {
	s.health.SetStatus("", grpchealth.StatusNotServing)
	if s.shutdownDelay > 0 {
		s.logger.InfoContext(ctx, "ConnectRPC server not ready, closing after delay",
			slog.Int64("shutdown_delay_ms", s.shutdownDelay.Milliseconds()))
		timer := time.NewTimer(s.shutdownDelay)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
		}
	}
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return oops.Wrap(err)
	}
	return nil
}

func (s *server) serve() {
	err := s.httpServer.Serve(s.listener)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
	s.logger.Error("ConnectRPC server stopped unexpectedly", sloghelper.Error(oops.Wrap(err)))
	if shutdownErr := s.shutdowner.Shutdown(fx.ExitCode(1)); shutdownErr != nil {
		s.logger.Error("failed to request process shutdown", sloghelper.Error(oops.Wrap(shutdownErr)))
	}
}

// Registry returns the connect.Server that handlers register on.
func (s *server) Registry() *connect.Server {
	return s.registry
}

// mount routes every registered procedure, such as
// "/user.private.v1.UserService/Create", to the connect.Server. Paths of
// unregistered services get a 404.
func (s *server) mount() {
	connecthttp.Mount(s.mux, s.registry)
	for spec := range s.registry.Specs() {
		s.logger.Info("ConnectRPC procedure mounted", slog.String("procedure", spec.Procedure))
	}
}

// Address returns the bound address after startup or the configured address
// before startup.
func (s *server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.address
}
