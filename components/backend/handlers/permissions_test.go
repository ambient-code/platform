//go:build test

package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8stesting "k8s.io/client-go/testing"
)

var _ = Describe("Permissions Handler", Ordered, Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelPermissions), func() {
	var (
		httpUtils                 *test_utils.HTTPTestUtils
		testClientFactory         *test_utils.TestClientFactory
		fakeClients               *test_utils.FakeClientSet
		k8sUtils                  *test_utils.K8sTestUtils // Store for use in tests
		originalK8sClient         kubernetes.Interface
		originalK8sClientMw       kubernetes.Interface
		originalK8sClientProjects kubernetes.Interface
		originalEnv               string
		originalNamespace         string
		createdNamespaces         []string
	)

	BeforeEach(func() {
		logger.Log("Setting up Permissions Handler test")

		// Save original state to restore in AfterEach
		originalK8sClient = K8sClient
		originalK8sClientMw = K8sClientMw
		originalK8sClientProjects = K8sClientProjects

		// Store original environment values for cleanup
		originalEnv = os.Getenv("ENVIRONMENT")
		originalNamespace = os.Getenv("NAMESPACE")

		// Use centralized handler dependencies setup
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// Pre-create test Roles with different permission sets for RBAC testing
		// This allows tests to use pre-existing Roles without creating RoleBindings during token setup
		ctx := context.Background()
		testNamespace := *config.TestNamespace

		// Create namespaces first (if they don't exist)
		testNamespaces := []string{testNamespace, "test-project", "default"}
		createdNamespaces = append([]string{}, testNamespaces...)
		for _, ns := range testNamespaces {
			_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: ns},
			}, metav1.CreateOptions{})
			// Ignore AlreadyExists errors
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Failed to create namespace %s", ns))
			}
		}

		// Create roles in both test namespace and common test project namespaces
		for _, ns := range testNamespaces {
			// Read-only role: only get and list permissions
			_, err := k8sUtils.CreateTestRole(ctx, ns, "test-read-only-role", []string{"get", "list"}, "*", "")
			Expect(err).NotTo(HaveOccurred())

			// Write role: includes create, update, delete
			_, err = k8sUtils.CreateTestRole(ctx, ns, "test-write-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
			Expect(err).NotTo(HaveOccurred())

			// RoleBindings-specific roles
			_, err = k8sUtils.CreateTestRole(ctx, ns, "test-rolebindings-read-role", []string{"get", "list"}, "rolebindings", "rbac.authorization.k8s.io")
			Expect(err).NotTo(HaveOccurred())

			_, err = k8sUtils.CreateTestRole(ctx, ns, "test-rolebindings-write-role", []string{"get", "list", "create", "update", "delete"}, "rolebindings", "rbac.authorization.k8s.io")
			Expect(err).NotTo(HaveOccurred())

			// Namespaces-specific roles
			_, err = k8sUtils.CreateTestRole(ctx, ns, "test-namespaces-read-role", []string{"get", "list"}, "namespaces", "")
			Expect(err).NotTo(HaveOccurred())

			_, err = k8sUtils.CreateTestRole(ctx, ns, "test-namespaces-write-role", []string{"get", "list", "create", "update", "delete"}, "namespaces", "")
			Expect(err).NotTo(HaveOccurred())
		}

		// For permissions tests, we need to set all the package-level K8s client variables
		// Different handlers use different client variables, so set them all
		// Use the fake client directly from the test setup instead of server.K8sClient which may be nil
		K8sClient = k8sUtils.K8sClient
		K8sClientMw = k8sUtils.K8sClient
		K8sClientProjects = k8sUtils.K8sClient

		// Use the same fake client for test data creation that handlers will use
		testClientFactory = test_utils.NewTestClientFactory()
		fakeClients = testClientFactory.GetFakeClients()

		// Override the fake clients to use the same instance as handlers
		fakeClients = &test_utils.FakeClientSet{
			K8sClient: k8sUtils.K8sClient,
		}

		httpUtils = test_utils.NewHTTPTestUtils()
	})

	AfterEach(func() {
		// Best-effort cleanup for test isolation (even though each spec uses a fresh fake client)
		if k8sUtils != nil {
			ctx := context.Background()
			for _, ns := range createdNamespaces {
				_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
			}
		}

		// Restore original environment values
		if originalEnv == "" {
			os.Unsetenv("ENVIRONMENT")
		} else {
			os.Setenv("ENVIRONMENT", originalEnv)
		}

		if originalNamespace == "" {
			os.Unsetenv("NAMESPACE")
		} else {
			os.Setenv("NAMESPACE", originalNamespace)
		}

		// Restore original state to prevent test pollution
		K8sClient = originalK8sClient
		K8sClientMw = originalK8sClientMw
		K8sClientProjects = originalK8sClientProjects

		logger.Log("Cleaned up Permissions Handler test environment")
	})

	Context("Project Permissions Management", func() {
		Describe("ListProjectPermissions", func() {
			It("Should return list of service accounts and role bindings", func() {
				// Create test service account
				sa := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-sa",
						Namespace: "test-project",
						Labels: map[string]string{
							"app.kubernetes.io/managed-by": "ambient-code",
						},
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ServiceAccounts("test-project").Create(
					context.Background(), sa, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Create test role binding
				rb := &rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rb",
						Namespace: "test-project",
						Labels: map[string]string{
							"app": "ambient-permission",
						},
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:     "User",
							Name:     "test-user",
							APIGroup: "rbac.authorization.k8s.io",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Kind:     "ClusterRole",
						Name:     "ambient-project-view",
						APIGroup: "rbac.authorization.k8s.io",
					},
				}
				_, err = fakeClients.GetK8sClient().RbacV1().RoleBindings("test-project").Create(
					context.Background(), rb, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Test endpoint
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/permissions", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				ListProjectPermissions(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("items"))

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(len(items)).To(BeNumerically(">=", 1))

				logger.Log("Successfully listed project permissions")
			})

			It("Should list permissions with valid RBAC token", func() {
				// Arrange - Create namespace and token with RBAC permissions
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/permissions", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}

				// Use the same client instance that handlers use (from fakeClients)
				// Create namespace for the test using handlers' client (if it doesn't exist)
				_, err := fakeClients.GetK8sClient().CoreV1().Namespaces().Create(
					ginContext.Request.Context(),
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

				// Create test service account using handlers' client (if it doesn't exist)
				sa := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-sa",
						Namespace: "test-project",
						Labels: map[string]string{
							"app.kubernetes.io/managed-by": "ambient-code",
						},
					},
				}
				_, err = fakeClients.GetK8sClient().CoreV1().ServiceAccounts("test-project").Create(
					ginContext.Request.Context(), sa, metav1.CreateOptions{})
				// Ignore AlreadyExists errors - ServiceAccount may have been created in a previous test
				if err != nil && !errors.IsAlreadyExists(err) {
					Expect(err).NotTo(HaveOccurred())
				}

				// Use the same k8sUtils instance from BeforeEach
				// Create token using pre-created read-only Role
				token, saName, err := httpUtils.SetValidTestToken(
					k8sUtils,
					"test-project",
					[]string{"get", "list"}, // Not used, but kept for clarity
					"*",                     // All resources (not used, but kept for clarity)
					"",
					"test-read-only-role", // Use pre-created Role with read-only permissions
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
				Expect(saName).NotTo(BeEmpty())

				// Act
				ListProjectPermissions(ginContext)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("items"))

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				_, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")

				logger.Log("Successfully listed project permissions with RBAC token (SA: %s)", saName)
			})

			It("Should handle project not found gracefully", func() {
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/nonexistent/permissions", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "nonexistent"},
				}
				httpUtils.SetAuthHeader("test-token")

				ListProjectPermissions(ginContext)

				// Should still return 200 with empty lists
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("items"))

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(items).To(HaveLen(0))
			})

			It("Should require project name parameter", func() {
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects//permissions", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: ""},
				}
				httpUtils.SetAuthHeader("test-token")

				ListProjectPermissions(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Project name is required")
			})
		})

		Describe("AddProjectPermission", func() {
			It("Should create role binding successfully", func() {
				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": "test-user",
					"role":        "edit",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusCreated)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("message"))
				Expect(response["message"]).To(ContainSubstring("Permission added"))

				// Verify role binding was created with correct name based on handler naming pattern
				expectedRbName := "ambient-permission-edit-test-user-user"
				rb, err := fakeClients.GetK8sClient().RbacV1().RoleBindings("test-project").Get(
					context.Background(), expectedRbName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(rb.Name).To(Equal(expectedRbName))
				Expect(rb.RoleRef.Name).To(Equal("ambient-project-edit"))
				Expect(rb.Labels["app"]).To(Equal("ambient-permission"))

				// Verify the RoleBinding has the correct User subject
				Expect(rb.Subjects).To(HaveLen(1))
				Expect(rb.Subjects[0].Kind).To(Equal("User"))
				Expect(rb.Subjects[0].Name).To(Equal("test-user"))

				logger.Log("Successfully added project permission")
			})

			It("Should create role binding with valid RBAC token", func() {
				// Arrange - Create namespace and token with RBAC permissions
				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": "test-user",
					"role":        "view",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}

				// Use the same client instance that handlers use (from fakeClients)
				// Create namespace for the test using handlers' client (if it doesn't exist)
				_, err := fakeClients.GetK8sClient().CoreV1().Namespaces().Create(
					ginContext.Request.Context(),
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

				// Create a K8sTestUtils wrapper around the existing fake client
				// so we can use SetValidTestToken, but it will use the same client instance
				k8sUtils := &test_utils.K8sTestUtils{
					K8sClient:     fakeClients.GetK8sClient(),
					DynamicClient: fakeClients.GetDynamicClient(),
					Namespace:     "test-project",
				}

				// Create token using pre-created write Role (with create permissions)
				token, saName, err := httpUtils.SetValidTestToken(
					k8sUtils,
					"test-project",
					[]string{"create", "get", "list"}, // Not used, but kept for clarity
					"rolebindings",
					"",
					"test-rolebindings-write-role", // Use pre-created Role with write permissions
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
				Expect(saName).NotTo(BeEmpty())

				// Act
				AddProjectPermission(ginContext)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusCreated)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("message"))
				Expect(response["message"]).To(ContainSubstring("Permission added"))

				// Verify role binding was created using handlers' client
				expectedRbName := "ambient-permission-view-test-user-user"
				rb, err := fakeClients.GetK8sClient().RbacV1().RoleBindings("test-project").Get(
					ginContext.Request.Context(), expectedRbName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(rb.Name).To(Equal(expectedRbName))
				Expect(rb.RoleRef.Name).To(Equal("ambient-project-view"))

				logger.Log("Successfully added project permission with RBAC token (SA: %s)", saName)
			})

			It("Should reject permission creation with insufficient RBAC permissions", func() {
				// Arrange - Create a token without create permissions for rolebindings
				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": "test-user",
					"role":        "view",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}

				// Use the same client instance that handlers use (from fakeClients)
				// Create namespace for the test using handlers' client (if it doesn't exist)
				_, err := fakeClients.GetK8sClient().CoreV1().Namespaces().Create(
					ginContext.Request.Context(),
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

				// Create token with only read permissions (no create)
				// NOTE: We create the token BEFORE setting SSARAllowedFunc to deny creation,
				// because SetValidTestToken needs to create RoleBindings for test setup.
				// The SSAR denial will apply to handler operations, not test setup.
				_, _, err = httpUtils.SetValidTestToken(
					k8sUtils,
					"test-project",
					[]string{"get", "list"}, // Only read permissions
					"rolebindings",
					"",
					"test-rolebindings-read-role", // Use pre-created Role with read-only permissions
				)
				Expect(err).NotTo(HaveOccurred())

				// NOW configure SSAR to return false for create operations on rolebindings
				// This will affect handler operations, not the test setup above
				// Use the same k8sUtils instance from BeforeEach so the reactor uses it
				k8sUtils.SSARAllowedFunc = func(action k8stesting.Action) bool {
					// Check resource directly (works for both CreateAction and mockSSARAction)
					resource := action.GetResource()
					if resource.Resource == "rolebindings" && action.GetVerb() == "create" {
						return false // Deny create permission for handler operations
					}
					return true
				}

				// Act
				AddProjectPermission(ginContext)

				// Assert - Our fake client reactors simulate RBAC for rolebindings/namespaces using SSARAllowedFunc
				httpUtils.AssertHTTPStatus(http.StatusForbidden)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Insufficient permissions to grant permission",
				})
				logger.Log("Correctly rejected permission creation with insufficient RBAC permissions")
			})

			It("Should reject invalid role names", func() {
				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": "test-user",
					"role":        "invalid-role",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "role must be one of: admin, edit, view",
				})
			})

			It("Should reject empty username", func() {
				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": "",
					"role":        "view",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("SubjectName")
			})

			It("Should reject empty role", func() {
				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": "test-user",
					"role":        "",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Role")
			})

			It("Should handle duplicate permissions gracefully", func() {
				// Create existing service account
				sa := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-user",
						Namespace: "test-project",
						Labels: map[string]string{
							"app.kubernetes.io/managed-by": "ambient-code",
						},
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ServiceAccounts("test-project").Create(
					context.Background(), sa, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": "existing-user",
					"role":        "view",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				// Should still succeed (update existing)
				httpUtils.AssertHTTPStatus(http.StatusCreated)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["message"]).To(ContainSubstring("Permission added"))
			})

			It("Should require valid JSON body", func() {
				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", "invalid-json")
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should validate all supported role types", func() {
				validRoles := []string{"view", "edit", "admin"}

				for _, role := range validRoles {
					httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

					requestBody := map[string]interface{}{
						"subjectType": "user",
						"subjectName": "test-user-" + role,
						"role":        role,
					}

					ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
					ginContext.Params = gin.Params{
						{Key: "projectName", Value: "test-project"},
					}
					httpUtils.SetAuthHeader("test-token")

					AddProjectPermission(ginContext)

					httpUtils.AssertHTTPStatus(http.StatusCreated)
					logger.Log("Successfully validated role: %s", role)
				}
			})
		})

		Describe("RemoveProjectPermission", func() {
			BeforeEach(func() {
				// Create test role binding to remove
				rb := &rbacv1.RoleBinding{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ambient-permission-edit-test-user-user",
						Namespace: "test-project",
						Labels: map[string]string{
							"app": "ambient-permission",
						},
					},
					Subjects: []rbacv1.Subject{
						{
							Kind:     "User",
							Name:     "test-user",
							APIGroup: "rbac.authorization.k8s.io",
						},
					},
					RoleRef: rbacv1.RoleRef{
						Kind:     "ClusterRole",
						Name:     "edit",
						APIGroup: "rbac.authorization.k8s.io",
					},
				}
				_, err := fakeClients.GetK8sClient().RbacV1().RoleBindings("test-project").Create(
					context.Background(), rb, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should remove role binding successfully", func() {
				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/permissions/user/test-user", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "subjectType", Value: "user"},
					{Key: "subjectName", Value: "test-user"},
				}
				httpUtils.SetAuthHeader("test-token")

				RemoveProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusNoContent)

				// Verify role binding was deleted
				_, err := fakeClients.GetK8sClient().RbacV1().RoleBindings("test-project").Get(
					context.Background(), "ambient-permission-edit-test-user-user", metav1.GetOptions{})
				Expect(errors.IsNotFound(err)).To(BeTrue())

				logger.Log("Successfully removed project permission")
			})

			It("Should handle non-existent role binding gracefully", func() {
				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/permissions/user/nonexistent-user", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "subjectType", Value: "user"},
					{Key: "subjectName", Value: "nonexistent-user"},
				}
				httpUtils.SetAuthHeader("test-token")

				RemoveProjectPermission(ginContext)

				// Handler returns 204 NoContent even if no matching binding found
				httpUtils.AssertHTTPStatus(http.StatusNoContent)
			})

			It("Should require subjectName parameter", func() {
				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/permissions/user/", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "subjectType", Value: "user"},
					{Key: "subjectName", Value: ""},
				}
				httpUtils.SetAuthHeader("test-token")

				RemoveProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("subjectName is required")
			})

			It("Should require project name parameter", func() {
				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects//permissions/user/test-user", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: ""},
					{Key: "subjectType", Value: "user"},
					{Key: "subjectName", Value: "test-user"},
				}
				httpUtils.SetAuthHeader("test-token")

				RemoveProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Project is required in path /api/projects/:projectName or X-OpenShift-Project header")
			})
		})
	})

	Context("Input Validation", func() {
		It("Should reject userNames with invalid characters", func() {
			invalidUserNames := []string{
				"user@domain.com",       // @ not allowed in k8s resource names
				"user/name",             // / not allowed
				"user\\name",            // \ not allowed
				"user:name",             // : not allowed
				"user name",             // space not allowed
				"user.name",             // . not allowed in k8s resource names
				"User-Name",             // uppercase not allowed
				"user-name-",            // can't end with dash
				"-user-name",            // can't start with dash
				"user..name",            // consecutive dots not allowed
				strings.Repeat("a", 64), // too long (>63 chars)
			}

			for _, userName := range invalidUserNames {
				httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": userName,
					"role":        "view",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Invalid userName format. Must be a valid Kubernetes resource name.",
				})

				logger.Log("Correctly rejected invalid userName: %s", userName)
			}
		})

		It("Should accept valid userNames", func() {
			validUserNames := []string{
				"user-name",
				"user123",
				"123user",
				"a",                     // single character
				strings.Repeat("a", 63), // exactly 63 chars (max allowed)
			}

			for _, userName := range validUserNames {
				httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

				requestBody := map[string]interface{}{
					"subjectType": "user",
					"subjectName": userName,
					"role":        "view",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader("test-token")

				AddProjectPermission(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusCreated)

				logger.Log("Successfully accepted valid userName: %s", userName)
			}
		})
	})

	Context("Error Handling", func() {
		It("Should handle Kubernetes API errors gracefully", func() {
			// Test with a fake client that will return errors for create operations
			// This would require modifying the fake client to return errors,
			// which is more complex - for now we test the basic error paths

			ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/permissions", nil)
			ginContext.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			// Don't set auth header to trigger auth error path

			ListProjectPermissions(ginContext)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})

		It("Should handle missing auth token", func() {
			requestBody := map[string]interface{}{
				"subjectType": "user",
				"subjectName": "test-user",
				"role":        "view",
			}

			ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
			ginContext.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			// Don't set auth header

			AddProjectPermission(ginContext)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})
	})

	Context("Resource Label Verification", func() {
		It("Should create resources with proper ambient-code labels", func() {
			requestBody := map[string]interface{}{
				"subjectType": "user",
				"subjectName": "labeled-user",
				"role":        "view",
			}

			ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/permissions", requestBody)
			ginContext.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader("test-token")

			AddProjectPermission(ginContext)

			httpUtils.AssertHTTPStatus(http.StatusCreated)

			// Verify role binding has proper labels (matches current handler naming pattern)
			expectedRbName := "ambient-permission-view-labeled-user-user"
			rb, err := fakeClients.GetK8sClient().RbacV1().RoleBindings("test-project").Get(
				context.Background(), expectedRbName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(rb.Labels["app"]).To(Equal("ambient-permission"))

			logger.Log("Verified resources have proper ambient-code labels")
		})
	})
})
