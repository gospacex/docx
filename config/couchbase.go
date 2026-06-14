package config

type CouchbaseConfig struct {
	ConnStr  string        `yaml:"conn_str"`
	Bucket   string        `yaml:"bucket"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	Timeout  int           `yaml:"timeout_ms"`
	Tracing  TracingConfig `yaml:"tracing"`
}
