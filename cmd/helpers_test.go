package cmd

import (
	"os"
	"testing"
)

func TestEnvOrDefaultWithEnvSet(t *testing.T) {
	t.Setenv("TEST_SUCCULENT_VAR", "custom-value")

	result := envOrDefault("TEST_SUCCULENT_VAR", "fallback")
	if result != "custom-value" {
		t.Errorf("Expected 'custom-value', got '%s'", result)
	}
}

func TestEnvOrDefaultWithEnvUnset(t *testing.T) {
	os.Unsetenv("TEST_SUCCULENT_UNSET_VAR")

	result := envOrDefault("TEST_SUCCULENT_UNSET_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("Expected 'fallback', got '%s'", result)
	}
}

func TestEnvOrDefaultWithEmptyEnv(t *testing.T) {
	t.Setenv("TEST_SUCCULENT_EMPTY", "")

	result := envOrDefault("TEST_SUCCULENT_EMPTY", "fallback")
	if result != "fallback" {
		t.Errorf("Expected 'fallback' for empty env var, got '%s'", result)
	}
}
