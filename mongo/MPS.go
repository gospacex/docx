package mongo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func MPS(ctx context.Context, configPath string) (*Collection, error) {
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
	if err := cfg.ValidateCollection(); err != nil {
		return nil, err
	}

	collectionKey, err := collectionFileKey(absPath, cfg)
	if err != nil {
		return nil, err
	}
	clientKey, err := clientFileKey(absPath, cfg)
	if err != nil {
		return nil, err
	}

	val, err := getOrCreate(ctx, collectionKey, kindCollection, func(ctx context.Context) (any, error) {
		return newCollection(ctx, cfg, collectionKey, clientKey)
	})
	if err != nil {
		return nil, err
	}
	coll, ok := val.(*Collection)
	if !ok {
		return nil, fmt.Errorf("mongo: cache value for %q is %T, want *Collection", collectionKey, val)
	}
	return coll, nil
}

func validateConfigPath(absPath string) error {
	ext := filepath.Ext(absPath)
	switch ext {
	case ".yaml", ".yml":
		return nil
	default:
		return fmt.Errorf("mongo: unsupported config extension %q (use .yaml or .yml)", ext)
	}
}
