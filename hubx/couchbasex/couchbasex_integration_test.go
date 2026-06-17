//go:build integration

package couchbasex

import (
	"context"
	"os"
	"testing"
)

// TestIntegration_BuildAndPing exercises the real couchbasedriver.COC
// path against a live Couchbase cluster. The test is gated by the
// `integration` build tag and the SKIP_INTEGRATION env var so it does
// not run in short CI runs.
//
//	COUCHBASE_ADDR=host:port go test -tags=integration ./hubx/couchbasex/...
func TestIntegration_BuildAndPing(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("SKIP_INTEGRATION set")
	}
	addr := os.Getenv("COUCHBASE_ADDR")
	if addr == "" {
		addr = "localhost:11210"
	}

	p := New()
	cfg := map[string]any{
		"config": map[string]any{
			"endpoints": []string{addr},
			"bucket":    "hubx_it",
		},
	}
	cli, err := p.Build("it", cfg)
	if err != nil {
		t.Fatalf("Build returned error: %v", err)
	}
	if err := cli.HealthCheck(context.Background()); err != nil {
		t.Fatalf("HealthCheck returned error: %v", err)
	}
	if err := cli.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}
