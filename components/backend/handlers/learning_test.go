//go:build test

package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	test_constants "ambient-code-backend/tests/constants"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Learning Endpoints", Label(test_constants.LabelUnit), func() {

	var router *gin.Engine

	BeforeEach(func() {
		gin.SetMode(gin.TestMode)
		router = gin.New()
		// Register routes with project context middleware
		group := router.Group("/api/projects/:projectName", ValidateProjectContext())
		group.GET("/learning/summary", GetLearningSummary)
		group.GET("/learning/timeline", GetLearningTimeline)
	})

	Describe("learning summary", func() {
		It("returns 401 without auth header", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/learning/summary", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns empty summary with auth", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/learning/summary", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var summary LearningSummary
			err := json.Unmarshal(w.Body.Bytes(), &summary)
			Expect(err).NotTo(HaveOccurred())
			Expect(summary.TotalCorrections).To(Equal(0))
			Expect(summary.CorrectionsByType).To(BeEmpty())
			Expect(summary.ImprovementSessions).To(Equal(0))
			Expect(summary.MemoriesCreated).To(Equal(0))
			Expect(summary.MemoryCitations).To(Equal(0))
		})
	})

	Describe("learning timeline", func() {
		It("returns 401 without auth header", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/learning/timeline", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusUnauthorized))
		})

		It("returns empty timeline with default pagination", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/learning/timeline", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response TimelineResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Items).To(BeEmpty())
			Expect(response.TotalCount).To(Equal(0))
			Expect(response.Page).To(Equal(1))
			Expect(response.PageSize).To(Equal(20))
		})

		It("respects custom pagination parameters", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/learning/timeline?page=3&pageSize=10", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response TimelineResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Page).To(Equal(3))
			Expect(response.PageSize).To(Equal(10))
		})

		It("clamps invalid pagination values", func() {
			req := httptest.NewRequest(http.MethodGet, "/api/projects/test-project/learning/timeline?page=-1&pageSize=999", nil)
			req.Header.Set("Authorization", "Bearer test-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response TimelineResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Page).To(Equal(1))
			Expect(response.PageSize).To(Equal(20))
		})
	})
})
