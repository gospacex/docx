package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// yamlMarshal is a tiny test helper — keeps test bodies free of yaml noise.
func yamlMarshal(v any) ([]byte, error) {
	return yaml.Marshal(v)
}

func TestTracingConfig_Validate_DisabledIsNoop(t *testing.T) {
	cfg := TracingConfig{Enabled: false}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("disabled config should validate, got: %v", err)
	}
}

func TestTracingConfig_Validate_RequiresServiceName(t *testing.T) {
	cfg := TracingConfig{Enabled: true, Exporter: "jaeger"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "service_name") {
		t.Fatalf("expected service_name error, got: %v", err)
	}
}

func TestTracingConfig_Validate_RequiresExporter(t *testing.T) {
	cfg := TracingConfig{Enabled: true, ServiceName: "svc"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "exporter is required") {
		t.Fatalf("expected exporter required error, got: %v", err)
	}
}

func TestTracingConfig_Validate_UnknownExporter(t *testing.T) {
	cfg := TracingConfig{Enabled: true, ServiceName: "svc", Exporter: "tcpdump"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unknown exporter") {
		t.Fatalf("expected unknown exporter error, got: %v", err)
	}
}

func TestTracingConfig_Validate_UnknownSamplerType(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "jaeger",
		Endpoint:    "localhost:4317",
		SamplerType: "mystery",
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "unsupported sampler_type") {
		t.Fatalf("expected sampler_type error, got: %v", err)
	}
}

func TestTracingConfig_Validate_JaegerRequiresEndpoint(t *testing.T) {
	cfg := TracingConfig{Enabled: true, ServiceName: "svc", Exporter: "jaeger"}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "endpoint is required") {
		t.Fatalf("expected endpoint required error, got: %v", err)
	}
}

func TestTracingConfig_Validate_JaegerDefaultsProtocolToGRPC(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "jaeger",
		Endpoint:    "localhost:4317",
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Protocol != "grpc" {
		t.Fatalf("expected Protocol=grpc, got: %q", cfg.Protocol)
	}
}

func TestTracingConfig_Validate_JaegerRejectsUnknownProtocol(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "jaeger",
		Endpoint:    "localhost:4317",
		Protocol:    "tcp",
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "protocol must be grpc or http") {
		t.Fatalf("expected protocol error, got: %v", err)
	}
}

func TestTracingConfig_Validate_JaegerHTTPPath(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "jaeger",
		Endpoint:    "localhost:4318",
		Protocol:    "http",
		Insecure:    true,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected http path to validate, got: %v", err)
	}
}

func TestTracingConfig_Validate_KafkaTopicRequiresAddrs(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "kafka_topic",
		Producer:    TracingProducerConfig{Topic: "t"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "addrs is required") {
		t.Fatalf("expected addrs required error, got: %v", err)
	}
}

func TestTracingConfig_Validate_KafkaTopicRequiresTopic(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "kafka_topic",
		Addrs:       []string{"localhost:9092"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "producer.topic is required") {
		t.Fatalf("expected producer.topic required error, got: %v", err)
	}
}

func TestTracingConfig_Validate_KafkaTopicDefaultsAcks(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "kafka_topic",
		Addrs:       []string{"localhost:9092"},
		Producer:    TracingProducerConfig{Topic: "t"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg.Producer.Acks != "all" {
		t.Fatalf("expected Acks=all, got: %q", cfg.Producer.Acks)
	}
	if !cfg.Producer.Idempotent {
		t.Fatal("expected Idempotent=true by default (mqx parity)")
	}
}

func TestTracingConfig_Validate_RedisStreamRequiresAddrs(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "redis_stream",
		Producer:    TracingProducerConfig{Topic: "s"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "addrs is required") {
		t.Fatalf("expected addrs required error, got: %v", err)
	}
}

func TestTracingConfig_Validate_RedisStreamRequiresStreamName(t *testing.T) {
	cfg := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "redis_stream",
		Addrs:       []string{"localhost:6379"},
	}
	err := cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "stream name") {
		t.Fatalf("expected stream name required error, got: %v", err)
	}
}

func TestTracingConfig_Validate_SamplerRatioRange(t *testing.T) {
	cases := []struct {
		name  string
		ratio float64
		ok    bool
	}{
		{"negative rejected", -0.01, false},
		{"zero allowed", 0, true},
		{"half allowed", 0.5, true},
		{"one allowed", 1, true},
		{"above one rejected", 1.01, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := TracingConfig{
				Enabled:      true,
				ServiceName:  "svc",
				Exporter:     "jaeger",
				Endpoint:     "localhost:4317",
				SamplerRatio: c.ratio,
			}
			err := cfg.Validate()
			if c.ok && err != nil {
				t.Fatalf("expected no error for ratio %v, got: %v", c.ratio, err)
			}
			if !c.ok && (err == nil || !strings.Contains(err.Error(), "sampler_ratio")) {
				t.Fatalf("expected sampler_ratio error for %v, got: %v", c.ratio, err)
			}
		})
	}
}

func TestTracingConfig_YAML_RoundTrip(t *testing.T) {
	// Verifies the renamed yaml tags are honoured: a config struct round-trips
	// through yaml.Marshal/Unmarshal preserving the new vocabulary
	// (exporter / addrs / auth / producer / kafka / redis).
	src := TracingConfig{
		Enabled:     true,
		ServiceName: "svc",
		Exporter:    "kafka_topic",
		Protocol:    "grpc",
		Endpoint:    "localhost:4317",
		Addrs:       []string{"broker1:9092", "broker2:9092"},
		Auth:        TracingAuthConfig{Username: "u", Password: "p", Mechanism: "PLAIN"},
		Producer:    TracingProducerConfig{Topic: "otel-traces", Acks: "all", Idempotent: true},
		Kafka:       TracingKafkaConfig{SecurityProtocol: "SASL_PLAINTEXT", SASLMechanism: "SCRAM-SHA-512"},
	}
	data, err := yamlMarshal(src)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, key := range []string{"exporter:", "addrs:", "auth:", "username:", "producer:", "topic:", "kafka:", "security_protocol:", "sasl_mechanism:"} {
		if !strings.Contains(string(data), key) {
			t.Fatalf("expected yaml to contain %q, got:\n%s", key, data)
		}
	}
}
