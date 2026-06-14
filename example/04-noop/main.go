// 04-noop — demonstrates the WithNoop() escape hatch: tracing is enabled
// at the API surface (so span() calls still work) but no exporter is
// ever wired up, so no spans are batched or sent anywhere.
//
// This is the recommended configuration for unit tests and for
// short-lived CLI tools that want OTel API compatibility without the
// cost of standing up a collector.
package main

import (
	"context"
	"log"

	couchbase "github.com/gospacex/hubx/cache/couchbase"
	"github.com/gospacex/hubx/cache/docx/config"
	"github.com/gospacex/hubx/cache/docx/observability"
)

func main() {
	log.Println("=== Couchbase + Noop Tracing Example ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Enabled: true so the rest of the program can call StartSpan / GetTrace
	// without conditional plumbing; WithNoop() swaps in an in-memory
	// TracerProvider that drops everything on the floor.
	if err := observability.InitTracing(ctx, config.TracingConfig{
		Enabled:     true,
		ServiceName: "noop-example",
		Exporter:    "jaeger", // ignored under WithNoop
		Endpoint:    "nowhere:4317",
	}, observability.WithNoop()); err != nil {
		log.Fatalf("InitTracing(noop) failed: %v", err)
	}
	defer func() {
		if err := observability.ShutdownTracing(ctx); err != nil {
			log.Printf("ShutdownTracing(noop) failed: %v", err)
		}
	}()
	log.Println("[noop] InitTracing ok; spans will be discarded")

	cfg := &couchbase.Config{
		Endpoints:      []string{"localhost:8091"},
		Bucket:         "default",
		Username:       "Administrator",
		Password:       "password",
		ConnectTimeout: 10000,
		SocketTimeout:  5000,
		Tracing: config.TracingConfig{
			// Connection opening no longer auto-initializes tracing; we keep
			// tracing config disabled here because the noop TracerProvider has
			// already been installed explicitly above.
			Enabled: false,
		},
	}

	bucket, err := couchbase.COS(ctx, cfg)
	if err != nil {
		// Expected when no Couchbase is running — log and move on.
		log.Printf("[noop] COS failed (ok in unit tests): %v", err)
		return
	}

	_, _ = couchbase.InsertTrace(ctx, bucket, "user:1", map[string]any{"k": "v"})
	_, _ = couchbase.GetTrace(ctx, bucket, "user:1")
	log.Println("[noop] done")
}
