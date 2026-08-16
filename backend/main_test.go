// Starter test for env() so `go test ./...` does real work in CI. No DB needed.
package main

import (
	"os"
	"testing"
)

func TestEnvReturnsFallbackWhenUnset(t *testing.T) {
	os.Unsetenv("DEVBOARD_TEST_KEY")
	if got := env("DEVBOARD_TEST_KEY", "fallback"); got != "fallback" {
		t.Errorf("env() = %q, want %q", got, "fallback")
	}
}

func TestEnvReturnsValueWhenSet(t *testing.T) {
	os.Setenv("DEVBOARD_TEST_KEY", "real")
	defer os.Unsetenv("DEVBOARD_TEST_KEY")
	if got := env("DEVBOARD_TEST_KEY", "fallback"); got != "real" {
		t.Errorf("env() = %q, want %q", got, "real")
	}
}

func TestToStringSlice(t *testing.T) {
	if _, ok := toStringSlice("not an array"); ok {
		t.Error("toStringSlice(non-array) should return ok=false")
	}
	if _, ok := toStringSlice([]interface{}{"a", 1}); ok {
		t.Error("toStringSlice with a non-string element should return ok=false")
	}
	got, ok := toStringSlice([]interface{}{"frontend", "urgent"})
	if !ok {
		t.Fatal("toStringSlice(valid array) should return ok=true")
	}
	if len(got) != 2 || got[0] != "frontend" || got[1] != "urgent" {
		t.Errorf("toStringSlice = %v, want [frontend urgent]", got)
	}
}
