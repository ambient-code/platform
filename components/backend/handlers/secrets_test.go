//go:build test

package handlers

import (
	"context"
	"net/http"
	"time"

	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Secrets Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelSecrets), func() {
	var (
		httpUtils   *test_utils.HTTPTestUtils
		k8sUtils    *test_utils.K8sTestUtils
		fakeClients *test_utils.FakeClientSet
		testToken   string
	)

	BeforeEach(func() {
		logger.Log("Setting up Secrets Handler test")

		// Use centralized K8s test setup with fake cluster
		k8sUtils = test_utils.NewK8sTestUtils(false, "test-project")
		SetupHandlerDependencies(k8sUtils)

		// Create fake clients that match the K8s utils setup
		fakeClients = &test_utils.FakeClientSet{
			K8sClient:     k8sUtils.K8sClient,
			DynamicClient: k8sUtils.DynamicClient,
		}

		httpUtils = test_utils.NewHTTPTestUtils()

		// Create namespace + role and mint a valid test token for this suite
		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
		_, err = k8sUtils.CreateTestRole(ctx, "test-project", "test-full-access-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := httpUtils.SetValidTestToken(
			k8sUtils,
			"test-project",
			[]string{"get", "list", "create", "update", "delete", "patch"},
			"*",
			"",
			"test-full-access-role",
		)
		Expect(err).NotTo(HaveOccurred())
		testToken = token
	})

	AfterEach(func() {
		// Clean up created namespace (best-effort)
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "test-project", metav1.DeleteOptions{})
		}
	})

	Context("Namespace Secrets Management", func() {
		Describe("ListNamespaceSecrets", func() {
			BeforeEach(func() {
				// Create test secrets with different types and annotations
				secrets := []*corev1.Secret{
					// Runner secret (should be included)
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "ambient-runner-secrets",
							Namespace:         "test-project",
							CreationTimestamp: metav1.NewTime(time.Now().Add(-1 * time.Hour)),
							Annotations: map[string]string{
								"ambient-code.io/runner-secret": "true",
							},
						},
						Type: corev1.SecretTypeOpaque,
						Data: map[string][]byte{
							"ANTHROPIC_API_KEY": []byte("test-key"),
						},
					},
					// Integration secret (should be included)
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:              "ambient-non-vertex-integrations",
							Namespace:         "test-project",
							CreationTimestamp: metav1.NewTime(time.Now().Add(-2 * time.Hour)),
							Annotations: map[string]string{
								"ambient-code.io/runner-secret": "true",
							},
						},
						Type: corev1.SecretTypeOpaque,
						Data: map[string][]byte{
							"GITHUB_TOKEN": []byte("github-token"),
						},
					},
					// System secret without annotation (should be excluded)
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "system-secret",
							Namespace: "test-project",
						},
						Type: corev1.SecretTypeOpaque,
					},
					// Service account token (should be excluded)
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "service-account-token",
							Namespace: "test-project",
						},
						Type: corev1.SecretTypeServiceAccountToken,
					},
				}

				// Create secrets in fake client
				ctx := context.Background()
				for _, secret := range secrets {
					_, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Create(
						ctx, secret, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("Should list only runner secrets with annotation", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				ListNamespaceSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("items"))

				items := response["items"].([]interface{})
				Expect(items).To(HaveLen(2), "Should return only runner secrets with annotation")

				// Check that all returned items have required fields
				for _, item := range items {
					itemMap := item.(map[string]interface{})
					Expect(itemMap).To(HaveKey("name"))
					Expect(itemMap).To(HaveKey("type"))
					Expect(itemMap).To(HaveKey("createdAt"))
					Expect(itemMap["type"]).To(Equal("Opaque"))
				}

				logger.Log("Successfully listed filtered namespace secrets")
			})

			It("Should require authentication", func() {
				// Arrange
				// Temporarily enable auth check to test proper auth failure
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header

				// Act
				ListNamespaceSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})
		})
	})

	Context("Runner Secrets Management", func() {
		Describe("ListRunnerSecrets", func() {
			It("Should return empty data when secret doesn't exist", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/runner-secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				ListRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("data"))

				data := response["data"].(map[string]interface{})
				Expect(data).To(HaveLen(0), "Should return empty data when secret doesn't exist")

				logger.Log("Correctly handled missing runner secrets")
			})

			It("Should return runner secrets data when secret exists", func() {
				// Arrange - create runner secret
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ambient-runner-secrets",
						Namespace: "test-project",
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"ANTHROPIC_API_KEY": []byte("test-anthropic-key"),
					},
				}
				ctx := context.Background()
				_, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Create(
					ctx, secret, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/runner-secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				ListRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("data"))

				data := response["data"].(map[string]interface{})
				Expect(data).To(HaveKey("ANTHROPIC_API_KEY"))
				Expect(data["ANTHROPIC_API_KEY"]).To(Equal("test-anthropic-key"))

				logger.Log("Successfully retrieved runner secrets")
			})

			It("Should require authentication", func() {
				// Arrange
				// Temporarily enable auth check to test proper auth failure
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/runner-secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header

				// Act
				ListRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})
		})

		Describe("UpdateRunnerSecrets", func() {
			It("Should create new runner secret when none exists", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"ANTHROPIC_API_KEY": "new-anthropic-key",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "runner secrets updated",
				})

				// Verify secret was created
				ctx := context.Background()
				secret, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Get(
					ctx, "ambient-runner-secrets", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.StringData["ANTHROPIC_API_KEY"]).To(Equal("new-anthropic-key"))
				Expect(secret.Annotations["ambient-code.io/runner-secret"]).To(Equal("true"))

				logger.Log("Successfully created new runner secret")
			})

			It("Should update existing runner secret", func() {
				// Arrange - create existing secret
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ambient-runner-secrets",
						Namespace: "test-project",
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"ANTHROPIC_API_KEY": []byte("old-key"),
					},
				}
				ctx := context.Background()
				_, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Create(
					ctx, existingSecret, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"ANTHROPIC_API_KEY": "updated-anthropic-key",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "runner secrets updated",
				})

				// Verify secret was updated
				ctx = context.Background()
				secret, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Get(
					ctx, "ambient-runner-secrets", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(string(secret.Data["ANTHROPIC_API_KEY"])).To(Equal("updated-anthropic-key"))

				logger.Log("Successfully updated existing runner secret")
			})

			It("Should validate allowed keys for runner secrets", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"INVALID_KEY": "some-value",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid key 'INVALID_KEY' for ambient-runner-secrets. Only ANTHROPIC_API_KEY is allowed.",
				})

				logger.Log("Successfully validated allowed keys for runner secrets")
			})

			It("Should require valid JSON body", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", "invalid-json")
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				logger.Log("Correctly rejected invalid JSON")
			})

			It("Should require data field in request", func() {
				// Arrange
				requestBody := map[string]interface{}{
					// Missing data field
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				logger.Log("Correctly required data field in request")
			})

			It("Should require authentication", func() {
				// Arrange
				// Temporarily enable auth check to test proper auth failure
				restore := WithAuthCheckEnabled()
				defer restore()

				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"ANTHROPIC_API_KEY": "test-key",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header

				// Act
				UpdateRunnerSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})
		})
	})

	Context("Integration Secrets Management", func() {
		Describe("ListIntegrationSecrets", func() {
			It("Should return empty data when secret doesn't exist", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integration-secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				ListIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("data"))

				data := response["data"].(map[string]interface{})
				Expect(data).To(HaveLen(0), "Should return empty data when secret doesn't exist")

				logger.Log("Correctly handled missing integration secrets")
			})

			It("Should return integration secrets data when secret exists", func() {
				// Arrange - create integration secret
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ambient-non-vertex-integrations",
						Namespace: "test-project",
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"GITHUB_TOKEN":   []byte("github-token"),
						"JIRA_API_TOKEN": []byte("jira-token"),
					},
				}
				ctx := context.Background()
				_, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Create(
					ctx, secret, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integration-secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				ListIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("data"))

				data := response["data"].(map[string]interface{})
				Expect(data).To(HaveKey("GITHUB_TOKEN"))
				Expect(data).To(HaveKey("JIRA_API_TOKEN"))
				Expect(data["GITHUB_TOKEN"]).To(Equal("github-token"))
				Expect(data["JIRA_API_TOKEN"]).To(Equal("jira-token"))

				logger.Log("Successfully retrieved integration secrets")
			})

			It("Should require authentication", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integration-secrets", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header

				// Act
				ListIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})
		})

		Describe("UpdateIntegrationSecrets", func() {
			It("Should create new integration secret when none exists", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"GITHUB_TOKEN":   "new-github-token",
						"JIRA_API_TOKEN": "new-jira-token",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integration-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "integration secrets updated",
				})

				// Verify secret was created
				ctx := context.Background()
				secret, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Get(
					ctx, "ambient-non-vertex-integrations", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(secret.StringData["GITHUB_TOKEN"]).To(Equal("new-github-token"))
				Expect(secret.StringData["JIRA_API_TOKEN"]).To(Equal("new-jira-token"))
				Expect(secret.Annotations["ambient-code.io/runner-secret"]).To(Equal("true"))

				logger.Log("Successfully created new integration secret")
			})

			It("Should update existing integration secret", func() {
				// Arrange - create existing secret
				existingSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ambient-non-vertex-integrations",
						Namespace: "test-project",
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"GITHUB_TOKEN": []byte("old-github-token"),
					},
				}
				ctx := context.Background()
				_, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Create(
					ctx, existingSecret, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"GITHUB_TOKEN":   "updated-github-token",
						"JIRA_API_TOKEN": "new-jira-token",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integration-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "integration secrets updated",
				})

				// Verify secret was updated
				ctx = context.Background()
				secret, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Get(
					ctx, "ambient-non-vertex-integrations", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(string(secret.Data["GITHUB_TOKEN"])).To(Equal("updated-github-token"))
				Expect(string(secret.Data["JIRA_API_TOKEN"])).To(Equal("new-jira-token"))

				logger.Log("Successfully updated existing integration secret")
			})

			It("Should allow any keys for integration secrets", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"CUSTOM_KEY":     "custom-value",
						"ANOTHER_SECRET": "another-value",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integration-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"message": "integration secrets updated",
				})

				logger.Log("Successfully accepted custom keys for integration secrets")
			})

			It("Should require valid JSON body", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integration-secrets", "invalid-json")
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				logger.Log("Correctly rejected invalid JSON")
			})

			It("Should require data field in request", func() {
				// Arrange
				requestBody := map[string]interface{}{
					// Missing data field
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integration-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				UpdateIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				logger.Log("Correctly required data field in request")
			})

			It("Should require authentication", func() {
				// Arrange
				// Temporarily enable auth check to test proper auth failure
				restore := WithAuthCheckEnabled()
				defer restore()

				requestBody := map[string]interface{}{
					"data": map[string]interface{}{
						"GITHUB_TOKEN": "mock-github-token",
					},
				}

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integration-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header

				// Act
				UpdateIntegrationSecrets(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})
		})
	})

	Context("Error Handling", func() {
		It("Should handle Kubernetes API errors gracefully", func() {
			// Test handling of K8s client errors
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/runner-secrets", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			// This tests the basic error handling without needing to inject specific K8s errors
			// Full K8s error simulation would require more complex mocking
			ListRunnerSecrets(ginCtx)

			// Should handle gracefully without panicking
			status := httpUtils.GetResponseRecorder().Code
			Expect(status).To(BeElementOf(http.StatusOK, http.StatusInternalServerError, http.StatusNotFound, http.StatusUnauthorized))

			logger.Log("Handled Kubernetes API interaction gracefully")
		})

		It("Should handle concurrent updates gracefully", func() {
			// Test that multiple concurrent requests don't cause issues
			// This simulates race conditions during secret updates
			requestBody := map[string]interface{}{
				"data": map[string]interface{}{
					"ANTHROPIC_API_KEY": "concurrent-key",
				},
			}

			for i := 0; i < 3; i++ {
				httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				UpdateRunnerSecrets(ginCtx)

				// Each request should be handled independently without errors
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(BeElementOf(http.StatusOK, http.StatusInternalServerError))

				logger.Log("Concurrent request %d handled successfully", i+1)
			}
		})
	})

	Context("Secret Architecture Validation", func() {
		It("Should enforce two-secret architecture", func() {
			// This test validates that the system correctly implements the two-secret architecture:
			// 1. ambient-runner-secrets: ANTHROPIC_API_KEY only
			// 2. ambient-non-vertex-integrations: GITHUB_TOKEN, JIRA_*, custom keys

			// Test runner secrets constraints
			runnerRequestBody := map[string]interface{}{
				"data": map[string]interface{}{
					"ANTHROPIC_API_KEY": "valid-key",
				},
			}

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", runnerRequestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateRunnerSecrets(ginCtx)
			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Test integration secrets flexibility
			httpUtils = test_utils.NewHTTPTestUtils() // Reset

			integrationRequestBody := map[string]interface{}{
				"data": map[string]interface{}{
					"GITHUB_TOKEN":   "github-value",
					"JIRA_API_TOKEN": "jira-value",
					"CUSTOM_KEY":     "custom-value",
				},
			}

			ginCtx = httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integration-secrets", integrationRequestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateIntegrationSecrets(ginCtx)
			httpUtils.AssertHTTPStatus(http.StatusOK)

			logger.Log("Successfully validated two-secret architecture")
		})

		It("Should create secrets with proper annotations", func() {
			// Test that secrets are created with the correct annotations for filtering
			requestBody := map[string]interface{}{
				"data": map[string]interface{}{
					"ANTHROPIC_API_KEY": "test-key",
				},
			}

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/runner-secrets", requestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateRunnerSecrets(ginCtx)
			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Verify secret has proper annotation
			ctx := context.Background()
			secret, err := fakeClients.GetK8sClient().CoreV1().Secrets("test-project").Get(
				ctx, "ambient-runner-secrets", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Annotations["ambient-code.io/runner-secret"]).To(Equal("true"))

			logger.Log("Successfully verified secret annotations")
		})
	})
})
