package tracing

import (
	"strings"
	"testing"

	"github.com/gospacex/hubx/cache/docx/config"
)

func TestNewExporter_Jaeger_InvalidEndpoint(t *testing.T) {
	// Control characters in the endpoint are rejected by gRPC's URL parser,
	// causing otlptracegrpc.New to return a synchronous error.
	_, err := NewExporter(config.TracingConfig{
		Exporter:  ExporterJaeger,
		Protocol:  ProtocolGRPC,
		Endpoint: "\n",
		Insecure: true,
	})
	if err == nil {
		t.Fatal("expected error for invalid Jaeger endpoint")
	}
	if !strings.Contains(err.Error(), "tracing: jaeger:") {
		t.Fatalf("error should wrap tracing:jaeger:, got: %v", err)
	}
}

func TestNewExporter_Jaeger_UnsupportedProtocol(t *testing.T) {
	_, err := NewExporter(config.TracingConfig{
		Exporter:  ExporterJaeger,
		Protocol:  "tcp",
		Endpoint: "localhost:4317",
		Insecure: true,
	})
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
	if !strings.Contains(err.Error(), "unsupported protocol") {
		t.Fatalf("expected unsupported protocol error, got: %v", err)
	}
}

func TestNewExporter_UnknownExporter(t *testing.T) {
	_, err := NewExporter(config.TracingConfig{
		Exporter: "unknown_exporter",
	})
	if err == nil {
		t.Fatal("expected error for unknown exporter")
	}
	expected := `tracing: unknown exporter "unknown_exporter"`
	if !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error containing %q, got: %v", expected, err)
	}
}
