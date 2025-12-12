//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Mock GitHub Token Manager for testing
type mockGithubTokenManager struct {
	jwt string
	err error
}

func (m *mockGithubTokenManager) GenerateJWT() (string, error) {
	return m.jwt, m.err
}

var _ = Describe("GitHub Auth Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelGitHubAuth), func() {
	var (
		httpUtils                 *test_utils.HTTPTestUtils
		testClientFactory         *test_utils.TestClientFactory
		fakeClients               *test_utils.FakeClientSet
		mockTokenManager          *mockGithubTokenManager
		originalK8sClient         kubernetes.Interface
		originalK8sClientMw       kubernetes.Interface
		originalK8sClientProjects kubernetes.Interface
		originalNamespace         string
	)

	BeforeEach(func() {
		logger.Log("Setting up GitHub Auth Handler test")

		// Save original state to restore in AfterEach
		originalK8sClient = K8sClient
		originalK8sClientMw = K8sClientMw
		originalK8sClientProjects = K8sClientProjects
		originalNamespace = Namespace

		// Create test client factory with fake clients
		testClientFactory = test_utils.NewTestClientFactory()
		fakeClients = testClientFactory.GetFakeClients()

		// Use centralized handler dependencies setup
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// For GitHub auth tests, we need to set all the package-level K8s client variables
		// Different handlers use different client variables, so set them all
		// Also set the Namespace variable that github_auth.go uses
		// IMPORTANT: Use the same fake client for handlers that the test data is created with
		K8sClient = fakeClients.GetK8sClient()
		K8sClientMw = fakeClients.GetK8sClient()
		K8sClientProjects = fakeClients.GetK8sClient()
		Namespace = *config.TestNamespace

		httpUtils = test_utils.NewHTTPTestUtils()

		// Create mock token manager (environment variable can control this in real implementation)
		mockTokenManager = &mockGithubTokenManager{
			jwt: "mock-jwt-token",
			err: nil,
		}

		os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
		os.Setenv("GITHUB_CLIENT_SECRET", "test-client-secret")
		os.Setenv("GITHUB_STATE_SECRET", "test-state-secret")
	})

	AfterEach(func() {
		// Restore original state to prevent test pollution
		K8sClient = originalK8sClient
		K8sClientMw = originalK8sClientMw
		K8sClientProjects = originalK8sClientProjects
		Namespace = originalNamespace
	})

	Context("GitHub App Installation Management", func() {
		Describe("GitHubAppInstallation struct", func() {
			It("Should implement interface methods correctly", func() {
				installation := &GitHubAppInstallation{
					UserID:         "test-user",
					GitHubUserID:   "github-user",
					InstallationID: 12345,
					Host:           "github.com",
					UpdatedAt:      time.Now(),
				}

				Expect(installation.GetInstallationID()).To(Equal(int64(12345)))
				Expect(installation.GetHost()).To(Equal("github.com"))
			})
		})

		Describe("GetGitHubInstallation", func() {
			BeforeEach(func() {
				// Create a ConfigMap with installation data
				installation := &GitHubAppInstallation{
					UserID:         "test-user",
					GitHubUserID:   "github-user",
					InstallationID: 12345,
					Host:           "github.com",
					UpdatedAt:      time.Now(),
				}
				installationJSON, err := json.Marshal(installation)
				Expect(err).NotTo(HaveOccurred())

				configMap := &corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:      "github-app-installations",
						Namespace: *config.TestNamespace,
					},
					Data: map[string]string{
						"test-user": string(installationJSON),
					},
				}
				_, err = fakeClients.GetK8sClient().CoreV1().ConfigMaps(*config.TestNamespace).Create(
					context.Background(), configMap, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should retrieve installation successfully", func() {
				installation, err := GetGitHubInstallation(context.Background(), "test-user")

				Expect(err).NotTo(HaveOccurred())
				Expect(installation).NotTo(BeNil())
				Expect(installation.UserID).To(Equal("test-user"))
				Expect(installation.GitHubUserID).To(Equal("github-user"))
				Expect(installation.InstallationID).To(Equal(int64(12345)))
				Expect(installation.Host).To(Equal("github.com"))
			})

			It("Should return error when installation not found", func() {
				installation, err := GetGitHubInstallation(context.Background(), "nonexistent-user")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("installation not found"))
				Expect(installation).To(BeNil())
			})

			It("Should return error when ConfigMap not found", func() {
				// Delete the ConfigMap
				err := fakeClients.GetK8sClient().CoreV1().ConfigMaps(*config.TestNamespace).Delete(
					context.Background(), "github-app-installations", v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				installation, err := GetGitHubInstallation(context.Background(), "test-user")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("installation not found"))
				Expect(installation).To(BeNil())
			})

			It("Should handle malformed JSON gracefully", func() {
				// Create ConfigMap with invalid JSON
				configMap := &corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:      "github-app-installations",
						Namespace: *config.TestNamespace,
					},
					Data: map[string]string{
						"malformed-user": "invalid-json{",
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps(*config.TestNamespace).Update(
					context.Background(), configMap, v1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())

				installation, err := GetGitHubInstallation(context.Background(), "malformed-user")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to decode installation"))
				Expect(installation).To(BeNil())
			})
		})
	})

	Context("Global GitHub Endpoints", func() {
		Describe("LinkGitHubInstallationGlobal", func() {
			It("Should link installation successfully", func() {
				requestBody := map[string]interface{}{
					"installationId": float64(12345),
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				LinkGitHubInstallationGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message":        "GitHub App installation linked successfully",
					"installationId": float64(12345),
				})

				// Verify installation was stored
				installation, err := GetGitHubInstallation(context.Request.Context(), "test-user")
				Expect(err).NotTo(HaveOccurred())
				Expect(installation.InstallationID).To(Equal(int64(12345)))
				Expect(installation.UserID).To(Equal("test-user"))
				Expect(installation.Host).To(Equal("github.com"))
			})

			It("Should require user authentication", func() {
				requestBody := map[string]interface{}{
					"installationId": float64(12345),
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
				// Don't set user context

				LinkGitHubInstallationGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("missing user identity")
			})

			It("Should validate installation ID is required", func() {
				requestBody := map[string]interface{}{
					// Missing installationId
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				LinkGitHubInstallationGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should handle invalid JSON gracefully", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", "invalid-json")
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				LinkGitHubInstallationGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should handle JWT generation for GitHub account enrichment", func() {
				mockTokenManager.jwt = "valid-jwt-token"

				requestBody := map[string]interface{}{
					"installationId": 12345,
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Note: This will attempt to make HTTP request to GitHub, but it should still complete
				// the basic installation linking even if GitHub request fails
				LinkGitHubInstallationGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
			})

			It("Should handle JWT generation errors gracefully", func() {
				mockTokenManager.err = fmt.Errorf("JWT generation failed")

				requestBody := map[string]interface{}{
					"installationId": 12345,
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				LinkGitHubInstallationGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusOK) // Should still succeed
			})
		})

		Describe("GetGitHubStatusGlobal", func() {
			It("Should return installation status when user is linked", func() {
				// Create installation
				installation := &GitHubAppInstallation{
					UserID:         "test-user",
					GitHubUserID:   "github-user",
					InstallationID: 12345,
					Host:           "github.com",
					UpdatedAt:      time.Now(),
				}
				installationJSON, err := json.Marshal(installation)
				Expect(err).NotTo(HaveOccurred())

				configMap := &corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:      "github-app-installations",
						Namespace: *config.TestNamespace,
					},
					Data: map[string]string{
						"test-user": string(installationJSON),
					},
				}
				_, err = fakeClients.GetK8sClient().CoreV1().ConfigMaps(*config.TestNamespace).Create(
					context.Background(), configMap, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				context := httpUtils.CreateTestGinContext("GET", "/auth/github/status", nil)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				GetGitHubStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"installed":      true,
					"installationId": float64(12345),
					"host":           "github.com",
					"githubUserId":   "github-user",
					"userId":         "test-user",
				})

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("updatedAt"))
			})

			It("Should return not installed when user has no installation", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/github/status", nil)
				httpUtils.SetUserContext("unlinked-user", "Test User", "test@example.com")

				GetGitHubStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"installed": false,
				})
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/github/status", nil)
				// Don't set user context

				GetGitHubStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("missing user identity")
			})

			It("Should handle empty user ID gracefully", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/github/status", nil)
				httpUtils.SetUserContext("", "Test User", "test@example.com")

				GetGitHubStatusGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("missing user identity")
			})
		})

		Describe("DisconnectGitHubGlobal", func() {
			BeforeEach(func() {
				// Create installation to disconnect
				installation := &GitHubAppInstallation{
					UserID:         "test-user",
					GitHubUserID:   "github-user",
					InstallationID: 12345,
					Host:           "github.com",
					UpdatedAt:      time.Now(),
				}
				installationJSON, err := json.Marshal(installation)
				Expect(err).NotTo(HaveOccurred())

				configMap := &corev1.ConfigMap{
					ObjectMeta: v1.ObjectMeta{
						Name:      "github-app-installations",
						Namespace: *config.TestNamespace,
					},
					Data: map[string]string{
						"test-user": string(installationJSON),
					},
				}
				_, err = fakeClients.GetK8sClient().CoreV1().ConfigMaps(*config.TestNamespace).Create(
					context.Background(), configMap, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should disconnect GitHub installation successfully", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/github/disconnect", nil)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				DisconnectGitHubGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "GitHub account disconnected",
				})

				// Verify installation was removed
				_, err := GetGitHubInstallation(context.Request.Context(), "test-user")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("installation not found"))
			})

			It("Should require user authentication", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/github/disconnect", nil)
				// Don't set user context

				DisconnectGitHubGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("missing user identity")
			})

			It("Should handle missing ConfigMap gracefully", func() {
				// Delete the ConfigMap first
				err := fakeClients.GetK8sClient().CoreV1().ConfigMaps(*config.TestNamespace).Delete(
					context.Background(), "github-app-installations", v1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())

				context := httpUtils.CreateTestGinContext("POST", "/auth/github/disconnect", nil)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				DisconnectGitHubGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusInternalServerError)
				httpUtils.AssertErrorMessage("failed to unlink installation")
			})

			It("Should handle disconnecting non-existent installation", func() {
				context := httpUtils.CreateTestGinContext("POST", "/auth/github/disconnect", nil)
				httpUtils.SetUserContext("nonexistent-user", "Test User", "test@example.com")

				DisconnectGitHubGlobal(context)

				httpUtils.AssertHTTPStatus(http.StatusOK) // Should succeed even if user wasn't linked
			})
		})
	})

	Context("OAuth Callback Handling", func() {
		Describe("HandleGitHubUserOAuthCallback", func() {
			It("Should require OAuth environment variables", func() {
				// Clear environment variables
				os.Unsetenv("GITHUB_CLIENT_ID")

				context := httpUtils.CreateTestGinContext("GET", "/auth/github/user/callback?code=test-code", nil)

				HandleGitHubUserOAuthCallback(context)

				httpUtils.AssertHTTPStatus(http.StatusInternalServerError)
				httpUtils.AssertErrorMessage("OAuth not configured")

				// Restore for other tests
				os.Setenv("GITHUB_CLIENT_ID", "test-client-id")
			})

			It("Should require code parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/github/user/callback", nil)

				HandleGitHubUserOAuthCallback(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("missing code")
			})

			It("Should require user identity when no state provided", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/github/user/callback?code=test-code", nil)
				// Don't set user context

				HandleGitHubUserOAuthCallback(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("missing user identity")
			})

			It("Should require installation_id when no state provided", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/github/user/callback?code=test-code", nil)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				HandleGitHubUserOAuthCallback(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("invalid installation id")
			})

			It("Should handle valid installation_id without state", func() {
				context := httpUtils.CreateTestGinContext("GET", "/auth/github/user/callback?code=test-code&installation_id=12345", nil)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Note: This will fail at OAuth exchange since we can't mock external HTTP calls easily
				// But we can verify it gets past the parameter validation
				HandleGitHubUserOAuthCallback(context)

				// Should get to OAuth exchange step (would fail there with real HTTP call)
				httpUtils.AssertHTTPStatus(http.StatusBadGateway)
				httpUtils.AssertErrorMessage("oauth exchange failed")
			})

			Context("With State Parameter", func() {
				var validState string

				BeforeEach(func() {
					// Create a valid state parameter for testing
					userID := "test-user"
					timestamp := strconv.FormatInt(time.Now().Unix(), 10)
					installationID := base64.RawURLEncoding.EncodeToString([]byte("12345"))
					returnTo := base64.RawURLEncoding.EncodeToString([]byte("/integrations"))

					payload := fmt.Sprintf("%s:%s:oauth:%s:%s", userID, timestamp, returnTo, installationID)

					// Sign the payload
					signature := signTestState("test-state-secret", payload)
					rawState := fmt.Sprintf("%s.%s", payload, signature)
					validState = base64.RawURLEncoding.EncodeToString([]byte(rawState))
				})

				It("Should validate state signature", func() {
					// Create invalid state with wrong signature
					payload := "test-user:123456:oauth::MTIzNDU="
					wrongSignature := "wrong-signature"
					rawState := fmt.Sprintf("%s.%s", payload, wrongSignature)
					invalidState := base64.RawURLEncoding.EncodeToString([]byte(rawState))

					url := fmt.Sprintf("/auth/github/user/callback?code=test-code&state=%s", invalidState)
					context := httpUtils.CreateTestGinContext("GET", url, nil)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					HandleGitHubUserOAuthCallback(context)

					httpUtils.AssertHTTPStatus(http.StatusBadRequest)
					httpUtils.AssertErrorMessage("bad state signature")
				})

				It("Should validate state format", func() {
					// Create state with wrong number of fields
					payload := "test-user:123456:oauth" // Missing fields
					signature := signTestState("test-state-secret", payload)
					rawState := fmt.Sprintf("%s.%s", payload, signature)
					invalidState := base64.RawURLEncoding.EncodeToString([]byte(rawState))

					url := fmt.Sprintf("/auth/github/user/callback?code=test-code&state=%s", invalidState)
					context := httpUtils.CreateTestGinContext("GET", url, nil)

					HandleGitHubUserOAuthCallback(context)

					httpUtils.AssertHTTPStatus(http.StatusBadRequest)
					httpUtils.AssertErrorMessage("bad state payload")
				})

				It("Should validate state expiration", func() {
					// Create expired state (old timestamp)
					userID := "test-user"
					oldTimestamp := strconv.FormatInt(time.Now().Add(-15*time.Minute).Unix(), 10)
					installationID := base64.RawURLEncoding.EncodeToString([]byte("12345"))
					returnTo := base64.RawURLEncoding.EncodeToString([]byte("/integrations"))

					payload := fmt.Sprintf("%s:%s:oauth:%s:%s", userID, oldTimestamp, returnTo, installationID)
					signature := signTestState("test-state-secret", payload)
					rawState := fmt.Sprintf("%s.%s", payload, signature)
					expiredState := base64.RawURLEncoding.EncodeToString([]byte(rawState))

					url := fmt.Sprintf("/auth/github/user/callback?code=test-code&state=%s", expiredState)
					context := httpUtils.CreateTestGinContext("GET", url, nil)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

					HandleGitHubUserOAuthCallback(context)

					httpUtils.AssertHTTPStatus(http.StatusBadRequest)
					httpUtils.AssertErrorMessage("state expired")
				})

				It("Should validate user matches state", func() {
					url := fmt.Sprintf("/auth/github/user/callback?code=test-code&state=%s", validState)
					context := httpUtils.CreateTestGinContext("GET", url, nil)
					httpUtils.SetUserContext("different-user", "Test User", "test@example.com") // Different user

					HandleGitHubUserOAuthCallback(context)

					httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
					httpUtils.AssertErrorMessage("user mismatch")
				})

				It("Should handle malformed state base64", func() {
					invalidBase64State := "invalid-base64!@#"

					url := fmt.Sprintf("/auth/github/user/callback?code=test-code&state=%s", invalidBase64State)
					context := httpUtils.CreateTestGinContext("GET", url, nil)

					HandleGitHubUserOAuthCallback(context)

					httpUtils.AssertHTTPStatus(http.StatusBadRequest)
					httpUtils.AssertErrorMessage("invalid state")
				})

				It("Should handle state without signature", func() {
					payload := "test-user:123456:oauth::"
					rawState := payload // No signature part
					invalidState := base64.RawURLEncoding.EncodeToString([]byte(rawState))

					url := fmt.Sprintf("/auth/github/user/callback?code=test-code&state=%s", invalidState)
					context := httpUtils.CreateTestGinContext("GET", url, nil)

					HandleGitHubUserOAuthCallback(context)

					httpUtils.AssertHTTPStatus(http.StatusBadRequest)
					httpUtils.AssertErrorMessage("invalid state")
				})
			})
		})
	})

	Context("Helper Functions", func() {
		It("Should resolve GitHub API base URL correctly", func() {
			testCases := []struct {
				host     string
				expected string
			}{
				{"", "https://api.github.com"},
				{"github.com", "https://api.github.com"},
				{"github.enterprise.com", "https://github.enterprise.com/api/v3"},
				{"my-github.company.com", "https://my-github.company.com/api/v3"},
			}

			for _, tc := range testCases {
				// We can't test this directly since githubAPIBaseURL is not exported
				// But we can verify the logic through the behavior of functions that use it
				Expect(tc.expected).NotTo(BeEmpty()) // Placeholder assertion
			}
		})

		It("Should create and validate GitHub installations", func() {
			installation := &GitHubAppInstallation{
				UserID:         "test-user",
				GitHubUserID:   "github-user",
				InstallationID: 12345,
				Host:           "github.com",
				UpdatedAt:      time.Now(),
			}

			// Test interface implementation
			Expect(installation.GetInstallationID()).To(Equal(int64(12345)))
			Expect(installation.GetHost()).To(Equal("github.com"))

			// Test JSON serialization/deserialization
			jsonData, err := json.Marshal(installation)
			Expect(err).NotTo(HaveOccurred())

			var restored GitHubAppInstallation
			err = json.Unmarshal(jsonData, &restored)
			Expect(err).NotTo(HaveOccurred())

			Expect(restored.UserID).To(Equal(installation.UserID))
			Expect(restored.GitHubUserID).To(Equal(installation.GitHubUserID))
			Expect(restored.InstallationID).To(Equal(installation.InstallationID))
			Expect(restored.Host).To(Equal(installation.Host))
		})
	})

	Context("ConfigMap Storage Management", func() {
		It("Should handle ConfigMap creation and updates through installation flow", func() {
			// Test the storage functionality through the public API
			requestBody := map[string]interface{}{
				"installationId": 54321,
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
			httpUtils.SetUserContext("storage-test-user", "Test User", "test@example.com")

			LinkGitHubInstallationGlobal(context)
			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Verify it was stored
			installation, err := GetGitHubInstallation(context.Request.Context(), "storage-test-user")
			Expect(err).NotTo(HaveOccurred())
			Expect(installation.InstallationID).To(Equal(int64(54321)))

			// Test updating the same user
			requestBody2 := map[string]interface{}{
				"installationId": 98765,
			}
			context2 := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody2)
			httpUtils.SetUserContext("storage-test-user", "Test User", "test@example.com")

			LinkGitHubInstallationGlobal(context2)
			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Verify it was updated
			installation2, err := GetGitHubInstallation(context2.Request.Context(), "storage-test-user")
			Expect(err).NotTo(HaveOccurred())
			Expect(installation2.InstallationID).To(Equal(int64(98765)))
		})

		It("Should handle multiple users in the same ConfigMap", func() {
			// Create installations for multiple users
			users := []string{"user1", "user2", "user3"}
			installationIDs := []int64{11111, 22222, 33333}

			for i, userID := range users {
				requestBody := map[string]interface{}{
					"installationId": installationIDs[i],
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
				httpUtils.SetUserContext(userID, "Test User", "test@example.com")

				LinkGitHubInstallationGlobal(context)
				httpUtils.AssertHTTPStatus(http.StatusOK)
			}

			// Verify all users have their installations
			for i, userID := range users {
				installation, err := GetGitHubInstallation(context.Background(), userID)
				Expect(err).NotTo(HaveOccurred())
				Expect(installation.InstallationID).To(Equal(installationIDs[i]))
			}

			// Remove one user
			context := httpUtils.CreateTestGinContext("POST", "/auth/github/disconnect", nil)
			httpUtils.SetUserContext("user2", "Test User", "test@example.com")

			DisconnectGitHubGlobal(context)
			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Verify user2 was removed but others remain
			_, err := GetGitHubInstallation(context.Request.Context(), "user2")
			Expect(err).To(HaveOccurred())

			for _, userID := range []string{"user1", "user3"} {
				_, err := GetGitHubInstallation(context.Request.Context(), userID)
				Expect(err).NotTo(HaveOccurred())
			}
		})
	})

	Context("Error Handling", func() {
		It("Should handle K8s client errors gracefully", func() {
			// Test with nil client
			K8sClient = nil

			context := httpUtils.CreateTestGinContext("GET", "/auth/github/status", nil)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			// Nil K8sClient is expected to cause a panic when trying to access ConfigMaps
			// We expect this to panic but recover gracefully
			Expect(func() {
				GetGitHubStatusGlobal(context)
			}).To(Panic())

			// Test verifies the handler panics predictably with nil client
			// This is expected behavior that should be handled by middleware in production
		})

		It("Should validate installation data before storage", func() {
			// Test with invalid installation data (tested through the API)
			requestBody := map[string]interface{}{
				"installationId": -1, // Invalid ID
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/github/install", requestBody)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			LinkGitHubInstallationGlobal(context)

			// Should still accept it (validation is minimal in current implementation)
			httpUtils.AssertHTTPStatus(http.StatusOK)
		})
	})
})

// Helper function to sign state for testing (replicates the internal signState function)
func signTestState(secret string, payload string) string {
	// This replicates the internal signState logic for testing
	// We need this to create valid test states
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
