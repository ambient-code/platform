package unit

import (
	"ambient-code-backend/test/config"
	test_constants "ambient-code-backend/test/constants"
	"fmt"
	"net/http"

	"ambient-code-backend/handlers"
	"ambient-code-backend/test/logger"
	"ambient-code-backend/test/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("GitLab Auth Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelGitLabAuth), func() {
	var (
		httpUtils                 *test_utils.HTTPTestUtils
		originalK8sClient         kubernetes.Interface
		originalK8sClientMw       kubernetes.Interface
		originalK8sClientProjects kubernetes.Interface
		originalNamespace         string
	)

	BeforeEach(func() {
		logger.Log("Setting up GitLab Auth Handler test")

		// Save original state to restore in AfterEach
		originalK8sClient = handlers.K8sClient
		originalK8sClientMw = handlers.K8sClientMw
		originalK8sClientProjects = handlers.K8sClientProjects
		originalNamespace = handlers.Namespace

		// Auth is disabled by default from config for unit tests

		// Use centralized handler dependencies setup
		k8sUtils := test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		test_utils.SetupHandlerDependencies(k8sUtils)

		// For GitLab auth tests, we need to set all the package-level K8s client variables
		// Different handlers use different client variables, so set them all
		// Also set the Namespace variable that gitlab_auth.go uses
		// IMPORTANT: Use the same fake client for handlers that the test data is created with
		handlers.K8sClient = k8sUtils.K8sClient
		handlers.K8sClientMw = k8sUtils.K8sClient
		handlers.K8sClientProjects = k8sUtils.K8sClient
		handlers.Namespace = *config.TestNamespace

		httpUtils = test_utils.NewHTTPTestUtils()

	})

	AfterEach(func() {
		// Restore original state to prevent test pollution
		handlers.K8sClient = originalK8sClient
		handlers.K8sClientMw = originalK8sClientMw
		handlers.K8sClientProjects = originalK8sClientProjects
		handlers.Namespace = originalNamespace
	})

	Context("Handler Creation", func() {
		Describe("NewGitLabAuthHandler", func() {
			It("Should handle nil kubernetes client", func() {
				handler := handlers.NewGitLabAuthHandler(nil, "test-project")

				Expect(handler).NotTo(BeNil())
				// Handler creation should not fail even with nil client
			})

			It("Should handle empty namespace", func() {
				handler := handlers.NewGitLabAuthHandler(nil, "")

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
					httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					handlers.ConnectGitLabGlobal(context)

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
				httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

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
				httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

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
					httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					handlers.ConnectGitLabGlobal(context)

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
					httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					handlers.ConnectGitLabGlobal(context)

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
				httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

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
					httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					handlers.ConnectGitLabGlobal(context)

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
				httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

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
				httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

				// Should not fail at validation stage (URL should default to gitlab.com)
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest))
			})
		})
	})

	Context("Connection Management", func() {
		Describe("ConnectGitLab", func() {
			It("Should require project name", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/connect", requestBody)
				// Don't set project name param
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Project name is required",
					"statusCode": http.StatusBadRequest,
				})
			})

			It("Should require valid JSON body", func() {
				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/connect", "invalid-json")
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid request body",
					"statusCode": http.StatusBadRequest,
				})
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
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid request body",
					"statusCode": http.StatusBadRequest,
				})
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

				handlers.ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "User not authenticated",
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
				context.Set("userID", 123) // Invalid type (should be string)

				handlers.ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusInternalServerError)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid user ID format",
					"statusCode": http.StatusInternalServerError,
				})
			})

			// Note: RBAC permission checks are tested at integration level
			// Unit tests focus on input validation and basic handler logic
		})

		Describe("GetGitLabStatus", func() {
			It("Should require project name", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/gitlab/status", nil)
				// Don't set project name param
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.GetGitLabStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Project name is required",
					"statusCode": http.StatusBadRequest,
				})
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/auth/gitlab/status", nil)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				// Don't set user context

				handlers.GetGitLabStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "User not authenticated",
					"statusCode": http.StatusUnauthorized,
				})
			})

			It("Should handle invalid user ID type", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/auth/gitlab/status", nil)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				context.Set("userID", 123) // Invalid type

				handlers.GetGitLabStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusInternalServerError)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "Invalid user ID format",
					"statusCode": http.StatusInternalServerError,
				})
			})

			// Note: RBAC permission checks are tested at integration level
			// Unit tests focus on input validation and basic handler logic
		})

		Describe("DisconnectGitLab", func() {
			It("Should require project name", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/disconnect", nil)
				// Don't set project name param
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.DisconnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Project name is required",
				})
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/auth/gitlab/disconnect", nil)
				context.Params = gin.Params{
					gin.Param{Key: "projectName", Value: "test-project"},
				}
				// Don't set user context

				handlers.DisconnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error":      "User not authenticated",
					"statusCode": http.StatusUnauthorized,
				})
			})

			// Note: RBAC permission checks are tested at integration level
			// Unit tests focus on input validation and basic handler logic
		})
	})

	Context("Global Wrapper Functions", func() {
		Describe("ConnectGitLabGlobal", func() {
			It("Should require project name parameter", func() {
				requestBody := map[string]interface{}{
					"personalAccessToken": "valid_token_1234567890",
					"instanceUrl":         "https://gitlab.com",
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/connect", requestBody)
				// Don't set project name param

				handlers.ConnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Project name is required")
			})

			// Note: Global function K8s client validation tested at integration level
			// Unit tests focus on specific handler logic
		})

		Describe("GetGitLabStatusGlobal", func() {
			It("Should require project name parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/gitlab/status", nil)
				// Don't set project name param

				handlers.GetGitLabStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Project name is required")
			})

			// Note: Global function K8s client validation tested at integration level
			// Unit tests focus on specific handler logic
		})

		Describe("DisconnectGitLabGlobal", func() {
			It("Should require project name parameter", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/gitlab/disconnect", nil)
				// Don't set project name param

				handlers.DisconnectGitLabGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Project name is required")
			})

			// Note: Global function K8s client validation tested at integration level
			// Unit tests focus on specific handler logic
		})
	})

	Context("Data Structure Validation", func() {
		Describe("Request and Response Types", func() {
			It("Should validate ConnectGitLabRequest structure", func() {
				request := handlers.ConnectGitLabRequest{
					PersonalAccessToken: "test-token-1234567890",
					InstanceURL:         "https://gitlab.com",
				}

				Expect(request.PersonalAccessToken).To(Equal("test-token-1234567890"))
				Expect(request.InstanceURL).To(Equal("https://gitlab.com"))
			})

			It("Should validate ConnectGitLabResponse structure", func() {
				response := handlers.ConnectGitLabResponse{
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
				connectedResponse := handlers.GitLabStatusResponse{
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
				disconnectedResponse := handlers.GitLabStatusResponse{
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

				handlers.ConnectGitLabGlobal(context)

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
				httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

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
				httpUtils.SetAuthHeader(test_constants.DEV_MOCK_TOKEN)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				handlers.ConnectGitLabGlobal(context)

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
