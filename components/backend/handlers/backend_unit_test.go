//go:build test

// Package test contains the Ginkgo test suite for ambient-code-backend
package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ambient-code-backend/tests/config"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Global test context and utilities
var (
	ctx        context.Context
	cancel     context.CancelFunc
	k8sUtils   *test_utils.K8sTestUtils
	httpUtils  *test_utils.HTTPTestUtils
	testClient kubernetes.Interface
	dynClient  dynamic.Interface
)

var (
	testLogsDirectory   = "logs"
	testReportDirectory = "reports"
)

// BeforeSuite runs before all tests in the suite
var _ = BeforeSuite(func() {
	logger.Log("Initializing test suite...")

	// Set up test context with timeout
	ctx, cancel = context.WithTimeout(context.Background(), *config.SuiteTimeout)

	// Set environment to local
	err := os.Setenv("ENVIRONMENT", "local")
	Expect(err).NotTo(HaveOccurred(), "Error setting environment to local")

	// NOTE: No auth bypass environment variables are supported/used in tests.
	// Handler tests must set Authorization headers / valid tokens explicitly.

	// Initialize Kubernetes test utilities
	logger.Log("Setting up Kubernetes test utilities...")
	k8sUtils = test_utils.NewK8sTestUtils(
		*config.UseRealCluster,
		*config.TestNamespace,
	)

	// Store clients for global access
	testClient = k8sUtils.K8sClient
	dynClient = k8sUtils.DynamicClient

	// Initialize HTTP test utilities
	logger.Log("Setting up HTTP test utilities...")
	httpUtils = test_utils.NewHTTPTestUtils()

	// Create test namespace if using real cluster
	if *config.UseRealCluster {
		logger.Log("Creating test namespace: %s", *config.TestNamespace)
		err := k8sUtils.CreateNamespace(ctx, *config.TestNamespace)
		Expect(err).NotTo(HaveOccurred(), "Failed to create test namespace")
	}

	// Log test configuration
	logger.Log("=== Test Suite Configuration ===")
	logger.Log("Test Namespace: %s", *config.TestNamespace)
	logger.Log("Use Real Cluster: %v", *config.UseRealCluster)
	logger.Log("Suite Timeout: %s", *config.SuiteTimeout)
	logger.Log("Test Timeout: %s", *config.TestTimeout)
	logger.Log("Skip Slow Tests: %v", *config.SkipSlowTests)
	logger.Log("================================")

	// Wait for environment to be ready
	Eventually(func() bool {
		return testClient != nil && dynClient != nil && httpUtils != nil
	}, 30*time.Second, 1*time.Second).Should(BeTrue(), "Test environment should be ready")

	logger.Log("Test suite initialization complete")
})

// AfterSuite runs after all tests in the suite
var _ = AfterSuite(func() {
	logger.Log("=== Test Suite Cleanup ===")

	// Clean up test resources
	if k8sUtils != nil {
		logger.Log("Cleaning up test resources in namespace: %s", *config.TestNamespace)
		k8sUtils.CleanupTestResources(ctx, *config.TestNamespace)

		// Delete test namespace if using real cluster and cleanup is enabled
		if *config.UseRealCluster && *config.CleanupResources {
			err := k8sUtils.DeleteNamespace(ctx, *config.TestNamespace)
			if err != nil {
				logger.Log("Warning: Failed to delete test namespace: %v", err)
			} else {
				logger.Log("Deleted test namespace: %s", *config.TestNamespace)
			}
		}
	}

	// Cancel context
	if cancel != nil {
		cancel()
	}

	logger.Log("=== Suite Cleanup Complete ===")
})

// ReportAfterEach captures test failures and logs following KFP pattern
var _ = ReportAfterEach(func(specReport SpecReport) {
	if specReport.Failed() {
		logger.Log("Test failed... Capturing logs")
		AddReportEntry("Test Log", specReport.CapturedGinkgoWriterOutput)

		// Write failure log to file
		currentDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred(), "Failed to get current directory")

		testName := GinkgoT().Name()
		testNameSplit := strings.Split(testName, ">")
		finalTestName := testNameSplit[len(testNameSplit)-1]

		test_utils.WriteLogFile(specReport, finalTestName, filepath.Join(currentDir, testLogsDirectory))
	} else {
		logger.Log("Test passed: %s", specReport.FullText())
	}
})

// TestBackend runs the Ginkgo test suite
func TestBackend(t *testing.T) {
	RegisterFailHandler(Fail)

	err := os.MkdirAll(testLogsDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Logs Directory: %s", testLogsDirectory))
	err = os.MkdirAll(testReportDirectory, 0755)
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error creating Reports Directory: %s", testReportDirectory))

	// SECURITY: Do not set DISABLE_AUTH/GO_TEST in unit tests. Use explicit tokens/headers instead.

	// Configure suite and reporter
	suiteConfig, reporterConfig := GinkgoConfiguration()

	// Apply configuration
	suiteConfig.RandomizeAllSpecs = true
	suiteConfig.FailOnPending = true
	suiteConfig.FailFast = false
	suiteConfig.FlakeAttempts = *config.FlakeAttempts

	// Run the test suite
	RunSpecs(t, "Ambient Code Backend Test Suite", suiteConfig, reporterConfig)
}
