package couchbase

import (
	"context"
	"fmt"
	"time"

	"github.com/couchbase/gocb/v2"
	"github.com/gospacex/hubx/cache/docx/observability"
)

func COS(ctx context.Context, cfg *Config) (*Bucket, error) {
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
		return newBucket(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Bucket), nil
}

func newBucket(ctx context.Context, cfg *Config) (*Bucket, error) {
	cl, err := newCluster(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if cfg.Tracing.Enabled {
		if err := observability.InitTracing(ctx, cfg.Tracing); err != nil {
			return nil, fmt.Errorf("couchbase: %w", err)
		}
	}

	var bucket *gocb.Bucket
	if cfg.Bucket != "" {
		bucket = cl.cluster.Bucket(cfg.Bucket)
		_ = bucket.WaitUntilReady(10e9, nil)
	} else {
		bucket = cl.cluster.Bucket("default")
	}

	return &Bucket{
		Name:    cfg.Bucket,
		cluster: cl,
		bucket:  bucket,
	}, nil
}

func newCluster(ctx context.Context, cfg *Config) (*Cluster, error) {
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
	if err := cluster.WaitUntilReady(10e9, nil); err != nil {
		_ = cluster.Close(nil)
		return nil, fmt.Errorf("couchbase: wait until ready: %w", err)
	}

	return &Cluster{cluster: cluster, cfg: cfg}, nil
}

func durationMS(ms int) time.Duration {
	if ms <= 0 {
		return 10 * time.Second
	}
	return time.Duration(ms) * time.Millisecond
}
