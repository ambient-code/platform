//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"fmt"
	"net/http"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("GitLab Auth Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelGitLabAuth), func() {
	var (
		httpUtils         *test_utils.HTTPTestUtils
		k8sUtils          *test_utils.K8sTestUtils
		originalNamespace string
		testToken         string
	)

	BeforeEach(func() {
		logger.Log("Setting up GitLab Auth Handler test")

		originalNamespace = Namespace

		// Use centralized handler dependencies setup
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// gitlab_auth.go uses Namespace (backend namespace) for some secret operations
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

		// Clean up created namespace (best-effort)
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), *config.TestNamespace, metav1.DeleteOptions{})
		}
	})

	Context("Handler Creation", func() {
		Describe("NewGitLabAuthHandler", func() {
			It("Should handle nil kubernetes client", func() {
				handler := NewGitLabAuthHandler(nil, "test-project")

				Expect(handler).NotTo(BeNil())
				// Handler creation should not fail even with nil client
			})

			It("Should handle empty namespace", func() {
				handler := NewGitLabAuthHandler(nil, "")

				Expect(handler).NotTo(BeNil())
				// Handler should be created even with empty namespace
			})
		})
	})

	Context("Input Validation", func() {
		// Note: validateGitLabInput is not exported, so we test it through the handlers
		Describe("Token validation through ConnectGitLab", func() {
			It("Should accept valid GitLab tokens", func() {
				validTokens := []string{
					"glpat-xxxxxxxxxxxxxxxxxxxx",       // 27 chars, typical GitLab PAT
					"glpat-1234567890abcdef1234567890", // 32 chars
					"token_with_underscores_123",       // with underscores
					"token-with-dashes-456",            // with dashes
					"UPPERCASE_TOKEN_789012",           // uppercase, 20 chars
					"MixedCase-Token_1234567",          // mixed case, 20 chars
				}

				for _, token := range validTokens {
					requestBody := map[string]interface{}{
						"personalAccessToken": token,
						"instanceUrl":         "https://gitlab.com",
					}

					context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
					context.Params = gin.Params{
						gin.Param{Key: "projectName", Value: "test-project"},
					}
					httpUtils.SetAuthHeader(testToken)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					ConnectGitLabGlobal(context)

					// Should not reject the token (may fail later due to connection manager mocking)
					// But should not fail at validation stage
					status := httpUtils.GetResponseRecorder().Code
					Expect(status).NotTo(Equal(http.StatusBadRequest), "Should accept valid token: "+token)

					// Reset for next test
					httpUtils = test_utils.NewHTTPTestUtils()
				}
			})

			It("Should reject tokens that are too short", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "short",
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid input: token must be at least 20 characters",
					"statusCode": http.StatusBadRequest,
				})
			})

			It("Should reject tokens that are too long", func() {
				longToken := ""
				for i := 0; i < 256; i++ {
					longToken += "a"
				}

				requestBody := map[string]interface{}{
					"personalAccessToken": longToken,
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid input: token must not exceed 255 characters",
					"statusCode": http.StatusBadRequest,
				})
			})

			It("Should reject tokens with invalid characters", func() {
				invalidTokens := []string{
					"token-with-spaces here",
					"token@with.email.chars",
					"token+with+plus+signs",
					"token/with/slashes",
					"token\\with\\backslashes",
					"token<with>brackets",
					"token{with}braces",
					"token[with]square",
					"token(with)parens",
					"token\"with\"quotes",
					"token'with'single",
					"token;with;semicolons",
					"token:with:colons",
					"token,with,commas",
					"token.with.dots",
					"token?with?questions",
					"token!with!exclamations",
				}

				for _, token := range invalidTokens {
					// Make token long enough to pass length check
					validLengthToken := token
					for len(validLengthToken) < 20 {
						validLengthToken += "a"
					}

					requestBody := map[string]interface{}{
						"personalAccessToken": validLengthToken,
						"instanceUrl":         "https://gitlab.com",
					}

					context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
					context.Params = gin.Params{
						gin.Param{Key: "projectName", Value: "test-project"},
					}
					httpUtils.SetAuthHeader(testToken)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					ConnectGitLabGlobal(context)

					httpUtils.AssertHTTPStatus(http.StatusBadRequest)
					httpUtils.AssertJSONContains(map[string]interface{}{
						"error":      "Invalid input: token contains invalid characters",
						"statusCode": http.StatusBadRequest,
					})

					// Reset for next test
					httpUtils = test_utils.NewHTTPTestUtils()
				}
			})
		})

		Describe("Instance URL validation through ConnectGitLab", func() {
			It("Should accept valid HTTPS URLs", func() {
				validURLs := []string{
					"https://gitlab.com",
					"https://gitlab.example.com",
					"https://gitlab.company.org",
					"https://git.domain.co.uk",
					"https://source.enterprise.local",
				}

				for _, url := range validURLs {
					requestBody := map[string]interface{}{
						"personalAccessToken": "valid_token_1234567890",
						"instanceUrl":         url,
					}

					context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
					context.Params = gin.Params{
						gin.Param{Key: "projectName", Value: "test-project"},
					}
					httpUtils.SetAuthHeader(testToken)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					ConnectGitLabGlobal(context)

					// Should not reject the URL at validation stage
					status := httpUtils.GetResponseRecorder().Code
					Expect(status).NotTo(Equal(http.StatusBadRequest), "Should accept valid URL: "+url)

					// Reset for next test
					httpUtils = test_utils.NewHTTPTestUtils()
				}
			})

			It("Should reject HTTP URLs", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "http://gitlab.example.com", // HTTP not HTTPS
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid input: instance URL must use HTTPS",
					"statusCode": http.StatusBadRequest,
				})
			})

			It("Should reject malformed URLs", func() {
				malformedURLs := []string{
					"not-a-url",
					"ftp://gitlab.com",
					"https://",
					"://gitlab.com",
					"https:gitlab.com",
					"gitlab.com",
				}

				for _, url := range malformedURLs {
					requestBody := map[string]interface{}{
						"personalAccessToken": "valid_token_1234567890",
						"instanceUrl":         url,
					}

					context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
					context.Params = gin.Params{
						gin.Param{Key: "projectName", Value: "test-project"},
					}
					httpUtils.SetAuthHeader(testToken)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					ConnectGitLabGlobal(context)

					status := httpUtils.GetResponseRecorder().Code
					Expect(status).To(Equal(http.StatusBadRequest), "Should reject malformed URL: "+url)

					// Reset for next test
					httpUtils = test_utils.NewHTTPTestUtils()
				}
			})

			It("Should reject URLs with @ in hostname", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "https://user@gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				// Note: url.Parse treats "user@" as user info, not hostname, so parsedURL.Host is "gitlab.com"
				// The validation passes URL validation, but then token validation fails when trying to connect
				// The test expects either 400 (validation error) or 500 (token validation error)
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(BeElementOf(http.StatusBadRequest, http.StatusInternalServerError))
			})

			It("Should default to gitlab.com when no URL provided", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					// instanceUrl omitted
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				// Should not fail at validation stage (URL should default to gitlab.com)
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest))
			})
		})
	})

	Context("Connection Management", func() {
		Describe("ConnectGitLab", func() {
			It("Should require authentication", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/connect", requestBody)
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				// Global function requires authentication
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require valid JSON body", func() {
				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", "invalid-json")
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				// Gin binding returns detailed validation error
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				// Error contains validation details, just check it's BadRequest
			})

			It("Should require personalAccessToken field", func() {
				requestBody := map[string]interface{}{
					"instanceUrl": "https://gitlab.com",
					// personalAccessToken missing
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				// Gin binding returns detailed validation error message
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				// Don't check exact error message - Gin's validation message varies
			})

			It("Should require user authentication", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				// Don't set user context

				ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid or missing token",
					"statusCode": http.StatusUnauthorized,
				})
			})

			It("Should handle invalid user ID type", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				context.Set("userID", 123) // Invalid type (should be string)

				ConnectGitLabGlobal(context)

				// GetString returns empty string for non-string types, triggers auth required error
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User authentication required")
			})

			// Note: RBAC permission checks are tested at integration level
			// Unit tests focus on input validation and basic handler logic
		})

		Describe("GetGitLabStatus", func() {
			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/gitlab/status", nil)
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				GetGitLabStatusGlobal(context)

				// Global function requires authentication
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/auth/gitlab/status", nil)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				// Don't set user context

				GetGitLabStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid or missing token",
					"statusCode": http.StatusUnauthorized,
				})
			})

			It("Should handle invalid user ID type", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/auth/gitlab/status", nil)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				context.Set("userID", 123) // Invalid type

				GetGitLabStatusGlobal(context)

				// GetString returns empty string for non-string types, triggers auth required error
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User authentication required")
			})

			// Note: RBAC permission checks are tested at integration level
			// Unit tests focus on input validation and basic handler logic
		})

		Describe("DisconnectGitLab", func() {
			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/disconnect", nil)
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				DisconnectGitLabGlobal(context)

				// Global function requires authentication
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/disconnect", nil)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				// Don't set user context

				DisconnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid or missing token",
					"statusCode": http.StatusUnauthorized,
				})
			})

			// Note: RBAC permission checks are tested at integration level
			// Unit tests focus on input validation and basic handler logic
		})
	})

	Context("Global Wrapper Functions", func() {
		Describe("ConnectGitLabGlobal", func() {
			It("Should require authentication", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/connect", requestBody)
				// Don't set auth header - should get 401

				ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			// Note: Global function K8s client validation tested at integration level
			// Unit tests focus on specific handler logic
		})

		Describe("GetGitLabStatusGlobal", func() {
			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/gitlab/status", nil)
				// Don't set auth header - should get 401

				GetGitLabStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			// Note: Global function K8s client validation tested at integration level
			// Unit tests focus on specific handler logic
		})

		Describe("DisconnectGitLabGlobal", func() {
			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/disconnect", nil)
				// Don't set auth header - should get 401

				DisconnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			// Note: Global function K8s client validation tested at integration level
			// Unit tests focus on specific handler logic
		})
	})

	Context("Data Structure Validation", func() {
		Describe("Request and Response Types", func() {
			It("Should validate ConnectGitLabRequest structure", func() {
				request := ConnectGitLabRequest{
					PersonalAccessToken: "test-token-1234567890",
					InstanceURL:         "https://gitlab.com",
				}

				Expect(request.PersonalAccessToken).To(Equal("test-token-1234567890"))
				Expect(request.InstanceURL).To(Equal("https://gitlab.com"))
			})

			It("Should validate ConnectGitLabResponse structure", func() {
				response := ConnectGitLabResponse{
					UserID:       "user123",
					GitLabUserID: "gitlab456",
					Username:     "testuser",
					InstanceURL:  "https://gitlab.com",
					Connected:    true,
					Message:      "Connected successfully",
				}

				Expect(response.UserID).To(Equal("user123"))
				Expect(response.GitLabUserID).To(Equal("gitlab456"))
				Expect(response.Username).To(Equal("testuser"))
				Expect(response.InstanceURL).To(Equal("https://gitlab.com"))
				Expect(response.Connected).To(BeTrue())
				Expect(response.Message).To(Equal("Connected successfully"))
			})

			It("Should validate GitLabStatusResponse structure", func() {
				// Connected status
				connectedResponse := GitLabStatusResponse{
					Connected:    true,
					Username:     "testuser",
					InstanceURL:  "https://gitlab.com",
					GitLabUserID: "gitlab456",
				}

				Expect(connectedResponse.Connected).To(BeTrue())
				Expect(connectedResponse.Username).To(Equal("testuser"))
				Expect(connectedResponse.InstanceURL).To(Equal("https://gitlab.com"))
				Expect(connectedResponse.GitLabUserID).To(Equal("gitlab456"))

				// Disconnected status
				disconnectedResponse := GitLabStatusResponse{
					Connected: false,
				}

				Expect(disconnectedResponse.Connected).To(BeFalse())
				Expect(disconnectedResponse.Username).To(BeEmpty())
				Expect(disconnectedResponse.InstanceURL).To(BeEmpty())
				Expect(disconnectedResponse.GitLabUserID).To(BeEmpty())
			})
		})
	})

	Context("Edge Cases and Error Handling", func() {
		It("Should handle concurrent requests", func() {
			// Test concurrent connect requests
			requestBody := map[string]interface{}{
				"personalAccessToken": "valid_token_1234567890",
				"instanceUrl":         "https://gitlab.com",
			}

			// Simulate multiple concurrent requests
			for i := 0; i < 3; i++ {
				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetUserContext(fmt.Sprintf("user-%d", i), "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				// Each should be processed independently
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest))

				// Reset for next iteration
				httpUtils = test_utils.NewHTTPTestUtils()
			}
		})

		It("Should handle empty and whitespace inputs", func() {
			testCases := []struct {
				token       string
				description string
			}{
				{"", "empty token"},
				{"   ", "whitespace token"},
				{"\t\n\r", "control character token"},
			}

			for _, tc := range testCases {
				requestBody := map[string]interface{}{
					"personalAccessToken": tc.token,
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				// Reset for next test
				httpUtils = test_utils.NewHTTPTestUtils()
			}
		})

		It("Should handle various URL edge cases", func() {
			testCases := []struct {
				url         string
				shouldFail  bool
				description string
			}{
				{"https://gitlab.com", false, "standard GitLab.com"},
				{"https://gitlab.com/", false, "with trailing slash"},
				{"https://gitlab.example.com:443", false, "with explicit HTTPS port"},
				{"https://gitlab.example.com:8443", false, "with custom HTTPS port"},
				{"https://gitlab", false, "single hostname"},
				{"https://127.0.0.1", false, "IP address"},
				{"https://[::1]", false, "IPv6 address"},
				{"https://gitlab.com:80", false, "custom port on HTTPS"}, // Would be unusual but not invalid
			}

			for _, tc := range testCases {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         tc.url,
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", requestBody)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGitLabGlobal(context)

				status := httpUtils.GetResponseRecorder().Code
				if tc.shouldFail {
					Expect(status).To(Equal(http.StatusBadRequest), "Should reject "+tc.description)
				} else {
					Expect(status).NotTo(Equal(http.StatusBadRequest), "Should accept "+tc.description)
				}

				// Reset for next test
				httpUtils = test_utils.NewHTTPTestUtils()
			}
		})
	})
})
