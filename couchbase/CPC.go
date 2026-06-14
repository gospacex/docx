package couchbase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func CPC(ctx context.Context, configPath string) (*Cluster, error) {
	if configPath == "" {
		return nil, fmt.Errorf("couchbase: config path is empty")
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("couchbase: %w", err)
	}
	if err := validateConfigPath(absPath); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("couchbase: read config: %w", err)
	}

	cfg, err := ParseConfig(content)
	if err != nil {
		return nil, err
	}
	if err := cfg.ValidateCluster(); err != nil {
		return nil, err
	}

	key, err := clusterFileKey(absPath, cfg)
	if err != nil {
		return nil, err
	}

	val, err := getOrCreate(ctx, key, kindCluster, func(ctx context.Context) (any, error) {
		return newCluster(ctx, cfg, key)
	})
	if err != nil {
		return nil, err
	}
	cluster, ok := val.(*Cluster)
	if !ok {
		return nil, fmt.Errorf("couchbase: cache value for %q is %T, want *Cluster", key, val)
	}
	return cluster, nil
}
