package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/baggage"
)

// TestSetBaggage_SetsSingleMember covers the happy path: one member
// is set, GetBaggage reads it back.
func TestSetBaggage_SetsSingleMember(t *testing.T) {
	m, err := baggage.NewMember("user_id", "alice")
	if err != nil {
		t.Fatalf("NewMember: %v", err)
	}
	ctx, err := SetBaggage(context.Background(), m)
	if err != nil {
		t.Fatalf("SetBaggage: %v", err)
	}
	got := GetBaggage(ctx)
	if got.Len() != 1 {
		t.Fatalf("expected 1 baggage member, got %d", got.Len())
	}
	v := got.Member("user_id")
	if v.Value() != "alice" {
		t.Fatalf("expected user_id=alice, got %q", v.Value())
	}
}

// TestSetBaggage_AppendsMultipleMembers verifies that successive
// members accumulate rather than overwrite.
func TestSetBaggage_AppendsMultipleMembers(t *testing.T) {
	m1, _ := baggage.NewMember("k1", "v1")
	m2, _ := baggage.NewMember("k2", "v2")
	ctx, err := SetBaggage(context.Background(), m1, m2)
	if err != nil {
		t.Fatalf("SetBaggage: %v", err)
	}
	got := GetBaggage(ctx)
	if got.Len() != 2 {
		t.Fatalf("expected 2 members, got %d", got.Len())
	}
	if got.Member("k1").Value() != "v1" || got.Member("k2").Value() != "v2" {
		t.Fatalf("unexpected baggage contents: %v", got)
	}
}

// TestSetBaggage_RejectsInvalidMember covers the early-return-on-error
// branch. NewMember rejects keys with whitespace; the production code
// must surface that error verbatim.
func TestSetBaggage_RejectsInvalidMember(t *testing.T) {
	_, err := baggage.NewMember("bad key with space", "v")
	if err == nil {
		t.Fatal("expected NewMember to reject a key with whitespace")
	}
	// Mirror the production code's behaviour: an invalid member
	// short-circuits the loop. We exercise the loop directly to
	// confirm the error is returned to the caller.
	bad := baggage.Member{}
	ctx, err := SetBaggage(context.Background(), bad)
	if err == nil {
		t.Fatal("expected SetBaggage to return an error for an invalid member")
	}
	if ctx == nil {
		t.Fatal("SetBaggage should return a non-nil context even on error")
	}
}

// TestGetBaggage_EmptyByDefault asserts the no-init case: with no
// SetBaggage in the chain, GetBaggage returns an empty (but non-nil)
// baggage.
func TestGetBaggage_EmptyByDefault(t *testing.T) {
	got := GetBaggage(context.Background())
	if got.Len() != 0 {
		t.Fatalf("expected empty baggage, got %d members", got.Len())
	}
}
