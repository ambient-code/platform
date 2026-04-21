//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"net/http"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CodeRabbit Auth Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelCodeRabbitAuth), func() {
	var (
		httpUtils                        *test_utils.HTTPTestUtils
		k8sUtils                         *test_utils.K8sTestUtils
		originalNamespace                string
		originalValidateCodeRabbitAPIKey func(context.Context, string) (bool, error)
		testToken                        string
	)

	BeforeEach(func() {
		logger.Log("Setting up CodeRabbit Auth Handler test")

		originalNamespace = Namespace
		originalValidateCodeRabbitAPIKey = ValidateCodeRabbitAPIKey
		ValidateCodeRabbitAPIKey = func(_ context.Context, _ string) (bool, error) { return true, nil }

		// Use centralized handler dependencies setup
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// coderabbit_auth.go uses Namespace (backend namespace) for some secret operations
		Namespace = *config.TestNamespace

		httpUtils = test_utils.NewHTTPTestUtils()

		// Create namespace + role and mint a valid test token for this suite
		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: *config.TestNamespace},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
		_, err = k8sUtils.CreateTestRole(ctx, *config.TestNamespace, "test-full-access-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := httpUtils.SetValidTestToken(
			k8sUtils,
			*config.TestNamespace,
			[]string{"get", "list", "create", "update", "delete", "patch"},
			"*",
			"",
			"test-full-access-role",
		)
		Expect(err).NotTo(HaveOccurred())
		testToken = token
	})

	AfterEach(func() {
		Namespace = originalNamespace
		ValidateCodeRabbitAPIKey = originalValidateCodeRabbitAPIKey

		// Clean up created namespace (best-effort)
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), *config.TestNamespace, metav1.DeleteOptions{})
		}
	})

	Context("Connection Management", func() {
		Describe("ConnectCodeRabbit", func() {
			It("Should require authentication", func() {
				requestBody := map[string]interface{}{
					"apiKey": "cr_test_key_1234567890",
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", requestBody)
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require user authentication", func() {
				requestBody := map[string]interface{}{
					"apiKey": "cr_test_key_1234567890",
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", requestBody)
				// Don't set auth header or user context

				ConnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid or missing token",
				})
			})

			It("Should reject empty API key", func() {
				requestBody := map[string]interface{}{
					"apiKey": "",
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should reject missing API key field", func() {
				requestBody := map[string]interface{}{
					// apiKey missing
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectCodeRabbit(context)

				// Gin binding returns detailed validation error message
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should reject invalid API key", func() {
				// Mock validation to return false
				ValidateCodeRabbitAPIKey = func(_ context.Context, _ string) (bool, error) { return false, nil }

				requestBody := map[string]interface{}{
					"apiKey": "invalid_key_123",
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid CodeRabbit API key",
				})
			})

			It("Should store credentials successfully with valid API key", func() {
				// Mock validation to return true
				ValidateCodeRabbitAPIKey = func(_ context.Context, _ string) (bool, error) { return true, nil }

				requestBody := map[string]interface{}{
					"apiKey": "cr_valid_key_1234567890",
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "CodeRabbit connected successfully",
				})
			})

			It("Should handle invalid user ID type", func() {
				requestBody := map[string]interface{}{
					"apiKey": "cr_test_key_1234567890",
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				context.Set("userID", 123) // Invalid type (should be string)

				ConnectCodeRabbit(context)

				// GetString returns empty string for non-string types, triggers auth required error
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User authentication required")
			})

			It("Should require valid JSON body", func() {
				context := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", "invalid-json")
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectCodeRabbit(context)

				// Gin binding returns detailed validation error
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})
		})

		Describe("GetCodeRabbitStatus", func() {
			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/auth/coderabbit/status", nil)
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				GetCodeRabbitStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/auth/coderabbit/status", nil)
				// Don't set auth header or user context

				GetCodeRabbitStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid or missing token",
				})
			})

			It("Should return connected:false when no credentials stored", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/auth/coderabbit/status", nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				GetCodeRabbitStatus(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"connected": false,
				})
			})

			It("Should handle invalid user ID type", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/auth/coderabbit/status", nil)
				httpUtils.SetAuthHeader(testToken)
				context.Set("userID", 123) // Invalid type

				GetCodeRabbitStatus(context)

				// GetString returns empty string for non-string types, triggers auth required error
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User authentication required")
			})
		})

		Describe("DisconnectCodeRabbit", func() {
			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("DELETE", "/api/auth/coderabbit/disconnect", nil)
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				DisconnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("DELETE", "/api/auth/coderabbit/disconnect", nil)
				// Don't set auth header or user context

				DisconnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid or missing token",
				})
			})

			It("Should succeed idempotently when no credentials exist", func() {
				context := httpUtils.CreateTestGinContext("DELETE", "/api/auth/coderabbit/disconnect", nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				DisconnectCodeRabbit(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "CodeRabbit disconnected successfully",
				})
			})
		})
	})

	Context("Data Structure Validation", func() {
		Describe("Request and Response Types", func() {
			It("Should validate CodeRabbitCredentials structure", func() {
				creds := CodeRabbitCredentials{
					UserID:    "user123",
					APIKey:    "cr_test_key_1234567890",
					UpdatedAt: metav1.Now().Time,
				}

				Expect(creds.UserID).To(Equal("user123"))
				Expect(creds.APIKey).To(Equal("cr_test_key_1234567890"))
				Expect(creds.UpdatedAt).NotTo(BeZero())
			})
		})
	})
})
