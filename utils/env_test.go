package utils

import (
	"os"
	"testing"
)

func TestExpandEnvVars_Hit(t *testing.T) {
	t.Setenv("FOO", "bar")
	got, err := ExpandEnvVars("hello ${env:FOO} world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello bar world" {
		t.Fatalf("expected 'hello bar world', got %q", got)
	}
}

func TestExpandEnvVars_Default(t *testing.T) {
	os.Unsetenv("MISSING")
	got, err := ExpandEnvVars("x=${env:MISSING:-fallback}")
	if err != nil {
		t.Fatalf("expected default to suppress error, got: %v", err)
	}
	if got != "x=fallback" {
		t.Fatalf("expected default, got %q", got)
	}
}

func TestExpandEnvVars_MissingNoDefaultErrors(t *testing.T) {
	os.Unsetenv("MISSING_ND")
	_, err := ExpandEnvVars("v=${env:MISSING_ND}")
	if err == nil {
		t.Fatal("expected error when env var is missing and no default")
	}
}

func TestExpandEnvVars_ExplicitEmptyDefault(t *testing.T) {
	os.Unsetenv("EMPTY_OK")
	got, err := ExpandEnvVars("x=${env:EMPTY_OK:-}")
	if err != nil {
		t.Fatalf("expected explicit empty default to suppress error, got: %v", err)
	}
	if got != "x=" {
		t.Fatalf("expected empty default, got %q", got)
	}
}

func TestExpandEnvVars_NoMarkers(t *testing.T) {
	got, err := ExpandEnvVars("plain text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "plain text" {
		t.Fatalf("expected passthrough, got %q", got)
	}
}
