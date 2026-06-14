package couchbase

import (
	"testing"

	"github.com/gospacex/hubx/cache/docx/config"
)

func TestParseConfigExpandsEnvVars(t *testing.T) {
	t.Setenv("CB_TEST_ENDPOINT", "cb.example:8091")
	cfg, err := ParseConfig([]byte(`
endpoints:
  - ${env:CB_TEST_ENDPOINT}
bucket: ${env:CB_TEST_BUCKET:-users}
username: ${env:CB_TEST_USER:-admin}
password: ${env:CB_TEST_PASS:-secret}
`))
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}
	if len(cfg.Endpoints) != 1 || cfg.Endpoints[0] != "cb.example:8091" {
		t.Fatalf("unexpected endpoints: %#v", cfg.Endpoints)
	}
	if cfg.Bucket != "users" {
		t.Fatalf("expected bucket users, got %q", cfg.Bucket)
	}
}

func TestValidateBucketRequiresBucket(t *testing.T) {
	cfg := &Config{Endpoints: []string{"localhost:8091"}}
	if err := cfg.ValidateBucket(); err == nil {
		t.Fatal("expected ValidateBucket to fail when bucket is empty")
	}
}

func TestConfigKeysDifferByResourceKind(t *testing.T) {
	cfg := &Config{Endpoints: []string{"localhost:8091"}, Bucket: "users"}
	clusterKey, err := clusterConfigKey(cfg)
	if err != nil {
		t.Fatalf("clusterConfigKey: %v", err)
	}
	bucketKey, err := bucketConfigKey(cfg)
	if err != nil {
		t.Fatalf("bucketConfigKey: %v", err)
	}
	if clusterKey == bucketKey {
		t.Fatalf("expected different keys, got %q", clusterKey)
	}
}

func TestClusterKeyIgnoresBucketName(t *testing.T) {
	a := &Config{Endpoints: []string{"localhost:8091"}, Bucket: "users"}
	b := &Config{Endpoints: []string{"localhost:8091"}, Bucket: "audit"}
	keyA, err := clusterConfigKey(a)
	if err != nil {
		t.Fatalf("clusterConfigKey(a): %v", err)
	}
	keyB, err := clusterConfigKey(b)
	if err != nil {
		t.Fatalf("clusterConfigKey(b): %v", err)
	}
	if keyA != keyB {
		t.Fatalf("expected cluster keys to match, got %q vs %q", keyA, keyB)
	}
}

func TestCacheFingerprintStableAfterTracingDefaults(t *testing.T) {
	cfg := &Config{
		Endpoints: []string{"localhost:8091"},
		Bucket:    "users",
		Tracing: config.TracingConfig{
			Enabled:     true,
			ServiceName: "svc",
			Exporter:    "jaeger",
			Endpoint:    "localhost:4317",
		},
	}
	first, err := cfg.CacheFingerprint()
	if err != nil {
		t.Fatalf("first fingerprint: %v", err)
	}
	second, err := cfg.CacheFingerprint()
	if err != nil {
		t.Fatalf("second fingerprint: %v", err)
	}
	if first != second {
		t.Fatalf("expected stable fingerprint, got %q vs %q", first, second)
	}
}
