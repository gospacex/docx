package mongo

import (
	"context"
	"fmt"
)

func MOC(ctx context.Context, cfg *Config) (*Client, error) {
	if err := cfg.ValidateClient(); err != nil {
		return nil, err
	}

	key, err := clientConfigKey(cfg)
	if err != nil {
		return nil, err
	}

	val, err := getOrCreate(ctx, key, kindClient, func(ctx context.Context) (any, error) {
		return newClient(ctx, cfg, key)
	})
	if err != nil {
		return nil, err
	}
	client, ok := val.(*Client)
	if !ok {
		return nil, fmt.Errorf("mongo: cache value for %q is %T, want *Client", key, val)
	}
	return client, nil
}
