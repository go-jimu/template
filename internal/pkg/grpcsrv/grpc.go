package grpcsrv

import (
	"context"
	"log/slog"
	"net"

	"github.com/go-jimu/components/sloghelper"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/samber/oops"
	"go.uber.org/fx"
	"google.golang.org/grpc"
)

type (
	grpcSrv struct {
		server *grpc.Server
		logger *slog.Logger
		opt    Option
		ln     net.Listener
	}

	Option struct {
		Addr string `json:"addr" toml:"addr" yaml:"addr"`
	}
)

func NewGRPCServ(lc fx.Lifecycle, opt Option, logger *slog.Logger) grpc.ServiceRegistrar {
	srv := &grpcSrv{logger: logger, opt: opt}
	unaryOpts := grpc.ChainUnaryInterceptor(
		logging.UnaryServerInterceptor(InterceptorLogger()),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandlerContext(PanicRecoveryHandler)),
	)
	streamOpt := grpc.ChainStreamInterceptor(
		logging.StreamServerInterceptor(InterceptorLogger()),
		recovery.StreamServerInterceptor(recovery.WithRecoveryHandlerContext(PanicRecoveryHandler)),
	)
	srv.server = grpc.NewServer(grpc.StatsHandler(NewCarrier(logger)), unaryOpts, streamOpt)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.opt.Addr)
			if err != nil {
				return oops.With("address", srv.opt.Addr).Wrap(err)
			}
			srv.ln = ln
			srv.logger.Info("gRPC server is running", slog.String("addr", ln.Addr().String()))

			go srv.Serve()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			srv.server.GracefulStop()
			srv.logger.Warn("gRPC server was shutdown")
			return nil
		},
	})
	return srv
}

func (g *grpcSrv) Serve() {
	if err := g.server.Serve(g.ln); err != nil {
		err = oops.With("addr", g.opt.Addr).Wrap(err)
		g.logger.Error("an error was encounted while the gRPC server was running", sloghelper.Error(err))
		return
	}
	g.logger.Info("grpc was shutdown")
}

func (g *grpcSrv) RegisterService(desc *grpc.ServiceDesc, impl any) {
	g.server.RegisterService(desc, impl)
	g.logger.Info("registered a new service", slog.String("service", desc.ServiceName))
}
