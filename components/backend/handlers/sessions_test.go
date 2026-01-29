//go:build test

package handlers

import (
	"ambient-code-backend/tests/config"
	test_constants "ambient-code-backend/tests/constants"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var _ = Describe("Sessions Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelSessions), func() {
	var (
		httpUtils     *test_utils.HTTPTestUtils
		k8sUtils      *test_utils.K8sTestUtils
		ctx           context.Context
		testNamespace string
		sessionGVR    schema.GroupVersionResource
		randomName    string
		testSession   string
		testToken     string
	)

	BeforeEach(func() {
		logger.Log("Setting up Sessions Handler test")

		httpUtils = test_utils.NewHTTPTestUtils()
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		ctx = context.Background()
		randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
		testNamespace = "test-project-" + randomName
		testSession = "test-session-" + randomName

		// Define AgenticSession GVR
		sessionGVR = schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}

		// Set up package-level variables for handlers
		SetupHandlerDependencies(k8sUtils)

		// Create namespace + role needed for this test suite, then mint a valid test token
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: testNamespace},
		}, v1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		// Broad test role (CRDs + common core resources) for this namespace
		_, err = k8sUtils.CreateTestRole(ctx, testNamespace, "test-full-access-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := httpUtils.SetValidTestToken(
			k8sUtils,
			testNamespace,
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
		if k8sUtils != nil && testNamespace != "" {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(ctx, testNamespace, v1.DeleteOptions{})
		}
	})

	Describe("ListSessions", func() {
		Context("When project has no sessions", func() {
			It("Should return empty list", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/"+testNamespace+"/agentic-sessions", nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				ListSessions(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("items"))

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(items).To(HaveLen(0), "Should return empty list when no sessions exist")

				logger.Log("Empty session list returned successfully")
			})
		})

		Context("When project has sessions", func() {
			BeforeEach(func() {
				// Create test sessions directly using DynamicClient (avoid CreateCustomResource which has Gomega issues)
				session1 := createTestSession("session-1-"+randomName, testNamespace, k8sUtils)
				session2 := createTestSession("session-2-"+randomName, testNamespace, k8sUtils)
				logger.Log("Created test sessions: session-1 (uid=%s), session-2 (uid=%s)", session1.GetUID(), session2.GetUID())

				// Verify sessions exist in the client being used by handlers
				gvr := schema.GroupVersionResource{
					Group:    "vteam.ambient-code",
					Version:  "v1alpha1",
					Resource: "agenticsessions",
				}

				list, err := DynamicClientProjects.Resource(gvr).Namespace(testNamespace).List(context.Background(), v1.ListOptions{})
				if err != nil {
					logger.Log("Error listing sessions in handler client: %v", err)
				} else {
					logger.Log("Handler client sees %d sessions in namespace %s", len(list.Items), testNamespace)
					for _, item := range list.Items {
						logger.Log("  - %s (uid=%s)", item.GetName(), item.GetUID())
					}
				}
			})

			It("Should return list of sessions", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/"+testNamespace+"/agentic-sessions", nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				ListSessions(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("items"))

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(items).To(HaveLen(2), "Should return all sessions in project")

				logger.Log("Session list with %d items returned successfully", len(items))
			})

			It("Should support pagination", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/"+testNamespace+"/agentic-sessions?offset=0&limit=1", nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				ListSessions(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("items"))
				Expect(response).To(HaveKey("hasMore"))
				Expect(response).To(HaveKey("totalCount"))

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(items).To(HaveLen(1), "Should return only one item due to limit")

				hasMoreInterface, exists := response["hasMore"]
				Expect(exists).To(BeTrue(), "Response should contain 'hasMore' field")
				hasMore, ok := hasMoreInterface.(bool)
				Expect(ok).To(BeTrue(), "HasMore should be a boolean")
				Expect(hasMore).To(BeTrue(), "Should indicate more items available")

				logger.Log("Paginated session list returned successfully")
			})

			It("Should support search filtering", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/"+testNamespace+"/agentic-sessions?search=session-1", nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				ListSessions(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(items).To(HaveLen(1), "Should filter sessions by search term")

				logger.Log("Filtered session list returned successfully")
			})
		})

		Context("When accessing a different project", func() {
			It("Should return empty list for unauthorized project (auth disabled in tests)", func() {
				// Arrange
				context := httpUtils.CreateTestGinContext("GET", "/api/projects/unauthorized-project/agentic-sessions", nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext("unauthorized-project")

				// Act
				ListSessions(context)

				// Assert - request is allowed in tests, but there are no sessions in this namespace
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("items"))

				itemsInterface, exists := response["items"]
				Expect(exists).To(BeTrue(), "Response should contain 'items' field")
				items, ok := itemsInterface.([]interface{})
				Expect(ok).To(BeTrue(), "Items should be an array")
				Expect(items).To(HaveLen(0), "Should return empty list for namespace without sessions")

				logger.Log("Unauthorized project returned empty list")
			})
		})
	})

	Describe("CreateSession", func() {
		Context("When creating a valid session", func() {
			It("Should create session with required fields", func() {
				// Arrange
				sessionRequest := map[string]interface{}{
					"initialPrompt": "Test prompt",
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "https://github.com/test/repo.git",
							"branch": "main",
						},
					},
					"interactive": false,
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/projects/"+testNamespace+"/agentic-sessions", sessionRequest)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				CreateSession(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusCreated)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("name"))
				Expect(response).To(HaveKey("uid"))

				sessionNameInterface, exists := response["name"]
				Expect(exists).To(BeTrue(), "Response should contain 'name' field")
				sessionName, ok := sessionNameInterface.(string)
				Expect(ok).To(BeTrue(), "Session name should be a string")
				Expect(sessionName).NotTo(BeEmpty(), "Session name should not be empty")

				logger.Log("Session created successfully: %s", sessionName)
			})

			It("Should generate unique session names", func() {
				sessionRequest := map[string]interface{}{
					"initialPrompt": "Test prompt",
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "https://github.com/test/repo.git",
							"branch": "main",
						},
					},
				}

				sessionNames := make([]string, 0)

				// Create multiple sessions with delays to ensure unique timestamps
				for i := 0; i < 3; i++ {
					// Add a delay to ensure unique timestamps (Unix() has 1-second precision)
					if i > 0 {
						time.Sleep(1001 * time.Millisecond) // Slightly over 1 second to ensure different Unix timestamps
					}

					context := httpUtils.CreateTestGinContext("POST", "/api/projects/"+testNamespace+"/agentic-sessions", sessionRequest)
					httpUtils.SetAuthHeader(testToken)
					httpUtils.SetProjectContext(testNamespace)

					CreateSession(context)

					httpUtils.AssertHTTPStatus(http.StatusCreated)

					var response map[string]interface{}
					httpUtils.GetResponseJSON(&response)
					sessionNameInterface, exists := response["name"]
					Expect(exists).To(BeTrue(), "Response should contain 'name' field")
					sessionName, ok := sessionNameInterface.(string)
					Expect(ok).To(BeTrue(), "Session name should be a string")
					sessionNames = append(sessionNames, sessionName)

					// Reset for next iteration
					httpUtils = test_utils.NewHTTPTestUtils()
				}

				// Assert all names are unique
				nameSet := make(map[string]bool)
				for _, name := range sessionNames {
					Expect(nameSet[name]).To(BeFalse(), fmt.Sprintf("Session name '%s' should be unique, but was generated multiple times", name))
					nameSet[name] = true
				}

				logger.Log("Generated %d unique session names: %v", len(sessionNames), sessionNames)
			})
		})

		Context("When creating session with edge case data", func() {
			It("Should handle empty initial prompt", func() {
				// Arrange
				sessionRequest := map[string]interface{}{
					"initialPrompt": "",
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "https://github.com/test/repo.git",
							"branch": "main",
						},
					},
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/projects/"+testNamespace+"/agentic-sessions", sessionRequest)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				CreateSession(context)

				// Assert - handler currently accepts empty initial prompt
				httpUtils.AssertHTTPStatus(http.StatusCreated)
			})

			It("Should handle sessions with no repositories", func() {
				// Arrange
				sessionRequest := map[string]interface{}{
					"initialPrompt": "Test prompt",
					"repos":         []interface{}{},
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/projects/"+testNamespace+"/agentic-sessions", sessionRequest)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				CreateSession(context)

				// Assert - handler currently accepts empty repos
				httpUtils.AssertHTTPStatus(http.StatusCreated)
			})

			It("Should handle invalid repository URLs", func() {
				// Arrange
				sessionRequest := map[string]interface{}{
					"initialPrompt": "Test prompt",
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "invalid-url",
							"branch": "main",
						},
					},
				}

				context := httpUtils.CreateTestGinContext("POST", "/api/projects/"+testNamespace+"/agentic-sessions", sessionRequest)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)

				// Act
				CreateSession(context)

				// Assert - handler currently accepts invalid URLs (validation at runtime)
				httpUtils.AssertHTTPStatus(http.StatusCreated)
			})
		})
	})

	Describe("GetSession", func() {
		var sessionName string

		BeforeEach(func() {
			sessionName = testSession
			createTestSession(sessionName, testNamespace, k8sUtils)
		})

		Context("When session exists", func() {
			It("Should return session details", func() {
				// Arrange
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", testNamespace, sessionName)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: sessionName},
				}

				// Act
				GetSession(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response types.AgenticSession
				httpUtils.GetResponseJSON(&response)
				Expect(response.Metadata).NotTo(BeNil(), "Response metadata should not be nil")

				nameValue, exists := response.Metadata["name"]
				Expect(exists).To(BeTrue(), "Response metadata should contain 'name'")
				Expect(nameValue).To(Equal(sessionName))

				namespaceValue, exists := response.Metadata["namespace"]
				Expect(exists).To(BeTrue(), "Response metadata should contain 'namespace'")
				Expect(namespaceValue).To(Equal(testNamespace))

				logger.Log("Session details retrieved successfully: %s", sessionName)
			})
		})

		Context("When session does not exist", func() {
			It("Should return 404 Not Found", func() {
				// Arrange
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/non-existent-session", testNamespace)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "non-existent-session"},
				}

				// Act
				GetSession(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("Session not found")
			})
		})
	})

	Describe("DeleteSession", func() {
		var sessionName string

		BeforeEach(func() {
			sessionName = "test-session-to-delete"
			createTestSession(sessionName, testNamespace, k8sUtils)
		})

		Context("When deleting existing session", func() {
			It("Should delete session successfully", func() {
				// Arrange
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s", testNamespace, sessionName)
				context := httpUtils.CreateTestGinContext("DELETE", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: sessionName},
				}

				// Act
				DeleteSession(context)

				// Assert - handler currently returns 200 due to using c.Status() instead of c.AbortWithStatus()
				httpUtils.AssertHTTPStatus(http.StatusOK)

				// Verify session was deleted
				k8sUtils.AssertResourceNotExists(ctx, sessionGVR, testNamespace, sessionName)

				logger.Log("Session deleted successfully: %s", sessionName)
			})
		})

		Context("When deleting non-existent session", func() {
			It("Should return 404 Not Found", func() {
				// Arrange
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/non-existent-session", testNamespace)
				context := httpUtils.CreateTestGinContext("DELETE", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "non-existent-session"},
				}

				// Act
				DeleteSession(context)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusNotFound)
			})
		})
	})

	// AutoPush functionality tests
	Context("AutoPush Field Parsing", func() {
		var (
			testClientFactory         *test_utils.TestClientFactory
			fakeClients               *test_utils.FakeClientSet
			originalK8sClient         kubernetes.Interface
			originalK8sClientMw       kubernetes.Interface
			originalK8sClientProjects kubernetes.Interface
			originalDynamicClient     dynamic.Interface
		)

		BeforeEach(func() {
			logger.Log("Setting up AutoPush test")

			// Save original state
			originalK8sClient = K8sClient
			originalK8sClientMw = K8sClientMw
			originalK8sClientProjects = K8sClientProjects
			originalDynamicClient = DynamicClient

			// Create fake clients
			testClientFactory = test_utils.NewTestClientFactory()
			fakeClients = testClientFactory.GetFakeClients()

			DynamicClient = fakeClients.GetDynamicClient()
			K8sClientProjects = fakeClients.GetK8sClient()
			K8sClient = fakeClients.GetK8sClient()
			K8sClientMw = fakeClients.GetK8sClient()
		})

		AfterEach(func() {
			// Restore original state
			K8sClient = originalK8sClient
			K8sClientMw = originalK8sClientMw
			K8sClientProjects = originalK8sClientProjects
			DynamicClient = originalDynamicClient
		})

		Describe("parseSpec", func() {
			It("Should parse autoPush=true from repo", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo.git",
							"branch":   "main",
							"autoPush": true,
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(1))
				Expect(parsed.Repos[0].URL).To(Equal("https://github.com/owner/repo.git"))
				Expect(parsed.Repos[0].Branch).NotTo(BeNil())
				Expect(*parsed.Repos[0].Branch).To(Equal("main"))
				Expect(parsed.Repos[0].AutoPush).NotTo(BeNil())
				Expect(*parsed.Repos[0].AutoPush).To(BeTrue())
			})

			It("Should parse autoPush=false from repo", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo.git",
							"branch":   "develop",
							"autoPush": false,
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(1))
				Expect(parsed.Repos[0].AutoPush).NotTo(BeNil())
				Expect(*parsed.Repos[0].AutoPush).To(BeFalse())
			})

			It("Should handle missing autoPush field", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "https://github.com/owner/repo.git",
							"branch": "main",
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(1))
				Expect(parsed.Repos[0].AutoPush).To(BeNil())
			})

			It("Should handle multiple repos with mixed autoPush settings", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo1.git",
							"autoPush": true,
						},
						map[string]interface{}{
							"url":      "https://github.com/owner/repo2.git",
							"autoPush": false,
						},
						map[string]interface{}{
							"url": "https://github.com/owner/repo3.git",
							// No autoPush field
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(3))

				// First repo: autoPush=true
				Expect(parsed.Repos[0].AutoPush).NotTo(BeNil())
				Expect(*parsed.Repos[0].AutoPush).To(BeTrue())

				// Second repo: autoPush=false
				Expect(parsed.Repos[1].AutoPush).NotTo(BeNil())
				Expect(*parsed.Repos[1].AutoPush).To(BeFalse())

				// Third repo: no autoPush field
				Expect(parsed.Repos[2].AutoPush).To(BeNil())
			})
		})

		Describe("ExtractSessionContext", func() {
			It("Should extract autoPush from repo spec", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo.git",
							"branch":   "main",
							"autoPush": true,
						},
					},
				}

				ctx := ExtractSessionContext(spec)
				Expect(ctx.Repos).To(HaveLen(1))
				Expect(ctx.Repos[0]["url"]).To(Equal("https://github.com/owner/repo.git"))
				Expect(ctx.Repos[0]["autoPush"]).To(BeTrue())
			})

			It("Should handle repos without autoPush field", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":    "https://github.com/owner/repo.git",
							"branch": "main",
						},
					},
				}

				ctx := ExtractSessionContext(spec)
				Expect(ctx.Repos).To(HaveLen(1))
				Expect(ctx.Repos[0]["autoPush"]).To(BeNil())
			})
		})

		Describe("BoolPtr helper", func() {
			It("Should create pointer to true", func() {
				ptr := types.BoolPtr(true)
				Expect(ptr).NotTo(BeNil())
				Expect(*ptr).To(BeTrue())
			})

			It("Should create pointer to false", func() {
				ptr := types.BoolPtr(false)
				Expect(ptr).NotTo(BeNil())
				Expect(*ptr).To(BeFalse())
			})
		})

		Describe("Error handling", func() {
			It("Should handle invalid autoPush type gracefully", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo.git",
							"autoPush": "invalid-string", // Should be bool
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(1))
				// Should skip invalid type and leave AutoPush as nil
				Expect(parsed.Repos[0].AutoPush).To(BeNil())
			})

			It("Should handle autoPush in malformed repo gracefully", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						"invalid-string-instead-of-map",
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(BeEmpty())
			})
		})

		Describe("Edge Cases in AddRepo Handler", func() {
			It("Should handle autoPush as null value", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo.git",
							"branch":   "main",
							"autoPush": nil,
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(1))
				// nil autoPush should be treated as not provided
				Expect(parsed.Repos[0].AutoPush).To(BeNil())
			})

			It("Should skip autoPush with invalid type (string)", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo.git",
							"branch":   "main",
							"autoPush": "true", // String instead of bool
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(1))
				// Invalid type should be skipped, leaving AutoPush as nil
				Expect(parsed.Repos[0].AutoPush).To(BeNil())
			})

			It("Should skip autoPush with invalid type (number)", func() {
				spec := map[string]interface{}{
					"repos": []interface{}{
						map[string]interface{}{
							"url":      "https://github.com/owner/repo.git",
							"branch":   "main",
							"autoPush": 1, // Number instead of bool
						},
					},
				}

				parsed := parseSpec(spec)
				Expect(parsed.Repos).To(HaveLen(1))
				// Invalid type should be skipped, leaving AutoPush as nil
				Expect(parsed.Repos[0].AutoPush).To(BeNil())
			})
		})
	})
})

// Helper functions

func createTestSession(name, namespace string, k8sUtils *test_utils.K8sTestUtils) *unstructured.Unstructured {
	session := &unstructured.Unstructured{}
	session.SetAPIVersion("vteam.ambient-code/v1alpha1")
	session.SetKind("AgenticSession")
	session.SetName(name)
	session.SetNamespace(namespace)

	// Set labels using unstructured helpers
	labels := map[string]string{
		"test-framework": "ambient-code-backend",
	}
	session.SetLabels(labels)

	// Set spec fields using unstructured nested field helpers
	unstructured.SetNestedField(session.Object, "Test prompt for "+name, "spec", "initialPrompt")

	// Set repos array - match the structure expected by the production handler
	repos := []interface{}{
		map[string]interface{}{
			"url":    "https://github.com/test/repo.git",
			"branch": "main",
		},
	}
	unstructured.SetNestedSlice(session.Object, repos, "spec", "repos")

	// Set interactive field properly for deep copy compatibility
	unstructured.SetNestedField(session.Object, false, "spec", "interactive")

	// Set status
	unstructured.SetNestedField(session.Object, "Pending", "status", "phase")

	sessionGVR := schema.GroupVersionResource{
		Group:    "vteam.ambient-code",
		Version:  "v1alpha1",
		Resource: "agenticsessions",
	}

	// Create directly using DynamicClient instead of CreateCustomResource to avoid Gomega issues
	created, err := k8sUtils.DynamicClient.Resource(sessionGVR).Namespace(namespace).Create(context.Background(), session, v1.CreateOptions{})
	if err != nil {
		// Use Ginkgo's Fail() instead of panic for proper test failure reporting
		Fail(fmt.Sprintf("Failed to create test session %s: %v", name, err))
		return nil // Will not be reached, but satisfies return type
	}
	return created
}
