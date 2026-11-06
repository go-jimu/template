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

	"connectrpc.com/connect/v2"
	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
)

const headerRequestID = "X-Request-ID"

// Carrier creates request-scoped loggers and one completion log per RPC: Info
// for success and for failures the client caused, Warn for transient
// failures, Error for server faults (see failureLevel).
type Carrier struct {
	logger  *slog.Logger
	counter uint64
	prefix  string
}

// NewCarrier creates the interceptor shared by both ConnectRPC servers.
func NewCarrier(logger *slog.Logger) *Carrier {
	return &Carrier{logger: logger, prefix: requestIDPrefix()}
}

// Intercept returns the server interceptor. It wraps the whole call, unary
// or streaming, and recovers a handler panic as CodeInternal.
func (c *Carrier) Intercept() connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) (err error) {
			startedAt := time.Now()
			var requestID string
			if info, ok := connect.CallInfoForServerContext(ctx); ok {
				requestID = info.RequestHeader().Get(headerRequestID)
			}
			if requestID == "" {
				requestID = fmt.Sprintf("%s-%06d", c.prefix, atomic.AddUint64(&c.counter, 1))
			}
			logger := c.logger.With(slog.String("request_id", requestID))
			ctx = sloghelper.NewContext(ctx, logger)

			func() {
				defer func() {
					if recovered := recover(); recovered != nil {
						err = connect.NewError(connect.CodeInternal, "internal server error").
							WithCause(oops.With("panic", fmt.Sprint(recovered)).Errorf("panic recovered"))
					}
				}()
				err = next(ctx, spec, stream)
			}()

			if err != nil {
				code := connect.CodeOf(err)
				logger.Log(
					ctx,
					failureLevel(code),
					"request complete",
					slog.String("operation", spec.Procedure),
					slog.String("outcome", "failed"),
					slog.String("connect_code", code.String()),
					slog.Int64("elapsed_ms", time.Since(startedAt).Milliseconds()),
					sloghelper.Error(diagnosis(err)),
				)
				return err
			}
			logger.InfoContext(
				ctx,
				"request complete",
				slog.String("operation", spec.Procedure),
				slog.String("outcome", "success"),
				slog.String("connect_code", "ok"),
				slog.Int64("elapsed_ms", time.Since(startedAt).Milliseconds()),
			)
			return nil
		}
	}
}

// diagnosis is the error to log for err: the cause behind a *connect.Error,
// which carries the fields where it entered our code, instead of the message
// sent to the client; err itself when there is no cause.
func diagnosis(err error) error {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		if cause := connectErr.Unwrap(); cause != nil {
			return cause
		}
	}
	return err
}

// failureLevel is the log level of a failed RPC by what its code means for
// the service: Info for an outcome the client caused, answered like any
// request; Warn for a transient failure of the service or a dependency;
// Error for a server fault, including a recovered panic.
func failureLevel(code connect.Code) slog.Level {
	switch code {
	case connect.CodeCanceled,
		connect.CodeInvalidArgument,
		connect.CodeNotFound,
		connect.CodeAlreadyExists,
		connect.CodePermissionDenied,
		connect.CodeUnauthenticated,
		connect.CodeFailedPrecondition,
		connect.CodeAborted,
		connect.CodeOutOfRange:
		return slog.LevelInfo
	case connect.CodeDeadlineExceeded,
		connect.CodeResourceExhausted,
		connect.CodeUnavailable:
		return slog.LevelWarn
	default:
		return slog.LevelError
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
