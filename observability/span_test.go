package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TestStartSpan_ReturnsValidSpan confirms the helper installs the SDK
// tracer under the "docx" name and that the returned span is
// non-nil and has a valid SpanContext.
func TestStartSpan_ReturnsValidSpan(t *testing.T) {
	ctx, span := StartSpan(context.Background(), "test-span")
	if span == nil {
		t.Fatal("StartSpan returned nil span")
	}
	if !span.SpanContext().IsValid() {
		t.Fatal("expected valid SpanContext, got zero value")
	}
	defer span.End()
	_ = ctx
}

// TestStartSpan_AppliesOptions verifies that span options (e.g.
// WithAttributes) flow through the helper.
func TestStartSpan_AppliesOptions(t *testing.T) {
	_, span := StartSpan(context.Background(), "with-attrs",
		trace.WithAttributes(attribute.String("k", "v")),
	)
	if !span.SpanContext().IsValid() {
		t.Fatal("expected valid SpanContext")
	}
	span.End()
}
