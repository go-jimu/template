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

func (g *grpcSrv) Serve() error {
	ln, err := net.Listen("tcp", g.opt.Addr)
	if err != nil {
		return oops.With("addr", g.opt.Addr).Wrap(err)
	}
	g.logger.Info("gRPC server is running", slog.String("addr", ln.Addr().String()))
	err = g.server.Serve(ln)
	if err != nil {
		err = oops.With("addr", g.opt.Addr).Wrap(err)
		g.logger.Error("an error was encounted while the gRPC server was running", sloghelper.Error(err))
	}
	return err
}

func (g *grpcSrv) RegisterService(desc *grpc.ServiceDesc, impl any) {
	g.server.RegisterService(desc, impl)
	g.logger.Info("registered a new service", slog.String("service", desc.ServiceName))
}
