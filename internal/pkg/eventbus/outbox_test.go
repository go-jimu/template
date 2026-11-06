package eventbus

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Intent: a must-run handler continues the trace of the request that
// dispatched, from the carrier the Outbox stores in its record.
func TestOutboxTraceRoundTrips(t *testing.T) {
	requestCtx, span := sdktrace.NewTracerProvider().Tracer("test").Start(context.Background(), "request")
	defer span.End()

	carrier := captureTrace(requestCtx)
	restored := trace.SpanContextFromContext(restoreTrace(context.Background(), carrier))

	if restored.TraceID() != span.SpanContext().TraceID() || restored.SpanID() != span.SpanContext().SpanID() {
		t.Fatalf("restored span context = %v, want the request's %v", restored, span.SpanContext())
	}
}
