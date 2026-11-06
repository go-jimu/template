package connectrpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/grpchealth/v2"
	"github.com/go-jimu/components/sloghelper"
	"github.com/samber/oops"
)

var healthProcedurePrefix = "/" + grpchealth.HealthV1ServiceName + "/"

// Carrier gives every RPC the process logger in its ctx and writes its one
// completion log, the access log of the service. The trace_id and span_id of
// the RPC's span come from the logger's handler (telemetry.LogAttrs). "API
// logging" in docs/ddd/design.md states its attributes and levels, what
// handlers log, and what no log may contain.
type Carrier struct {
	logger *slog.Logger
}

// NewCarrier creates the interceptor every RPC on the ConnectRPC server runs through.
func NewCarrier(logger *slog.Logger) *Carrier {
	return &Carrier{logger: logger}
}

// Intercept returns the server interceptor. It wraps the whole call, unary
// or streaming, and recovers a handler panic as CodeInternal. Health checks
// pass through unlogged: probes are not requests.
func (c *Carrier) Intercept() connect.ServerInterceptor {
	return func(next connect.ServerFunc) connect.ServerFunc {
		return func(ctx context.Context, spec connect.Spec, stream connect.ServerStream) (err error) {
			if strings.HasPrefix(spec.Procedure, healthProcedurePrefix) {
				return next(ctx, spec, stream)
			}
			startedAt := time.Now()
			logger := c.logger
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
