// Package config provides centralized test configuration for ambient-code-backend tests
package config

import (
	test_constants "ambient-code-backend/tests/constants"
	"flag"
	"os"
	"strconv"
	"time"
)

// Command line flags with environment variable fallback defaults
var (
	// Environment settings
	TestNamespace    = flag.String("testNamespace", getEnvOrDefault("TEST_NAMESPACE", "test-namespace"), "Kubernetes namespace for tests")
	UseRealCluster   = flag.Bool("useRealCluster", getBoolEnvOrDefault("USE_REAL_CLUSTER", false), "Use real Kubernetes cluster instead of fake client")
	CleanupResources = flag.Bool("cleanup", getBoolEnvOrDefault("CLEANUP_RESOURCES", true), "Clean up test resources after completion")
	FlakeAttempts    = flag.Int("flakeAttempts", getIntEnvOrDefault("FLAKE_ATTEMPTS", 0), "Number of retry attempts for API calls")

	// Timeout settings
	SuiteTimeout = flag.Duration("suiteTimeout", getDurationEnvOrDefault("SUITE_TIMEOUT", 30*time.Minute), "Test suite timeout")
	TestTimeout  = flag.Duration("testTimeout", getDurationEnvOrDefault("TEST_TIMEOUT", 5*time.Minute), "Individual test timeout")
	APITimeout   = flag.Duration("apiTimeout", getDurationEnvOrDefault("API_TIMEOUT", 30*time.Second), "API request timeout")

	// Execution settings
	SkipSlowTests     = flag.Bool("skipSlow", getBoolEnvOrDefault("SKIP_SLOW_TESTS", false), "Skip slow-running tests")
	TestDataDirectory = flag.String("testDataDir", getEnvOrDefault("TEST_DATA_DIR", "testdata"), "Test data directory")

	// HTTP client settings
	RetryAttempts = flag.Int("retryAttempts", getIntEnvOrDefault("RETRY_ATTEMPTS", 3), "Number of retry attempts for API calls")
	RetryDelay    = flag.Duration("retryDelay", getDurationEnvOrDefault("RETRY_DELAY", 1*time.Second), "Delay between retries")
	MaxRetryDelay = flag.Duration("maxRetryDelay", getDurationEnvOrDefault("MAX_RETRY_DELAY", 10*time.Second), "Maximum retry delay")

	// Kubernetes settings
	KubeConfigPath = flag.String("kubeconfig", getEnvOrDefault("KUBECONFIG", ""), "Path to kubeconfig file")
	ContextName    = flag.String("kubeContext", getEnvOrDefault("KUBE_CONTEXT", ""), "Kubernetes context name")

	// Authentication settings
	TestUserToken   = flag.String("testUserToken", getEnvOrDefault("TEST_USER_TOKEN", ""), "Test user token for authentication")
	TestUserSubject = flag.String("testUserSubject", getEnvOrDefault("TEST_USER_SUBJECT", "test-user"), "Test user subject")

	// Test environment settings
	DisableAuth = flag.Bool("disableAuth", getBoolEnvOrDefault("DISABLE_AUTH", true), "Disable authentication for testing")
	GoTestMode  = flag.Bool("goTestMode", getBoolEnvOrDefault("GO_TEST", true), "Enable Go test mode")
)

// Helper functions for environment variable parsing
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnvOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getIntEnvOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getDurationEnvOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

// IsIntegrationTest returns true if integration tests should use a real cluster.
func IsIntegrationTest() bool {
	return *UseRealCluster
}

func ShouldSkipSlowTests() bool {
	return *SkipSlowTests
}

func GetTestNamespace() string {
	return *TestNamespace
}

func IsAuthDisabled() bool {
	return os.Getenv(test_constants.EnvDisableAuth) == "true"
}

func IsGoTestMode() bool {
	return os.Getenv(test_constants.EnvGoTest) == "true"
}
