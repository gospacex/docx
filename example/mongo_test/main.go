package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gospacex/hubx/cache/docx/config"
	"github.com/gospacex/hubx/cache/docx/observability"
	mongo "github.com/gospacex/hubx/cache/mongo"
)

func main() {
	log.Println("=== MongoDB Example (MOS/MOC/MPS/MPC) ===")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := &mongo.Config{
		URI:            "mongodb://localhost:27017",
		Database:       "testdb",
		Collection:     "users",
		ConnectTimeout: 10000,
		MaxPoolSize:    100,
		Tracing: config.TracingConfig{
			Enabled:     true,
			ServiceName: "mongo-example",
			Exporter:    "jaeger",
			Endpoint:    "localhost:4317",
			Protocol:    "grpc",
		},
	}

	if err := observability.InitTracing(ctx, cfg.Tracing); err != nil {
		log.Fatalf("InitTracing failed: %v", err)
	}
	defer func() {
		if err := observability.ShutdownTracing(ctx); err != nil {
			log.Printf("ShutdownTracing failed: %v", err)
		}
	}()

	client, err := mongo.MOC(ctx, cfg)
	if err != nil {
		log.Fatalf("MOC failed: %v", err)
	}
	log.Println("[MOC] client connected")

	if err := client.HealthCheck(ctx); err != nil {
		log.Printf("[MOC] health check failed: %v", err)
	} else {
		log.Println("[MOC] health check OK")
	}

	coll := client.Collection(cfg.Database, cfg.Collection)
	log.Printf("[MOC] collection=%s ready", coll.Name)

	_, err = mongo.InsertTrace(ctx, coll, map[string]interface{}{
		"name":  "alice",
		"email": "alice@example.com",
	})
	if err != nil {
		log.Printf("[MOC] InsertTrace failed: %v", err)
	} else {
		log.Println("[MOC] InsertTrace OK")
	}

	cursor, err := mongo.FindTrace(ctx, coll, map[string]interface{}{"name": "alice"})
	if err != nil {
		log.Printf("[MOC] FindTrace failed: %v", err)
	} else {
		cursor.Close(ctx)
		log.Println("[MOC] FindTrace OK")
	}

	singleColl, err := mongo.MOS(ctx, cfg)
	if err != nil {
		log.Printf("[MOS] failed: %v", err)
	} else {
		log.Printf("[MOS] collection=%s ready", singleColl.Name)
	}

	psClient, err := mongo.MPC(ctx, "mongo.yaml")
	if err != nil {
		log.Printf("[MPC] failed: %v (expected without server)", err)
	} else {
		log.Printf("[MPC] client connected from config file")
		_ = psClient
	}

	psColl, err := mongo.MPS(ctx, "mongo.yaml")
	if err != nil {
		log.Printf("[MPS] failed: %v (expected without server)", err)
	} else {
		log.Printf("[MPS] collection=%s ready from config file", psColl.Name)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigCh:
		log.Println("interrupted")
	case <-time.After(3 * time.Second):
		log.Println("demo timeout")
	}

	if err := client.Close(ctx); err != nil {
		log.Printf("close error: %v", err)
	}
	log.Println("=== MongoDB Example done ===")
}
