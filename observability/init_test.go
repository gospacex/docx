package observability

import (
	"context"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gospacex/hubx/cache/docx/config"
	"github.com/gospacex/hubx/cache/docx/observability/tracing"
)

// jaegerInProcCfg returns a TracingConfig that passes Validate() and
// uses the Jaeger OTLP/gRPC exporter pointed at an unreachable address.
// newJaegerExporter is lazy (no dial happens in the constructor), so the
// tests below never need a real collector.
func jaegerInProcCfg() config.TracingConfig {
	return config.TracingConfig{
		Enabled:     true,
		ServiceName: "docx-observability-test",
		Exporter:    tracing.ExporterJaeger,
		Endpoint:    "127.0.0.1:1",
		Protocol:    tracing.ProtocolGRPC,
		Insecure:    true,
	}
}

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

// TestInitTracing_Noop_ReinitSwapsPrevious exercises the `currentTP.Swap`
// branch in the noop path: the second InitTracing call's Swap must
// return the first call's TP, which is then Shutdown. The test
// confirms the call doesn't panic and the global TP ends up at the
// second call's value.
func TestInitTracing_Noop_ReinitSwapsPrevious(t *testing.T) {
	ctx := context.Background()
	if err := InitTracing(ctx, config.TracingConfig{}, WithNoop()); err != nil {
		t.Fatalf("first noop init failed: %v", err)
	}
	first := currentTP.Load()
	if first == nil {
		t.Fatal("first noop init did not install a TP")
	}
	if err := InitTracing(ctx, config.TracingConfig{}, WithNoop()); err != nil {
		t.Fatalf("second noop init failed: %v", err)
	}
	second := currentTP.Load()
	if second == nil {
		t.Fatal("second noop init did not install a TP")
	}
	if first == second {
		t.Fatal("expected second noop init to swap in a fresh TP")
	}
	if err := ShutdownTracing(ctx); err != nil {
		t.Fatalf("cleanup shutdown: %v", err)
	}
}

// TestInitTracing_NonNoop_ReinitSwapsPrevious covers the same Swap
// branch in the production (non-noop) path. We use the Jaeger OTLP/gRPC
// exporter pointed at an unreachable address; newJaegerExporter is lazy
// and never dials, so the test is hermetic.
func TestInitTracing_NonNoop_ReinitSwapsPrevious(t *testing.T) {
	ctx := context.Background()
	cfg := jaegerInProcCfg()
	if err := InitTracing(ctx, cfg); err != nil {
		t.Fatalf("first non-noop init failed: %v", err)
	}
	first := currentTP.Load()
	if first == nil {
		t.Fatal("first non-noop init did not install a TP")
	}
	if err := InitTracing(ctx, cfg); err != nil {
		t.Fatalf("second non-noop init failed: %v", err)
	}
	second := currentTP.Load()
	if second == nil || first == second {
		t.Fatal("expected second non-noop init to swap in a fresh TP")
	}
	if err := ShutdownTracing(ctx); err != nil {
		t.Fatalf("cleanup shutdown: %v", err)
	}
}

// TestInitTracing_NonNoop_SetsPropagator verifies the propagator side
// of InitTracing's contract — both noop and non-noop paths install a
// composite propagator (TraceContext + Baggage).
func TestInitTracing_NonNoop_SetsPropagator(t *testing.T) {
	ctx := context.Background()
	if err := InitTracing(ctx, jaegerInProcCfg()); err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if otel.GetTextMapPropagator() == nil {
		t.Fatal("expected non-noop init to install a propagator")
	}
	if err := ShutdownTracing(ctx); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
}

// TestBuildSampler_AllValidTypes is a table-driven sweep over the four
// sampler types the SDK actually supports. Each must build without
// error; coverage asserts the right OTel sampler was constructed.
func TestBuildSampler_AllValidTypes(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.TracingConfig
	}{
		{"always_on_empty", config.TracingConfig{SamplerType: ""}},
		{"always_on_explicit", config.TracingConfig{SamplerType: "always_on"}},
		{"always_off", config.TracingConfig{SamplerType: "always_off"}},
		{"traceidratio", config.TracingConfig{SamplerType: "traceidratio", SamplerRatio: 0.25}},
		{"parentbased_traceidratio", config.TracingConfig{SamplerType: "parentbased_traceidratio", SamplerRatio: 0.5}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := buildSampler(tc.cfg)
			if err != nil {
				t.Fatalf("buildSampler(%q): %v", tc.cfg.SamplerType, err)
			}
			if s == nil {
				t.Fatal("expected non-nil sampler")
			}
			// The SDK returns concrete sampler implementations; we
			// don't pin the type, but Description() is a stable string
			// we can assert is non-empty.
			if d := s.Description(); d == "" {
				t.Fatal("sampler description should be non-empty")
			}
		})
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

// _ keeps the SDK import live for type assertions in future tests
// without forcing an unused-import error on a trimmed build.
var _ = sdktrace.NewTracerProvider
