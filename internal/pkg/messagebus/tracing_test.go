package messagebus

import (
	"context"
	"testing"

	"github.com/twmb/franz-go/pkg/kgo"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Intent: a record published inside a span carries its traceparent, and the
// consumer's handler ctx, built by continueRecordTrace from the fetched
// record, continues that trace; one trace spans producer and consumer.
func TestRecordHandlersContinueThePublishersTrace(t *testing.T) {
	provider := sdktrace.NewTracerProvider()
	hooks := tracingHooks(provider, Option{ClientID: "test", ConsumerGroup: "test"})

	publishCtx, span := provider.Tracer("test").Start(context.Background(), "use case")
	defer span.End()
	produced := &kgo.Record{Topic: "t", Context: publishCtx}
	for _, hook := range hooks {
		if h, ok := hook.(kgo.HookProduceRecordBuffered); ok {
			h.OnProduceRecordBuffered(produced)
		}
	}

	fetched := &kgo.Record{Topic: "t", Headers: produced.Headers}
	for _, hook := range hooks {
		if h, ok := hook.(kgo.HookFetchRecordBuffered); ok {
			h.OnFetchRecordBuffered(fetched)
		}
	}
	handlerCtx := continueRecordTrace(context.Background(), fetched)

	want := span.SpanContext().TraceID()
	if got := trace.SpanContextFromContext(handlerCtx).TraceID(); got != want {
		t.Fatalf("handler trace = %s, want the publisher's %s (headers %v)", got, want, produced.Headers)
	}
}

// Intent: a record without trace context leaves the handler ctx as it is.
func TestRecordWithoutTraceLeavesHandlerContext(t *testing.T) {
	ctx := context.Background()
	if got := continueRecordTrace(ctx, &kgo.Record{}); got != ctx {
		t.Fatal("handler ctx changed for a record without trace context")
	}
}
