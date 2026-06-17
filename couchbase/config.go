package couchbase

import (
	"fmt"

	"github.com/gospacex/hubx/cache/docx/config"
	"github.com/gospacex/hubx/cache/docx/utils"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Endpoints []string `yaml:"endpoints"`
	Bucket    string   `yaml:"bucket"`
	Username  string   `yaml:"username"`
	Password  string   `yaml:"password"`

	ConnectTimeout int `yaml:"connect_timeout_ms"`
	SocketTimeout  int `yaml:"socket_timeout_ms"`

	Tracing config.TracingConfig `yaml:"tracing"`

	contentHash string
}

func ParseConfig(content []byte) (*Config, error) {
	expanded, err := utils.ExpandEnvVars(string(content))
	if err != nil {
		return nil, fmt.Errorf("couchbase: expand env: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("couchbase: parse config: %w", err)
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
		return fmt.Errorf("couchbase: config is nil")
	}
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("couchbase: endpoints is required")
	}
	return nil
}

func (c *Config) ValidateCluster() error {
	return c.validateCommon()
}

func (c *Config) ValidateBucket() error {
	if err := c.validateCommon(); err != nil {
		return err
	}
	if c.Bucket == "" {
		return fmt.Errorf("couchbase: bucket is required")
	}
	return nil
}

func (c *Config) CacheFingerprint() (string, error) {
	if c == nil {
		return "", fmt.Errorf("couchbase: config is nil")
	}
	copy := *c
	copy.contentHash = ""
	if err := copy.Tracing.Validate(); err != nil {
		return "", fmt.Errorf("couchbase: fingerprint: %w", err)
	}
	fp, err := utils.Fingerprint(copy)
	if err != nil {
		return "", fmt.Errorf("couchbase: fingerprint: %w", err)
	}
	c.contentHash = fp
	return fp, nil
}

// ContentHash is the cached fingerprint of the last CacheFingerprint call.
// Errors from CacheFingerprint are swallowed here so callers can use the
// method for cheap equality checks without surfacing fingerprint errors;
// callers that need to surface the error should call CacheFingerprint
// directly.
func (c *Config) ContentHash() string {
	fp, _ := c.CacheFingerprint()
	return fp
}
