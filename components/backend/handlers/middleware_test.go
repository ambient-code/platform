//go:build test

package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stesting "k8s.io/client-go/testing"
)

var _ = Describe("Middleware Handlers", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelMiddleware), func() {
	var (
		httpUtils         *test_utils.HTTPTestUtils
		k8sUtils          *test_utils.K8sTestUtils
		createdNamespaces []string
	)

	BeforeEach(func() {
		logger.Log("Setting up Middleware Handler test")

		httpUtils = test_utils.NewHTTPTestUtils()
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		createdNamespaces = []string{}

		// Set up handler dependencies (now lives in handlers test helpers)
		SetupHandlerDependencies(k8sUtils)

		// Pre-create test Roles with different permission sets for RBAC testing
		ctx := context.Background()
		testNamespace := *config.TestNamespace

		// Create namespaces first (if they don't exist)
		testNamespaces := []string{testNamespace, "test-project"}
		createdNamespaces = append(createdNamespaces, testNamespaces...)
		for _, ns := range testNamespaces {
			_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}, metav1.CreateOptions{})
			// Ignore AlreadyExists errors
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create namespace %s", ns))
			}
		}

		// Read-only role: only get and list permissions
		for _, ns := range testNamespaces {
			_, err := k8sUtils.CreateTestRole(ctx, ns, "test-read-only-role", []string{"get", "list"}, "*", "")
			Expect(err).NotTo(HaveOccurred())

			// AgenticSessions-specific roles
			_, err = k8sUtils.CreateTestRole(ctx, ns, "test-agenticsessions-read-role", []string{"get", "list"}, "agenticsessions", "")
			Expect(err).NotTo(HaveOccurred())
		}
	})

	AfterEach(func() {
		// Best-effort cleanup for test isolation (each test has a fresh fake client, but keep hygiene)
		if k8sUtils == nil {
			return
		}
		ctx := context.Background()
		for _, ns := range createdNamespaces {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
		}
	})

	Describe("ValidateProjectContext", func() {
		var middleware gin.HandlerFunc

		BeforeEach(func() {
			middleware = ValidateProjectContext()
		})

		Context("When validating project names", func() {
			It("Should accept valid Kubernetes namespace names", func() {
				testCases := []struct {
					name        string
					projectName string
					shouldPass  bool
				}{
					{name: "Valid lowercase name", projectName: "valid-project-name", shouldPass: true},
					{name: "Valid name with numbers", projectName: "project123", shouldPass: true},
					{name: "Valid name with hyphens", projectName: "my-project-v2", shouldPass: true},
					{name: "Invalid uppercase letters", projectName: "Invalid-Project-Name", shouldPass: false},
					{name: "Invalid underscores", projectName: "invalid_project_name", shouldPass: false},
					{name: "Invalid special characters", projectName: "invalid@project!", shouldPass: false},
					{name: "Invalid starting with hyphen", projectName: "-invalid-project", shouldPass: false},
					{name: "Invalid ending with hyphen", projectName: "invalid-project-", shouldPass: false},
					{name: "Empty name", projectName: "", shouldPass: false},
					{name: "Too long name", projectName: strings.Repeat("a", 64), shouldPass: false},
				}

				for _, tc := range testCases {
					By(tc.name, func() {
						context := httpUtils.CreateTestGinContext("GET", "/api/projects/"+tc.projectName+"/sessions", nil)
						context.Params = gin.Params{
							{Key: "projectName", Value: tc.projectName},
						}
						httpUtils.SetAuthHeader("test-token")

						middleware(context)

						if tc.shouldPass {
							Expect(context.IsAborted()).To(BeFalse(), "Valid project name should not abort request")
						} else {
							Expect(context.IsAborted()).To(BeTrue(), "Invalid project name should abort request")
						}

						logger.Log("Test case '%s' completed successfully", tc.name)
					})
				}
			})
		})

		Context("When handling authentication", func() {
			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

				middleware(context)

				Expect(context.IsAborted()).To(BeTrue(), "Request without auth should be aborted")
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
			})

			It("Should accept valid Bearer token", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
				httpUtils.SetAuthHeader("valid-test-token")

				middleware(context)

				Expect(context.IsAborted()).To(BeFalse(), "Request with valid auth should not be aborted")
			})

			It("Should accept token with valid RBAC permissions", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

				_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(
					context.Request.Context(),
					&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-project"}},
					metav1.CreateOptions{},
				)
				if err != nil && !errors.IsAlreadyExists(err) {
					Expect(err).NotTo(HaveOccurred())
				}

				token, saName, err := httpUtils.SetValidTestToken(
					k8sUtils,
					"test-project",
					[]string{"get", "list"},
					"agenticsessions",
					"",
					"test-agenticsessions-read-role",
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
				Expect(saName).NotTo(BeEmpty())

				middleware(context)

				Expect(context.IsAborted()).To(BeFalse(), "Request with valid RBAC token should not be aborted")
				logger.Log("Successfully validated RBAC token for ServiceAccount: %s", saName)
			})

			It("Should reject token with insufficient RBAC permissions", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

				_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(
					context.Request.Context(),
					&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-project"}},
					metav1.CreateOptions{},
				)
				if err != nil && !errors.IsAlreadyExists(err) {
					Expect(err).NotTo(HaveOccurred())
				}

				// Create the token first (setup must succeed)
				_, _, err = httpUtils.SetValidTestToken(
					k8sUtils,
					"test-project",
					[]string{"get"},
					"agenticsessions",
					"",
					"test-read-only-role",
				)
				Expect(err).NotTo(HaveOccurred())

				// Then deny SSAR
				k8sUtils.SSARAllowedFunc = func(action k8stesting.Action) bool { return false }

				middleware(context)

				Expect(context.IsAborted()).To(BeTrue(), "Request with insufficient permissions should be aborted")
				httpUtils.AssertHTTPStatus(http.StatusForbidden)
				httpUtils.AssertErrorMessage("Unauthorized to access project")
				logger.Log("Correctly rejected token with insufficient RBAC permissions")
			})

			It("Should reject invalid token", func() {
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
				httpUtils.SetAuthHeader("invalid-token")

				middleware(context)

				Expect(context.IsAborted()).To(BeTrue(), "Invalid token should be aborted")
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})
		})
	})

	Describe("extractRequestToken", func() {
		It("Should prefer X-Forwarded-Access-Token over Authorization: Bearer", func() {
			c := httpUtils.CreateTestGinContext("GET", "/health", nil)
			c.Request.Header.Set("Authorization", "Bearer bearer-token")
			c.Request.Header.Set("X-Forwarded-Access-Token", "forwarded-token")

			token, src, hasAuth, hasFwd := extractRequestToken(c)
			Expect(token).To(Equal("forwarded-token"))
			Expect(src).To(Equal("x-forwarded-access-token"))
			Expect(hasAuth).To(BeTrue())
			Expect(hasFwd).To(BeTrue())
		})

		It("Should accept raw Authorization token (non-Bearer)", func() {
			c := httpUtils.CreateTestGinContext("GET", "/health", nil)
			c.Request.Header.Set("Authorization", "raw-token")

			token, src, hasAuth, hasFwd := extractRequestToken(c)
			Expect(token).To(Equal("raw-token"))
			Expect(src).To(Equal("authorization"))
			Expect(hasAuth).To(BeTrue())
			Expect(hasFwd).To(BeFalse())
		})

		It("Should fall back to X-Forwarded-Access-Token when Authorization is missing/empty", func() {
			c := httpUtils.CreateTestGinContext("GET", "/health", nil)
			c.Request.Header.Set("X-Forwarded-Access-Token", "forwarded-token")

			token, src, hasAuth, hasFwd := extractRequestToken(c)
			Expect(token).To(Equal("forwarded-token"))
			Expect(src).To(Equal("x-forwarded-access-token"))
			Expect(hasAuth).To(BeFalse())
			Expect(hasFwd).To(BeTrue())
		})
	})

	Describe("ExtractServiceAccountFromAuth", func() {
		It("Should extract service account from token review", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)

			_, _, err := httpUtils.SetValidTestToken(
				k8sUtils,
				"test-project",
				[]string{"get", "list"},
				"agenticsessions",
				"test-sa",
				"test-agenticsessions-read-role",
			)
			Expect(err).NotTo(HaveOccurred())

			namespace, serviceAccount, found := ExtractServiceAccountFromAuth(context)
			Expect(found).To(BeTrue(), "Should find service account from token")
			Expect(namespace).To(Equal("test-project"))
			Expect(serviceAccount).To(Equal("test-sa"))
			logger.Log("Extracted service account: %s/%s", namespace, serviceAccount)
		})

		It("Should return false for non-service account users", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
			httpUtils.SetAuthHeader("regular-user-token")

			_, _, found := ExtractServiceAccountFromAuth(context)
			Expect(found).To(BeFalse(), "Should not find service account for regular user")
		})

		It("Should handle malformed service account headers", func() {
			testCases := []string{"", "Bearer", "Bearer invalid.token", "NotBearer token"}

			for _, header := range testCases {
				By(fmt.Sprintf("Testing malformed header: %s", header), func() {
					context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
					context.Request.Header.Set("Authorization", header)

					_, _, found := ExtractServiceAccountFromAuth(context)
					Expect(found).To(BeFalse(), "Malformed header should not be parsed as service account")
				})
			}
		})
	})
})
