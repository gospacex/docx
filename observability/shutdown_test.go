package observability

import (
	"context"
	"testing"
)

// TestShutdownTracing_NoopWhenNeverInited covers the early-return
// branch: with currentTP == nil, ShutdownTracing must return nil
// without touching the global tracer provider.
func TestShutdownTracing_NoopWhenNeverInited(t *testing.T) {
	// Force the package state to "never inited" by swapping out
	// whatever a previous test left behind.
	currentTP.Store(nil)
	if err := ShutdownTracing(context.Background()); err != nil {
		t.Fatalf("ShutdownTracing on nil state should be a no-op, got: %v", err)
	}
	if currentTP.Load() != nil {
		t.Fatal("expected currentTP to remain nil")
	}
}
