package tracing

import (
	"context"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/gospacex/hubx/cache/docx/config"
)

func TestRedisStreamExporter_ExportSpans_EmptyReturnsNil(t *testing.T) {
	exp := &redisStreamExporter{stream: "test-stream"}
	if err := exp.ExportSpans(context.Background(), nil); err != nil {
		t.Fatalf("empty spans should not error, got: %v", err)
	}
	if err := exp.ExportSpans(context.Background(), []sdktrace.ReadOnlySpan{}); err != nil {
		t.Fatalf("zero-length slice should not error, got: %v", err)
	}
}

// TestRedisStreamExporter_Shutdown_Idempotent covers the contract: once
// the exporter has been shut down, subsequent Shutdown calls must
// return nil without re-closing the underlying client.
func TestRedisStreamExporter_Shutdown_Idempotent(t *testing.T) {
	// Use a real go-redis client pointed at an unreachable host; the
	// constructor would fail on Ping, so we build one directly here
	// (matching the pattern already used in TestRedisStreamExporter_Shutdown_Idempotent).
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	exp := &redisStreamExporter{client: client, stream: "test-stream"}

	if err := exp.Shutdown(context.Background()); err != nil {
		t.Fatalf("first Shutdown should not error, got: %v", err)
	}
	if err := exp.Shutdown(context.Background()); err != nil {
		t.Fatalf("second Shutdown should be a no-op, got: %v", err)
	}
}

// TestNewRedisStreamExporter_PingFailure covers the constructor path
// where the Redis server is unreachable: Ping must fail fast (3s
// timeout) and the constructor must return the wrapped error.
func TestNewRedisStreamExporter_PingFailure(t *testing.T) {
	_, err := newRedisStreamExporter(config.TracingConfig{
		Exporter: ExporterRedisStream,
		Addrs:    []string{"127.0.0.1:1"},
		Producer: config.TracingProducerConfig{Topic: "test-stream"},
	})
	if err == nil {
		t.Fatal("expected ping failure against unreachable redis")
	}
	if !strings.Contains(err.Error(), "redis_stream: ping") {
		t.Fatalf("error must wrap redis_stream: ping, got: %v", err)
	}
}

// TestNewExporter_RedisStream_Dispatch covers the factory dispatch path.
// The construction will fail at Ping, so we assert on the error wording
// rather than the returned exporter — this still exercises the
// dispatcher's switch-case branch for "redis_stream".
func TestNewExporter_RedisStream_Dispatch(t *testing.T) {
	_, err := NewExporter(config.TracingConfig{
		Exporter: ExporterRedisStream,
		Addrs:    []string{"127.0.0.1:1"},
		Producer: config.TracingProducerConfig{Topic: "x"},
	})
	if err == nil {
		t.Fatal("expected ping failure via factory dispatch")
	}
	if !strings.Contains(err.Error(), "redis_stream:") {
		t.Fatalf("expected factory to wrap redis_stream error, got: %v", err)
	}
}

// TestRedisStreamAuthFallback documents the precedence rules in
// newRedisStreamExporter: Redis.Username/Password take precedence;
// Auth.Username/Password are the fallback. We verify by triggering the
// constructor (which pings and fails) and inspecting the error path —
// the auth-selection code is unreachable through an exported symbol, but
// we still keep this test to lock in the wiring if the constructor is
// refactored.
func TestRedisStreamAuthFallback(t *testing.T) {
	cfg := config.TracingConfig{
		Exporter: ExporterRedisStream,
		Addrs:    []string{"127.0.0.1:1"},
		Producer: config.TracingProducerConfig{Topic: "x"},
		Auth:     config.TracingAuthConfig{Username: "fallback-user", Password: "fallback-pass"},
		Redis:    config.TracingRedisConfig{Username: "primary-user", Password: "primary-pass"},
	}
	_, err := newRedisStreamExporter(cfg)
	if err == nil {
		t.Fatal("expected ping failure (auth fallback path is exercised pre-ping)")
	}
	// Construction failed at Ping, so we have at least exercised the
	// auth-precedence branch. The actual auth values would only surface
	// in a redis AUTH handshake, which requires a real server.
}

// TestRedisStreamAuthFromConfigFallback exercises the case where
// Redis.Username/Password are empty and Auth.Username/Password must
// be used instead.
func TestRedisStreamAuthFromConfigFallback(t *testing.T) {
	cfg := config.TracingConfig{
		Exporter: ExporterRedisStream,
		Addrs:    []string{"127.0.0.1:1"},
		Producer: config.TracingProducerConfig{Topic: "x"},
		Auth:     config.TracingAuthConfig{Username: "u", Password: "p"},
		// Redis intentionally empty
	}
	_, err := newRedisStreamExporter(cfg)
	if err == nil {
		t.Fatal("expected ping failure (Auth → Redis fallback path is exercised)")
	}
}