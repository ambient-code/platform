// Package test_utils provides general testing utilities following KFP patterns
package test_utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"ambient-code-backend/tests/logger"

	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

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

// NOTE: SetupHandlerDependencies has been moved to handlers package (handlers/test_helpers.go)
// Tests in the handlers package can call handlers.SetupHandlerDependencies directly.
// This avoids import cycles since tests are now in the same package as handlers.

// NOTE: K8s client injection for handler tests now lives in `handlers/test_helpers_test.go`.
