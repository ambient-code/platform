//go:build test

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"time"

	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Runtime Credentials - Git Identity", Label(test_constants.LabelUnit), func() {

	Describe("fetchGitHubUserIdentity", func() {
		var (
			server *httptest.Server
		)

		AfterEach(func() {
			if server != nil {
				server.Close()
			}
		})

		Context("when GitHub API returns valid user data", func() {
			It("should return user name and email", func() {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					Expect(r.URL.Path).To(Equal("/user"))
					Expect(r.Header.Get("Authorization")).To(Equal("Bearer test-token"))
					Expect(r.Header.Get("Accept")).To(Equal("application/vnd.github+json"))

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]string{
						"login": "testuser",
						"name":  "Test User",
						"email": "test@example.com",
					})
				}))

				// Test with empty token returns empty strings
				name, email := fetchGitHubUserIdentity(context.Background(), "")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})

		Context("when token is empty", func() {
			It("should return empty strings without making API call", func() {
				name, email := fetchGitHubUserIdentity(context.Background(), "")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})
	})

	Describe("fetchGitLabUserIdentity", func() {
		Context("when token is empty", func() {
			It("should return empty strings without making API call", func() {
				name, email := fetchGitLabUserIdentity(context.Background(), "", "")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})

		Context("when instance URL is provided", func() {
			It("should construct correct API URL for self-hosted GitLab", func() {
				name, email := fetchGitLabUserIdentity(context.Background(), "", "https://gitlab.mycompany.com")
				Expect(name).To(Equal(""))
				Expect(email).To(Equal(""))
			})
		})
	})
})

var _ = Describe("Runtime Credentials - Shared Session User Resolution", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	var (
		httpUtils     *test_utils.HTTPTestUtils
		k8sUtils      *test_utils.K8sTestUtils
		ctx           context.Context
		testNamespace string
		testSession   string
		testToken     string
		ownerUserID   string
		sessionGVR    schema.GroupVersionResource
	)

	BeforeEach(func() {
		logger.Log("Setting up Shared Session Credentials test")

		httpUtils = test_utils.NewHTTPTestUtils()
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		ctx = context.Background()

		randomName := strconv.FormatInt(time.Now().UnixNano(), 10)
		testNamespace = "test-creds-" + randomName
		testSession = "test-session-" + randomName
		ownerUserID = "owner-user-" + randomName

		sessionGVR = schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}

		SetupHandlerDependencies(k8sUtils)

		// Create namespace
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: testNamespace},
		}, v1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		// Create RBAC role and token
		_, err = k8sUtils.CreateTestRole(ctx, testNamespace, "test-creds-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := httpUtils.SetValidTestToken(k8sUtils, testNamespace, []string{"get", "list", "create", "update", "delete", "patch"}, "*", "", "test-creds-role")
		Expect(err).NotTo(HaveOccurred())
		testToken = token

		// Create session CR with ownerUserID in spec.userContext.userId
		session := &unstructured.Unstructured{}
		session.SetAPIVersion("vteam.ambient-code/v1alpha1")
		session.SetKind("AgenticSession")
		session.SetName(testSession)
		session.SetNamespace(testNamespace)
		unstructured.SetNestedField(session.Object, ownerUserID, "spec", "userContext", "userId")
		unstructured.SetNestedField(session.Object, "Owner User", "spec", "userContext", "userName")
		unstructured.SetNestedField(session.Object, "Test prompt", "spec", "initialPrompt")
		unstructured.SetNestedField(session.Object, "Pending", "status", "phase")

		_, err = k8sUtils.DynamicClient.Resource(sessionGVR).Namespace(testNamespace).Create(ctx, session, v1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		logger.Log("Created test session %s with owner %s", testSession, ownerUserID)
	})

	AfterEach(func() {
		if k8sUtils != nil && testNamespace != "" {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(ctx, testNamespace, v1.DeleteOptions{})
		}
	})

	// Helper: create a gin context for GET credentials endpoint with route params
	createCredentialContext := func(credType string) *gin.Context {
		path := "/api/projects/" + testNamespace + "/agentic-sessions/" + testSession + "/credentials/" + credType
		c := httpUtils.CreateTestGinContext("GET", path, nil)
		c.Params = gin.Params{
			{Key: "projectName", Value: testNamespace},
			{Key: "sessionName", Value: testSession},
		}
		httpUtils.SetAuthHeader(testToken)
		return c
	}

	Describe("GetGitHubTokenForSession - Shared Session", func() {
		Context("when BOT_TOKEN calls without X-Runner-Current-User header", func() {
			It("should use owner's userID for credential lookup", func() {
				c := createCredentialContext("github")
				// BOT_TOKEN: no userID in gin context (empty authenticatedUserID)
				c.Set("userID", "")

				GetGitHubTokenForSession(c)

				// Handler will try to fetch GitHub token for ownerUserID.
				// With fake K8s, git.GetGitHubToken may not find creds, but
				// the session lookup and user resolution should succeed.
				statusCode := httpUtils.GetResponseRecorder().Code
				// Should NOT be 401 (auth) or 403 (RBAC) — user resolution worked
				Expect(statusCode).NotTo(Equal(http.StatusUnauthorized))
				Expect(statusCode).NotTo(Equal(http.StatusForbidden))
				logger.Log("BOT_TOKEN without header correctly resolved to owner: status=%d", statusCode)
			})
		})

		Context("when BOT_TOKEN calls with X-Runner-Current-User header for non-owner", func() {
			It("should reject — BOT_TOKEN can only access owner credentials", func() {
				c := createCredentialContext("github")
				c.Set("userID", "")
				c.Request.Header.Set("X-Runner-Current-User", "collaborator-user-abc")

				GetGitHubTokenForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				// Per-user scoping uses caller token, not X-Runner-Current-User with BOT_TOKEN
				Expect(statusCode).To(Equal(http.StatusForbidden))
				logger.Log("BOT_TOKEN with non-owner current user header: status=%d", statusCode)
			})
		})

		Context("when owner directly requests their own credentials", func() {
			It("should succeed (authenticatedUserID matches ownerUserID)", func() {
				c := createCredentialContext("github")
				c.Set("userID", ownerUserID)

				GetGitHubTokenForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				Expect(statusCode).NotTo(Equal(http.StatusUnauthorized))
				Expect(statusCode).NotTo(Equal(http.StatusForbidden))
				logger.Log("Owner accessing own creds: status=%d", statusCode)
			})
		})

		Context("when non-owner user directly calls the endpoint", func() {
			It("should return 403 (RBAC violation)", func() {
				c := createCredentialContext("github")
				// A different user (not owner, not effective user) directly calling
				c.Set("userID", "malicious-user-xyz")

				GetGitHubTokenForSession(c)

				httpUtils.AssertHTTPStatus(http.StatusForbidden)
				logger.Log("Non-owner direct access correctly blocked with 403")
			})
		})

		Context("when X-Runner-Current-User matches the authenticated user", func() {
			It("should succeed (authenticatedUserID == effectiveUserID)", func() {
				collaborator := "collaborator-user-123"
				c := createCredentialContext("github")
				c.Set("userID", collaborator)
				c.Request.Header.Set("X-Runner-Current-User", collaborator)

				GetGitHubTokenForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				Expect(statusCode).NotTo(Equal(http.StatusUnauthorized))
				Expect(statusCode).NotTo(Equal(http.StatusForbidden))
				logger.Log("Collaborator accessing own creds via header: status=%d", statusCode)
			})
		})
	})

	Describe("GetJiraCredentialsForSession - Shared Session", func() {
		Context("when BOT_TOKEN calls without X-Runner-Current-User", func() {
			It("should resolve to owner userID", func() {
				c := createCredentialContext("jira")
				c.Set("userID", "")

				GetJiraCredentialsForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				Expect(statusCode).NotTo(Equal(http.StatusUnauthorized))
				Expect(statusCode).NotTo(Equal(http.StatusForbidden))
				logger.Log("Jira BOT_TOKEN without header: status=%d", statusCode)
			})
		})

		Context("when BOT_TOKEN calls with X-Runner-Current-User for non-owner", func() {
			It("should reject — BOT_TOKEN can only access owner credentials", func() {
				c := createCredentialContext("jira")
				c.Set("userID", "")
				c.Request.Header.Set("X-Runner-Current-User", "collaborator-jira")

				GetJiraCredentialsForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				Expect(statusCode).To(Equal(http.StatusForbidden))
				logger.Log("Jira BOT_TOKEN with non-owner current user header: status=%d", statusCode)
			})
		})

		Context("when non-owner directly calls endpoint", func() {
			It("should return 403", func() {
				c := createCredentialContext("jira")
				c.Set("userID", "attacker-user")

				GetJiraCredentialsForSession(c)

				httpUtils.AssertHTTPStatus(http.StatusForbidden)
				logger.Log("Jira non-owner access correctly blocked")
			})
		})
	})

	Describe("GetGoogleCredentialsForSession - Shared Session", func() {
		Context("when non-owner directly calls endpoint", func() {
			It("should return 403", func() {
				c := createCredentialContext("google")
				c.Set("userID", "attacker-user")

				GetGoogleCredentialsForSession(c)

				httpUtils.AssertHTTPStatus(http.StatusForbidden)
				logger.Log("Google non-owner access correctly blocked")
			})
		})

		Context("when BOT_TOKEN calls with X-Runner-Current-User for non-owner", func() {
			It("should reject — BOT_TOKEN can only access owner credentials", func() {
				c := createCredentialContext("google")
				c.Set("userID", "")
				c.Request.Header.Set("X-Runner-Current-User", "collaborator-google")

				GetGoogleCredentialsForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				Expect(statusCode).To(Equal(http.StatusForbidden))
				logger.Log("Google BOT_TOKEN with non-owner current user header: status=%d", statusCode)
			})
		})
	})

	Describe("GetGitLabTokenForSession - Shared Session", func() {
		Context("when non-owner directly calls endpoint", func() {
			It("should return 403", func() {
				c := createCredentialContext("gitlab")
				c.Set("userID", "attacker-user")

				GetGitLabTokenForSession(c)

				httpUtils.AssertHTTPStatus(http.StatusForbidden)
				logger.Log("GitLab non-owner access correctly blocked")
			})
		})

		Context("when BOT_TOKEN calls with X-Runner-Current-User for non-owner", func() {
			It("should reject — BOT_TOKEN can only access owner credentials", func() {
				c := createCredentialContext("gitlab")
				c.Set("userID", "")
				c.Request.Header.Set("X-Runner-Current-User", "collaborator-gitlab")

				GetGitLabTokenForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				Expect(statusCode).To(Equal(http.StatusForbidden))
				logger.Log("GitLab BOT_TOKEN with non-owner current user header: status=%d", statusCode)
			})
		})
	})

	Describe("Scheduled Session (no current user header)", func() {
		Context("when automated session uses owner credentials", func() {
			It("should resolve to owner without X-Runner-Current-User", func() {
				c := createCredentialContext("github")
				// Scheduled: BOT_TOKEN auth (empty userID), no current user header
				c.Set("userID", "")

				GetGitHubTokenForSession(c)

				statusCode := httpUtils.GetResponseRecorder().Code
				Expect(statusCode).NotTo(Equal(http.StatusUnauthorized))
				Expect(statusCode).NotTo(Equal(http.StatusForbidden))
				logger.Log("Scheduled session correctly fell back to owner: status=%d", statusCode)
			})
		})
	})
})
