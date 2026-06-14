package couchbase

import (
	"context"
	"fmt"

	"github.com/gospacex/hubx/cache/docx/observability"
)

func COC(ctx context.Context, cfg *Config) (*Cluster, error) {
	if cfg == nil {
		return nil, fmt.Errorf("couchbase: config is nil")
	}
	if len(cfg.Endpoints) == 0 {
		return nil, fmt.Errorf("couchbase: endpoints is required")
	}

	key := cfg.ContentHash()
	if key == "" {
		key = fmt.Sprintf("%s|%s", cfg.Endpoints[0], cfg.Bucket)
	}

	val, err := getOrCreate(key, func() (interface{}, error) {
		return newClusterWithTracing(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Cluster), nil
}

func newClusterWithTracing(ctx context.Context, cfg *Config) (*Cluster, error) {
	if cfg.Tracing.Enabled {
		if err := observability.InitTracing(ctx, cfg.Tracing); err != nil {
			return nil, fmt.Errorf("couchbase: %w", err)
		}
	}
	return newCluster(ctx, cfg)
}
