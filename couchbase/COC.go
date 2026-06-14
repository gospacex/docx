package couchbase

import (
	"context"
	"fmt"
)

func COC(ctx context.Context, cfg *Config) (*Cluster, error) {
	if err := cfg.ValidateCluster(); err != nil {
		return nil, err
	}

	key, err := clusterConfigKey(cfg)
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
