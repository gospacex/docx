package utils

import "testing"

func TestFingerprint_StableForEqualInput(t *testing.T) {
	a, err := Fingerprint(map[string]any{"k": "v", "n": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := Fingerprint(map[string]any{"k": "v", "n": 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a != b {
		t.Fatalf("expected stable fingerprint, got %q vs %q", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("expected hex SHA-256 (64 chars), got %d", len(a))
	}
}

func TestFingerprint_DiffersForDifferentInput(t *testing.T) {
	a, _ := Fingerprint(map[string]any{"k": "v1"})
	b, _ := Fingerprint(map[string]any{"k": "v2"})
	if a == b {
		t.Fatal("expected different fingerprints for different inputs")
	}
}
