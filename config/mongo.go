package config

type MongoConfig struct {
	URI            string        `yaml:"uri"`
	Database       string        `yaml:"database"`
	Collection     string        `yaml:"collection"`
	Username       string        `yaml:"username"`
	Password       string        `yaml:"password"`
	ConnectTimeout int           `yaml:"connect_timeout_ms"`
	MaxPoolSize    int           `yaml:"max_pool_size"`
	Tracing        TracingConfig `yaml:"tracing"`
}
