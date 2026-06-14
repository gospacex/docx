package config

import "fmt"

// TracingConfig describes how docx emits OTel spans.
//
// The shape is intentionally aligned with the mqx config tree so that callers
// familiar with kafkax/redisx YAML can reuse the same vocabulary. Where the
// mqx KafkaConfig / RedisConfig are driver-specific, this struct only embeds
// the fields docx's tracing exporters actually need — see TracingKafkaConfig /
// TracingRedisConfig below.
//
// Field renames vs the legacy struct (2026-06-13):
//
//	Backend     → Exporter
//	Mode        → REMOVED (sampling lives in OTel SDK; cfg.SamplerRatio covers it)
//	KafkaBrokers  → Addrs
//	KafkaTopic    → Producer.Topic
//	KafkaSASLMechanism → Kafka.SASLMechanism
//	KafkaSASLUsername/Password → Auth.Username/Password
//	RedisAddr     → Addrs[0]
//	RedisUsername/Password → Redis.Username/Password (with Auth fallback)
//	Stream        → Producer.Topic (shared between kafka and redis)
//	KafkaProducer / RedisClient (any inject) → REMOVED
type TracingConfig struct {
	Enabled     bool              `yaml:"enabled"`
	ServiceName string            `yaml:"service_name"`
	Exporter    string            `yaml:"exporter"` // jaeger | kafka_topic | redis_stream
	Protocol    string            `yaml:"protocol"` // grpc | http (jaeger only); default grpc
	Endpoint    string            `yaml:"endpoint"`
	SamplerType string            `yaml:"sampler_type"`
	SamplerRatio float64          `yaml:"sampler_ratio"`
	Insecure    bool              `yaml:"insecure"`
	Headers     map[string]string `yaml:"headers"`

	// Addrs is the connection address list. For redis exporters, Addrs[0]
	// is used as the redis server address (mqx redisx also accepts []string
	// for cluster mode, but docx only uses the first entry).
	Addrs []string `yaml:"addrs"`

	// Auth holds SASL/username/password used by Kafka. For Redis, see
	// Redis.Username/Password (with Auth as a fallback when Redis is empty).
	Auth TracingAuthConfig `yaml:"auth"`

	// Producer holds the span destination. For Kafka exporters this is the
	// Kafka topic; for Redis exporters this is the stream name. The field
	// name matches mqx.ProducerConfig.Topic.
	Producer TracingProducerConfig `yaml:"producer"`

	// Kafka holds Kafka-specific knobs (security protocol + SASL mechanism).
	Kafka TracingKafkaConfig `yaml:"kafka"`

	// Redis holds Redis-specific knobs (db, pool, credentials).
	Redis TracingRedisConfig `yaml:"redis"`
}

// TracingAuthConfig mirrors mqx.AuthConfig fields used by tracing exporters.
type TracingAuthConfig struct {
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Mechanism string `yaml:"mechanism"`
	Token     string `yaml:"token"`
}

// TracingKafkaConfig mirrors the subset of mqx.KafkaConfig consumed by the
// tracing Kafka producer.
type TracingKafkaConfig struct {
	SecurityProtocol string `yaml:"security_protocol"`
	SASLMechanism    string `yaml:"sasl_mechanism"`
}

// TracingRedisConfig mirrors the subset of mqx.RedisConfig consumed by the
// tracing Redis client. Username/Password are kept here (go-redis Options)
// and fall back to Auth.Username/Password when empty.
type TracingRedisConfig struct {
	DB       int    `yaml:"db"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	PoolSize int    `yaml:"pool_size"`
}

// TracingProducerConfig mirrors the subset of mqx.ProducerConfig used by
// tracing Kafka producer. Topic is the destination name (kafka topic /
// redis stream); Acks/Idempotent/LingerMs forward to librdkafka.
type TracingProducerConfig struct {
	Topic      string `yaml:"topic"`
	Acks       string `yaml:"acks"`
	Idempotent bool   `yaml:"idempotent"`
	LingerMs   int    `yaml:"linger_ms"`
}

// Validate enforces the rules shared by every tracing exporter plus the
// per-exporter required fields. It mutates Protocol and Producer.Acks
// in place to apply defaults (matching mqx behaviour: ACKS defaults to
// "all", idempotence default true).
func (c *TracingConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.ServiceName == "" {
		return fmt.Errorf("config: service_name is required when tracing is enabled")
	}
	if c.Protocol == "" {
		c.Protocol = "grpc"
	}
	switch c.Exporter {
	case "jaeger":
		if c.Endpoint == "" {
			return fmt.Errorf("config: endpoint is required for jaeger exporter")
		}
		if c.Protocol != "grpc" && c.Protocol != "http" {
			return fmt.Errorf("config: protocol must be grpc or http, got %q", c.Protocol)
		}
	case "kafka_topic":
		if len(c.Addrs) == 0 {
			return fmt.Errorf("config: addrs is required for kafka_topic exporter")
		}
		if c.Producer.Topic == "" {
			return fmt.Errorf("config: producer.topic is required for kafka_topic exporter")
		}
		if c.Producer.Acks == "" {
			c.Producer.Acks = "all"
		}
		if !c.Producer.Idempotent {
			// mqx parity: idempotence is on by default for tracing producers
			// to guarantee exactly-once delivery on retry; operators can opt
			// out by setting idem=... no — they must explicitly disable.
			c.Producer.Idempotent = true
		}
	case "redis_stream":
		if len(c.Addrs) == 0 {
			return fmt.Errorf("config: addrs is required for redis_stream exporter")
		}
		if c.Producer.Topic == "" {
			return fmt.Errorf("config: producer.topic (stream name) is required for redis_stream exporter")
		}
	case "":
		return fmt.Errorf("config: exporter is required when tracing is enabled")
	default:
		return fmt.Errorf("config: unknown exporter %q", c.Exporter)
	}
	if c.SamplerRatio < 0 || c.SamplerRatio > 1 {
		return fmt.Errorf("config: sampler_ratio out of range [0,1]: %v", c.SamplerRatio)
	}
	return nil
}
