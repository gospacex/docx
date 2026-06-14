package couchbase

import (
	"context"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
)

func COS(ctx context.Context, cfg *Config) (*Bucket, error) {
	if err := cfg.ValidateBucket(); err != nil {
		return nil, err
	}

	bucketKey, err := bucketConfigKey(cfg)
	if err != nil {
		return nil, err
	}
	clusterKey, err := clusterConfigKey(cfg)
	if err != nil {
		return nil, err
	}

	val, err := getOrCreate(ctx, bucketKey, kindBucket, func(ctx context.Context) (any, error) {
		return newBucket(ctx, cfg, bucketKey, clusterKey)
	})
	if err != nil {
		return nil, err
	}
	bucket, ok := val.(*Bucket)
	if !ok {
		return nil, fmt.Errorf("couchbase: cache value for %q is %T, want *Bucket", bucketKey, val)
	}
	return bucket, nil
}

func newBucket(ctx context.Context, cfg *Config, bucketKey, clusterKey string) (*Bucket, error) {
	cl, err := getOrCreateCluster(ctx, cfg, clusterKey)
	if err != nil {
		return nil, err
	}

	bucket := cl.cluster.Bucket(cfg.Bucket)
	if err := bucket.WaitUntilReady(10*time.Second, nil); err != nil {
		return nil, fmt.Errorf("couchbase: bucket wait until ready: %w", err)
	}

	return &Bucket{
		Name:     cfg.Bucket,
		cluster:  cl,
		bucket:   bucket,
		cacheKey: bucketKey,
	}, nil
}

func openCluster(ctx context.Context, cfg *Config) (*Cluster, error) {
	if err := cfg.ValidateCluster(); err != nil {
		return nil, err
	}
	key, err := clusterConfigKey(cfg)
	if err != nil {
		return nil, err
	}
	return getOrCreateCluster(ctx, cfg, key)
}

func getOrCreateCluster(ctx context.Context, cfg *Config, key string) (*Cluster, error) {
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

func newCluster(ctx context.Context, cfg *Config, clusterKey string) (*Cluster, error) {
	_ = ctx
	uri := cfg.Endpoints[0]
	opts := gocb.ClusterOptions{
		Username: cfg.Username,
		Password: cfg.Password,
	}
	if cfg.ConnectTimeout > 0 {
		opts.TimeoutsConfig.ConnectTimeout = durationMS(cfg.ConnectTimeout)
	}
	if cfg.SocketTimeout > 0 {
		opts.TimeoutsConfig.KVTimeout = durationMS(cfg.SocketTimeout)
	}

	cluster, err := gocb.Connect(uri, opts)
	if err != nil {
		return nil, fmt.Errorf("couchbase: connect: %w", err)
	}
	if err := cluster.WaitUntilReady(10*time.Second, nil); err != nil {
		_ = cluster.Close(nil)
		return nil, fmt.Errorf("couchbase: wait until ready: %w", err)
	}

	return &Cluster{cluster: cluster, cfg: cfg, cacheKey: clusterKey}, nil
}

func clusterConfigKey(cfg *Config) (string, error) {
	normalized := *cfg
	normalized.Bucket = ""
	fp, err := normalized.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return "cluster:" + fp, nil
}

func bucketConfigKey(cfg *Config) (string, error) {
	fp, err := cfg.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return "bucket:" + fp, nil
}

func clusterFileKey(absPath string, cfg *Config) (string, error) {
	normalized := *cfg
	normalized.Bucket = ""
	fp, err := normalized.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("cluster:file:%s:%s", absPath, fp), nil
}

func bucketFileKey(absPath string, cfg *Config) (string, error) {
	fp, err := cfg.CacheFingerprint()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("bucket:file:%s:%s", absPath, fp), nil
}

func durationMS(ms int) time.Duration {
	if ms <= 0 {
		return 10 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}
