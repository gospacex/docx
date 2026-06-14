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
	log.Println("=== Couchbase Example (COS/COC/CPS/CPC) ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &couchbase.Config{
		Endpoints: []string{"localhost:8091"},
		Bucket:    "default",
		Username:  "Administrator",
		Password:  "password",
		ConnectTimeout: 10000,
		SocketTimeout:  5000,
		Tracing: config.TracingConfig{
			Enabled:     true,
			ServiceName: "couchbase-example",
			Exporter:    "jaeger",
			Endpoint:    "localhost:4317",
			Protocol:    "grpc",
		},
	}

	bucket, err := couchbase.COS(ctx, cfg)
	if err != nil {
		log.Fatalf("COS failed: %v", err)
	}
	log.Printf("[COS] bucket=%s connected", bucket.Name)

	_, err = bucket.Upsert("user:1", map[string]interface{}{
		"name":  "alice",
		"email": "alice@example.com",
	})
	if err != nil {
		log.Printf("[COS] Upsert failed: %v", err)
	} else {
		log.Println("[COS] Upsert user:1 OK")
	}

	result, err := couchbase.GetTrace(ctx, bucket, "user:1")
	if err != nil {
		log.Printf("[COS] GetTrace failed: %v", err)
	} else {
		log.Printf("[COS] GetTrace user:1 -> %v", result.Content)
	}

	cluster, err := couchbase.COC(ctx, cfg)
	if err != nil {
		log.Printf("[COC] failed: %v (continuing)", err)
	} else {
		log.Printf("[COC] cluster connected")
		ping, err := cluster.Ping()
		if err != nil {
			log.Printf("[COC] Ping failed: %v", err)
		} else {
			log.Printf("[COC] Ping OK: %d services", len(ping.Services))
		}
	}

	psBucket, err := couchbase.CPS(ctx, "couchbase.yaml")
	if err != nil {
		log.Printf("[CPS] failed: %v (expected without server)", err)
	} else {
		log.Printf("[CPS] bucket=%s connected from config file", psBucket.Name)
	}

	_, err = couchbase.CPC(ctx, "couchbase.yaml")
	if err != nil {
		log.Printf("[CPC] failed: %v (expected without server)", err)
	} else {
		log.Println("[CPC] cluster connected from config file")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("interrupted")
	case <-time.After(3 * time.Second):
		log.Println("demo timeout")
	}

	if cluster != nil {
		_ = cluster.Close()
		log.Println("[COC] cluster closed")
	}
	log.Println("=== Couchbase Example done ===")
}