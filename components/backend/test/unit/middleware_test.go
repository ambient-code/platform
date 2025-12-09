package unit

import (
	"context"

	"ambient-code-backend/test/config"
	test_constants "ambient-code-backend/test/constants"
	"fmt"
	"net/http"
	"strings"

	"ambient-code-backend/handlers"
	"ambient-code-backend/test/logger"
	"ambient-code-backend/test/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	k8stesting "k8s.io/client-go/testing"
)

var _ = Describe("Middleware Handlers", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelMiddleware), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
		k8sUtils  *test_utils.K8sTestUtils
	)

	BeforeEach(func() {
		logger.Log("Setting up Middleware Handler test")

		// Auth is disabled by default from config for unit tests

		httpUtils = test_utils.NewHTTPTestUtils()
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace) // Use fake clients for unit tests

		// Set up all handler dependencies using clean pattern
		test_utils.SetupHandlerDependencies(k8sUtils)

		// Pre-create test Roles with different permission sets for RBAC testing
		// Create roles in both test namespace and common test project namespaces
		ctx := context.Background()
		testNamespace := *config.TestNamespace

		// Create namespaces first (if they don't exist)
		testNamespaces := []string{testNamespace, "test-project"}
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

	Describe("ValidateProjectContext", func() {
		var (
			middleware gin.HandlerFunc
		)

		BeforeEach(func() {
			middleware = handlers.ValidateProjectContext()
		})

		Context("When validating project names", func() {
			It("Should accept valid Kubernetes namespace names", func() {
				testCases := []struct {
					name        string
					projectName string
					shouldPass  bool
				}{
					{
						name:        "Valid lowercase name",
						projectName: "valid-project-name",
						shouldPass:  true,
					},
					{
						name:        "Valid name with numbers",
						projectName: "project123",
						shouldPass:  true,
					},
					{
						name:        "Valid name with hyphens",
						projectName: "my-project-v2",
						shouldPass:  true,
					},
					{
						name:        "Invalid uppercase letters",
						projectName: "Invalid-Project-Name",
						shouldPass:  false,
					},
					{
						name:        "Invalid underscores",
						projectName: "invalid_project_name",
						shouldPass:  false,
					},
					{
						name:        "Invalid special characters",
						projectName: "invalid@project!",
						shouldPass:  false,
					},
					{
						name:        "Invalid starting with hyphen",
						projectName: "-invalid-project",
						shouldPass:  false,
					},
					{
						name:        "Invalid ending with hyphen",
						projectName: "invalid-project-",
						shouldPass:  false,
					},
					{
						name:        "Empty project name",
						projectName: "",
						shouldPass:  false,
					},
					{
						name:        "Too long project name",
						projectName: "this-is-a-very-long-project-name-that-exceeds-the-kubernetes-namespace-limit-of-63-characters",
						shouldPass:  false,
					},
				}

				for _, tc := range testCases {
					By(tc.name, func() {
						// Arrange
						path := "/api/projects/" + tc.projectName + "/sessions"
						context := httpUtils.CreateTestGinContext("GET", path, nil)

						// Set up route parameters
						context.Params = gin.Params{
							{Key: "projectName", Value: tc.projectName},
						}

						// Always set auth header - validation happens after auth check
						// For invalid names, we still need auth to get past the auth check
						// so we can test the validation logic
						httpUtils.SetAuthHeader("test-token")
						httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

						// Act
						middleware(context)

						// Assert
						if tc.shouldPass {
							// Should not abort the request
							Expect(context.IsAborted()).To(BeFalse(), "Valid project name should not abort request")

							// Should set project in context
							project, exists := context.Get("project")
							Expect(exists).To(BeTrue(), "Project should be set in context")
							Expect(project).To(Equal(tc.projectName), "Project name should match")
						} else {
							// Should abort the request with bad request status
							Expect(context.IsAborted()).To(BeTrue(), "Invalid project name should abort request")
							httpUtils.AssertHTTPStatus(http.StatusBadRequest)
						}

						logger.Log("Test case '%s' completed successfully", tc.name)
					})
				}
			})
		})

		Context("When handling authentication", func() {
			It("Should require authorization header", func() {
				// Temporarily override GetK8sClientsFunc to require auth
				originalFunc := handlers.GetK8sClientsFunc
				handlers.SetGetK8sClientsFunc(func(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
					// Force auth check - return nil if no auth header
					auth := c.GetHeader("Authorization")
					if auth == "" {
						auth = c.GetHeader("X-Forwarded-Access-Token")
					}
					if auth == "" {
						return nil, nil
					}
					// Call original function if auth header present
					return originalFunc(c)
				})
				defer func() {
					handlers.SetGetK8sClientsFunc(originalFunc)
				}()

				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}

				// Act (no auth header set)
				middleware(context)

				// Assert
				Expect(context.IsAborted()).To(BeTrue(), "Request without auth should be aborted")
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
			})

			It("Should accept valid Bearer token", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("valid-test-token")

				// Act
				middleware(context)

				// Assert
				Expect(context.IsAborted()).To(BeFalse(), "Request with valid auth should not be aborted")
			})

			It("Should accept token with valid RBAC permissions", func() {
				// Arrange - Create a token with actual RBAC permissions
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}

				// Use the same client instance that handlers use (from k8sUtils set up in BeforeEach)
				// Create namespace for the test using handlers' client (if it doesn't exist)
				_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(
					context.Request.Context(),
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-project",
						},
					},
					metav1.CreateOptions{},
				)
				// Ignore AlreadyExists errors - namespace may have been created in BeforeEach
				if err != nil && !errors.IsAlreadyExists(err) {
					Expect(err).NotTo(HaveOccurred())
				}

				// Create token using pre-created read Role with list permissions for agenticsessions
				token, saName, err := httpUtils.SetValidTestToken(
					k8sUtils,
					"test-project",
					[]string{"get", "list"}, // Not used, but kept for clarity
					"agenticsessions",
					"",
					"test-agenticsessions-read-role", // Use pre-created Role with read permissions
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
				Expect(saName).NotTo(BeEmpty())

				// Act
				middleware(context)

				// Assert
				Expect(context.IsAborted()).To(BeFalse(), "Request with valid RBAC token should not be aborted")
				project, exists := context.Get("project")
				Expect(exists).To(BeTrue(), "Project should be set in context")
				Expect(project).To(Equal("test-project"))

				logger.Log("Successfully validated RBAC token for ServiceAccount: %s", saName)
			})

			It("Should reject token with insufficient RBAC permissions", func() {
				// Arrange - Create a token without required permissions
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}

				// Use the same client instance that handlers use (from k8sUtils set up in BeforeEach)
				// Create namespace for the test using handlers' client (if it doesn't exist)
				_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(
					context.Request.Context(),
					&corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: "test-project",
						},
					},
					metav1.CreateOptions{},
				)
				// Ignore AlreadyExists errors - namespace may have been created in BeforeEach
				if err != nil && !errors.IsAlreadyExists(err) {
					Expect(err).NotTo(HaveOccurred())
				}

				// Configure SSAR to return false (insufficient permissions)
				// We can set this BEFORE creating the token because we're using a pre-created Role
				k8sUtils.SSARAllowedFunc = func(action k8stesting.Action) bool {
					return false // Simulate insufficient permissions for handler operations
				}

				// Create token using pre-created read-only Role (only get, not list)
				// This avoids creating RoleBindings during token setup, allowing SSAR denial to be set first
				_, _, err = httpUtils.SetValidTestToken(
					k8sUtils,
					"test-project",
					[]string{"get"}, // Only get, not list (not used, but kept for clarity)
					"agenticsessions",
					"",
					"test-read-only-role", // Use pre-created Role with read-only permissions (get only, no list)
				)
				Expect(err).NotTo(HaveOccurred())

				// Act
				middleware(context)

				// Assert - Should abort due to insufficient permissions
				Expect(context.IsAborted()).To(BeTrue(), "Request with insufficient permissions should be aborted")
				httpUtils.AssertHTTPStatus(http.StatusForbidden)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("error"))
				Expect(response["error"]).To(ContainSubstring("Unauthorized"))

				logger.Log("Correctly rejected token with insufficient RBAC permissions")
			})

			It("Should reject malformed authorization header", func() {
				// Temporarily override GetK8sClientsFunc to require valid auth
				originalFunc := handlers.GetK8sClientsFunc
				handlers.SetGetK8sClientsFunc(func(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
					// Force auth check - validate Bearer format
					auth := c.GetHeader("Authorization")
					if auth == "" {
						return nil, nil
					}
					parts := strings.SplitN(auth, " ", 2)
					if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
						return nil, nil // Malformed header
					}
					// Call original function if valid Bearer token
					return originalFunc(c)
				})
				defer func() {
					handlers.SetGetK8sClientsFunc(originalFunc)
				}()

				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/sessions", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				context.Request.Header.Set("Authorization", "InvalidFormat token123")

				// Act
				middleware(context)

				// Assert
				Expect(context.IsAborted()).To(BeTrue(), "Malformed auth header should abort request")
				// Expect 403 (Forbidden) instead of 401 because GetK8sClientForPermissions returns a client
				// but the RBAC check fails, or 401 if client is nil
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(BeElementOf(http.StatusUnauthorized, http.StatusForbidden))
			})
		})

	})

	Describe("ExtractServiceAccountFromAuth", func() {
		It("Should extract service account from X-Remote-User header", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("GET", "/test", nil)
			context.Request.Header.Set("X-Remote-User", "system:serviceaccount:namespace:service-account")

			// Act
			namespace, serviceAccount, found := handlers.ExtractServiceAccountFromAuth(context)

			// Assert
			Expect(found).To(BeTrue(), "Should find service account in header")
			Expect(namespace).To(Equal("namespace"), "Should extract correct namespace")
			Expect(serviceAccount).To(Equal("service-account"), "Should extract correct service account name")

			logger.Log("Extracted service account: %s/%s", namespace, serviceAccount)
		})

		It("Should return false for non-service account users", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("GET", "/test", nil)
			context.Request.Header.Set("X-Remote-User", "regular-user")

			// Act
			_, _, found := handlers.ExtractServiceAccountFromAuth(context)

			// Assert
			Expect(found).To(BeFalse(), "Should not find service account for regular user")
		})

		It("Should handle malformed service account headers", func() {
			testCases := []string{
				"system:serviceaccount:namespace", // Missing service account name
				"system:serviceaccount:",          // Missing namespace and service account
				"system:serviceaccount",           // Missing all parts
				"serviceaccount:namespace:sa",     // Wrong prefix
				"",                                // Empty header
			}

			for _, testCase := range testCases {
				By(fmt.Sprintf("Testing malformed header: '%s'", testCase), func() {
					// Arrange
					context := httpUtils.CreateTestGinContext("GET", "/test", nil)
					if testCase != "" {
						context.Request.Header.Set("X-Remote-User", testCase)
					}

					// Act
					_, _, found := handlers.ExtractServiceAccountFromAuth(context)

					// Assert
					Expect(found).To(BeFalse(), "Malformed header should not be parsed as service account")
				})
			}
		})
	})

})
