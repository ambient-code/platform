//go:build test

package handlers

import (
	"context"
	"fmt"
	"net/http"
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

var _ = Describe("Session Sub-Resource Handlers", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelSessions), func() {
	var (
		httpUtils     *test_utils.HTTPTestUtils
		k8sUtils      *test_utils.K8sTestUtils
		ctx           context.Context
		testNamespace string
		sessionGVR    schema.GroupVersionResource
		randomName    string
		testToken     string
	)

	BeforeEach(func() {
		logger.Log("Setting up Session Sub-Resource Handler test")

		httpUtils = test_utils.NewHTTPTestUtils()
		k8sUtils = test_utils.NewK8sTestUtils(false, *config.TestNamespace)
		ctx = context.Background()
		randomName = strconv.FormatInt(time.Now().UnixNano(), 10)
		testNamespace = "test-sub-" + randomName

		sessionGVR = schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}

		SetupHandlerDependencies(k8sUtils)

		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: v1.ObjectMeta{Name: testNamespace},
		}, v1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

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
		if k8sUtils != nil && testNamespace != "" {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(ctx, testNamespace, v1.DeleteOptions{})
		}
	})

	Describe("GetSessionMetrics", func() {
		Context("When session exists with status fields", func() {
			BeforeEach(func() {
				session := &unstructured.Unstructured{}
				session.SetAPIVersion("vteam.ambient-code/v1alpha1")
				session.SetKind("AgenticSession")
				session.SetName("metrics-session-"+randomName)
				session.SetNamespace(testNamespace)
				session.SetAnnotations(map[string]string{
					"ambient-code.io/input-tokens":  "1500",
					"ambient-code.io/output-tokens": "3200",
					"ambient-code.io/total-cost":    "0.05",
					"ambient-code.io/tool-calls":    "12",
				})

				_ = unstructured.SetNestedField(session.Object, "Running", "status", "phase")
				_ = unstructured.SetNestedField(session.Object, "2026-03-04T10:00:00Z", "status", "startTime")
				_ = unstructured.SetNestedField(session.Object, float64(300), "spec", "timeout")

				_, err := k8sUtils.DynamicClient.Resource(sessionGVR).Namespace(testNamespace).Create(
					ctx, session, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				logger.Log("Created metrics test session with annotations")
			})

			It("Should return metrics with usage data", func() {
				sessionName := "metrics-session-" + randomName
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/metrics", testNamespace, sessionName)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: sessionName},
				}

				GetSessionMetrics(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("sessionId"))
				Expect(response["sessionId"]).To(Equal(sessionName))
				Expect(response).To(HaveKey("phase"))
				Expect(response["phase"]).To(Equal("Running"))
				Expect(response).To(HaveKey("startTime"))
				Expect(response).To(HaveKey("durationSeconds"))
				Expect(response).To(HaveKey("timeoutSeconds"))
				Expect(response["timeoutSeconds"]).To(BeNumerically("==", 300))

				// Check usage annotations
				Expect(response).To(HaveKey("usage"))
				usage, ok := response["usage"].(map[string]interface{})
				Expect(ok).To(BeTrue(), "usage should be a map")
				Expect(usage["inputTokens"]).To(Equal("1500"))
				Expect(usage["outputTokens"]).To(Equal("3200"))
				Expect(usage["totalCost"]).To(Equal("0.05"))
				Expect(usage["toolCalls"]).To(Equal("12"))

				logger.Log("Metrics with usage data returned successfully")
			})
		})

		Context("When session exists with completion time", func() {
			BeforeEach(func() {
				session := &unstructured.Unstructured{}
				session.SetAPIVersion("vteam.ambient-code/v1alpha1")
				session.SetKind("AgenticSession")
				session.SetName("completed-session-"+randomName)
				session.SetNamespace(testNamespace)

				_ = unstructured.SetNestedField(session.Object, "Completed", "status", "phase")
				_ = unstructured.SetNestedField(session.Object, "2026-03-04T10:00:00Z", "status", "startTime")
				_ = unstructured.SetNestedField(session.Object, "2026-03-04T10:05:00Z", "status", "completionTime")

				_, err := k8sUtils.DynamicClient.Resource(sessionGVR).Namespace(testNamespace).Create(
					ctx, session, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				logger.Log("Created completed test session")
			})

			It("Should calculate duration from start and completion times", func() {
				sessionName := "completed-session-" + randomName
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/metrics", testNamespace, sessionName)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: sessionName},
				}

				GetSessionMetrics(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["durationSeconds"]).To(BeNumerically("==", 300), "5 minutes = 300 seconds")
				Expect(response).To(HaveKey("completionTime"))

				logger.Log("Duration calculated correctly for completed session")
			})
		})

		Context("When session does not exist", func() {
			It("Should return 404 Not Found", func() {
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/non-existent/metrics", testNamespace)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "non-existent"},
				}

				GetSessionMetrics(context)

				httpUtils.AssertHTTPStatus(http.StatusNotFound)
				httpUtils.AssertErrorMessage("Session not found")

				logger.Log("404 returned for non-existent session metrics")
			})
		})

		Context("When session has no usage annotations", func() {
			BeforeEach(func() {
				session := &unstructured.Unstructured{}
				session.SetAPIVersion("vteam.ambient-code/v1alpha1")
				session.SetKind("AgenticSession")
				session.SetName("no-usage-session-"+randomName)
				session.SetNamespace(testNamespace)

				_ = unstructured.SetNestedField(session.Object, "Pending", "status", "phase")

				_, err := k8sUtils.DynamicClient.Resource(sessionGVR).Namespace(testNamespace).Create(
					ctx, session, v1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			It("Should return metrics without usage field", func() {
				sessionName := "no-usage-session-" + randomName
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/%s/metrics", testNamespace, sessionName)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: sessionName},
				}

				GetSessionMetrics(context)

				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("sessionId"))
				Expect(response).NotTo(HaveKey("usage"), "Should not include usage when no usage annotations exist")

				logger.Log("Metrics without usage returned correctly")
			})
		})

		Context("When no auth token is provided", func() {
			It("Should return 401 Unauthorized", func() {
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/any-session/metrics", testNamespace)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				// Deliberately NOT setting auth header
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "any-session"},
				}

				GetSessionMetrics(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				logger.Log("401 returned for unauthenticated metrics request")
			})
		})
	})

	Describe("GetSessionLogs", func() {
		// Note: GetSessionLogs calls k8sClt.CoreV1().Pods().GetLogs() which requires
		// a real or fake pod. The fake k8s client doesn't implement pod log streaming,
		// so we test the input validation and auth paths here. Integration tests
		// cover the full streaming path.

		Context("When no auth token is provided", func() {
			It("Should return 401 Unauthorized", func() {
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/any-session/logs", testNamespace)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "any-session"},
				}

				GetSessionLogs(context)

				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

				logger.Log("401 returned for unauthenticated logs request")
			})
		})

		Context("When tailLines parameter is invalid", func() {
			It("Should return 400 for non-numeric tailLines", func() {
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/any-session/logs?tailLines=abc", testNamespace)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "any-session"},
				}

				GetSessionLogs(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("tailLines must be a positive integer")

				logger.Log("400 returned for non-numeric tailLines")
			})

			It("Should return 400 for negative tailLines", func() {
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/any-session/logs?tailLines=-5", testNamespace)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "any-session"},
				}

				GetSessionLogs(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("tailLines must be a positive integer")

				logger.Log("400 returned for negative tailLines")
			})

			It("Should return 400 for zero tailLines", func() {
				path := fmt.Sprintf("/api/projects/%s/agentic-sessions/any-session/logs?tailLines=0", testNamespace)
				context := httpUtils.CreateTestGinContext("GET", path, nil)
				httpUtils.SetAuthHeader(testToken)
				httpUtils.SetProjectContext(testNamespace)
				context.Params = gin.Params{
					{Key: "sessionName", Value: "any-session"},
				}

				GetSessionLogs(context)

				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("tailLines must be a positive integer")

				logger.Log("400 returned for zero tailLines")
			})
		})
	})
})
