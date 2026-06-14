package couchbase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	key := absPath
	val, err := getOrCreate(key, func() (interface{}, error) {
		return newBucket(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Bucket), nil
}

func validateConfigPath(absPath string) error {
	base := strings.ToLower(filepath.Base(absPath))
	if !strings.Contains(base, "couch") {
		return fmt.Errorf("couchbase: config %q does not appear to be a Couchbase config", absPath)
	}

	ext := filepath.Ext(absPath)
	switch ext {
	case ".yaml", ".yml":
		return nil
	default:
		return fmt.Errorf("couchbase: unsupported config extension %q (use .yaml or .yml)", ext)
	}
}
