package grpcsrv

import (
	"context"
	"log/slog"
	"net"
	"runtime/debug"

	"github.com/go-jimu/components/sloghelper"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		logging.UnaryServerInterceptor(srv.InterceptorLogger()),
		recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(srv.PanicRecoveryHandler)),
	)
	streamOpt := grpc.ChainStreamInterceptor(
		logging.StreamServerInterceptor(srv.InterceptorLogger()),
		recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(srv.PanicRecoveryHandler)),
	)
	srv.server = grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()), unaryOpts, streamOpt)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go srv.Serve()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			srv.server.GracefulStop()
			return nil
		},
	})
	return srv
}

func (g *grpcSrv) Serve() error {
	ln, err := net.Listen("tcp", g.opt.Addr)
	if err != nil {
		return err
	}
	g.logger.Info("running gRPC server", slog.String("addr", ln.Addr().String()))
	return g.server.Serve(ln)
}

func (g *grpcSrv) RegisterService(desc *grpc.ServiceDesc, impl any) {
	g.server.RegisterService(desc, impl)
	g.logger.Info("registered a new service", slog.String("service", desc.ServiceName))
}

func (g *grpcSrv) InterceptorLogger() logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, level logging.Level, msg string, fields ...any) {
		g.logger.Log(ctx, slog.Level(level), msg, fields...)
	})
}

func (g *grpcSrv) PanicRecoveryHandler(p any) error {
	if err, ok := p.(error); ok {
		g.logger.Error("recovered from panic", sloghelper.Error(err))
	} else {
		g.logger.Error("recovered from panic", slog.Group("error", slog.Any("message", p), slog.String("trace", string(debug.Stack()))))
	}
	return status.Errorf(codes.Internal, "%s", p)
}
