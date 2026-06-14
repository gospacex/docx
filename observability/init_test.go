package observability

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"

	"github.com/gospacex/hubx/cache/docx/config"
)

func TestInitTracing_NoopSetsProvider(t *testing.T) {
	if err := InitTracing(context.Background(), config.TracingConfig{}, WithNoop()); err != nil {
		t.Fatalf("noop init should not error, got: %v", err)
	}
	if otel.GetTracerProvider() == nil {
		t.Fatal("expected noop init to install a tracer provider")
	}
}

func TestInitTracing_RejectsInvalidConfig(t *testing.T) {
	err := InitTracing(context.Background(), config.TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		// Exporter empty — Validate() should fail before we ever touch a real backend.
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "exporter is required") {
		t.Fatalf("expected validation error to bubble up, got: %v", err)
	}
}

func TestBuildSampler_RejectsUnknownSampler(t *testing.T) {
	_, err := buildSampler(config.TracingConfig{SamplerType: "mystery"})
	if err == nil {
		t.Fatal("expected buildSampler to reject unknown sampler type")
	}
	if !strings.Contains(err.Error(), "unsupported sampler_type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestShutdownTracing_Idempotent(t *testing.T) {
	if err := InitTracing(context.Background(), config.TracingConfig{}, WithNoop()); err != nil {
		t.Fatalf("InitTracing(noop) failed: %v", err)
	}
	if err := ShutdownTracing(context.Background()); err != nil {
		t.Fatalf("first ShutdownTracing should succeed: %v", err)
	}
	if err := ShutdownTracing(context.Background()); err != nil {
		t.Fatalf("second ShutdownTracing should succeed: %v", err)
	}
}
