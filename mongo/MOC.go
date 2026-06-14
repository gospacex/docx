package mongo

import (
	"context"
	"fmt"

	"github.com/gospacex/hubx/cache/docx/observability"
)

func MOC(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("mongo: config is nil")
	}
	if cfg.URI == "" {
		return nil, fmt.Errorf("mongo: URI is required")
	}

	key := cfg.ContentHash()
	if key == "" {
		key = cfg.URI
	}

	val, err := getOrCreate(key, func() (interface{}, error) {
		return newClientWithTracing(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Client), nil
}

func newClientWithTracing(ctx context.Context, cfg *Config) (*Client, error) {
	if cfg.Tracing.Enabled {
		if err := observability.InitTracing(ctx, cfg.Tracing); err != nil {
			return nil, fmt.Errorf("mongo: %w", err)
		}
	}
	return newClient(ctx, cfg)
}
