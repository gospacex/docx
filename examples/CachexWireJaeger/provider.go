// Package main wires cachex into a Redis client and exposes it to go test.
package main

import (
	"context"
	"fmt"

	"github.com/gospacex/cachex"
	"github.com/gospacex/cachex/drivers/redisx"
	cachexInitx "github.com/gospacex/cachex/initx"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// ProvideRedisClient reaches into the driver directly via GetSingle,
// which causes drivers/redisx's init() to run and register the global
// pool used by the cachex.Cache high-level API.

// CachexProvider loads cachex configs and produces native Redis handles
// and OTel tracer providers. It is stateless and safe for concurrent use.
type CachexProvider struct{}

// NewCachexProvider returns a fresh CachexProvider for wire injection.
func NewCachexProvider() *CachexProvider { return &CachexProvider{} }

// ProvideRedisClient loads cfgPath via cachex.LoadConfig and returns the
// pooled native *redis.Client. The client is reused across callers that
// supply structurally equivalent *cachex.Config (config fingerprint).
func (p *CachexProvider) ProvideRedisClient(cfgPath string) (*redis.Client, error) {
	if cfgPath == "" {
		return nil, fmt.Errorf("provide redis client: cfgPath is empty")
	}
	cfg, err := cachex.LoadConfig(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("provide redis client: load cachex config %q: %w", cfgPath, err)
	}
	cli, err := redisx.GetSingle(cfg)
	if err != nil {
		return nil, fmt.Errorf("provide redis client: redisx.GetSingle addrs=%v: %w", cfg.Addrs, err)
	}
	// Attach the OTel tracing hook so every command emits a span into
	// the global TracerProvider that cachexInitx.InitTracing set up.
	// Without this, *redis.Client is invisible to the trace pipeline.
	if err := redisotel.InstrumentTracing(cli); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("provide redis client: redisotel.InstrumentTracing: %w", err)
	}
	return cli, nil
}

// SetupTracing reads the trace: block from cfgPath and initialises OTel
// globals (TracerProvider + Propagators). Returned cleanup flushes and
// shuts down the exporter; safe to call multiple times.
func SetupTracing(ctx context.Context, cfgPath string) (func(context.Context), error) {
	return cachexInitx.InitTracing(ctx, cfgPath)
}
