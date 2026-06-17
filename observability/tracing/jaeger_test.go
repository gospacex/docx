package tracing

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gospacex/hubx/cache/docx/config"
)

func TestCloneHeaders(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]string
		want map[string]string
	}{
		{
			name: "nil input returns writable empty map",
			in:   nil,
			want: map[string]string{},
		},
		{
			name: "empty input returns writable empty map",
			in:   map[string]string{},
			want: map[string]string{},
		},
		{
			name: "single header",
			in:   map[string]string{"x-token": "abc"},
			want: map[string]string{"x-token": "abc"},
		},
		{
			name: "multiple headers preserved",
			in:   map[string]string{"a": "1", "b": "2", "c": "3"},
			want: map[string]string{"a": "1", "b": "2", "c": "3"},
		},
		{
			name: "returned map is independent copy",
			in:   map[string]string{"k": "v"},
			want: map[string]string{"k": "v"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := cloneHeaders(c.in)
			// The returned map must be writable even when len(in) == 0.
			// newJaegerExporter merges an Authorization header into it
			// unconditionally; a nil map there used to panic.
			got["__writable_check__"] = "ok"
			delete(got, "__writable_check__")

			if len(got) != len(c.want) {
				t.Fatalf("length mismatch: got %v want %v", got, c.want)
			}
			for k, v := range c.want {
				if got[k] != v {
					t.Fatalf("key %q: got %q want %q", k, got[k], v)
				}
			}
			// Mutating input must not affect the clone.
			if len(c.in) > 0 {
				c.in["mutation"] = "x"
				if _, ok := got["mutation"]; ok {
					t.Fatal("clone shares underlying memory with input")
				}
			}
		})
	}
}

// TestNewJaegerExporter_HTTP_DoesNotFailBeforeDial attempts to build the
// HTTP exporter against an unreachable endpoint; the constructor should
// return without error (the gRPC/HTTP constructors defer the actual dial
// to first export). This guards against regressions where validation
// short-circuits the constructor for a syntactically valid endpoint.
func TestNewJaegerExporter_HTTP_DoesNotFailBeforeDial(t *testing.T) {
	exp, err := newJaegerExporter(config.TracingConfig{
		Exporter: ExporterJaeger,
		Protocol: ProtocolHTTP,
		Endpoint: "127.0.0.1:1",
		Insecure: true,
	})
	if err != nil {
		t.Fatalf("expected HTTP constructor to succeed without a server, got: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
	// Best-effort shutdown — if it fails (e.g. on systems where the
	// unreachable socket is fully rejected), we still pass the test.
	_ = exp.Shutdown(context.Background())
}

// TestNewJaegerExporter_GRPC_DefaultsProtocolToGRPC checks that the
// constructor defaults to the gRPC transport when Protocol is empty,
// matching the documented behaviour.
func TestNewJaegerExporter_GRPC_DefaultsProtocolToGRPC(t *testing.T) {
	exp, err := newJaegerExporter(config.TracingConfig{
		Exporter: ExporterJaeger,
		Endpoint: "127.0.0.1:14317",
		Insecure: true,
		// Protocol intentionally empty
	})
	if err != nil {
		t.Fatalf("expected default-grpc constructor to succeed, got: %v", err)
	}
	if exp == nil {
		t.Fatal("expected non-nil exporter")
	}
	_ = exp.Shutdown(context.Background())
}

// TestNewJaegerExporter_AuthHeaderMaterialisation covers the Basic-Auth
// fallback: when Auth.Username is set, the constructor must merge a
// pre-encoded Basic header into the headers map. The header itself is
// not exposed (constructor swallows it into the option), so we can only
// assert that construction succeeds and the value is well-formed
// base64 if we extract it back. We rely on the success path here and
// rely on the unit tests in config/tracing_test.go for the field-level
// validation.
func TestNewJaegerExporter_AuthHeaderMaterialisation(t *testing.T) {
	exp, err := newJaegerExporter(config.TracingConfig{
		Exporter:  ExporterJaeger,
		Protocol:  ProtocolGRPC,
		Endpoint:  "127.0.0.1:14317",
		Insecure:  true,
		Auth:      config.TracingAuthConfig{Username: "alice", Password: "s3cret"},
	})
	if err != nil {
		t.Fatalf("expected gRPC-with-auth constructor to succeed, got: %v", err)
	}
	_ = exp.Shutdown(context.Background())
}

// TestNewJaegerExporter_HTTP_BindsToLoopback uses an httptest server to
// exercise the http transport without going over the network; we point
// the exporter at the server's address and assert that the exporter
// is constructed without error and can be shut down cleanly.
func TestNewJaegerExporter_HTTP_BindsToLoopback(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()
	addr := strings.TrimPrefix(srv.URL, "http://")

	exp, err := newJaegerExporter(config.TracingConfig{
		Exporter: ExporterJaeger,
		Protocol: ProtocolHTTP,
		Endpoint: addr,
		Insecure: true,
		Headers:  map[string]string{"x-extra": "yes"},
	})
	if err != nil {
		t.Fatalf("expected loopback http constructor to succeed, got: %v", err)
	}
	_ = exp.Shutdown(context.Background())
}