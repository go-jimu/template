package connectrpc

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
	"time"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/sloghelper"
)

type Carrier struct {
	logger  *slog.Logger
	counter uint64
	prefix  string
}

func NewCarrier(logger *slog.Logger) *Carrier {
	ca := &Carrier{logger: logger}
	ca.genPrefix()
	return ca
}

func (ca *Carrier) genPrefix() {
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

func (ca *Carrier) Intercept() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			requestID := fmt.Sprintf("%s-%06d", ca.prefix, ca.nextRequestID())
			logger := ca.logger.With("request_id", requestID)
			ctx = sloghelper.NewContext(ctx, ca.logger)

			defer func() {
				if rec := recover(); rec != nil {
					err, ok := rec.(error)
					if ok {
						logger.Error("recovered from panic", sloghelper.Error(err))
						return
					}
					logger.Error("recovered from panic", slog.Group("error", slog.Any("message", rec), slog.String("trace", string(debug.Stack()))))
				}
			}()

			res, err := next(ctx, req)

			logger.Info("request complete",
				slog.Group("request", slog.Bool("is_client", req.Spec().IsClient), slog.String("method", req.HTTPMethod()), slog.String("procedure", req.Spec().Procedure)),
				slog.String("elapsed", time.Since(start).String()))

			return res, err
		}
	}
}
