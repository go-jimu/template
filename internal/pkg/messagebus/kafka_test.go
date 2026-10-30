package messagebus

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/ddd/message"
	"github.com/go-jimu/components/sloghelper"
	"google.golang.org/protobuf/types/known/emptypb"
)

type handlerStub struct{ err error }

func (handlerStub) Listening() []message.Kind { return nil }

func (h handlerStub) Handle(context.Context, message.Message) error { return h.err }

func TestLoggedHandlerLogsOnlyHandledMessages(t *testing.T) {
	integrationMessage, err := message.New("test.v1.Empty", &emptypb.Empty{})
	if err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	ctx := sloghelper.NewContext(context.Background(), slog.New(slog.NewJSONHandler(&logs, nil)))

	if err = (loggedHandler{Handler: handlerStub{}}).Handle(ctx, integrationMessage); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"message_id":"` + integrationMessage.ID() + `"`, `"duration_ms":`} {
		if !strings.Contains(logs.String(), field) {
			t.Fatalf("completion log %q does not contain %s", logs.String(), field)
		}
	}

	logs.Reset()
	handleErr := errors.New("handler failed")
	if err = (loggedHandler{Handler: handlerStub{err: handleErr}}).Handle(ctx, integrationMessage); !errors.Is(err, handleErr) {
		t.Fatalf("Handle() error = %v, want %v", err, handleErr)
	}
	if logs.Len() != 0 {
		t.Fatalf("failed message logged a completion record: %q", logs.String())
	}
}
