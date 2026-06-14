package test

import (
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/gospacex/hubx/cache/docx/config"
)

func MockExporter() *tracetest.SpanRecorder {
	return tracetest.NewSpanRecorder()
}

func WithMockTP(rec *tracetest.SpanRecorder) *sdktrace.TracerProvider {
	return sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
}

// FixtureTracingConfig returns a minimal, valid TracingConfig pointing at a
// local insecure OTLP/gRPC endpoint. Useful for unit tests that exercise
// the public layer without spinning up a real collector.
func FixtureTracingConfig() config.TracingConfig {
	return config.TracingConfig{
		Enabled:     true,
		ServiceName: "docx-test",
		Exporter:    "jaeger",
		Protocol:    "grpc",
		Endpoint:    "localhost:4317",
		Insecure:    true,
	}
}
