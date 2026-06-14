package mongo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func MPC(ctx context.Context, configPath string) (*Client, error) {
	if configPath == "" {
		return nil, fmt.Errorf("mongo: config path is empty")
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("mongo: %w", err)
	}

	base := strings.ToLower(filepath.Base(absPath))
	if !strings.Contains(base, "mongo") {
		return nil, fmt.Errorf("mongo: config %q does not appear to be a Mongo config", absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("mongo: read config: %w", err)
	}

	cfg, err := ParseConfig(content)
	if err != nil {
		return nil, err
	}

	key := absPath
	val, err := getOrCreate(key, func() (interface{}, error) {
		return newClientWithTracing(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Client), nil
}
