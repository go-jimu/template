package connectrpc

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect/v2"
	"github.com/go-jimu/components/transaction"
	"github.com/samber/oops"
)

func TestMapErrorHidesInternalMessageAndPreservesCause(t *testing.T) {
	cause := oops.With("user_id", "user-1").
		Wrap(errors.New("database credentials rejected"))
	mapped := mapError(cause)

	if connect.CodeOf(mapped) != connect.CodeInternal {
		t.Fatalf("Connect code = %s, want %s", connect.CodeOf(mapped), connect.CodeInternal)
	}
	if strings.Contains(mapped.Error(), "credentials") {
		t.Fatalf("public Connect error leaked internal cause: %q", mapped.Error())
	}
	if !errors.Is(mapped, cause) {
		t.Fatalf("mapped error does not preserve diagnostic cause: %v", mapped)
	}
}

// Intent: a transaction outcome reaches the client as a code it can act on,
// without the driver's message, and an unknown commit outcome is never
// reported as a plain cancellation the client would retry blindly.
func TestMapErrorTransactionOutcomes(t *testing.T) {
	driverErr := errors.New("Error 1213: Deadlock found when trying to get lock")
	for name, tc := range map[string]struct {
		err  error
		code connect.Code
	}{
		"conflict":        {errors.Join(transaction.ErrConflict, driverErr), connect.CodeAborted},
		"outcome unknown": {errors.Join(transaction.ErrOutcomeUnknown, driverErr), connect.CodeUnknown},
		"outcome unknown on cancel": {
			errors.Join(transaction.ErrOutcomeUnknown, context.Canceled), connect.CodeUnknown,
		},
	} {
		t.Run(name, func(t *testing.T) {
			mapped := mapError(tc.err)
			if connect.CodeOf(mapped) != tc.code {
				t.Fatalf("Connect code = %s, want %s", connect.CodeOf(mapped), tc.code)
			}
			if strings.Contains(mapped.Error(), "Deadlock") {
				t.Fatalf("public Connect error leaked the driver error: %q", mapped.Error())
			}
			if !errors.Is(mapped, tc.err) {
				t.Fatalf("mapped error does not preserve the cause: %v", mapped)
			}
		})
	}
}
