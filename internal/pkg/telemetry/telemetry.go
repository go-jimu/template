// Package telemetry starts OpenTelemetry tracing for the process: W3C trace
// context propagation, and a TracerProvider that always creates trace and
// span IDs but exports spans only when an OTLP endpoint is configured. The
// trace ID is what correlates logs across RPCs, Domain Events, Kafka, and
// services; see "Trace context" in docs/ddd/design.md.
//
// Configuration is the standard OpenTelemetry environment: OTEL_SERVICE_NAME
// and OTEL_RESOURCE_ATTRIBUTES name the service, OTEL_TRACES_SAMPLER picks
// the sampler (parent-based always-on by default), and
// OTEL_EXPORTER_OTLP_ENDPOINT or OTEL_EXPORTER_OTLP_TRACES_ENDPOINT turns
// export over OTLP/HTTP on.
package telemetry

import (
	"context"
	"log/slog"
	"os"

	"github.com/samber/oops"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
)

// defaultServiceName names the service when OTEL_SERVICE_NAME is not set.
const defaultServiceName = "go-jimu-template"

// Propagator is the trace context propagation every adapter uses: W3C
// traceparent and tracestate.
var Propagator propagation.TextMapPropagator = propagation.TraceContext{}

// NewTracerProvider creates the process TracerProvider, also set as the
// OpenTelemetry global with Propagator. Its stop hook flushes and shuts it
// down; components that trace take the provider as a dependency, so it is
// built before them and shut down after them.
func NewTracerProvider(lifecycle fx.Lifecycle) (trace.TracerProvider, error) {
	ctx := context.Background()
	res, err := resource.New(ctx,
		resource.WithAttributes(attribute.String("service.name", defaultServiceName)),
		resource.WithFromEnv(), // OTEL_SERVICE_NAME and OTEL_RESOURCE_ATTRIBUTES win
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, oops.With("operation", "telemetry.resource").Wrap(err)
	}
	options := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if exporting() {
		exporter, err := otlptracehttp.New(ctx)
		if err != nil {
			return nil, oops.With("operation", "telemetry.exporter").Wrap(err)
		}
		options = append(options, sdktrace.WithBatcher(exporter))
	}
	provider := sdktrace.NewTracerProvider(options...)
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(Propagator)
	lifecycle.Append(fx.StopHook(provider.Shutdown))
	return provider, nil
}

func exporting() bool {
	return os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" ||
		os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != ""
}

// LogAttrs is a sloghelper.HandleFunc that adds the trace_id and span_id of
// the span in ctx to every record logged with that ctx.
func LogAttrs(ctx context.Context, record *slog.Record) {
	span := trace.SpanContextFromContext(ctx)
	if !span.IsValid() {
		return
	}
	record.AddAttrs(
		slog.String("trace_id", span.TraceID().String()),
		slog.String("span_id", span.SpanID().String()),
	)
}

// ContinueTrace returns ctx carrying the span context of from, so work on ctx
// continues from's trace, or ctx unchanged when from carries none. It copies
// only the span context: no other value of from, and not its cancellation.
func ContinueTrace(ctx, from context.Context) context.Context {
	if from == nil {
		return ctx
	}
	span := trace.SpanContextFromContext(from)
	if !span.IsValid() {
		return ctx
	}
	return trace.ContextWithSpanContext(ctx, span)
}
