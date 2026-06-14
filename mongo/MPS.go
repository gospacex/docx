package mongo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	key := absPath
	val, err := getOrCreate(key, func() (interface{}, error) {
		return newCollection(ctx, cfg)
	})
	if err != nil {
		return nil, err
	}
	return val.(*Collection), nil
}

func validateConfigPath(absPath string) error {
	base := strings.ToLower(filepath.Base(absPath))
	if !strings.Contains(base, "mongo") {
		return fmt.Errorf("mongo: config %q does not appear to be a Mongo config", absPath)
	}

	ext := filepath.Ext(absPath)
	switch ext {
	case ".yaml", ".yml":
		return nil
	default:
		return fmt.Errorf("mongo: unsupported config extension %q (use .yaml or .yml)", ext)
	}
}
