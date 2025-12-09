// Package test_utils provides general testing utilities following KFP patterns
package test_utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ambient-code-backend/handlers"
	"ambient-code-backend/server"
	"ambient-code-backend/test/logger"

	"github.com/gin-gonic/gin"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Global variable to store fake K8s client for tests
var fakeK8sClientForTests kubernetes.Interface

// GetRandomString generates a random string of specified length
func GetRandomString(length int) string {
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

// WriteLogFile writes test failure logs to file following KFP pattern
func WriteLogFile(specReport types.SpecReport, testName, logDirectory string) {
	stdOutput := specReport.CapturedGinkgoWriterOutput
	testLogFile := filepath.Join(logDirectory, testName+".log")

	logFile, err := os.Create(testLogFile)
	if err != nil {
		logger.Log("Failed to create log file due to: %s", err.Error())
		return
	}
	defer logFile.Close()

	_, err = logFile.Write([]byte(stdOutput))
	if err != nil {
		logger.Log("Failed to write to the log file, due to: %s", err.Error())
		return
	}

	logger.Log("Test failure log written to: %s", testLogFile)
}

// GenerateTestID creates a unique test identifier
func GenerateTestID(prefix string) string {
	timestamp := time.Now().Unix()
	randomSuffix := GetRandomString(6)
	return fmt.Sprintf("%s-%d-%s", prefix, timestamp, randomSuffix)
}

// ParsePointerToString converts a string pointer to string value
func ParsePointerToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// CheckIfSkipping checks if test should be skipped based on conditions
func CheckIfSkipping(testName string) {
	// Skip tests with specific patterns if needed
	// This follows the KFP pattern for conditional test skipping
	if testName == "" {
		return
	}

	// Add any skip conditions here as needed
	// Example: Skip tests marked with certain tags
}

// StringPtr returns a pointer to the given string
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to the given bool
func BoolPtr(b bool) *bool {
	return &b
}

// WaitWithTimeout waits for a condition with timeout
func WaitWithTimeout(conditionFn func() bool, timeout time.Duration, message string) {
	Eventually(conditionFn, timeout, 1*time.Second).Should(BeTrue(), message)
}

// RetryOperation retries an operation with exponential backoff
func RetryOperation(operation func() error, maxRetries int, initialDelay time.Duration) error {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
			if attempt < maxRetries-1 {
				delay := time.Duration(1<<attempt) * initialDelay
				time.Sleep(delay)
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxRetries, lastErr)
}

// SetupHandlerDependencies sets up the package-level variables that handlers depend on
// This function should be called in BeforeEach for all handler tests to ensure proper fake client setup
func SetupHandlerDependencies(k8sUtils *K8sTestUtils) {
	// Set up the package-level variables that handlers depend on
	handlers.DynamicClient = k8sUtils.DynamicClient

	// Set up K8sClientProjects for permissions handlers
	handlers.K8sClientProjects = k8sUtils.K8sClient

	// Set up DynamicClientProjects for sessions and other project-scoped operations
	handlers.DynamicClientProjects = k8sUtils.DynamicClient

	// Set up common GVR functions
	handlers.GetAgenticSessionV1Alpha1Resource = func() schema.GroupVersionResource {
		return schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}
	}

	// SECURITY: Override GetK8sClientsFunc (the only mutable function) to return fake clients
	// All client selection functions are now immutable and call GetK8sClientsForRequest,
	// which uses GetK8sClientsFunc. This ensures authentication logic is centralized.
	// Tests can control auth behavior by checking the Authorization header here.
	//
	// Note: GetK8sClientsForRequest returns (*kubernetes.Clientset, dynamic.Interface), but
	// fake clients are *k8sfake.Clientset. We store the fake clientset separately and return
	// it using a type assertion. The client selection functions that use this return value
	// expect kubernetes.Interface, so they'll work correctly with the fake client.
	fakeClientset := k8sUtils.K8sClient
	// Use setGetK8sClientsFunc which includes runtime safety checks
	// This ensures we can only override in test mode
	handlers.SetGetK8sClientsFunc(func(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
		// In test setup, we default to allowing auth bypass (DISABLE_AUTH=true by default in tests)
		// Individual tests can override GetK8sClientsFunc to test auth behavior
		bypassAuth := os.Getenv("DISABLE_AUTH") == "true"

		if !bypassAuth {
			// Validate auth header exists (for tests that want to test auth)
			auth := c.GetHeader("Authorization")
			if auth == "" {
				// Also check X-Forwarded-Access-Token
				auth = c.GetHeader("X-Forwarded-Access-Token")
			}
			if auth == "" {
				return nil, nil // No auth header = return nil to trigger 401
			}
			// Check for valid Bearer format if using Authorization header
			if strings.HasPrefix(auth, "Bearer ") || strings.HasPrefix(auth, "bearer ") {
				parts := strings.SplitN(auth, " ", 2)
				if len(parts) != 2 {
					return nil, nil // Malformed auth header = return nil to trigger 401
				}
			}
		}

		// Return fake clients directly as kubernetes.Interface (no unsafe conversion needed)
		// fakeClientset is already kubernetes.Interface, so we can return it directly
		return fakeClientset, k8sUtils.DynamicClient
	})

	// Set up server package variables needed for getLocalDevK8sClients()
	// In tests, we need to ensure the server clients point to our fake clients
	// so getLocalDevK8sClients() returns the fake clients instead of real ones

	// Set up the server clients to use fake clients
	server.DynamicClient = k8sUtils.DynamicClient

	// Store the fake clients for tests
	setFakeK8sClientForTests(k8sUtils.K8sClient)

	// IMPORTANT: Set server.K8sClient to the fake client instead of creating a real one
	// The fake clientset implements kubernetes.Interface, which is what getLocalDevK8sClients() returns
	// We need to type-assert it to *kubernetes.Clientset for server.K8sClient
	if fakeClientset, ok := k8sUtils.K8sClient.(*kubernetes.Clientset); ok {
		server.K8sClient = fakeClientset
		logger.Log("Set server.K8sClient to real fake clientset")
	} else {
		// k8sUtils.K8sClient is a fake.Clientset, not *kubernetes.Clientset
		// For tests, we need to ensure getLocalDevK8sClients() doesn't use a real HTTP client
		// Since we can't easily convert types, we'll set server.K8sClient to nil
		// and handle this in getLocalDevK8sClients() by checking the interface first
		server.K8sClient = nil
		logger.Log("Set server.K8sClient to nil (fake client interface mismatch)")
	}

	// Set up other handler dependencies with mock implementations
	handlers.GetGitHubToken = func(ctx context.Context, k8sClient *kubernetes.Clientset, dynClient dynamic.Interface, namespace, userID string) (string, error) {
		return "fake-github-token", nil
	}

	handlers.DeriveRepoFolderFromURL = func(url string) string {
		// Simple mock implementation
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return "repo"
	}

	// Set up specific handler clients if they exist
	// Note: handlers.K8sClientProjects may also have type conversion issues,
	// but many handlers will use GetK8sClientsForRequest() instead
	// so we'll skip setting it for now

	// Set up additional handler functions that may be needed
	handlers.SendMessageToSession = func(sessionID, userID string, message map[string]interface{}) {
		// Mock implementation - no-op for tests
	}

	// Set up environment variables for tests
	// Note: GO_TEST and DISABLE_AUTH are true by default from config
	// This ensures GetK8sClientsForRequest doesn't route to getLocalDevK8sClients()
	// and instead returns fake clients, which allows tests to control client behavior directly
	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("NAMESPACE", "vteam-dev") // Use whitelisted dev namespace
	os.Setenv("DISABLE_AUTH", "true")   // Enable auth bypass by default in tests

	logger.Log("Handler dependencies set up with fake clients and local dev mode enabled")
}

// WithAuthCheckEnabled temporarily enables auth checks for a test by overriding GetK8sClientsFunc
// Returns a restore function that should be called in defer
// Usage:
//
//	restore := test_utils.WithAuthCheckEnabled()
//	defer restore()
func WithAuthCheckEnabled() func() {
	originalFunc := handlers.GetK8sClientsFunc
	handlers.SetGetK8sClientsFunc(func(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
		// Force auth check - return nil if no auth header
		auth := c.GetHeader("Authorization")
		if auth == "" {
			auth = c.GetHeader("X-Forwarded-Access-Token")
		}
		if auth == "" {
			return nil, nil
		}
		// Call original function with auth check
		return originalFunc(c)
	})
	return func() {
		handlers.SetGetK8sClientsFunc(originalFunc)
	}
}

// WithAuthCheckDisabled temporarily disables auth checks for a test by overriding GetK8sClientsFunc
// Returns a restore function that should be called in defer
// Usage:
//
//	restore := test_utils.WithAuthCheckDisabled()
//	defer restore()
//
// Note: This requires k8sUtils to be passed or stored. For now, tests should use DISABLE_AUTH=true
// or override GetK8sClientsFunc directly in their BeforeEach.
func WithAuthCheckDisabled() func() {
	originalFunc := handlers.GetK8sClientsFunc
	// This function is deprecated - tests should set DISABLE_AUTH=true or override GetK8sClientsFunc directly
	// Keeping for backward compatibility but it won't work without access to k8sUtils
	handlers.SetGetK8sClientsFunc(func(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
		// Bypass auth check - call original function which should return fake clients
		return originalFunc(c)
	})
	return func() {
		handlers.SetGetK8sClientsFunc(originalFunc)
	}
}

// setFakeK8sClientForTests stores the fake K8s client for use in tests
func setFakeK8sClientForTests(client kubernetes.Interface) {
	fakeK8sClientForTests = client
}

// GetFakeK8sClientForTests returns the fake K8s client for tests
func GetFakeK8sClientForTests() kubernetes.Interface {
	return fakeK8sClientForTests
}
