//go:build test

package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("CodeRabbit Auth Handlers", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelCodeRabbitAuth), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
		k8sUtils  *test_utils.K8sTestUtils
		namespace string
	)

	BeforeEach(func() {
		logger.Log("Setting up CodeRabbit Auth Handler test")

		httpUtils = test_utils.NewHTTPTestUtils()
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		namespace = *config.TestNamespace

		// Set up handler dependencies
		SetupHandlerDependencies(k8sUtils)
		Namespace = namespace

		// Create namespace
		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespace},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create namespace %s", namespace))
		}
	})

	AfterEach(func() {
		// Cleanup
		if k8sUtils == nil {
			return
		}
		ctx := context.Background()
		_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{})
	})

	Describe("ConnectCodeRabbit", func() {
		var (
			mockServer         *httptest.Server
			originalValidation func(context.Context, string) error
		)

		BeforeEach(func() {
			// Mock the CodeRabbit health endpoint
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/health" {
					auth := r.Header.Get("Authorization")
					if auth == "Bearer valid-api-key" {
						w.WriteHeader(http.StatusOK)
						return
					}
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(http.StatusNotFound)
			}))

			// Save original validation function
			originalValidation = ValidateCodeRabbitAPIKey

			// Replace with mock that uses our test server
			ValidateCodeRabbitAPIKey = func(ctx context.Context, apiKey string) error {
				if apiKey == "" {
					return fmt.Errorf("API key is empty")
				}
				if apiKey == "valid-api-key" {
					return nil
				}
				return fmt.Errorf("invalid API key")
			}
		})

		AfterEach(func() {
			if mockServer != nil {
				mockServer.Close()
			}
			// Restore original validation
			ValidateCodeRabbitAPIKey = originalValidation
		})

		Context("When storing credentials with valid request", func() {
			It("Should store credentials successfully", func() {
				reqBody := map[string]string{
					"apiKey": "valid-api-key",
				}

				c := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", reqBody)
				httpUtils.SetAuthHeader("valid-test-token")
				c.Set("userID", "test-user")

				ConnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("message"))
				Expect(response["message"]).To(Equal("CodeRabbit connected successfully"))

				// Verify credentials were stored
				ctx := context.Background()
				creds, err := GetCodeRabbitCredentials(ctx, "test-user")
				Expect(err).NotTo(HaveOccurred())
				Expect(creds).NotTo(BeNil())
				Expect(creds.UserID).To(Equal("test-user"))
				Expect(creds.APIKey).To(Equal("valid-api-key"))

				logger.Log("Successfully stored CodeRabbit credentials")
			})
		})

		Context("When validating request input", func() {
			It("Should reject request with empty API key", func() {
				reqBody := map[string]string{
					"apiKey": "",
				}

				c := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", reqBody)
				httpUtils.SetAuthHeader("valid-test-token")
				c.Set("userID", "test-user")

				ConnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))

				logger.Log("Correctly rejected empty API key")
			})

			It("Should reject request with invalid API key", func() {
				reqBody := map[string]string{
					"apiKey": "invalid-api-key",
				}

				c := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", reqBody)
				httpUtils.SetAuthHeader("valid-test-token")
				c.Set("userID", "test-user")

				ConnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
				Expect(response["error"]).To(Equal("Invalid CodeRabbit API key"))

				logger.Log("Correctly rejected invalid API key")
			})
		})

		Context("When handling authentication", func() {
			It("Should reject unauthenticated requests", func() {
				reqBody := map[string]string{
					"apiKey": "valid-api-key",
				}

				c := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", reqBody)
				// No auth header set

				ConnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
				Expect(response["error"]).To(Equal("Invalid or missing token"))

				logger.Log("Correctly rejected unauthenticated request")
			})

			It("Should reject request without userID", func() {
				reqBody := map[string]string{
					"apiKey": "valid-api-key",
				}

				c := httpUtils.CreateTestGinContext("POST", "/api/auth/coderabbit/connect", reqBody)
				// Set auth header manually without SetAuthHeader (which auto-sets userID)
				c.Request.Header.Set("Authorization", "Bearer valid-test-token")
				// Do NOT set userID in context

				ConnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
				Expect(response["error"]).To(Equal("User authentication required"))

				logger.Log("Correctly rejected request without userID")
			})
		})
	})

	Describe("GetCodeRabbitStatus", func() {
		Context("When credentials exist", func() {
			It("Should return connected status with timestamp", func() {
				// First store credentials
				ctx := context.Background()
				creds := &CodeRabbitCredentials{
					UserID: "test-user",
					APIKey: "test-api-key",
				}
				err := storeCodeRabbitCredentials(ctx, creds)
				Expect(err).NotTo(HaveOccurred())

				// Now check status
				c := httpUtils.CreateTestGinContext("GET", "/api/auth/coderabbit/status", nil)
				httpUtils.SetAuthHeader("valid-test-token")
				c.Set("userID", "test-user")

				GetCodeRabbitStatus(c)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("connected"))
				Expect(response["connected"]).To(BeTrue())
				Expect(response).To(HaveKey("updatedAt"))

				logger.Log("Successfully retrieved connected status")
			})
		})

		Context("When credentials do not exist", func() {
			It("Should return disconnected status", func() {
				c := httpUtils.CreateTestGinContext("GET", "/api/auth/coderabbit/status", nil)
				httpUtils.SetAuthHeader("valid-test-token")
				c.Set("userID", "nonexistent-user")

				GetCodeRabbitStatus(c)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("connected"))
				Expect(response["connected"]).To(BeFalse())

				logger.Log("Successfully returned disconnected status for user without credentials")
			})
		})

		Context("When handling authentication", func() {
			It("Should reject unauthenticated requests", func() {
				c := httpUtils.CreateTestGinContext("GET", "/api/auth/coderabbit/status", nil)
				// No auth header

				GetCodeRabbitStatus(c)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
				Expect(response["error"]).To(Equal("Invalid or missing token"))

				logger.Log("Correctly rejected unauthenticated request")
			})
		})
	})

	Describe("DisconnectCodeRabbit", func() {
		Context("When removing existing credentials", func() {
			It("Should delete credentials successfully", func() {
				// First store credentials
				ctx := context.Background()
				creds := &CodeRabbitCredentials{
					UserID: "test-user",
					APIKey: "test-api-key",
				}
				err := storeCodeRabbitCredentials(ctx, creds)
				Expect(err).NotTo(HaveOccurred())

				// Verify they exist
				retrievedCreds, err := GetCodeRabbitCredentials(ctx, "test-user")
				Expect(err).NotTo(HaveOccurred())
				Expect(retrievedCreds).NotTo(BeNil())

				// Now disconnect
				c := httpUtils.CreateTestGinContext("DELETE", "/api/auth/coderabbit/disconnect", nil)
				httpUtils.SetAuthHeader("valid-test-token")
				c.Set("userID", "test-user")

				DisconnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("message"))
				Expect(response["message"]).To(Equal("CodeRabbit disconnected successfully"))

				// Verify credentials were deleted
				deletedCreds, err := GetCodeRabbitCredentials(ctx, "test-user")
				Expect(err).NotTo(HaveOccurred())
				Expect(deletedCreds).To(BeNil())

				logger.Log("Successfully deleted CodeRabbit credentials")
			})
		})

		Context("When credentials do not exist", func() {
			It("Should succeed idempotently", func() {
				c := httpUtils.CreateTestGinContext("DELETE", "/api/auth/coderabbit/disconnect", nil)
				httpUtils.SetAuthHeader("valid-test-token")
				c.Set("userID", "nonexistent-user")

				DisconnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("message"))
				Expect(response["message"]).To(Equal("CodeRabbit disconnected successfully"))

				logger.Log("Successfully handled disconnect for nonexistent credentials")
			})
		})

		Context("When handling authentication", func() {
			It("Should reject unauthenticated requests", func() {
				c := httpUtils.CreateTestGinContext("DELETE", "/api/auth/coderabbit/disconnect", nil)
				// No auth header

				DisconnectCodeRabbit(c)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
				Expect(response["error"]).To(Equal("Invalid or missing token"))

				logger.Log("Correctly rejected unauthenticated request")
			})
		})
	})
})
