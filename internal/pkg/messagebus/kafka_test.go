package messagebus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/go-jimu/components/ddd/message"
	jimukafka "github.com/go-jimu/contrib/message/kafka"
	"google.golang.org/protobuf/types/known/emptypb"
)

const testKind message.Kind = "test.v1.Empty"

type handlerStub struct{ err error }

func (handlerStub) Listening() []message.Kind { return []message.Kind{testKind} }

func (h handlerStub) Handle(context.Context, message.Message) error { return h.err }

// Intent: every handler subscribed through the runtime is wrapped in
// message.Logging, so each handled message gets one Debug record without the
// handler logging it; a failure is returned unchanged for the consumer's
// failure policy, which logs it at its own level.
func TestLoggingSubscriberWrapsHandlers(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{"handled", nil, "message handled"},
		{"failed", errors.New("handler failed"), "message handler failed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			router := message.NewRouter()
			if err := (loggingSubscriber{subscriber: router, logger: logger}).Subscribe(handlerStub{err: test.err}); err != nil {
				t.Fatal(err)
			}
			integrationMessage, err := message.New(testKind, &emptypb.Empty{})
			if err != nil {
				t.Fatal(err)
			}

			if err = router.Handle(context.Background(), integrationMessage); !errors.Is(err, test.err) {
				t.Fatalf("Handle() error = %v, want %v", err, test.err)
			}
			if strings.Count(logs.String(), "\n") != 1 {
				t.Fatalf("logs = %q, want one record", logs.String())
			}
			var record map[string]any
			if err = json.Unmarshal(logs.Bytes(), &record); err != nil {
				t.Fatal(err)
			}
			if record["level"] != "DEBUG" || record["msg"] != test.message {
				t.Fatalf("record = %v, want DEBUG %q", record, test.message)
			}
			if record["message_id"] != integrationMessage.ID() {
				t.Fatalf("message_id = %v, want %s", record["message_id"], integrationMessage.ID())
			}
			if _, ok := record["elapsed_ms"]; !ok {
				t.Fatalf("record %v has no elapsed_ms", record)
			}
		})
	}
}

// Intent: failure-policy retry retries a handler failure in place, while drop
// drops it at once.
func TestFailurePolicy(t *testing.T) {
	failure := jimukafka.Error{Stage: jimukafka.StageHandle, Err: errors.New("db down"), Attempt: 1}
	for name, want := range map[string]jimukafka.FailureAction{
		failurePolicyRetry: jimukafka.FailureActionRetry,
		failurePolicyDrop:  jimukafka.FailureActionDrop,
	} {
		if got := failurePolicy(name).Decide(context.Background(), failure); got != want {
			t.Errorf("failurePolicy(%q) decided %v, want %v", name, got, want)
		}
	}
}
