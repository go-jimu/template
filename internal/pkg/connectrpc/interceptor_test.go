package connectrpc_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"connectrpc.com/connect"
	sharedconnectrpc "github.com/go-jimu/template/internal/pkg/connectrpc"
	"github.com/samber/oops"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestCarrierLogsReturnedErrorOnceAtTheOuterBoundary(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	cause := errors.New("database unavailable")
	interceptor := sharedconnectrpc.NewCarrier(logger).Intercept()(
		func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
			return nil, oops.With("user_id", "user-1").Wrap(cause)
		},
	)

	_, err := interceptor(
		context.Background(),
		connect.NewRequest(&emptypb.Empty{}),
	)
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
				func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
					return nil, connect.NewError(test.code, errors.New("failed"))
				},
			)
			_, _ = interceptor(context.Background(), connect.NewRequest(&emptypb.Empty{}))

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
		func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) { panic("boom") },
	)
	_, err := interceptor(context.Background(), connect.NewRequest(&emptypb.Empty{}))
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("code = %v, want internal", connect.CodeOf(err))
	}
	if !strings.Contains(logs.String(), `"level":"ERROR"`) {
		t.Fatalf("log = %s, want ERROR", logs.String())
	}
}
