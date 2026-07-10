package connectrpc

import (
	"errors"
	"strings"
	"testing"

	connect "connectrpc.com/connect"
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
