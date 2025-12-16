# Ambient Code Backend Testing Guide

This comprehensive guide will help you understand, run, and write tests for the ambient-code-backend using our Ginkgo-based test framework.

## 🎯 Quick Start

### Prerequisites

1. **Go 1.24+**: Ensure you have Go installed
   ```bash
   go version  # Should show 1.24 or higher
   ```

2. **Install Test Tools**:
   ```bash
   cd components/backend
   make install-tools  # Installs Ginkgo CLI and other tools
   ```

3. **Verify Setup**:
   ```bash
   ginkgo version  # Should show Ginkgo v2.x.x
   ```

### Run Your First Test

```bash
# Navigate to backend directory
cd components/backend

# Run all unit tests
make test-unit

# Or run with Ginkgo directly
ginkgo run --label-filter="unit" -v
```

Expected output:
```
Running Suite: Ambient Code Backend Test Suite
===============================================
[unit, handlers, health] Health Handler
  ✓ Should return 200 OK with health status
[unit, handlers, middleware] Middleware Handlers
  ✓ Should accept valid Kubernetes namespace names

SUCCESS! -- 5 Passed | 0 Failed | 0 Pending | 0 Skipped
```

## 📁 Understanding the Test Structure

### Directory Layout
```
components/backend/
├── tests/                   # Shared test framework utilities (not production code)
│   ├── config/             # Test configuration management
│   ├── logger/             # Test logging utilities
│   ├── test_utils/         # Reusable test utilities (HTTP/K8s fakes, token helpers)
├── handlers/               # Business logic + tests
│   ├── health.go
│   ├── health_test.go     # ✅ Tests for health.go
│   ├── sessions.go
│   └── sessions_test.go   # ✅ Tests for sessions.go
├── types/
│   ├── common.go
│   └── common_test.go     # ✅ Tests for common.go
└── git/
    ├── operations.go
    └── operations_test.go # ✅ Tests for operations.go
```

### Why This Structure?

1. **Co-location**: Tests live next to the code they test (easier to find and maintain)
2. **Shared Utilities**: Common test logic in `tests/` directory (avoid duplication)
3. **Import Pattern**: Test packages import utilities with `"ambient-code-backend/tests/..."`

## 🏷️ Test Labels and Categories

Our tests use labels for organization and filtering:

| Label | Purpose | Example Usage |
|-------|---------|---------------|
| `unit` | Pure unit tests, no external dependencies | Testing business logic |
| `integration` | Tests requiring real Kubernetes cluster | End-to-end workflows |
| `handlers` | HTTP handler tests | API endpoint testing |
| `types` | Type and utility function tests | Data structure validation |
| `git` | Git operation tests | Repository operations |
| `auth` | Authentication/authorization tests | Security testing |
| `slow` | Time-consuming tests | Performance tests |

### Running Specific Test Categories

```bash
# Run only handler tests
make test-handlers
# or
ginkgo run --label-filter="handlers" -v

# Run everything except slow tests
make test-fast
# or
ginkgo run --label-filter="!slow"

# Run auth tests only
make test-auth
# or
ginkgo run --label-filter="auth" -v

# Combine filters (unit tests for handlers, excluding slow ones)
ginkgo run --label-filter="unit && handlers && !slow" -v
```

### Integration Tests (real cluster)

Integration tests live under `components/backend/tests/integration/` and are intended to run against a real Kubernetes/OpenShift cluster.

For local development authentication setup (since `DISABLE_AUTH` is not supported), see:
- `components/backend/README.md` → **Local development authentication (DISABLE_AUTH removed)**

```bash
cd components/backend

# Run Go integration tests directly (these are standard `go test` suites)
go test ./tests/integration/... -count=1

# If you are running a Ginkgo integration suite (labelled `integration`), use:
ginkgo run --label-filter="integration" -v
```

Common expectations for integration tests:

- You may need to set **`TEST_NAMESPACE`** and ensure your kubeconfig points to a cluster you can modify.
- Prefer cleaning up resources created during tests (namespaces, rolebindings, secrets).
- For RBAC validation, use `SetValidTestToken(...)` with real Roles/RoleBindings created in the test namespace.

## 🔧 Test Execution Options

### Basic Execution

```bash
# Most common: Run unit tests with reports
make test-unit

# Run all tests (unit + integration)
make test-all

# Run with verbose output
make test-ginkgo-verbose

# Run tests in parallel (faster)
make test-ginkgo-parallel
```

### Advanced Execution

```bash
# Focus on specific test by name
make test-focus FOCUS="Should return 200 OK"

# Run with custom configuration
VERBOSE=true SKIP_SLOW_TESTS=true ginkgo run

# Run with timeout
ginkgo run --timeout=10m

# Generate coverage report
go test -cover ./handlers ./types ./git
```

### Environment Variables

Configure test behavior with environment variables:

```bash
# Test execution
export VERBOSE="true"                    # Enable verbose logging
export SKIP_SLOW_TESTS="true"           # Skip performance tests
export PARALLEL_NODES="4"               # Run 4 tests in parallel

# Test environment
export TEST_NAMESPACE="my-test-ns"       # Custom test namespace
export USE_REAL_CLUSTER="false"         # Use fake K8s clients (default)
export CLEANUP_RESOURCES="true"         # Clean up after tests

# Reporting
export ENABLE_REPORTING="true"          # Generate test reports
export REPORTS_DIR="custom-reports"     # Custom report directory
export LOGS_DIR="custom-logs"          # Custom log directory

# Timeouts
export SUITE_TIMEOUT="30m"             # Max time for entire test suite
export TEST_TIMEOUT="5m"               # Max time per individual test
export API_TIMEOUT="30s"               # Max time for API calls
```

## ✍️ Writing Your First Test

### Step 1: Create the Test File

If you're adding tests for `components/backend/handlers/projects.go`:

```bash
# Create the test file
touch components/backend/handlers/projects_test.go
```

### Step 2: Basic Test Structure

```go
package handlers_test

import (
    "net/http"

    "ambient-code-backend/handlers"
    "ambient-code-backend/tests/logger"
    "ambient-code-backend/tests/test_utils"

    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
)

var _ = Describe("Projects Handler", Label("unit", "handlers", "projects"), func() {
    var (
        httpUtils *test_utils.HTTPTestUtils
        k8sUtils  *test_utils.K8sTestUtils
    )

    BeforeEach(func() {
        logger.Log("Setting up Projects Handler test")
        httpUtils = test_utils.NewHTTPTestUtils()
        k8sUtils = test_utils.NewK8sTestUtils(false, "test-namespace")
    })

    Context("When creating a project", func() {
        It("Should create project successfully", func() {
            // Arrange - Set up test data
            projectRequest := map[string]interface{}{
                "name": "test-project",
                "description": "Test project description",
            }

            context := httpUtils.CreateTestGinContext("POST", "/api/projects", projectRequest)
            httpUtils.SetAuthHeader("test-token")
            httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

            // Act - Call the handler
            handlers.CreateProject(context)

            // Assert - Check the results
            httpUtils.AssertHTTPStatus(http.StatusCreated)

            var response map[string]interface{}
            httpUtils.GetResponseJSON(&response)
            Expect(response).To(HaveKey("name"))
            Expect(response["name"]).To(Equal("test-project"))

            logger.Log("Project created successfully: %s", response["name"])
        })

        It("Should reject invalid project names", func() {
            // Test edge case - invalid input
            projectRequest := map[string]interface{}{
                "name": "Invalid Project Name!", // Invalid characters
            }

            context := httpUtils.CreateTestGinContext("POST", "/api/projects", projectRequest)
            httpUtils.SetAuthHeader("test-token")

            // Act
            handlers.CreateProject(context)

            // Assert
            httpUtils.AssertHTTPStatus(http.StatusBadRequest)
            httpUtils.AssertErrorMessage("Invalid project name")
        })
    })
})
```

### Step 3: Test Different Scenarios

```go
Context("When listing projects", func() {
    BeforeEach(func() {
        // Create test data for each test in this context
        createTestProject("project-1", "test-namespace")
        createTestProject("project-2", "test-namespace")
    })

    It("Should return all projects", func() {
        context := httpUtils.CreateTestGinContext("GET", "/api/projects", nil)
        httpUtils.SetAuthHeader("test-token")

        handlers.ListProjects(context)

        httpUtils.AssertHTTPStatus(http.StatusOK)

        var response map[string]interface{}
        httpUtils.GetResponseJSON(&response)
        Expect(response).To(HaveKey("items"))

        items := response["items"].([]interface{})
        Expect(items).To(HaveLen(2))
    })

    It("Should support pagination", func() {
        context := httpUtils.CreateTestGinContext("GET", "/api/projects?limit=1", nil)
        httpUtils.SetAuthHeader("test-token")

        handlers.ListProjects(context)

        httpUtils.AssertHTTPStatus(http.StatusOK)

        var response map[string]interface{}
        httpUtils.GetResponseJSON(&response)

        items := response["items"].([]interface{})
        Expect(items).To(HaveLen(1))
        Expect(response).To(HaveKey("hasMore"))
        Expect(response["hasMore"]).To(BeTrue())
    })
})
```

### Step 4: Add Helper Functions

```go
// Helper function to create test projects
func createTestProject(name, namespace string) {
    project := &unstructured.Unstructured{
        Object: map[string]interface{}{
            "apiVersion": "v1",
            "kind":       "Namespace",
            "metadata": map[string]interface{}{
                "name": name,
                "labels": map[string]interface{}{
                    "test-framework": "ambient-code-backend",
                },
            },
        },
    }

    k8sUtils := test_utils.NewK8sTestUtils(false, namespace)
    k8sUtils.CreateCustomResource(context.Background(),
        schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"},
        "", project)
}
```

## 🧪 Common Testing Patterns

### 1. HTTP Handler Testing

```go
It("Should handle POST request with JSON body", func() {
    // Arrange
    requestBody := map[string]interface{}{
        "field1": "value1",
        "field2": 123,
        "nested": map[string]interface{}{
            "key": "value",
        },
    }

    context := httpUtils.CreateTestGinContext("POST", "/api/endpoint", requestBody)
    httpUtils.SetAuthHeader("bearer-token")
    httpUtils.SetProjectContext("my-project")

    // Act
    handlers.MyHandler(context)

    // Assert
    httpUtils.AssertHTTPStatus(http.StatusOK)
    httpUtils.AssertJSONContains(map[string]interface{}{
        "status": "success",
        "id": BeNumerically(">", 0), // Using Gomega matcher
    })
})
```

### 2. Authentication Testing

```go
It("Should require authentication", func() {
    // No auth header set
    context := httpUtils.CreateTestGinContext("GET", "/api/secure-endpoint", nil)

    handlers.SecureHandler(context)

    httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
    httpUtils.AssertErrorMessage("Authentication required")
})

It("Should accept valid token", func() {
    context := httpUtils.CreateTestGinContext("GET", "/api/secure-endpoint", nil)
    // For simple tests, arbitrary token is fine:
    httpUtils.SetAuthHeader("valid-token")

    handlers.SecureHandler(context)

    httpUtils.AssertHTTPSuccess() // Any 2xx status
})

It("Should validate RBAC permissions", func() {
    // Create a token with actual RBAC permissions
    // This ensures tests match production RBAC behavior
    context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/agentic-sessions", nil)
    // NOTE: create the namespace + Role needed for the test in BeforeEach
    token, _, err := httpUtils.SetValidTestToken(
        k8sUtils,
        "test-project",                    // namespace
        []string{"get", "list"},           // verbs
        "agenticsessions",                 // resource
        "",                                // auto-generate SA name
        "test-agenticsessions-read-role",  // pre-created Role name
    )
    Expect(err).NotTo(HaveOccurred())
    // Token is automatically set in Authorization header by SetValidTestToken

    handlers.ListSessions(context)

    httpUtils.AssertHTTPSuccess()
    // This test verifies that the handler works with real RBAC permissions,
    // not just arbitrary tokens that bypass security checks
})
```

### 3. Kubernetes Resource Testing

```go
It("Should create Kubernetes resource", func() {
    // Arrange
    resource := &unstructured.Unstructured{
        Object: map[string]interface{}{
            "apiVersion": "vteam.ambient-code/v1alpha1",
            "kind":       "AgenticSession",
            "metadata": map[string]interface{}{
                "name": "test-session",
            },
            "spec": map[string]interface{}{
                "initialPrompt": "Test prompt",
            },
        },
    }

    gvr := schema.GroupVersionResource{
        Group:    "vteam.ambient-code",
        Version:  "v1alpha1",
        Resource: "agenticsessions",
    }

    // Act
    created := k8sUtils.CreateCustomResource(ctx, gvr, "test-namespace", resource)

    // Assert
    Expect(created).NotTo(BeNil())
    Expect(created.GetName()).To(Equal("test-session"))

    // Verify it exists
    k8sUtils.AssertResourceExists(ctx, gvr, "test-namespace", "test-session")
})
```

### 4. Error Handling Testing

```go
Context("When handling errors", func() {
    It("Should return 400 for invalid input", func() {
        // Test with malformed JSON
        context := httpUtils.CreateTestGinContext("POST", "/api/endpoint", "invalid-json")

        handlers.MyHandler(context)

        httpUtils.AssertHTTPStatus(http.StatusBadRequest)
    })

    It("Should return 404 for missing resource", func() {
        context := httpUtils.CreateTestGinContext("GET", "/api/projects/nonexistent", nil)
        // This sets an arbitrary token to satisfy handlers that require an auth header.
        // It does NOT validate RBAC permissions. For RBAC tests, use SetValidTestToken.
        httpUtils.SetAuthHeader("any-token")

        handlers.GetProject(context)

        httpUtils.AssertHTTPStatus(http.StatusNotFound)
        httpUtils.AssertErrorMessage("Project not found")
    })
})
```

### 5. Async and Retry Testing

```go
It("Should retry failed operations", func() {
    attempt := 0
    operation := func() error {
        attempt++
        if attempt < 3 {
            return fmt.Errorf("simulated failure")
        }
        return nil
    }

    // Use test utility for retry logic
    err := test_utils.RetryOperation(operation, 5, 100*time.Millisecond)

    Expect(err).NotTo(HaveOccurred())
    Expect(attempt).To(Equal(3))
})
```

## 🛠️ Test Utilities Reference

### HTTP Utils (`test_utils.HTTPTestUtils`)

**Important**: For tests that need to validate RBAC permissions (matching production security model), use `SetValidTestToken` instead of `SetAuthHeader` with arbitrary tokens. This ensures tests use tokens that would work with real RBAC, not just strings that bypass security checks.

```go
httpUtils := test_utils.NewHTTPTestUtils()

// Create contexts
context := httpUtils.CreateTestGinContext("GET", "/path", body)

// Set headers
httpUtils.SetAuthHeader("token")  // Simple token (no RBAC validation)
// For tests that need RBAC validation, use SetValidTestToken:
token, saName, err := httpUtils.SetValidTestToken(
    k8sUtils,
    "namespace",
    []string{"get", "list", "create"},
    "agenticsessions",
    "",                       // optional SA name
    "test-agenticsessions-write-role", // pre-created Role name
)
Expect(err).NotTo(HaveOccurred())
// Token is automatically set in Authorization header
httpUtils.SetUserContext("userID", "userName", "user@email.com")
httpUtils.SetProjectContext("projectName")

// Assertions
httpUtils.AssertHTTPStatus(200)
httpUtils.AssertHTTPSuccess()          // Any 2xx
httpUtils.AssertHTTPError()            // Any 4xx/5xx
httpUtils.AssertJSONContains(map[string]interface{}{"key": "value"})
httpUtils.AssertJSONStructure([]string{"id", "name", "status"})
httpUtils.AssertErrorMessage("Expected error message")

// Get responses
body := httpUtils.GetResponseBody()
var data MyStruct
httpUtils.GetResponseJSON(&data)
```

### Kubernetes Utils (`test_utils.K8sTestUtils`)

```go
k8sUtils := test_utils.NewK8sTestUtils(false, "namespace") // false = use fake clients

// Resource operations
created := k8sUtils.CreateCustomResource(ctx, gvr, namespace, resource)
resource, err := k8sUtils.GetCustomResource(ctx, gvr, namespace, name)
updated, err := k8sUtils.UpdateCustomResource(ctx, gvr, resource)
err := k8sUtils.DeleteCustomResource(ctx, gvr, namespace, name)

// Assertions
k8sUtils.AssertResourceExists(ctx, gvr, namespace, name)
k8sUtils.AssertResourceNotExists(ctx, gvr, namespace, name)
k8sUtils.AssertResourceHasStatus(ctx, gvr, namespace, name, map[string]interface{}{
    "phase": "Running",
    "ready": true,
})

// Secrets and ConfigMaps
secret := k8sUtils.CreateSecret(ctx, namespace, name, data)
configMap := k8sUtils.CreateConfigMap(ctx, namespace, name, data)

// Cleanup
k8sUtils.CleanupTestResources(ctx, namespace)
```

### General Utils (`test_utils`)

```go
// Random data generation
randomString := test_utils.GetRandomString(10)
testID := test_utils.GenerateTestID("test")

// Pointer helpers
stringPtr := test_utils.StringPtr("value")
intPtr := test_utils.IntPtr(42)
boolPtr := test_utils.BoolPtr(true)

// Operations
err := test_utils.RetryOperation(func() error {
    return someOperation()
}, 3, time.Second)

// Logging
test_utils.WriteLogFile(specReport, "test-name", "logs/")
```

## 🔍 Debugging Tests

### 1. Running Single Tests

```bash
# Run specific test by name
ginkgo run --focus="Should create project successfully"

# Run specific describe block
ginkgo run --focus="Projects Handler"

# Run tests matching pattern
ginkgo run --focus="create.*project"
```

### 2. Debugging Output

```bash
# Verbose output with test progress
ginkgo run -v

# Show stack traces on failure
ginkgo run --trace

# Keep going after first failure (default stops)
ginkgo run --keep-going
```

### 3. Test Logs and Reports

After running tests, check these locations:

```bash
# Test reports
ls reports/
# junit.xml - For CI integration
# results.json - Machine-readable results
# test_summary.txt - Human-readable summary

# Failure logs
ls logs/
# Contains detailed logs for failed tests
# Stack traces and captured output
```

### 4. Common Debugging Patterns

Add debug output to your tests:

```go
It("Should debug issue", func() {
    logger.Log("Debug: Starting test with value %v", testValue)

    // Add intermediate assertions
    Expect(preliminaryResult).NotTo(BeNil(), "Preliminary result should exist")
    logger.Log("Debug: Preliminary result: %v", preliminaryResult)

    // Use GinkgoWriter for output that appears in reports
    GinkgoWriter.Printf("Debug info: %+v\n", complexObject)

    // Final assertion
    Expect(finalResult).To(Equal(expectedValue))
})
```

### 5. Investigating Failures

When a test fails:

1. **Check the failure message**: Shows expected vs actual values
2. **Review the logs**: Look in `logs/` directory for detailed output
3. **Run with verbose**: `ginkgo run -v --focus="failing test"`
4. **Add debug logging**: Use `logger.Log()` to trace execution
5. **Isolate the test**: Run just that one test to avoid interference

## 📊 Test Reports and CI Integration

### Local Report Generation

```bash
# Generate reports
ginkgo run --junit-report=reports/junit.xml --json-report=reports/results.json

# View coverage
go test -cover ./handlers ./types ./git
go test -coverprofile=coverage.out ./handlers ./types ./git
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### CI Integration Example (GitHub Actions)

```yaml
name: Backend Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest

    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.24

    - name: Install dependencies
      run: |
        cd components/backend
        go mod download
        make install-tools

    - name: Run tests
      run: |
        cd components/backend
        export ENABLE_REPORTING="true"
        export SKIP_SLOW_TESTS="true"
        make test-unit

    - name: Upload test results
      uses: actions/upload-artifact@v3
      if: always()
      with:
        name: test-results
        path: components/backend/reports/

    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: components/backend/coverage.out
```

## 🚨 Troubleshooting

### Common Issues

#### "ginkgo: command not found"
```bash
# Install Ginkgo CLI
go install github.com/onsi/ginkgo/v2/ginkgo@latest

# Or use make target
make install-tools
```

#### "package not found" errors
```bash
# Update dependencies
go mod tidy
go mod download
```

#### Tests hang or timeout
```bash
# Check for goroutine leaks
export GOMAXPROCS=1  # Limit concurrency for debugging

# Run with shorter timeout
ginkgo run --timeout=30s

# Add timeout to specific test
It("Should complete quickly", func(SpecContext) {
    // Test will be cancelled after default timeout (5min)
}, SpecTimeout(30*time.Second))
```

#### Import cycle errors
```bash
# Common cause: importing handler package from handler test
# Solution: Use separate _test package name

package handlers_test  // Not: package handlers

import (
    "ambient-code-backend/handlers"  // Import the package under test
    . "github.com/onsi/ginkgo/v2"
)
```

#### Tests fail due to permissions
```bash
# For integration tests, ensure proper RBAC
kubectl auth can-i create agenticsessions.vteam.ambient-code --namespace=test-namespace

# Check test namespace exists
kubectl get namespace test-namespace

# Reset test environment
make k8s-teardown
make k8s-setup
```

### Getting Help

1. **Review test logs**: Check `logs/` directory for detailed error information
2. **Run with verbose output**: `ginkgo run -v` shows test progress
3. **Review this guide**: See `TEST_GUIDE.md` for comprehensive testing documentation
4. **Examine existing tests**: Look at `handlers/*_test.go` for patterns
5. **Ginkgo documentation**: https://onsi.github.io/ginkgo/
6. **Gomega matchers**: https://onsi.github.io/gomega/

### Performance Optimization

If tests are running slowly:

```bash
# Run in parallel
ginkgo run -p

# Skip slow tests during development
export SKIP_SLOW_TESTS=true
make test-fast

# Profile test execution
ginkgo run --json-report=results.json
# Check results.json for test timings
```

## 📝 Test Writing Checklist

Before submitting your tests:

- [ ] **Test file named correctly**: `*_test.go`
- [ ] **Package name**: Use `package xyz_test` pattern
- [ ] **Imports**: Include required test utilities
- [ ] **Labels**: Add appropriate labels (`unit`, `handlers`, etc.)
- [ ] **Descriptive names**: Test descriptions explain what is being tested
- [ ] **AAA pattern**: Arrange, Act, Assert structure
- [ ] **Edge cases**: Test both success and failure scenarios
- [ ] **Cleanup**: Use `BeforeEach`/`AfterEach` for setup/teardown
- [ ] **No side effects**: Tests don't affect each other
- [ ] **Assertions**: Use descriptive Gomega matchers
- [ ] **Logging**: Add `logger.Log()` statements for debugging

### Code Review Guidelines

When reviewing test code:

- [ ] **Test coverage**: Are all important code paths tested?
- [ ] **Test quality**: Do tests actually verify the intended behavior?
- [ ] **Maintainability**: Are tests easy to understand and modify?
- [ ] **Performance**: Are slow tests marked with `slow` label?
- [ ] **Documentation**: Are complex test scenarios explained?
- [ ] **Consistency**: Do tests follow established patterns?

---

## 🎉 Conclusion

You now have a comprehensive understanding of the ambient-code-backend test framework!

**Quick reminder of the most important commands:**

```bash
# Install tools and run tests
make install-tools
make test-unit

# Debug failing test
ginkgo run --focus="failing test name" -v

# Run specific category
make test-handlers

# Skip slow tests
make test-fast
```

The framework is designed to make testing easy and comprehensive. When in doubt, look at existing tests in `handlers/*_test.go` for patterns, and don't hesitate to add debug logging with `logger.Log()` to understand what's happening in your tests.

Happy testing! 🚀