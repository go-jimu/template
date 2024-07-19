package grpcsrv

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"

	"github.com/go-jimu/components/sloghelper"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

type Carrier struct {
	prefix  string
	counter uint64
	logger  *slog.Logger
}

// Key to use when setting the request ID.
type ctxKeyRequestID int

const (
	HeaderRequestID = "x-request-id"
	// RequestIDKey is the key that holds the unique request ID in a request context.
	RequestIDKey ctxKeyRequestID = 0
)

func NewCarrier(logger *slog.Logger) stats.Handler {
	ca := &Carrier{logger: logger}
	ca.init()
	return ca
}

func (ca *Carrier) init() {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}
	var buf [12]byte
	var b64 string
	for len(b64) < 10 {
		rand.Read(buf[:])
		b64 = base64.StdEncoding.EncodeToString(buf[:])
		b64 = strings.NewReplacer("+", "", "/", "").Replace(b64)
	}
	ca.prefix = fmt.Sprintf("%s/%s", hostname, b64[0:10])
}

func (ca *Carrier) nextRequestID() uint64 {
	return atomic.AddUint64(&ca.counter, 1)
}

func (ca *Carrier) TagRPC(ctx context.Context, rpc *stats.RPCTagInfo) context.Context {
	var requestID string
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		requestID = fmt.Sprintf("%s-%d", ca.prefix, ca.nextRequestID())
	} else {
		header, ok := md[HeaderRequestID]
		if !ok || len(header) == 0 || header[0] == "" {
			requestID = fmt.Sprintf("%s-%06d", ca.prefix, ca.nextRequestID())
		} else {
			requestID = header[0]
		}
	}

	logger := ca.logger.With(slog.String("request_id", requestID))
	return sloghelper.NewContext(ctx, logger)
}

func (ca *Carrier) HandleRPC(ctx context.Context, rs stats.RPCStats) {}
func (ca *Carrier) TagConn(ctx context.Context, conn *stats.ConnTagInfo) context.Context {
	return ctx
}
func (ca *Carrier) HandleConn(ctx context.Context, cs stats.ConnStats) {}

func InterceptorLogger() logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, level logging.Level, msg string, fields ...any) {
		sloghelper.FromContext(ctx).Log(ctx, slog.Level(level), msg, fields...)
	})
}

func PanicRecoveryHandler(ctx context.Context, p any) error {
	if err, ok := p.(error); ok {
		sloghelper.FromContext(ctx).Error("recovered from panic", sloghelper.Error(err))
	} else {
		sloghelper.FromContext(ctx).Error("recovered from panic", slog.Group("error", slog.Any("message", p), slog.String("trace", string(debug.Stack()))))
	}
	return status.Errorf(codes.Internal, "%s", p)
}
