// 02-kafka-topic — demonstrates emitting OTel spans as JSON messages to a
// Kafka topic via docx's self-built Kafka topic SpanExporter.
//
// Run:
//
//	docker run -d --name kafka -p 9092:9092 \
//	  apache/kafka:3.7.0
//	go run .
//	kafka-console-consumer --bootstrap-server localhost:9092 \
//	  --topic otel-traces --from-beginning
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
	log.Println("=== Couchbase + Kafka Topic Tracing Example ===")

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
			ServiceName: "couchbase-kafka-topic-example",
			Exporter:    "kafka_topic",
			Addrs:       []string{envOr("KAFKA_BROKER", "localhost:9092")},
			Producer: config.TracingProducerConfig{
				Topic:      envOr("KAFKA_TRACE_TOPIC", "otel-traces"),
				Acks:       "all",
				Idempotent: true,
			},
			SamplerRatio: 1.0,
		},
	}

	bucket, err := couchbase.COS(ctx, cfg)
	if err != nil {
		log.Fatalf("COS failed: %v", err)
	}
	log.Printf("[COS] bucket=%s connected", bucket.Name)

	// Drive a few CRUD ops; each call goes through docx/tracing.GetTrace
	// which opens an OTel span, gets batched by the SDK, and finally emitted
	// as a JSON record to the Kafka topic.
	for i := 0; i < 3; i++ {
		key := "user:" + time.Now().Format("150405.000000")
		_, _ = couchbase.InsertTrace(ctx, bucket, key, map[string]any{
			"id":   i,
			"name": "alice",
		})
		_, _ = couchbase.GetTrace(ctx, bucket, key)
		_, _ = couchbase.DeleteTrace(ctx, bucket, key)
	}

	// Give the BatchSpanProcessor a moment to flush before we exit.
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
