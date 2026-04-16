//go:build test

package handlers

import (
	"context"
	"net/http"
	"time"

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

var _ = Describe("Corrections Pipeline Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelCorrections), func() {
	var (
		localHTTP *test_utils.HTTPTestUtils
		localK8s  *test_utils.K8sTestUtils
		testToken string
	)

	BeforeEach(func() {
		logger.Log("Setting up Corrections Pipeline test")

		localK8s = test_utils.NewK8sTestUtils(false, "test-project")
		SetupHandlerDependencies(localK8s)

		localHTTP = test_utils.NewHTTPTestUtils()

		// Create namespace + role and mint token
		ctx := context.Background()
		_, err := localK8s.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
		_, err = localK8s.CreateTestRole(ctx, "test-project", "test-corrections-role",
			[]string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := localHTTP.SetValidTestToken(
			localK8s, "test-project",
			[]string{"get", "list", "create", "update", "delete", "patch"},
			"*", "", "test-corrections-role",
		)
		Expect(err).NotTo(HaveOccurred())
		testToken = token

		// Enable the feature flag via ConfigMap override
		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      FeatureFlagOverridesConfigMap,
				Namespace: "test-project",
			},
			Data: map[string]string{
				correctionsFeatureFlag: "true",
			},
		}
		_, err = localK8s.K8sClient.CoreV1().ConfigMaps("test-project").Create(ctx, cm, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		// Reset the in-memory buffer between tests
		ResetCorrectionsBuffers()
	})

	AfterEach(func() {
		if localK8s != nil {
			_ = localK8s.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "test-project", metav1.DeleteOptions{})
		}
	})

	// ---------------------------------------------------------------
	// POST /corrections
	// ---------------------------------------------------------------
	Context("POST /corrections", func() {
		It("Should accept a valid correction and return 201", func() {
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "incorrect",
				"agentAction":    "Used if/else",
				"userCorrection": "Should have used try/except",
				"target":         "my-workflow",
				"source":         "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusCreated)
			logger.Log("POST /corrections returned 201 for valid input")
		})

		It("Should reject missing correctionType with 400", func() {
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"agentAction":    "test",
				"userCorrection": "test",
				"source":         "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusBadRequest)
			logger.Log("POST /corrections rejected missing correctionType")
		})

		It("Should reject invalid correctionType with 400", func() {
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "bogus",
				"agentAction":    "test",
				"userCorrection": "test",
				"source":         "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusBadRequest)
			localHTTP.AssertErrorMessage("Invalid correctionType")
			logger.Log("POST /corrections rejected invalid correctionType")
		})

		It("Should reject invalid source with 400", func() {
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "incorrect",
				"agentAction":    "test",
				"userCorrection": "test",
				"source":         "invalid-source",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusBadRequest)
			localHTTP.AssertErrorMessage("Invalid source")
			logger.Log("POST /corrections rejected invalid source")
		})

		It("Should accept source=ui for frontend corrections", func() {
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "style",
				"agentAction":    "Used wrong pattern",
				"userCorrection": "Use the standard pattern",
				"source":         "ui",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusCreated)
			logger.Log("POST /corrections accepted source=ui")
		})

		It("Should accept source=rubric", func() {
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "style",
				"agentAction":    "Originality scored low",
				"userCorrection": "Use fresh humor",
				"source":         "rubric",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusCreated)
			logger.Log("POST /corrections accepted source=rubric")
		})

		It("Should accept empty target", func() {
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "incomplete",
				"agentAction":    "Missed a step",
				"userCorrection": "Include step 3",
				"source":         "human",
				"target":         "",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusCreated)
			logger.Log("POST /corrections accepted empty target")
		})

		It("Should require authentication", func() {
			restore := WithAuthCheckEnabled()
			defer restore()

			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "incorrect",
				"agentAction":    "test",
				"userCorrection": "test",
				"source":         "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			// Don't set auth header

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusNotFound)
			logger.Log("POST /corrections requires authentication")
		})
	})

	// ---------------------------------------------------------------
	// GET /corrections
	// ---------------------------------------------------------------
	Context("GET /corrections", func() {
		It("Should return empty list when no corrections exist", func() {
			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			ListCorrections(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)

			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			corrections := response["corrections"].([]interface{})
			Expect(corrections).To(HaveLen(0))
			logger.Log("GET /corrections returns empty list when no corrections")
		})

		It("Should return posted corrections", func() {
			// Post a correction first
			body := map[string]interface{}{
				"sessionName":    "session-1",
				"correctionType": "incorrect",
				"agentAction":    "Did X",
				"userCorrection": "Should do Y",
				"target":         "workflow-a",
				"source":         "human",
			}
			postCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			postCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)
			PostCorrection(postCtx)

			// Now list
			localHTTP = test_utils.NewHTTPTestUtils()
			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			ListCorrections(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)

			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			corrections := response["corrections"].([]interface{})
			Expect(corrections).To(HaveLen(1))

			first := corrections[0].(map[string]interface{})
			Expect(first["sessionName"]).To(Equal("session-1"))
			Expect(first["correctionType"]).To(Equal("incorrect"))
			Expect(first["target"]).To(Equal("workflow-a"))
			Expect(first["source"]).To(Equal("human"))
			logger.Log("GET /corrections returns posted corrections")
		})

		It("Should filter by session query param", func() {
			// Post corrections for different sessions
			for _, sess := range []string{"session-a", "session-b", "session-a"} {
				localHTTP = test_utils.NewHTTPTestUtils()
				body := map[string]interface{}{
					"sessionName": sess, "correctionType": "incorrect",
					"agentAction": "test", "userCorrection": "test", "source": "human",
				}
				ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
				ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
				localHTTP.SetAuthHeader(testToken)
				PostCorrection(ginCtx)
			}

			// Filter for session-a
			localHTTP = test_utils.NewHTTPTestUtils()
			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections?session=session-a", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			ginCtx.Request.URL.RawQuery = "session=session-a"
			localHTTP.SetAuthHeader(testToken)

			ListCorrections(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			corrections := response["corrections"].([]interface{})
			Expect(corrections).To(HaveLen(2))
			logger.Log("GET /corrections filters by session")
		})

		It("Should filter by target query param", func() {
			// Post corrections for different targets
			for _, t := range []string{"wf-a", "wf-b", "wf-a"} {
				localHTTP = test_utils.NewHTTPTestUtils()
				body := map[string]interface{}{
					"sessionName": "s1", "correctionType": "style",
					"agentAction": "test", "userCorrection": "test",
					"target": t, "source": "human",
				}
				ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
				ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
				localHTTP.SetAuthHeader(testToken)
				PostCorrection(ginCtx)
			}

			// Filter for wf-a
			localHTTP = test_utils.NewHTTPTestUtils()
			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections?target=wf-a", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			ginCtx.Request.URL.RawQuery = "target=wf-a"
			localHTTP.SetAuthHeader(testToken)

			ListCorrections(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			corrections := response["corrections"].([]interface{})
			Expect(corrections).To(HaveLen(2))
			logger.Log("GET /corrections filters by target")
		})

		It("Should return corrections from both sources", func() {
			// Post from runner (human)
			localHTTP = test_utils.NewHTTPTestUtils()
			body := map[string]interface{}{
				"sessionName": "s1", "correctionType": "incorrect",
				"agentAction": "test", "userCorrection": "test",
				"source": "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)
			PostCorrection(ginCtx)

			// Post from UI
			localHTTP = test_utils.NewHTTPTestUtils()
			body = map[string]interface{}{
				"sessionName": "s1", "correctionType": "style",
				"agentAction": "wrong approach", "userCorrection": "use standard pattern",
				"source": "ui",
			}
			ginCtx = localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)
			PostCorrection(ginCtx)

			// List all
			localHTTP = test_utils.NewHTTPTestUtils()
			ginCtx = localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			ListCorrections(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			corrections := response["corrections"].([]interface{})
			Expect(corrections).To(HaveLen(2))

			// Verify both sources are present
			sources := make(map[string]bool)
			for _, c := range corrections {
				evt := c.(map[string]interface{})
				sources[evt["source"].(string)] = true
			}
			Expect(sources).To(HaveKey("human"))
			Expect(sources).To(HaveKey("ui"))
			logger.Log("GET /corrections returns corrections from both sources")
		})

		It("Should require authentication", func() {
			restore := WithAuthCheckEnabled()
			defer restore()

			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

			ListCorrections(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusNotFound)
			logger.Log("GET /corrections requires authentication")
		})
	})

	// ---------------------------------------------------------------
	// GET /corrections/summary
	// ---------------------------------------------------------------
	Context("GET /corrections/summary", func() {
		It("Should return empty summary when no corrections", func() {
			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections/summary", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			GetCorrectionsSummary(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			summary := response["summary"].(map[string]interface{})
			Expect(summary).To(BeEmpty())
			logger.Log("GET /corrections/summary returns empty when no corrections")
		})

		It("Should return counts grouped by target", func() {
			// Post corrections for different targets
			targets := []string{"workflow-a", "workflow-a", "workflow-a", "repo-b", "repo-b"}
			for _, t := range targets {
				localHTTP = test_utils.NewHTTPTestUtils()
				body := map[string]interface{}{
					"sessionName": "s1", "correctionType": "incorrect",
					"agentAction": "test", "userCorrection": "test",
					"target": t, "source": "human",
				}
				ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
				ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
				localHTTP.SetAuthHeader(testToken)
				PostCorrection(ginCtx)
			}

			localHTTP = test_utils.NewHTTPTestUtils()
			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections/summary", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			GetCorrectionsSummary(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			summary := response["summary"].(map[string]interface{})
			Expect(summary["workflow-a"]).To(BeNumerically("==", 3))
			Expect(summary["repo-b"]).To(BeNumerically("==", 2))
			logger.Log("GET /corrections/summary returns grouped counts")
		})

		It("Should filter summary by target query param", func() {
			// Post corrections
			for _, t := range []string{"workflow-a", "workflow-a", "repo-b"} {
				localHTTP = test_utils.NewHTTPTestUtils()
				body := map[string]interface{}{
					"sessionName": "s1", "correctionType": "incorrect",
					"agentAction": "test", "userCorrection": "test",
					"target": t, "source": "human",
				}
				ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
				ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
				localHTTP.SetAuthHeader(testToken)
				PostCorrection(ginCtx)
			}

			localHTTP = test_utils.NewHTTPTestUtils()
			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections/summary?target=workflow-a", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			ginCtx.Request.URL.RawQuery = "target=workflow-a"
			localHTTP.SetAuthHeader(testToken)

			GetCorrectionsSummary(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			summary := response["summary"].(map[string]interface{})
			Expect(summary).To(HaveLen(1))
			Expect(summary["workflow-a"]).To(BeNumerically("==", 2))
			logger.Log("GET /corrections/summary filters by target")
		})

		It("Should group empty targets under (none)", func() {
			localHTTP = test_utils.NewHTTPTestUtils()
			body := map[string]interface{}{
				"sessionName": "s1", "correctionType": "style",
				"agentAction": "test", "userCorrection": "test",
				"target": "", "source": "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)
			PostCorrection(ginCtx)

			localHTTP = test_utils.NewHTTPTestUtils()
			ginCtx = localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections/summary", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			GetCorrectionsSummary(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			localHTTP.GetResponseJSON(&response)
			summary := response["summary"].(map[string]interface{})
			Expect(summary["(none)"]).To(BeNumerically("==", 1))
			logger.Log("GET /corrections/summary groups empty targets under (none)")
		})

		It("Should require authentication", func() {
			restore := WithAuthCheckEnabled()
			defer restore()

			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections/summary", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}

			GetCorrectionsSummary(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusNotFound)
			logger.Log("GET /corrections/summary requires authentication")
		})
	})

	// ---------------------------------------------------------------
	// Feature flag gating
	// ---------------------------------------------------------------
	Context("Feature Flag Gating", func() {
		It("Should return 404 for POST when feature flag is disabled", func() {
			// Delete the ConfigMap override to disable the flag
			ctx := context.Background()
			_ = localK8s.K8sClient.CoreV1().ConfigMaps("test-project").Delete(ctx, FeatureFlagOverridesConfigMap, metav1.DeleteOptions{})

			body := map[string]interface{}{
				"sessionName": "s1", "correctionType": "incorrect",
				"agentAction": "test", "userCorrection": "test", "source": "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			PostCorrection(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusNotFound)
			logger.Log("POST /corrections returns 404 when flag disabled")
		})

		It("Should return 404 for GET when feature flag is disabled", func() {
			ctx := context.Background()
			_ = localK8s.K8sClient.CoreV1().ConfigMaps("test-project").Delete(ctx, FeatureFlagOverridesConfigMap, metav1.DeleteOptions{})

			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			ListCorrections(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusNotFound)
			logger.Log("GET /corrections returns 404 when flag disabled")
		})

		It("Should return 404 for summary when feature flag is disabled", func() {
			ctx := context.Background()
			_ = localK8s.K8sClient.CoreV1().ConfigMaps("test-project").Delete(ctx, FeatureFlagOverridesConfigMap, metav1.DeleteOptions{})

			ginCtx := localHTTP.CreateTestGinContext("GET", "/api/projects/test-project/corrections/summary", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)

			GetCorrectionsSummary(ginCtx)

			localHTTP.AssertHTTPStatus(http.StatusNotFound)
			logger.Log("GET /corrections/summary returns 404 when flag disabled")
		})
	})

	// ---------------------------------------------------------------
	// Buffer behavior
	// ---------------------------------------------------------------
	Context("Buffer Behavior", func() {
		It("Should evict oldest events when buffer is full", func() {
			buf := getProjectBuffer("eviction-test")

			// Fill buffer beyond max
			for i := 0; i < maxEventsPerProject+5; i++ {
				buf.append(CorrectionEvent{
					SessionName:    "s1",
					CorrectionType: "incorrect",
					AgentAction:    "test",
					UserCorrection: "test",
					Source:         "human",
					ReceivedAt:     time.Now(),
				})
			}

			buf.mu.RLock()
			Expect(len(buf.events)).To(Equal(maxEventsPerProject))
			buf.mu.RUnlock()
			logger.Log("Buffer evicts oldest events at max capacity")
		})

		It("Should not return expired events", func() {
			buf := getProjectBuffer("expiry-test")

			// Add an event with a ReceivedAt in the past (>24h ago)
			buf.mu.Lock()
			buf.events = append(buf.events, CorrectionEvent{
				SessionName:    "old-session",
				CorrectionType: "incorrect",
				AgentAction:    "old action",
				UserCorrection: "old correction",
				Source:         "human",
				ReceivedAt:     time.Now().Add(-25 * time.Hour),
			})
			buf.mu.Unlock()

			// Add a fresh event
			buf.append(CorrectionEvent{
				SessionName:    "new-session",
				CorrectionType: "style",
				AgentAction:    "new action",
				UserCorrection: "new correction",
				Source:         "human",
				ReceivedAt:     time.Now(),
			})

			events := buf.list("", "")
			Expect(events).To(HaveLen(1))
			Expect(events[0].SessionName).To(Equal("new-session"))
			logger.Log("Buffer excludes expired events from list")
		})

		It("Should not count expired events in summary", func() {
			buf := getProjectBuffer("expiry-summary-test")

			// Add an expired event
			buf.mu.Lock()
			buf.events = append(buf.events, CorrectionEvent{
				SessionName:    "old",
				CorrectionType: "incorrect",
				Target:         "wf-a",
				Source:         "human",
				ReceivedAt:     time.Now().Add(-25 * time.Hour),
			})
			buf.mu.Unlock()

			// Add a fresh event
			buf.append(CorrectionEvent{
				SessionName:    "new",
				CorrectionType: "style",
				Target:         "wf-a",
				Source:         "human",
				ReceivedAt:     time.Now(),
			})

			counts := buf.summary("")
			Expect(counts["wf-a"]).To(Equal(1))
			logger.Log("Buffer excludes expired events from summary")
		})

		It("Should isolate corrections between projects", func() {
			// Post to project test-project
			body := map[string]interface{}{
				"sessionName": "s1", "correctionType": "incorrect",
				"agentAction": "test", "userCorrection": "test",
				"source": "human",
			}
			ginCtx := localHTTP.CreateTestGinContext("POST", "/api/projects/test-project/corrections", body)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			localHTTP.SetAuthHeader(testToken)
			PostCorrection(ginCtx)

			// Check another project buffer is empty
			otherBuf := getProjectBuffer("other-project")
			events := otherBuf.list("", "")
			Expect(events).To(HaveLen(0))
			logger.Log("Corrections are isolated between projects")
		})
	})
})
