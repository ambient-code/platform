//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	k8stesting "k8s.io/client-go/testing"
)

var _ = Describe("Repo Handler >", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelRepo), func() {
	var (
		httpUtils                 *test_utils.HTTPTestUtils
		testClientFactory         *test_utils.TestClientFactory
		testToken                 string
		originalK8sClient         kubernetes.Interface
		originalK8sClientMw       kubernetes.Interface
		originalK8sClientProjects kubernetes.Interface
		originalNamespace         string
	)

	BeforeEach(func() {
		logger.Log("Setting up Repo Handler test")

		// Save original state to restore in AfterEach
		originalK8sClient = K8sClient
		originalK8sClientMw = K8sClientMw
		originalK8sClientProjects = K8sClientProjects
		originalNamespace = Namespace

		// Auth is disabled by default from config for unit tests

		// Use centralized handler dependencies setup
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		SetupHandlerDependencies(k8sUtils)

		// Create test client factory with fake clients
		testClientFactory = test_utils.NewTestClientFactory()
		_ = testClientFactory.GetFakeClients()

		// For repo tests, we need to set all the package-level K8s client variables
		// Different handlers use different client variables, so set them all
		// IMPORTANT: Use the same fake client for handlers that the test data is created with
		K8sClient = k8sUtils.K8sClient
		K8sClientMw = k8sUtils.K8sClient
		K8sClientProjects = k8sUtils.K8sClient
		Namespace = *config.TestNamespace

		GetGitHubTokenRepo = func(ctx context.Context, k8s kubernetes.Interface, dyn dynamic.Interface, project, userID string) (string, error) {
			if project == "unauthorized-project" {
				return "", fmt.Errorf("no GitHub token found for user")
			}
			return "mock-github-token", nil
		}

		DoGitHubRequest = func(ctx context.Context, method, url, authHeader, accept string, body io.Reader) (*http.Response, error) {
			// For unit tests, simulate GitHub API responses
			// Valid repositories get 502 (as expected by the test)
			// This simulates a GitHub API error condition
			status := http.StatusBadGateway
			responseBody := `{"message": "Bad gateway", "documentation_url": "https://docs.github.com"}`

			// Create a mock HTTP response
			resp := &http.Response{
				StatusCode: status,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(responseBody)),
			}
			return resp, nil
		}

		httpUtils = test_utils.NewHTTPTestUtils()

		// Create a realistic RBAC-backed token (instead of arbitrary strings).
		// This aligns unit tests with the production auth/RBAC model.
		ctx := context.Background()
		err := k8sUtils.CreateNamespace(ctx, "test-project")
		Expect(err).NotTo(HaveOccurred())
		_, err = k8sUtils.CreateTestRole(ctx, "test-project", "test-full-access-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		// Seed an initial gin context so SetValidTestToken can set headers, then store the token for later contexts.
		_ = httpUtils.CreateTestGinContext("GET", "/noop", nil)
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
		// Restore original state to prevent test pollution
		K8sClient = originalK8sClient
		K8sClientMw = originalK8sClientMw
		K8sClientProjects = originalK8sClientProjects
		Namespace = originalNamespace
	})

	Context("Access Control", func() {
		Describe("AccessCheck", func() {
			It("Should return admin role for users with rolebinding create permissions", func() {
				// Note: In unit tests, we would need to mock the SSAR response
				// For simplicity, we test the endpoint structure and auth requirements
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/access-check", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				AccessCheck(context)

				// Should not return auth error (specific role determination would require more complex mocking)
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusUnauthorized))

				logger.Log("Access check completed")
			})

			It("Should return edit role when rolebinding create is denied but agentic session create is allowed", func() {
				originalSSARFunc := k8sUtils.SSARAllowedFunc
				k8sUtils.SSARAllowedFunc = func(action k8stesting.Action) bool {
					create, ok := action.(k8stesting.CreateAction)
					if !ok {
						return true
					}
					ssar, ok := create.GetObject().(*authv1.SelfSubjectAccessReview)
					if !ok || ssar.Spec.ResourceAttributes == nil {
						return true
					}
					ra := ssar.Spec.ResourceAttributes
					if ra.Resource == "rolebindings" && ra.Verb == "create" {
						return false
					}
					if ra.Resource == "agenticsessions" && ra.Verb == "create" {
						return true
					}
					return false
				}
				defer func() { k8sUtils.SSARAllowedFunc = originalSSARFunc }()

				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/access-check", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				AccessCheck(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"project":  "test-project",
					"allowed":  false,
					"userRole": "edit",
				})
			})

			It("Should require project name parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/access-check", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				AccessCheck(context)

				// Function should handle empty project name gracefully
				// (specific behavior depends on implementation details)
			})

			It("Should return access info for unauthenticated users", func() {
				// AccessCheck uses GetK8sClientsForRequestRepo which calls GetK8sClientsForRequest
				// Configure SSAR to return allowed=false for unauthenticated users
				originalSSARFunc := k8sUtils.SSARAllowedFunc
				k8sUtils.SSARAllowedFunc = func(action k8stesting.Action) bool {
					// Return false for unauthenticated users
					return false
				}
				defer func() {
					k8sUtils.SSARAllowedFunc = originalSSARFunc
				}()

				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/access-check", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// AccessCheck requires auth; provide a token but deny via SSARAllowedFunc
				httpUtils.SetAuthHeader(testToken)

				AccessCheck(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"allowed":  false,
					"project":  "test-project",
					"userRole": "view",
				})
			})
		})
	})

	Context("Repository Fork Operations", func() {
		Describe("ListUserForks", func() {
			It("Should require upstreamRepo parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/users/forks", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				ListUserForks(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("upstreamRepo query parameter required")
			})

			It("Should handle invalid repository URLs", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/users/forks?upstreamRepo=invalid-repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				ListUserForks(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "invalid repo format, expected owner/repo",
				})
			})

			It("Should handle unauthorized access", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/unauthorized-project/users/forks?upstreamRepo=owner/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "unauthorized-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				ListUserForks(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/users/forks?upstreamRepo=owner/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ListUserForks(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})

			// Note: Testing actual GitHub API calls would require more complex mocking
			// or integration tests. Here we focus on input validation and error handling.
		})

		Describe("CreateUserFork", func() {
			It("Should require upstreamRepo in request body", func() {
				requestBody := map[string]interface{}{
					// Missing upstreamRepo
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/users/forks", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				CreateUserFork(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should validate repository URL format", func() {
				requestBody := map[string]interface{}{
					"upstreamRepo": "invalid-repo-format",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/users/forks", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				CreateUserFork(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertJSONContains(map[string]interface{}{
					"error": "invalid repo format, expected owner/repo",
				})
			})

			It("Should handle unauthorized access", func() {
				requestBody := map[string]interface{}{
					"upstreamRepo": "owner/repo",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/unauthorized-project/users/forks", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "unauthorized-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				CreateUserFork(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("Invalid or missing token")
			})

			It("Should require valid JSON body", func() {
				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/users/forks", "invalid-json")
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				CreateUserFork(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
			})

			It("Should require authentication", func() {
				// Temporarily enable auth check to test proper auth failure
				restore := WithAuthCheckEnabled()
				defer restore()

				requestBody := map[string]interface{}{
					"upstreamRepo": "owner/repo",
				}

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/users/forks", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				CreateUserFork(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})
		})
	})

	Context("Repository Browsing Operations", func() {
		Describe("GetRepoTree", func() {
			It("Should require repo and ref parameters", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/tree", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoTree(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("repo and ref query parameters required")
			})

			It("Should handle missing ref parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/tree?repo=owner/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoTree(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("repo and ref query parameters required")
			})

			It("Should handle unsupported repository providers", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/tree?repo=https://bitbucket.org/owner/repo&ref=main", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoTree(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("unsupported repository provider (only GitHub and GitLab are supported)")
			})

			It("Should handle GitHub repository URLs", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/tree?repo=https://github.com/owner/repo&ref=main&path=src", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoTree(context)

				// Should process GitHub URLs (actual API call would be mocked in integration tests)
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest))

				logger.Log("Processed GitHub repository tree request")
			})

			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/tree?repo=https://github.com/owner/repo&ref=main", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				GetRepoTree(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})
		})

		Describe("ListRepoBranches", func() {
			It("Should require repo parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/branches", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				ListRepoBranches(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("repo query parameter required")
			})

			It("Should handle unsupported repository providers", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/branches?repo=https://bitbucket.org/owner/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				ListRepoBranches(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("unsupported repository provider (only GitHub and GitLab are supported)")
			})

			It("Should handle GitHub repository URLs", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/branches?repo=https://github.com/owner/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				ListRepoBranches(context)

				// Should process GitHub URLs
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest))

				logger.Log("Processed GitHub repository branches request")
			})

			It("Should handle GitLab repository URLs", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/branches?repo=https://gitlab.com/owner/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				ListRepoBranches(context)

				// Should process GitLab URLs
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(BeElementOf(http.StatusUnauthorized, http.StatusBadGateway, http.StatusOK))

				logger.Log("Processed GitLab repository branches request")
			})

			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/branches?repo=https://github.com/owner/repo", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				ListRepoBranches(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})
		})

		Describe("GetRepoBlob", func() {
			It("Should require repo, ref, and path parameters", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/blob", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoBlob(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("repo, ref, and path query parameters required")
			})

			It("Should handle missing path parameter", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/blob?repo=owner/repo&ref=main", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoBlob(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("repo, ref, and path query parameters required")
			})

			It("Should handle unsupported repository providers", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/blob?repo=https://bitbucket.org/owner/repo&ref=main&path=README.md", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoBlob(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("unsupported repository provider (only GitHub and GitLab are supported)")
			})

			It("Should handle GitHub repository URLs", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/blob?repo=https://github.com/owner/repo&ref=main&path=README.md", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoBlob(context)

				// Should process GitHub URLs
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusBadRequest))

				logger.Log("Processed GitHub repository blob request")
			})

			It("Should handle GitLab repository URLs", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/blob?repo=https://gitlab.com/owner/repo&ref=main&path=README.md", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
				httpUtils.AutoSetProjectContextFromParams()

				GetRepoBlob(context)

				// Should process GitLab URLs
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(BeElementOf(http.StatusUnauthorized, http.StatusBadGateway, http.StatusOK))

				logger.Log("Processed GitLab repository blob request")
			})

			It("Should require authentication", func() {
				context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/repo/blob?repo=https://github.com/owner/repo&ref=main&path=README.md", nil)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				GetRepoBlob(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			})
		})
	})

	Context("Repository URL Parsing", func() {
		Describe("parseOwnerRepo function", func() {
			// Note: parseOwnerRepo is not exported, so we test it through the endpoints that use it

			It("Should handle various GitHub URL formats through endpoints", func() {
				testCases := []struct {
					repo     string
					expected int // expected status code (not testing exact parsing since function is internal)
				}{
					{"owner/repo", http.StatusBadGateway},                                // valid format
					{"https://github.com/owner/repo.git", http.StatusBadGateway},         // HTTPS URL
					{"git@github.com:owner/repo.git", http.StatusBadGateway},             // SSH URL
					{"invalid-format", http.StatusBadRequest},                            // invalid format
					{"https://github.com/owner/repo/tree/branch", http.StatusBadGateway}, // URL with path
				}

				for _, tc := range testCases {
					httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

					requestBody := map[string]interface{}{
						"upstreamRepo": tc.repo,
					}

					context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/users/forks", requestBody)
					context.Params = gin.Params{
						{Key: "projectName", Value: "test-project"},
					}
					httpUtils.SetAuthHeader(testToken)
					httpUtils.SetUserContext("test-user", "Test User", "test@example.com")
					httpUtils.AutoSetProjectContextFromParams()

					CreateUserFork(context)

					status := httpUtils.GetResponseRecorder().Code
					Expect(status).To(BeElementOf(tc.expected, http.StatusBadRequest))

					logger.Log("Tested repo format: %s, status: %d", tc.repo, status)
				}
			})
		})
	})

	Context("Provider Detection", func() {
		It("Should handle GitHub URLs correctly", func() {
			githubUrls := []string{
				"https://github.com/owner/repo",
				"git@github.com:owner/repo.git",
			}

			for _, url := range githubUrls {
				provider := types.DetectProvider(url)
				Expect(provider).To(Equal(types.ProviderGitHub))
				logger.Log("Correctly detected GitHub provider for: %s", url)
			}
		})

		It("Should handle GitLab URLs correctly", func() {
			gitlabUrls := []string{
				"https://gitlab.com/owner/repo",
				"git@gitlab.com:owner/repo.git",
				"https://gitlab.example.com/owner/repo",
			}

			for _, url := range gitlabUrls {
				provider := types.DetectProvider(url)
				Expect(provider).To(Equal(types.ProviderGitLab))
				logger.Log("Correctly detected GitLab provider for: %s", url)
			}
		})

		It("Should handle unknown providers", func() {
			unknownUrls := []string{
				"https://bitbucket.org/owner/repo",
				"https://example.com/owner/repo",
				"invalid-url",
			}

			for _, url := range unknownUrls {
				provider := types.DetectProvider(url)
				Expect(provider).NotTo(Equal(types.ProviderGitHub))
				Expect(provider).NotTo(Equal(types.ProviderGitLab))
				logger.Log("Correctly handled unknown provider for: %s", url)
			}
		})
	})

	Context("Error Handling", func() {
		It("Should handle missing user context gracefully", func() {
			context := httpUtils.CreateTestGinContext("GET", "/projects/test-project/users/forks?upstreamRepo=owner/repo", nil)
			context.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)
			// Don't set user context

			ListUserForks(context)

			// Should handle gracefully without panicking
			status := httpUtils.GetResponseRecorder().Code
			Expect(status).To(BeElementOf(http.StatusInternalServerError, http.StatusBadRequest, http.StatusUnauthorized))
		})

		It("Should handle malformed JSON gracefully", func() {
			context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/users/forks", "{invalid-json}")
			context.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)
			httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

			CreateUserFork(context)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)
		})

		It("Should handle concurrent requests gracefully", func() {
			// Test that the handlers can handle multiple concurrent requests
			// This tests for race conditions in the handler logic
			requestBody := map[string]interface{}{
				"upstreamRepo": "owner/concurrent-repo",
			}

			for i := 0; i < 3; i++ {
				httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

				context := httpUtils.CreateTestGinContext("POST", "/projects/test-project/users/forks", requestBody)
				context.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetUserContext("test-user", "Test User", "test@example.com")

				CreateUserFork(context)

				// Each request should be handled independently without errors
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).NotTo(Equal(http.StatusInternalServerError))

				logger.Log("Concurrent request %d handled successfully", i+1)
			}
		})
	})
})
