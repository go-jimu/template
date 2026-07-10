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
		`"error":`,
	} {
		if !strings.Contains(output, field) {
			t.Fatalf("completion log %q does not contain %s", output, field)
		}
	}
}
