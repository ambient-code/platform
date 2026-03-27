//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
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

var _ = Describe("Gerrit Auth Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, "gerrit-auth"), func() {
	var (
		httpUtils         *test_utils.HTTPTestUtils
		k8sUtils          *test_utils.K8sTestUtils
		originalNamespace string
		testToken         string
	)

	BeforeEach(func() {
		logger.Log("Setting up Gerrit Auth Handler test")

		originalNamespace = Namespace

		// Use centralized handler dependencies setup
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// gerrit_auth.go uses Namespace (backend namespace) for secret operations
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

	Context("ConnectGerrit", func() {
		It("Should require authentication", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "testuser",
				"httpToken":    "secret-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			// Don't set auth header
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "testuser",
				"httpToken":    "secret-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			// Don't set user context

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertJSONContains(map[string]interface{}{
				"error": "Invalid or missing token",
			})
		})

		It("Should require valid JSON body", func() {
			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", "invalid-json")
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should require instanceName field", func() {
			requestBody := map[string]interface{}{
				"url":        "https://review.opendev.org",
				"authMethod": "http_basic",
				"username":   "testuser",
				"httpToken":  "secret-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should require url field", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"authMethod":   "http_basic",
				"username":     "testuser",
				"httpToken":    "secret-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should require authMethod field", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"username":     "testuser",
				"httpToken":    "secret-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should reject invalid instance names", func() {
			invalidNames := []string{
				"a",       // too short (single char)
				"INVALID", // uppercase
				"my@name", // special characters
			}

			for _, name := range invalidNames {
				requestBody := map[string]interface{}{
					"instanceName": name,
					"url":          "https://review.opendev.org",
					"authMethod":   "http_basic",
					"username":     "testuser",
					"httpToken":    "secret-token",
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGerrit(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Instance name must be lowercase alphanumeric with hyphens (2-63 chars)")

				// Reset for next test
				httpUtils = test_utils.NewHTTPTestUtils()
			}
		})

		It("Should accept valid instance names", func() {
			validNames := []string{
				"my-gerrit",
				"openstack",
				"review-01",
				"a1",
			}

			for _, name := range validNames {
				requestBody := map[string]interface{}{
					"instanceName": name,
					"url":          "https://review.opendev.org",
					"authMethod":   "http_basic",
					"username":     "testuser",
					"httpToken":    "secret-token",
				}

				context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGerrit(context)

				// Should not fail at instance name validation stage
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest), "Should accept valid instance name: "+name)

				// Reset for next test
				httpUtils = test_utils.NewHTTPTestUtils()
			}
		})

		It("Should reject invalid auth method", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"authMethod":   "oauth2",
				"username":     "testuser",
				"httpToken":    "secret-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Auth method must be 'http_basic' or 'git_cookies'")
		})

		It("Should require username and httpToken for http_basic", func() {
			// Missing both username and httpToken
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Username and HTTP token are required for HTTP basic auth")
		})

		It("Should require username for http_basic when only httpToken provided", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"httpToken":    "secret-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Username and HTTP token are required for HTTP basic auth")
		})

		It("Should require httpToken for http_basic when only username provided", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "testuser",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Username and HTTP token are required for HTTP basic auth")
		})

		It("Should require gitcookiesContent for git_cookies", func() {
			requestBody := map[string]interface{}{
				"instanceName": "my-gerrit",
				"url":          "https://review.opendev.org",
				"authMethod":   "git_cookies",
			}

			context := httpUtils.CreateTestGinContext("POST", "/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Gitcookies content is required for git_cookies auth")
		})
	})

	Context("GetGerritStatus", func() {
		It("Should require authentication", func() {
			context := httpUtils.CreateTestGinContext("GET", "/auth/gerrit/openstack/status", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}
			// Don't set auth header
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			GetGerritStatus(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			context := httpUtils.CreateTestGinContext("GET", "/auth/gerrit/openstack/status", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}
			// Don't set user context

			GetGerritStatus(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertJSONContains(map[string]interface{}{
				"error": "Invalid or missing token",
			})
		})
	})

	Context("DisconnectGerrit", func() {
		It("Should require authentication", func() {
			context := httpUtils.CreateTestGinContext("DELETE", "/auth/gerrit/openstack/disconnect", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}
			// Don't set auth header
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			DisconnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			context := httpUtils.CreateTestGinContext("DELETE", "/auth/gerrit/openstack/disconnect", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}
			// Don't set user context

			DisconnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertJSONContains(map[string]interface{}{
				"error": "Invalid or missing token",
			})
		})
	})

	Context("ListGerritInstances", func() {
		It("Should require authentication", func() {
			context := httpUtils.CreateTestGinContext("GET", "/auth/gerrit/instances", nil)
			// Don't set auth header
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ListGerritInstances(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			context := httpUtils.CreateTestGinContext("GET", "/auth/gerrit/instances", nil)
			// Don't set user context

			ListGerritInstances(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertJSONContains(map[string]interface{}{
				"error": "Invalid or missing token",
			})
		})
	})
})
