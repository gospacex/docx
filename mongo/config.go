package mongo

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/gospacex/hubx/cache/docx/config"
	"gopkg.in/yaml.v3"
)

type Config struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`

	ConnectTimeout int `yaml:"connect_timeout_ms"`
	MaxPoolSize    int `yaml:"max_pool_size"`

	Tracing config.TracingConfig `yaml:"tracing"`

	contentHash string
}

func ParseConfig(content []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("mongo: parse config: %w", err)
	}
	cfg.ComputeContentHash()
	return &cfg, nil
}

func (c *Config) ComputeContentHash() string {
	safe := struct {
		URI            string `json:"uri"`
		Database       string `json:"database"`
		Username       string `json:"username"`
		ConnectTimeout int    `json:"connect_timeout_ms"`
		MaxPoolSize    int    `json:"max_pool_size"`
		TracingEnabled bool   `json:"tracing_enabled"`
	}{
		URI:            c.URI,
		Database:       c.Database,
		Username:       c.Username,
		ConnectTimeout: c.ConnectTimeout,
		MaxPoolSize:    c.MaxPoolSize,
		TracingEnabled: c.Tracing.Enabled,
	}
	data, _ := json.Marshal(safe)
	h := sha256.Sum256(data)
	c.contentHash = fmt.Sprintf("%x", h)
	return c.contentHash
}

func (c *Config) ContentHash() string {
	return c.contentHash
}