package featureflags

import (
	"testing"
)

// These tests verify behavior when Unleash is NOT configured (the default in tests).
// The initialized atomic.Bool is false, so functions hit the early-return path.

func TestIsEnabled_Unconfigured(t *testing.T) {
	result := IsEnabled("any-flag")
	if result {
		t.Error("IsEnabled should return false when Unleash is not configured")
	}
}

func TestIsEnabledWithContext_Unconfigured(t *testing.T) {
	result := IsEnabledWithContext("any-flag", "user1", "session1", "127.0.0.1")
	if result {
		t.Error("IsEnabledWithContext should return false when Unleash is not configured")
	}
}

func TestIsModelEnabled_Unconfigured(t *testing.T) {
	result := IsModelEnabled("model.claude-opus-4-6.enabled")
	if !result {
		t.Error("IsModelEnabled should return true (fail-open) when Unleash is not configured")
	}
}

func TestIsModelEnabledWithContext_Unconfigured(t *testing.T) {
	result := IsModelEnabledWithContext("model.claude-opus-4-6.enabled", "user1", "session1", "127.0.0.1")
	if !result {
		t.Error("IsModelEnabledWithContext should return true (fail-open) when Unleash is not configured")
	}
}

func TestInit_NoEnvVars(t *testing.T) {
	// Ensure Init is safe to call when env vars are not set
	t.Setenv("UNLEASH_URL", "")
	t.Setenv("UNLEASH_CLIENT_KEY", "")

	Init() // should not panic

	// initialized should still be false
	if initialized.Load() {
		t.Error("Init should not set initialized when env vars are empty")
	}
}
