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

var _ = Describe("Gerrit Auth Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelGerritAuth), func() {
	var (
		httpUtils                   *test_utils.HTTPTestUtils
		k8sUtils                    *test_utils.K8sTestUtils
		originalNamespace           string
		originalValidateGerritToken func(context.Context, string, string, string, string, string) (bool, error)
		testToken                   string
	)

	BeforeEach(func() {
		logger.Log("Setting up Gerrit Auth Handler test")

		originalNamespace = Namespace
		originalValidateGerritToken = validateGerritTokenFn
		// Stub out credential validation to avoid live HTTP calls
		validateGerritTokenFn = func(_ context.Context, _, _, _, _, _ string) (bool, error) {
			return true, nil
		}

		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		Namespace = *config.TestNamespace

		httpUtils = test_utils.NewHTTPTestUtils()

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
		validateGerritTokenFn = originalValidateGerritToken

		if k8sUtils != nil {
			// Clean up per-user secrets (best-effort)
			_ = k8sUtils.K8sClient.CoreV1().Secrets(*config.TestNamespace).Delete(context.Background(), gerritSecretName("test-user"), metav1.DeleteOptions{})
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), *config.TestNamespace, metav1.DeleteOptions{})
		}
	})

	Context("ConnectGerrit", func() {
		It("Should require authentication token", func() {
			requestBody := map[string]interface{}{
				"instanceName": "openstack",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "john",
				"httpToken":    "abc123",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			// Don't set auth header

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			requestBody := map[string]interface{}{
				"instanceName": "openstack",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "john",
				"httpToken":    "abc123",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			// Don't set user context - should reach the userID check

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("User authentication required")
		})

		It("Should reject invalid instance names", func() {
			invalidNames := []string{
				"A",            // too short and uppercase
				"-bad",         // starts with hyphen
				"UPPER",        // uppercase
				"has spaces",   // spaces
				"special!char", // special characters
			}

			for _, name := range invalidNames {
				requestBody := map[string]interface{}{
					"instanceName": name,
					"url":          "https://review.opendev.org",
					"authMethod":   "http_basic",
					"username":     "john",
					"httpToken":    "abc123",
				}

				ctx := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ConnectGerrit(ctx)

				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(Equal(http.StatusBadRequest), "Should reject invalid instance name: "+name)

				httpUtils = test_utils.NewHTTPTestUtils()
			}
		})

		It("Should reject HTTP URLs (SSRF protection)", func() {
			requestBody := map[string]interface{}{
				"instanceName": "test-instance",
				"url":          "http://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "john",
				"httpToken":    "abc123",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should reject mixed auth credentials for http_basic", func() {
			requestBody := map[string]interface{}{
				"instanceName":      "openstack",
				"url":               "https://review.opendev.org",
				"authMethod":        "http_basic",
				"username":          "john",
				"httpToken":         "abc123",
				"gitcookiesContent": "should-not-be-here",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("gitcookiesContent must not be provided with http_basic auth")
		})

		It("Should reject mixed auth credentials for git_cookies", func() {
			requestBody := map[string]interface{}{
				"instanceName":      "android",
				"url":               "https://android-review.googlesource.com",
				"authMethod":        "git_cookies",
				"gitcookiesContent": ".googlesource.com\tTRUE\t/\tTRUE\t0\to\tgit-user.cookies=val",
				"username":          "should-not-be-here",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("username and httpToken must not be provided with git_cookies auth")
		})

		It("Should require username and httpToken for http_basic", func() {
			requestBody := map[string]interface{}{
				"instanceName": "openstack",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "",
				"httpToken":    "",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Username and HTTP token are required for HTTP basic auth")
		})

		It("Should require gitcookiesContent for git_cookies", func() {
			requestBody := map[string]interface{}{
				"instanceName":      "android",
				"url":               "https://android-review.googlesource.com",
				"authMethod":        "git_cookies",
				"gitcookiesContent": "",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Gitcookies content is required for git_cookies auth")
		})

		It("Should reject unsupported auth method", func() {
			requestBody := map[string]interface{}{
				"instanceName": "openstack",
				"url":          "https://review.opendev.org",
				"authMethod":   "oauth",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertErrorMessage("Auth method must be 'http_basic' or 'git_cookies'")
		})

		It("Should accept valid http_basic credentials", func() {
			requestBody := map[string]interface{}{
				"instanceName": "openstack",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "john",
				"httpToken":    "abc123",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			status := httpUtils.GetResponseRecorder().Code
			Expect(status).NotTo(Equal(http.StatusBadRequest), "Should accept valid http_basic credentials")
		})

		It("Should return 401 when credentials are invalid", func() {
			validateGerritTokenFn = func(_ context.Context, _, _, _, _, _ string) (bool, error) {
				return false, nil
			}

			requestBody := map[string]interface{}{
				"instanceName": "openstack",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "john",
				"httpToken":    "wrong-token",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid Gerrit credentials")
		})

		It("Should return error when validation fails", func() {
			validateGerritTokenFn = func(_ context.Context, _, _, _, _, _ string) (bool, error) {
				return false, fmt.Errorf("connection timeout")
			}

			requestBody := map[string]interface{}{
				"instanceName": "openstack",
				"url":          "https://review.opendev.org",
				"authMethod":   "http_basic",
				"username":     "john",
				"httpToken":    "abc123",
			}

			context := httpUtils.CreateTestGinContext("POST", "/api/auth/gerrit/connect", requestBody)
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			ConnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})
	})

	Context("GetGerritStatus", func() {
		It("Should require authentication token", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/auth/gerrit/openstack/status", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}

			GetGerritStatus(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/auth/gerrit/openstack/status", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}
			httpUtils.SetAuthHeader(testToken)
			// Don't set user context

			GetGerritStatus(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("User authentication required")
		})

		It("Should return not connected when no credentials exist", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/auth/gerrit/nonexistent/status", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "nonexistent"},
			}
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			GetGerritStatus(context)

			status := httpUtils.GetResponseRecorder().Code
			Expect(status).To(BeElementOf(http.StatusOK, http.StatusNotFound))
		})
	})

	Context("DisconnectGerrit", func() {
		It("Should require authentication token", func() {
			context := httpUtils.CreateTestGinContext("DELETE", "/api/auth/gerrit/openstack/disconnect", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}

			DisconnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			context := httpUtils.CreateTestGinContext("DELETE", "/api/auth/gerrit/openstack/disconnect", nil)
			context.Params = gin.Params{
				gin.Param{Key: "instanceName", Value: "openstack"},
			}
			httpUtils.SetAuthHeader(testToken)
			// Don't set user context

			DisconnectGerrit(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("User authentication required")
		})
	})

	Context("ListGerritInstances", func() {
		It("Should require authentication token", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/auth/gerrit/instances", nil)

			ListGerritInstances(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should require user authentication", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/auth/gerrit/instances", nil)
			httpUtils.SetAuthHeader(testToken)
			// Don't set user context

			ListGerritInstances(context)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("User authentication required")
		})
	})

	Context("Per-User Secret Naming", func() {
		It("Should derive secret name from user ID", func() {
			name := gerritSecretName("user123")
			Expect(name).To(Equal("gerrit-credentials-user123"))
		})

		It("Should produce valid K8s Secret names", func() {
			name := gerritSecretName("system-serviceaccount-ns-user")
			Expect(name).To(HavePrefix("gerrit-credentials-"))
			Expect(name).To(MatchRegexp(`^[a-z0-9][-a-z0-9.]*[a-z0-9]$`))
		})
	})

	Context("URL Validation", func() {
		It("Should reject HTTP URLs", func() {
			err := validateGerritURL("http://review.opendev.org")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("HTTPS"))
		})

		It("Should reject URLs without hostname", func() {
			err := validateGerritURL("https://")
			Expect(err).To(HaveOccurred())
		})

		It("Should reject FTP URLs", func() {
			err := validateGerritURL("ftp://review.opendev.org")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("HTTPS"))
		})
	})
})
