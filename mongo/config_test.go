package mongo

import (
	"testing"

	"github.com/gospacex/hubx/cache/docx/config"
)

func TestParseConfigExpandsEnvVars(t *testing.T) {
	t.Setenv("MG_TEST_URI", "mongodb://db.example:27017")
	cfg, err := ParseConfig([]byte(`
uri: ${env:MG_TEST_URI}
database: ${env:MG_TEST_DB:-app}
collection: ${env:MG_TEST_COLLECTION:-users}
username: ${env:MG_TEST_USER:-admin}
password: ${env:MG_TEST_PASS:-secret}
`))
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}
	if cfg.URI != "mongodb://db.example:27017" {
		t.Fatalf("unexpected uri: %q", cfg.URI)
	}
	if cfg.Database != "app" || cfg.Collection != "users" {
		t.Fatalf("unexpected database/collection: %q/%q", cfg.Database, cfg.Collection)
	}
}

func TestValidateCollectionRequiresDatabaseAndCollection(t *testing.T) {
	cfg := &Config{URI: "mongodb://localhost:27017"}
	if err := cfg.ValidateCollection(); err == nil {
		t.Fatal("expected ValidateCollection to fail when database/collection are empty")
	}
}

func TestConfigKeysDifferByResourceKind(t *testing.T) {
	cfg := &Config{URI: "mongodb://localhost:27017", Database: "app", Collection: "users"}
	clientKey, err := clientConfigKey(cfg)
	if err != nil {
		t.Fatalf("clientConfigKey: %v", err)
	}
	collectionKey, err := collectionConfigKey(cfg)
	if err != nil {
		t.Fatalf("collectionConfigKey: %v", err)
	}
	if clientKey == collectionKey {
		t.Fatalf("expected different keys, got %q", clientKey)
	}
}

func TestClientKeyIgnoresDatabaseAndCollection(t *testing.T) {
	a := &Config{URI: "mongodb://localhost:27017", Database: "app", Collection: "users"}
	b := &Config{URI: "mongodb://localhost:27017", Database: "audit", Collection: "events"}
	keyA, err := clientConfigKey(a)
	if err != nil {
		t.Fatalf("clientConfigKey(a): %v", err)
	}
	keyB, err := clientConfigKey(b)
	if err != nil {
		t.Fatalf("clientConfigKey(b): %v", err)
	}
	if keyA != keyB {
		t.Fatalf("expected client keys to match, got %q vs %q", keyA, keyB)
	}
}

func TestCacheFingerprintStableAfterTracingDefaults(t *testing.T) {
	cfg := &Config{
		URI:        "mongodb://localhost:27017",
		Database:   "app",
		Collection: "users",
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
