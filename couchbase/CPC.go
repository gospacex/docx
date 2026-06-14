package couchbase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func CPC(ctx context.Context, configPath string) (*Cluster, error) {
	if configPath == "" {
		return nil, fmt.Errorf("couchbase: config path is empty")
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("couchbase: %w", err)
	}

	base := strings.ToLower(filepath.Base(absPath))
	if !strings.Contains(base, "couch") {
		return nil, fmt.Errorf("couchbase: config %q does not appear to be a Couchbase config", absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("couchbase: read config: %w", err)
	}

	cfg, err := ParseConfig(content)
	if err != nil {
		return nil, err
	}

	key := absPath
	val, err := getOrCreate(key, func() (interface{}, error) {
		return newClusterWithTracing(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Cluster), nil
}
