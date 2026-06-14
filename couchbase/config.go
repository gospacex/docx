package couchbase

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/gospacex/hubx/cache/docx/config"
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
	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("couchbase: parse config: %w", err)
	}
	cfg.ComputeContentHash()
	return &cfg, nil
}

func (c *Config) ComputeContentHash() string {
	safe := struct {
		Endpoints      []string `json:"endpoints"`
		Bucket         string   `json:"bucket"`
		Username       string   `json:"username"`
		ConnectTimeout int      `json:"connect_timeout_ms"`
		SocketTimeout  int      `json:"socket_timeout_ms"`
		TracingEnabled bool     `json:"tracing_enabled"`
	}{
		Endpoints:      c.Endpoints,
		Bucket:         c.Bucket,
		Username:       c.Username,
		ConnectTimeout: c.ConnectTimeout,
		SocketTimeout:  c.SocketTimeout,
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
