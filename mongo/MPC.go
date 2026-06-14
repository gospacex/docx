package mongo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func MPC(ctx context.Context, configPath string) (*Client, error) {
	if configPath == "" {
		return nil, fmt.Errorf("mongo: config path is empty")
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("mongo: %w", err)
	}
	if err := validateConfigPath(absPath); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("mongo: read config: %w", err)
	}

	cfg, err := ParseConfig(content)
	if err != nil {
		return nil, err
	}
	if err := cfg.ValidateClient(); err != nil {
		return nil, err
	}

	key, err := clientFileKey(absPath, cfg)
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
