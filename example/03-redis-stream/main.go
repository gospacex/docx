// 03-redis-stream — demonstrates emitting OTel spans as entries to a
// Redis Stream via docx's self-built Redis Stream SpanExporter.
//
// Run:
//
//	docker run -d --name redis -p 6379:6379 redis:7-alpine
//	go run .
//	redis-cli XREAD COUNT 5 BLOCK 0 STREAMS otel-traces '$'
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	couchbase "github.com/gospacex/hubx/cache/couchbase"
	"github.com/gospacex/hubx/cache/docx/config"
)

func main() {
	log.Println("=== Couchbase + Redis Stream Tracing Example ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &couchbase.Config{
		Endpoints:      []string{envOr("CB_ENDPOINT", "localhost:8091")},
		Bucket:         envOr("CB_BUCKET", "default"),
		Username:       envOr("CB_USERNAME", "Administrator"),
		Password:       envOr("CB_PASSWORD", "password"),
		ConnectTimeout: 10000,
		SocketTimeout:  5000,
		Tracing: config.TracingConfig{
			Enabled:     true,
			ServiceName: "couchbase-redis-stream-example",
			Exporter:    "redis_stream",
			Addrs:       []string{envOr("REDIS_ADDR", "localhost:6379")},
			Producer: config.TracingProducerConfig{
				// Producer.Topic doubles as the stream name for redis_stream.
				Topic: envOr("REDIS_STREAM", "otel-traces"),
			},
			Auth: config.TracingAuthConfig{
				Username: envOr("REDIS_USERNAME", ""),
				Password: envOr("REDIS_PASSWORD", ""),
			},
			Redis: config.TracingRedisConfig{
				DB:       0,
				PoolSize: 10,
			},
			SamplerRatio: 1.0,
		},
	}

	bucket, err := couchbase.COS(ctx, cfg)
	if err != nil {
		log.Fatalf("COS failed: %v", err)
	}
	log.Printf("[COS] bucket=%s connected", bucket.Name)

	for i := 0; i < 3; i++ {
		key := "user:" + time.Now().Format("150405.000000")
		_, _ = couchbase.InsertTrace(ctx, bucket, key, map[string]any{"id": i})
		_, _ = couchbase.GetTrace(ctx, bucket, key)
		_, _ = couchbase.DeleteTrace(ctx, bucket, key)
	}

	time.Sleep(2 * time.Second)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
	case <-time.After(2 * time.Second):
	}
	log.Println("=== done ===")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
