package connectrpc_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	sharedconnectrpc "github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/samber/oops"
)

var spec = connect.Spec{Procedure: "/test.v1.Service/Call"}

func TestCarrierLogsReturnedErrorOnceAtTheOuterBoundary(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	cause := errors.New("database unavailable")
	interceptor := sharedconnectrpc.NewCarrier(logger).Intercept()(
		func(context.Context, connect.Spec, connect.ServerStream) error {
			return oops.With("user_id", "user-1").Wrap(cause)
		},
	)

	err := interceptor(context.Background(), spec, nil)
	if !errors.Is(err, cause) {
		t.Fatalf("interceptor error = %v, want %v", err, cause)
	}

	output := logs.String()
	if strings.Count(output, "\n") != 1 {
		t.Fatalf("completion log count = %d, want 1: %q", strings.Count(output, "\n"), output)
	}
	for _, field := range []string{
		`"level":"ERROR"`,
		`"outcome":"failed"`,
		`"elapsed_ms":`,
		`"error":`,
		`"operation":"/test.v1.Service/Call"`,
	} {
		if !strings.Contains(output, field) {
			t.Fatalf("completion log %q does not contain %s", output, field)
		}
	}
}

// Intent: an RPC failure is logged at the level of what it means for the
// service: an outcome the client caused, such as an unknown id on the public
// API, is Info like any answered request; a transient dependency failure is
// Warn; a server fault is Error, so error alerts mean a fault.
func TestCarrierLevelsFailuresByCode(t *testing.T) {
	tests := []struct {
		code connect.Code
		want string
	}{
		{connect.CodeCanceled, "INFO"},
		{connect.CodeInvalidArgument, "INFO"},
		{connect.CodeNotFound, "INFO"},
		{connect.CodeAlreadyExists, "INFO"},
		{connect.CodePermissionDenied, "INFO"},
		{connect.CodeUnauthenticated, "INFO"},
		{connect.CodeFailedPrecondition, "INFO"},
		{connect.CodeAborted, "INFO"},
		{connect.CodeOutOfRange, "INFO"},
		{connect.CodeDeadlineExceeded, "WARN"},
		{connect.CodeResourceExhausted, "WARN"},
		{connect.CodeUnavailable, "WARN"},
		{connect.CodeUnknown, "ERROR"},
		{connect.CodeInternal, "ERROR"},
		{connect.CodeUnimplemented, "ERROR"},
		{connect.CodeDataLoss, "ERROR"},
	}
	for _, test := range tests {
		t.Run(test.code.String(), func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			interceptor := sharedconnectrpc.NewCarrier(logger).Intercept()(
				func(context.Context, connect.Spec, connect.ServerStream) error {
					return connect.NewError(test.code, "failed")
				},
			)
			_ = interceptor(context.Background(), spec, nil)

			if !strings.Contains(logs.String(), `"level":"`+test.want+`"`) {
				t.Fatalf("log = %s, want level %s", logs.String(), test.want)
			}
			if !strings.Contains(logs.String(), `"connect_code":"`+test.code.String()+`"`) {
				t.Fatalf("log = %s, want connect_code %s", logs.String(), test.code)
			}
		})
	}
}

// Intent: a recovered panic is a fault, logged at Error.
func TestCarrierLogsPanicAsError(t *testing.T) {
	var logs bytes.Buffer
	interceptor := sharedconnectrpc.NewCarrier(slog.New(slog.NewJSONHandler(&logs, nil))).Intercept()(
		func(context.Context, connect.Spec, connect.ServerStream) error { panic("boom") },
	)
	err := interceptor(context.Background(), spec, nil)
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
	if !strings.Contains(logs.String(), `"level":"ERROR"`) {
		t.Fatalf("log = %s, want ERROR", logs.String())
	}
}

// Intent: the client sees only the fixed message of an internal error, but
// the log records its cause with the fields attached where it entered, so an
// operator can diagnose the fault.
func TestCarrierLogsTheCauseBehindAConnectError(t *testing.T) {
	var logs bytes.Buffer
	interceptor := sharedconnectrpc.NewCarrier(slog.New(slog.NewJSONHandler(&logs, nil))).Intercept()(
		func(context.Context, connect.Spec, connect.ServerStream) error {
			cause := oops.With("user_id", "user-1").Wrap(errors.New("dial tcp: connection refused"))
			return connect.NewError(connect.CodeInternal, "internal server error").WithCause(cause)
		},
	)
	err := interceptor(context.Background(), spec, nil)

	var connectErr *connect.Error
	if !errors.As(err, &connectErr) || connectErr.Message() != "internal server error" {
		t.Fatalf("error = %v, want the fixed internal message", err)
	}
	for _, want := range []string{"connection refused", `"user_id":"user-1"`} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log does not contain %s: %s", want, logs.String())
		}
	}
}
