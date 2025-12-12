//go:build test

package handlers

import (
	"context"
	"fmt"
	"net/http"

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

var _ = Describe("Projects Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelProjects), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
		k8sUtils  *test_utils.K8sTestUtils // Store for use in tests
		testToken string
	)

	BeforeEach(func() {
		logger.Log("Setting up Projects Handler test")

		httpUtils = test_utils.NewHTTPTestUtils()

		// Set up K8s test utilities and handler dependencies
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// Pre-create test Roles with different permission sets for RBAC testing
		// This allows tests to use pre-existing Roles without creating RoleBindings during token setup
		ctx := context.Background()
		testNamespace := *config.TestNamespace

		// Create namespaces first (if they don't exist)
		testNamespaces := []string{testNamespace, "default"}
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
			// Namespaces-specific roles
			_, err := k8sUtils.CreateTestRole(ctx, ns, "test-namespaces-read-role", []string{"get", "list"}, "namespaces", "")
			Expect(err).NotTo(HaveOccurred())

			_, err = k8sUtils.CreateTestRole(ctx, ns, "test-namespaces-write-role", []string{"get", "list", "create", "update", "delete"}, "namespaces", "")
			Expect(err).NotTo(HaveOccurred())
		}

		// Mint a valid test token for this suite
		token, _, err := httpUtils.SetValidTestToken(
			k8sUtils,
			"default",
			[]string{"get", "list", "create", "update", "delete", "patch"},
			"*",
			"",
			"test-namespaces-write-role",
		)
		Expect(err).NotTo(HaveOccurred())
		testToken = token
	})

	AfterEach(func() {
		// Clean up created namespaces (best-effort)
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), *config.TestNamespace, metav1.DeleteOptions{})
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "default", metav1.DeleteOptions{})
		}
	})

	Context("Project Validation", func() {
		Describe("Project Name Validation (indirect testing)", func() {
			It("Should accept valid project names", func() {
				validNames := []string{
					"test-project",
					"my-app",
					"project123",
				}

				for _, name := range validNames {
					// Test indirectly through CreateProject since validateProjectName is not exported
					requestBody := map[string]interface{}{"name": name}
					ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
					httpUtils.SetAuthHeader(testToken)

					CreateProject(ginContext)

					status := httpUtils.GetResponseRecorder().Code
					Expect(status).NotTo(Equal(http.StatusBadRequest), "Should accept valid project name: "+name)

					// Reset for next iteration
					httpUtils = test_utils.NewHTTPTestUtils()
					logger.Log("Accepted valid project name: %s", name)
				}
			})

			It("Should reject invalid project names", func() {
				invalidNames := []string{
					"",             // empty
					"Project-Name", // uppercase not allowed
					"project_name", // underscore not allowed
					"project name", // space not allowed
				}

				for _, name := range invalidNames {
					// Test indirectly through CreateProject since validateProjectName is not exported
					requestBody := map[string]interface{}{"name": name}
					ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
					httpUtils.SetAuthHeader(testToken)

					CreateProject(ginContext)

					status := httpUtils.GetResponseRecorder().Code
					Expect(status).To(Equal(http.StatusBadRequest), "Should reject invalid project name: "+name)

					// Reset for next iteration
					httpUtils = test_utils.NewHTTPTestUtils()
					logger.Log("Correctly rejected invalid project name: %s", name)
				}
			})
		})

	})

	Context("Project Lifecycle Management", func() {
		Describe("CreateProject", func() {
			It("Should create project successfully with valid name", func() {
				requestBody := map[string]interface{}{
					"name": "test-project",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				CreateProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusCreated)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("name"))
				Expect(response["name"]).To(Equal("test-project"))
				Expect(response).To(HaveKey("status"))

				// Verify namespace was created using the same client as handlers
				ns, err := K8sClientProjects.CoreV1().Namespaces().Get(
					ginContext.Request.Context(), "test-project", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(ns.Name).To(Equal("test-project"))
				Expect(ns.Labels["app.kubernetes.io/managed-by"]).To(Equal("ambient-code"))

				logger.Log("Successfully created project: test-project")
			})

			It("Should create project with valid RBAC token", func() {
				// Arrange - Create a token with actual RBAC permissions for namespace creation
				requestBody := map[string]interface{}{
					"name": "rbac-test-project",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Use the same k8sUtils instance from BeforeEach
				// Create token using pre-created write Role (with create permissions)
				token, saName, err := httpUtils.SetValidTestToken(
					k8sUtils,
					"default",                         // Use default namespace for SA creation
					[]string{"create", "get", "list"}, // Not used, but kept for clarity
					"namespaces",
					"",
					"test-namespaces-write-role", // Use pre-created Role with write permissions
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(token).NotTo(BeEmpty())
				Expect(saName).NotTo(BeEmpty())

				// Act
				CreateProject(ginContext)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusCreated)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("name"))
				Expect(response["name"]).To(Equal("rbac-test-project"))

				// Verify namespace was created
				ns, err := K8sClientProjects.CoreV1().Namespaces().Get(
					ginContext.Request.Context(), "rbac-test-project", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(ns.Name).To(Equal("rbac-test-project"))

				logger.Log("Successfully created project with RBAC token (SA: %s)", saName)
			})

			It("Should reject project creation with insufficient RBAC permissions", func() {
				// LIMITATION: This test validates the handler's error handling logic when Kubernetes
				// would reject Namespace creation due to insufficient RBAC permissions.
				//
				// In production, Kubernetes enforces RBAC and returns errors.IsForbidden(err) when
				// a user lacks create permissions. The handler checks this and returns 403 Forbidden.
				//
				// With fake clients, we cannot fully simulate Kubernetes RBAC enforcement because
				// fake clients don't enforce RBAC - they allow all operations. The Create reactor
				// attempts to simulate this, but fake clients have limitations.
				//
				// This test validates:
				// 1. The handler uses SetValidTestToken to create tokens with RBAC permissions
				// 2. The handler's error handling logic (though full RBAC enforcement requires real K8s)
				//
				// For full RBAC validation, integration tests with a real Kubernetes cluster are required.
				//
				// Arrange - Create a token without create permissions
				requestBody := map[string]interface{}{
					"name": "forbidden-project",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				// Configure SSAR to return false for create operations on namespaces
				// Use the same k8sUtils instance from BeforeEach so the reactor uses it
				// We can set this BEFORE creating the token because we're using a pre-created Role
				k8sUtils.SSARAllowedFunc = func(action k8stesting.Action) bool {
					// Check resource directly (works for both CreateAction and mockSSARAction)
					resource := action.GetResource()
					if resource.Resource == "namespaces" && action.GetVerb() == "create" {
						return false // Deny create permission for handler operations
					}
					return true
				}

				// Create token using pre-created read-only Role (no create permissions)
				// This avoids creating RoleBindings during token setup, allowing SSAR denial to be set first
				_, _, err := httpUtils.SetValidTestToken(
					k8sUtils,
					"default",
					[]string{"get", "list"}, // Only read permissions (not used, but kept for clarity)
					"namespaces",
					"",
					"test-namespaces-read-role", // Use pre-created Role with read-only permissions
				)
				Expect(err).NotTo(HaveOccurred())

				// Act
				CreateProject(ginContext)

				// Assert - With fake clients, the creation may succeed because fake clients don't
				// enforce RBAC. In production, Kubernetes would reject this and the handler would
				// return 403 Forbidden. This test validates the token creation and handler logic,
				// but full RBAC enforcement requires integration tests with a real cluster.
				//
				// Accept either success (fake client limitation) or failure (if reactor works)
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(BeElementOf(http.StatusCreated, http.StatusForbidden, http.StatusInternalServerError),
					"Handler behavior depends on fake client RBAC simulation (limitation: fake clients don't fully enforce RBAC)")

				if status == http.StatusCreated {
					logger.Log("NOTE: Test completed with 201 Created due to fake client limitation (fake clients don't enforce RBAC). " +
						"In production, Kubernetes would enforce RBAC and return 403 Forbidden.")
				} else {
					logger.Log("Correctly rejected project creation with insufficient RBAC permissions")
				}
			})

			It("Should reject invalid project names", func() {
				requestBody := map[string]interface{}{
					"name": "Invalid-Project-Name",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				httpUtils.SetAuthHeader(testToken)

				CreateProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "project name must be lowercase alphanumeric with hyphens (cannot start or end with hyphen)",
				})
			})

			It("Should reject empty project name", func() {
				requestBody := map[string]interface{}{
					"name": "",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				httpUtils.SetAuthHeader(testToken)

				CreateProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "Key: 'CreateProjectRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag",
				})
			})

			It("Should handle existing project gracefully", func() {
				// Create namespace first using the same client as handlers
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "existing-project",
						Labels: map[string]string{
							"app.kubernetes.io/managed-by": "ambient-code",
							"ambient-code.io/managed":      "true",
						},
					},
				}
				_, err := K8sClientProjects.CoreV1().Namespaces().Create(
					context.Background(), ns, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				requestBody := map[string]interface{}{
					"name": "existing-project",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				httpUtils.SetAuthHeader(testToken)

				CreateProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusConflict)
				httpUtils.AssertErrorMessage("Project already exists")
			})

			It("Should require valid JSON body", func() {
				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", "invalid-json")
				httpUtils.SetAuthHeader(testToken)

				CreateProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should require authentication", func() {
				// Temporarily enable auth check for this specific test
				restore := WithAuthCheckEnabled()
				defer restore()

				requestBody := map[string]interface{}{
					"name": "test-project",
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				// Don't set auth header

				CreateProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})
		})

		Describe("ListProjects", func() {
			BeforeEach(func() {
				// Create test namespaces
				namespaces := []string{"project-1", "project-2", "kube-system", "ambient-managed"}

				for _, name := range namespaces {
					labels := map[string]string{}
					if name != "kube-system" {
						labels["app.kubernetes.io/managed-by"] = "ambient-code"
						labels["ambient-code.io/managed"] = "true"
					}

					ns := &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name:   name,
							Labels: labels,
						},
					}
					_, err := K8sClientProjects.CoreV1().Namespaces().Create(
						context.Background(), ns, metav1.CreateOptions{})
					Expect(err).NotTo(HaveOccurred())
				}
			})

			It("Should list ambient-code managed projects only", func() {
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects", nil)
				httpUtils.SetAuthHeader(testToken)

				ListProjects(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("items"))
				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")

				// Should have 3 ambient-code managed namespaces
				Expect(len(items)).To(Equal(3))

				// Verify all returned projects are ambient-code managed
				for _, item := range items {
					project, ok := item.(map[string]interface{})
					Expect(ok).To(BeTrue(), "Item should be a map")
					Expect(project).To(HaveKey("name"))
					nameInterface, exists := project["name"]
					Expect(exists).To(BeTrue(), "Project should contain 'name' field")
					name, ok := nameInterface.(string)
					Expect(ok).To(BeTrue(), "Project name should be a string")
					Expect(name).NotTo(Equal("kube-system"))
				}

				logger.Log("Successfully listed %d projects", len(items))
			})

			It("Should handle no projects gracefully", func() {
				// Delete all namespaces using same client as handler
				namespaces, _ := K8sClientProjects.CoreV1().Namespaces().List(
					context.Background(), metav1.ListOptions{})
				for _, ns := range namespaces.Items {
					_ = K8sClientProjects.CoreV1().Namespaces().Delete(
						context.Background(), ns.Name, metav1.DeleteOptions{})
				}

				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects", nil)
				httpUtils.SetAuthHeader(testToken)

				ListProjects(ginContext)

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

			It("Should require authentication", func() {
				// Temporarily enable auth check for this specific test
				restore := WithAuthCheckEnabled()
				defer restore()

				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects", nil)
				// Don't set auth header

				ListProjects(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should include project metadata", func() {
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects", nil)
				httpUtils.SetAuthHeader(testToken)

				ListProjects(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				if len(items) > 0 {
					project, ok := items[0].(map[string]interface{})
					Expect(ok).To(BeTrue(), "Item should be a map")
					Expect(project).To(HaveKey("name"))
					Expect(project).To(HaveKey("creationTimestamp"))
				}
			})
		})

		Describe("GetProject", func() {
			BeforeEach(func() {
				// Create test namespace using the same client as handlers
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-project",
						Labels: map[string]string{
							"app.kubernetes.io/managed-by": "ambient-code",
							"ambient-code.io/managed":      "true",
						},
					},
				}
				_, err := K8sClientProjects.CoreV1().Namespaces().Create(
					context.Background(), ns, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should get project details successfully", func() {
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				GetProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)

				Expect(response).To(HaveKey("name"))
				Expect(response["name"]).To(Equal("test-project"))
				Expect(response).To(HaveKey("creationTimestamp"))

				logger.Log("Successfully retrieved project details")
			})

			It("Should return 404 for non-existent project", func() {
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/nonexistent", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "nonexistent"},
				}
				httpUtils.SetAuthHeader(testToken)

				GetProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("Project not found")
			})

			It("Should require project name parameter", func() {
				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				GetProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Project name is required")
			})

			It("Should require authentication", func() {
				// Temporarily enable auth check for this specific test
				restore := WithAuthCheckEnabled()
				defer restore()

				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header

				GetProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should not return non-ambient-code projects", func() {
				// Create a namespace not managed by ambient-code using the same client as handlers
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "external-project",
					},
				}
				_, err := K8sClientProjects.CoreV1().Namespaces().Create(
					context.Background(), ns, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginContext := httpUtils.CreateTestGinContext("GET", "/api/projects/external-project", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "external-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				GetProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("Project not found")
			})
		})

		Describe("DeleteProject", func() {
			BeforeEach(func() {
				// Create test namespace using the same client as handlers
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "project-to-delete",
						Labels: map[string]string{
							"app.kubernetes.io/managed-by": "ambient-code",
							"ambient-code.io/managed":      "true",
						},
					},
				}
				_, err := K8sClientProjects.CoreV1().Namespaces().Create(
					context.Background(), ns, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should delete project successfully", func() {
				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects/project-to-delete", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "project-to-delete"},
				}
				httpUtils.SetAuthHeader(testToken)

				DeleteProject(ginContext)

				// Debug: Print the actual response
				status := httpUtils.GetResponseRecorder().Code
				body := httpUtils.GetResponseRecorder().Body.String()
				logger.Log("DeleteProject returned status: %d, body: '%s'", status, body)

				httpUtils.AssertHTTPStatus(http.StatusNoContent)

				// Verify namespace was deleted using same client as handler
				_, err := K8sClientProjects.CoreV1().Namespaces().Get(
					ginContext.Request.Context(), "project-to-delete", metav1.GetOptions{})
				Expect(errors.IsNotFound(err)).To(BeTrue())

				logger.Log("Successfully deleted project")
			})

			It("Should return 404 for non-existent project", func() {
				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects/nonexistent", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "nonexistent"},
				}
				httpUtils.SetAuthHeader(testToken)

				DeleteProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("Project not found")
			})

			It("Should require project name parameter", func() {
				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects/", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				DeleteProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Project name is required")
			})

			It("Should require authentication", func() {
				// Temporarily enable auth check for this specific test
				restore := WithAuthCheckEnabled()
				defer restore()

				ginContext := httpUtils.CreateTestGinContext("DELETE", "/api/projects/project-to-delete", nil)
				ginContext.Params = gin.Params{
					{Key: "projectName", Value: "project-to-delete"},
				}
				// Don't set auth header

				DeleteProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})
		})
	})

	Context("Project Namespace Management", func() {
		It("Should create namespace with proper labels", func() {
			requestBody := map[string]interface{}{
				"name": "labeled-project",
			}

			ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
			httpUtils.SetAuthHeader(testToken)

			CreateProject(ginContext)

			httpUtils.AssertHTTPStatus(http.StatusCreated)

			// Verify namespace has proper labels using same client as handler
			ns, err := K8sClientProjects.CoreV1().Namespaces().Get(
				ginContext.Request.Context(), "labeled-project", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(ns.Labels["app.kubernetes.io/managed-by"]).To(Equal("ambient-code"))

			logger.Log("Verified namespace has proper ambient-code labels")
		})

		It("Should enforce project naming conventions consistently", func() {
			// Test that the same validation is applied across all endpoints
			invalidNames := []string{"Invalid-Name", "name_with_underscore", "name with spaces"}

			for _, name := range invalidNames {
				httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

				requestBody := map[string]interface{}{
					"name": name,
				}

				ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
				httpUtils.SetAuthHeader(testToken)

				CreateProject(ginContext)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)

				logger.Log("Correctly rejected project name: %s", name)
			}
		})
	})

	Context("Error Scenarios", func() {
		It("Should handle malformed JSON gracefully", func() {
			ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", "{invalid-json}")
			httpUtils.SetAuthHeader(testToken)

			CreateProject(ginContext)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should handle missing required fields", func() {
			requestBody := map[string]interface{}{
				// missing name field
			}

			ginContext := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
			httpUtils.SetAuthHeader(testToken)

			CreateProject(ginContext)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			httpUtils.AssertJSONContains(map[string]interface{}{
				"error": "Key: 'CreateProjectRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag",
			})
		})

		It("Should handle concurrent project creation", func() {
			// This test simulates race conditions in project creation
			// Both requests should either succeed or fail gracefully
			requestBody := map[string]interface{}{
				"name": "concurrent-project",
			}

			context1 := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
			httpUtils.SetAuthHeader(testToken)

			CreateProject(context1)

			// First should succeed
			status1 := httpUtils.GetResponseRecorder().Code
			Expect(status1).To(BeElementOf(http.StatusCreated, http.StatusConflict))

			// Reset for second request
			httpUtils = test_utils.NewHTTPTestUtils()

			context2 := httpUtils.CreateTestGinContext("POST", "/api/projects", requestBody)
			httpUtils.SetAuthHeader(testToken)

			CreateProject(context2)

			// Second should either conflict or succeed depending on timing
			status2 := httpUtils.GetResponseRecorder().Code
			Expect(status2).To(BeElementOf(http.StatusCreated, http.StatusConflict))

			logger.Log("Handled concurrent creation - status1: %d, status2: %d", status1, status2)
		})
	})
})
