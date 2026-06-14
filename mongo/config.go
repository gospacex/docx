package mongo

import (
	"fmt"

	"github.com/gospacex/hubx/cache/docx/config"
	"github.com/gospacex/hubx/cache/docx/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	URI        string `yaml:"uri"`
	Database   string `yaml:"database"`
	Collection string `yaml:"collection"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`

	ConnectTimeout int `yaml:"connect_timeout_ms"`
	MaxPoolSize    int `yaml:"max_pool_size"`

	Tracing config.TracingConfig `yaml:"tracing"`

	contentHash string
}

func ParseConfig(content []byte) (*Config, error) {
	expanded, err := utils.ExpandEnvVars(string(content))
	if err != nil {
		return nil, fmt.Errorf("mongo: expand env: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("mongo: parse config: %w", err)
	}
	if err := cfg.validateCommon(); err != nil {
		return nil, err
	}
	if _, err := cfg.CacheFingerprint(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validateCommon() error {
	if c == nil {
		return fmt.Errorf("mongo: config is nil")
	}
	if c.URI == "" {
		return fmt.Errorf("mongo: URI is required")
	}
	return nil
}

func (c *Config) ValidateClient() error {
	return c.validateCommon()
}

func (c *Config) ValidateCollection() error {
	if err := c.validateCommon(); err != nil {
		return err
	}
	if c.Database == "" {
		return fmt.Errorf("mongo: database is required for standard collection mode")
	}
	if c.Collection == "" {
		return fmt.Errorf("mongo: collection is required for standard collection mode")
	}
	return nil
}

func (c *Config) CacheFingerprint() (string, error) {
	if c == nil {
		return "", fmt.Errorf("mongo: config is nil")
	}
	copy := *c
	copy.contentHash = ""
	if err := copy.Tracing.Validate(); err != nil {
		return "", fmt.Errorf("mongo: fingerprint: %w", err)
	}
	fp, err := utils.Fingerprint(copy)
	if err != nil {
		return "", fmt.Errorf("mongo: fingerprint: %w", err)
	}
	c.contentHash = fp
	return fp, nil
}

func (c *Config) ComputeContentHash() string {
	fp, _ := c.CacheFingerprint()
	return fp
}

func (c *Config) ContentHash() string {
	fp, _ := c.CacheFingerprint()
	return fp
}
