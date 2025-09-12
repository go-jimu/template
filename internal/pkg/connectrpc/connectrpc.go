package connectrpc

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/sloghelper"
	"go.uber.org/fx"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type (
	connectRPCSrv struct {
		option       Option
		logger       *slog.Logger
		interceptors []connect.Interceptor
		mux          *http.ServeMux
		server       *http.Server
		ln           net.Listener
	}

	Option struct {
		Addr     string `json:"addr" yaml:"addr" toml:"addr"`
		CertFile string `json:"cert_file" toml:"cert_file" yaml:"cert_file"`
		KeyFile  string `json:"key_file" toml:"key_file" yaml:"key_file"`
	}

	ConnectServer interface {
		GetGlobalInterceptors() []connect.Interceptor
		Register(string, http.Handler)
		Serve() error
	}
)

func NewConnectRPCServer(lc fx.Lifecycle, opt Option, logger *slog.Logger) ConnectServer {
	logger.Info("create a new Connect server", slog.Any("option", opt))

	srv := &connectRPCSrv{
		option:       opt,
		logger:       logger,
		interceptors: []connect.Interceptor{NewCarrier(logger).Intercept()},
		mux:          http.NewServeMux(),
	}

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			ln, err := net.Listen("tcp", srv.option.Addr)
			if err != nil {
				return err
			}
			srv.ln = ln
			srv.logger.Info("the Connect server is running", slog.String("address", srv.option.Addr))

			go srv.Serve()
			return nil
		},

		OnStop: func(ctx context.Context) error {
			if srv.server != nil {
				return srv.server.Shutdown(ctx)
			}
			return nil
		},
	})
	return srv
}

func (c *connectRPCSrv) GetGlobalInterceptors() []connect.Interceptor {
	return c.interceptors[:]
}

func (c *connectRPCSrv) Register(pattern string, hdl http.Handler) {
	c.mux.Handle(pattern, hdl)
	c.logger.Info("registered a new handler", slog.String("pattern", pattern))
}

func (c *connectRPCSrv) Serve() error {
	c.server = &http.Server{
		Handler:           h2c.NewHandler(c.mux, &http2.Server{}),
		ReadHeaderTimeout: 1 * time.Second,
		MaxHeaderBytes:    8 * 1024,
	}

	// running
	err := c.server.Serve(c.ln)
	if errors.Is(err, http.ErrServerClosed) {
		c.logger.Warn("the Connect server was shutdown")
		return nil
	}
	c.logger.Error("the Connect server encountered an error while serving", sloghelper.Error(err))
	return err
}
