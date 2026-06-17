//go:build integration

package mongox

import (
	"context"
	"os"
	"testing"
)

// TestIntegration_BuildAndPing exercises the real mongodriver.MOC path
// against a live MongoDB server. The test is gated by the `integration`
// build tag and the SKIP_INTEGRATION env var so it does not run in
// short CI runs.
//
//	MONGO_ADDR=mongodb://host:port go test -tags=integration ./hubx/mongox/...
func TestIntegration_BuildAndPing(t *testing.T) {
	if os.Getenv("SKIP_INTEGRATION") != "" {
		t.Skip("SKIP_INTEGRATION set")
	}
	addr := os.Getenv("MONGO_ADDR")
	if addr == "" {
		addr = "mongodb://localhost:27017"
	}

	p := New()
	cfg := map[string]any{
		"config": map[string]any{
			"uri":        addr,
			"database":   "hubx_it",
			"collection": "smoke",
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
