package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// carrier is a minimal TextMapCarrier that stores the trace headers in
// a map. It is used to assert that the propagator round-trips a trace
// context through Inject + Extract.
type carrier map[string]string

func (c carrier) Get(k string) string { return c[k] }
func (c carrier) Set(k, v string)     { c[k] = v }
func (c carrier) Keys() []string {
	out := make([]string, 0, len(c))
	for k := range c {
		out = append(out, k)
	}
	return out
}

// TestInjectTrace_WritesHeaders asserts that a context carrying a
// valid SpanContext produces traceparent / tracestate headers when
// passed through InjectTrace.
func TestInjectTrace_WritesHeaders(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	_, span := StartSpan(context.Background(), "inject-test")
	defer span.End()

	c := carrier{}
	InjectTrace(trace.ContextWithSpan(context.Background(), span), c)
	if len(c) == 0 {
		t.Fatal("expected carrier to receive trace headers after Inject")
	}
	if c["traceparent"] == "" {
		t.Fatalf("expected traceparent header, got carrier: %v", c)
	}
}

// TestExtractTrace_RestoresSpanContext asserts the inverse: a carrier
// carrying traceparent is extracted back into a context whose
// SpanContext matches the original span.
func TestExtractTrace_RestoresSpanContext(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	_, span := StartSpan(context.Background(), "extract-test")
	defer span.End()

	c := carrier{}
	InjectTrace(trace.ContextWithSpan(context.Background(), span), c)
	want := span.SpanContext()

	extracted := ExtractTrace(context.Background(), c)
	got := trace.SpanContextFromContext(extracted)
	if !got.IsValid() {
		t.Fatal("extracted context has invalid SpanContext")
	}
	if got.TraceID() != want.TraceID() {
		t.Fatalf("trace id mismatch: want %s, got %s", want.TraceID(), got.TraceID())
	}
	if got.SpanID() != want.SpanID() {
		t.Fatalf("span id mismatch: want %s, got %s", want.SpanID(), got.SpanID())
	}
}

// TestInjectTrace_EmptyContextStaysEmpty verifies the no-tracing
// case: a context with no SpanContext produces an empty carrier.
func TestInjectTrace_EmptyContextStaysEmpty(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})

	c := carrier{}
	InjectTrace(context.Background(), c)
	if len(c) != 0 {
		t.Fatalf("expected empty carrier for context without SpanContext, got: %v", c)
	}
}
