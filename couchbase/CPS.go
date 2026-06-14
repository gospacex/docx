package couchbase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func CPS(ctx context.Context, configPath string) (*Bucket, error) {
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
	if err := cfg.ValidateBucket(); err != nil {
		return nil, err
	}

	bucketKey, err := bucketFileKey(absPath, cfg)
	if err != nil {
		return nil, err
	}
	clusterKey, err := clusterFileKey(absPath, cfg)
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

func validateConfigPath(absPath string) error {
	ext := filepath.Ext(absPath)
	switch ext {
	case ".yaml", ".yml":
		return nil
	default:
		return fmt.Errorf("couchbase: unsupported config extension %q (use .yaml or .yml)", ext)
	}
}
