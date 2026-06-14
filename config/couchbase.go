package config

type CouchbaseConfig struct {
	Endpoints      []string      `yaml:"endpoints"`
	Bucket         string        `yaml:"bucket"`
	Username       string        `yaml:"username"`
	Password       string        `yaml:"password"`
	ConnectTimeout int           `yaml:"connect_timeout_ms"`
	SocketTimeout  int           `yaml:"socket_timeout_ms"`
	Tracing        TracingConfig `yaml:"tracing"`
}
