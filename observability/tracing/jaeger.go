package tracing

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/url"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gospacex/hubx/cache/docx/config"
)

// newJaegerExporter builds an OTLP gRPC/HTTP exporter for an OTel collector
// (Jaeger 1.35+ ingests OTLP natively; for older Jaeger deployments point
// Endpoint at an otel-collector that forwards to jaeger-thrift).
//
// Protocol is selected from cfg.Protocol ("grpc" | "http"); grpc is the
// default. If both cfg.Headers["authorization"] and cfg.Auth.Username are
// set, the username/password pair is materialised as a Basic Auth header
// and merged into the headers map (password wins over an existing
// authorization header to keep the operator in control of credentials).
func newJaegerExporter(cfg config.TracingConfig) (sdktrace.SpanExporter, error) {
	if _, err := url.Parse("otlp://" + cfg.Endpoint); err != nil {
		return nil, fmt.Errorf("tracing: jaeger: invalid endpoint %q: %w", cfg.Endpoint, err)
	}

	headers := cloneHeaders(cfg.Headers)
	if cfg.Auth.Username != "" {
		token := base64.StdEncoding.EncodeToString([]byte(cfg.Auth.Username + ":" + cfg.Auth.Password))
		headers["authorization"] = "Basic " + token
	}

	ctx := context.Background()
	switch cfg.Protocol {
	case ProtocolHTTP:
		opts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracehttp.WithHeaders(headers))
		}
		exp, err := otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("tracing: jaeger: http: %w", err)
		}
		return exp, nil
	case ProtocolGRPC, "":
		opts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(cfg.Endpoint)}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithTLSCredentials(insecure.NewCredentials()))
		}
		if len(headers) > 0 {
			opts = append(opts, otlptracegrpc.WithHeaders(headers))
		}
		exp, err := otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("tracing: jaeger: grpc: %w", err)
		}
		return exp, nil
	default:
		return nil, fmt.Errorf("tracing: jaeger: unsupported protocol %q", cfg.Protocol)
	}
}

// _ ensures otlptrace package symbol is reachable for future use without
// breaking imports when only one transport is wired.
var _ = otlptrace.New

func cloneHeaders(in map[string]string) map[string]string {
	// Always allocate so the caller can safely write into the returned
	// map (e.g. merging an Authorization header below). Returning nil
	// here previously caused a nil-map-write panic in newJaegerExporter
	// whenever Auth.Username was set without Headers.
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
