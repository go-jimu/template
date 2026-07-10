package connectrpc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
)

const headerRequestID = "X-Request-ID"

// Carrier creates request-scoped loggers and one completion log per RPC.
type Carrier struct {
	logger  *slog.Logger
	counter uint64
	prefix  string
}

// NewCarrier creates the shared unary interceptor.
func NewCarrier(logger *slog.Logger) *Carrier {
	return &Carrier{logger: logger, prefix: requestIDPrefix()}
}

// Intercept returns the Connect unary interceptor.
func (c *Carrier) Intercept() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, request connect.AnyRequest) (
			response connect.AnyResponse,
			err error,
		) {
			startedAt := time.Now()
			requestID := request.Header().Get(headerRequestID)
			if requestID == "" {
				requestID = fmt.Sprintf("%s-%06d", c.prefix, atomic.AddUint64(&c.counter, 1))
			}
			logger := c.logger.With(slog.String("request_id", requestID))
			ctx = sloghelper.NewContext(ctx, logger)

			var loggedErr error
			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						loggedErr = oops.With("panic", fmt.Sprint(recovered)).
							Errorf("panic recovered")
						response = nil
						err = connect.NewError(
							connect.CodeInternal,
							errors.New("internal server error"),
						)
					}
				}()
				response, err = next(ctx, request)
				loggedErr = err
			}()

			if err != nil {
				logger.ErrorContext(
					ctx,
					"request complete",
					slog.String("operation", request.Spec().Procedure),
					slog.String("outcome", "failed"),
					slog.String("connect_code", connect.CodeOf(err).String()),
					slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
					sloghelper.Error(loggedErr),
				)
				return response, err
			}
			logger.InfoContext(
				ctx,
				"request complete",
				slog.String("operation", request.Spec().Procedure),
				slog.String("outcome", "success"),
				slog.String("connect_code", "ok"),
				slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			)
			return response, nil
		}
	}
}

func requestIDPrefix() string {
	hostname, err := os.Hostname()
	if hostname == "" || err != nil {
		hostname = "localhost"
	}
	var random [12]byte
	if _, err = rand.Read(random[:]); err != nil {
		return hostname
	}
	encoded := base64.RawURLEncoding.EncodeToString(random[:])
	return fmt.Sprintf("%s/%s", hostname, strings.TrimRight(encoded, "=")[:10])
}
