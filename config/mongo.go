package config

type MongoConfig struct {
	URI         string        `yaml:"uri"`
	DB          string        `yaml:"db"`
	Collection  string        `yaml:"collection"`
	Timeout     int           `yaml:"timeout_ms"`
	MaxPoolSize int           `yaml:"max_pool_size"`
	Tracing     TracingConfig `yaml:"tracing"`
}
