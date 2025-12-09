//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"
	"net/http"
	"time"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Health Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelHealth), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
	)

	BeforeEach(func() {
		logger.Log("Setting up Health Handler test")
		httpUtils = test_utils.NewHTTPTestUtils()
	})

	Context("When checking application health", func() {
		It("Should return 200 OK with health status", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("GET", "/health", nil)

			// Act
			Health(context)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)

			expectedResponse := map[string]interface{}{
				"status": "healthy",
			}
			httpUtils.AssertJSONContains(expectedResponse)

			logger.Log("Health endpoint returned expected response")
		})

		It("Should respond quickly", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("GET", "/health", nil)

			// Act & Assert - should complete within reasonable time
			startTime := time.Now()
			Health(context)
			duration := time.Since(startTime)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			Expect(duration.Milliseconds()).To(BeNumerically("<", 100), "Health endpoint should respond in under 100ms")

			logger.Log("Health endpoint responded in %v", duration)
		})
	})

	Context("When handling different HTTP methods", func() {
		It("Should handle GET requests", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("GET", "/health", nil)

			// Act
			Health(context)

			// Assert
			httpUtils.AssertHTTPSuccess()
		})

		It("Should handle POST requests (if endpoint supports them)", func() {
			// Arrange
			context := httpUtils.CreateTestGinContext("POST", "/health", nil)

			// Act
			Health(context)

			// Assert
			httpUtils.AssertHTTPSuccess()
		})
	})

	Context("Edge cases", func() {
		It("Should handle concurrent requests", func() {
			// Arrange
			const numGoroutines = 10
			results := make(chan int, numGoroutines)

			// Act
			for i := 0; i < numGoroutines; i++ {
				go func() {
					httpUtils := test_utils.NewHTTPTestUtils()
					context := httpUtils.CreateTestGinContext("GET", "/health", nil)
					Health(context)
					results <- httpUtils.GetResponseRecorder().Code
				}()
			}

			// Assert
			for i := 0; i < numGoroutines; i++ {
				statusCode := <-results
				Expect(statusCode).To(Equal(http.StatusOK))
			}

			logger.Log("All concurrent health requests returned 200 OK")
		})
	})
})
