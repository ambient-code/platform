//go:build test

package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Feedback Loop Config Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
		k8sUtils  *test_utils.K8sTestUtils
		testToken string
	)

	BeforeEach(func() {
		logger.Log("Setting up Feedback Loop Config test")

		k8sUtils = test_utils.NewK8sTestUtils(false, "test-project")
		SetupHandlerDependencies(k8sUtils)

		httpUtils = test_utils.NewHTTPTestUtils()

		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
		_, err = k8sUtils.CreateTestRole(ctx, "test-project", "feedback-loop-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := httpUtils.SetValidTestToken(
			k8sUtils,
			"test-project",
			[]string{"get", "list", "create", "update", "delete", "patch"},
			"*",
			"",
			"feedback-loop-role",
		)
		Expect(err).NotTo(HaveOccurred())
		testToken = token
	})

	AfterEach(func() {
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "test-project", metav1.DeleteOptions{})
		}
	})

	Describe("GetFeedbackLoopConfig", func() {
		It("Should return defaults when no config exists", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/config", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp FeedbackLoopConfig
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.MinCorrections).To(Equal(defaultMinCorrections))
			Expect(resp.TimeWindowHours).To(Equal(defaultTimeWindowHours))
			Expect(resp.AutoTriggerEnabled).To(BeTrue())

			logger.Log("GetFeedbackLoopConfig returns defaults correctly")
		})

		It("Should return stored config when it exists", func() {
			ctx := context.Background()
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      feedbackLoopConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"config": `{"minCorrections":5,"timeWindowHours":12,"autoTriggerEnabled":false}`,
				},
			}
			_, err := k8sUtils.K8sClient.CoreV1().ConfigMaps("test-project").Create(ctx, cm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/config", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp FeedbackLoopConfig
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.MinCorrections).To(Equal(5))
			Expect(resp.TimeWindowHours).To(Equal(12))
			Expect(resp.AutoTriggerEnabled).To(BeFalse())

			logger.Log("GetFeedbackLoopConfig returns stored config")
		})

		It("Should require authentication", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/config", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			// No auth header

			GetFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

			logger.Log("GetFeedbackLoopConfig requires auth")
		})
	})

	Describe("PutFeedbackLoopConfig", func() {
		It("Should store valid config", func() {
			body := map[string]interface{}{
				"minCorrections":     3,
				"timeWindowHours":    12,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Verify it was persisted
			ctx := context.Background()
			cm, err := k8sUtils.K8sClient.CoreV1().ConfigMaps("test-project").Get(ctx, feedbackLoopConfigMap, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Data["config"]).To(ContainSubstring(`"minCorrections":3`))

			logger.Log("PutFeedbackLoopConfig stores valid config")
		})

		It("Should reject minCorrections less than 1", func() {
			body := map[string]interface{}{
				"minCorrections":     0,
				"timeWindowHours":    24,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("PutFeedbackLoopConfig rejects invalid minCorrections")
		})

		It("Should reject timeWindowHours less than 1", func() {
			body := map[string]interface{}{
				"minCorrections":     2,
				"timeWindowHours":    0,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("PutFeedbackLoopConfig rejects invalid timeWindowHours")
		})

		It("Should require authentication", func() {
			body := map[string]interface{}{
				"minCorrections":     2,
				"timeWindowHours":    24,
				"autoTriggerEnabled": true,
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feedback-loop/config", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			// No auth header

			PutFeedbackLoopConfig(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

			logger.Log("PutFeedbackLoopConfig requires auth")
		})
	})

	Describe("GetFeedbackLoopHistory", func() {
		It("Should return empty list when no history exists", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/history", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopHistory(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp map[string]json.RawMessage
			httpUtils.GetResponseJSON(&resp)
			var sessions []FeedbackLoopHistoryEntry
			err := json.Unmarshal(resp["sessions"], &sessions)
			Expect(err).NotTo(HaveOccurred())
			Expect(sessions).To(BeEmpty())

			logger.Log("GetFeedbackLoopHistory returns empty list")
		})

		It("Should return stored history entries", func() {
			ctx := context.Background()
			entries := []FeedbackLoopHistoryEntry{
				{
					SessionName:   "session-123",
					CreatedAt:     "2026-04-16T10:00:00Z",
					Source:        "event-driven",
					TargetType:    "workflow",
					TargetRepoURL: "https://github.com/org/repo",
					TargetBranch:  "main",
					TargetPath:    ".ambient/workflows/review",
					CorrectionIDs: []string{"trace-1", "trace-2"},
				},
			}
			data, err := json.Marshal(entries)
			Expect(err).NotTo(HaveOccurred())

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      feedbackLoopConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"history": string(data),
				},
			}
			_, err = k8sUtils.K8sClient.CoreV1().ConfigMaps("test-project").Create(ctx, cm, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/history", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			httpUtils.SetProjectContext("test-project")
			httpUtils.SetAuthHeader(testToken)

			GetFeedbackLoopHistory(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp map[string]json.RawMessage
			httpUtils.GetResponseJSON(&resp)
			var sessions []FeedbackLoopHistoryEntry
			err = json.Unmarshal(resp["sessions"], &sessions)
			Expect(err).NotTo(HaveOccurred())
			Expect(sessions).To(HaveLen(1))
			Expect(sessions[0].SessionName).To(Equal("session-123"))
			Expect(sessions[0].Source).To(Equal("event-driven"))

			logger.Log("GetFeedbackLoopHistory returns stored entries")
		})

		It("Should require authentication", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feedback-loop/history", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

			GetFeedbackLoopHistory(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)

			logger.Log("GetFeedbackLoopHistory requires auth")
		})
	})
})
