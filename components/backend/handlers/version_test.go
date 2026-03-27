//go:build test

package handlers

import (
	test_constants "ambient-code-backend/tests/constants"
	"net/http"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Version Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
	)

	BeforeEach(func() {
		logger.Log("Setting up Version Handler test")
		httpUtils = test_utils.NewHTTPTestUtils()
	})

	Context("When querying application version", func() {
		It("Should return the default version", func() {
			context := httpUtils.CreateTestGinContext("GET", "/api/version", nil)

			GetVersion(context)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			httpUtils.AssertJSONContains(map[string]interface{}{
				"version": "dev",
			})
			logger.Log("Version endpoint returned default version")
		})

		It("Should return a version set via SetVersion", func() {
			SetVersion("v1.2.3")
			defer SetVersion("dev")

			context := httpUtils.CreateTestGinContext("GET", "/api/version", nil)

			GetVersion(context)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			httpUtils.AssertJSONContains(map[string]interface{}{
				"version": "v1.2.3",
			})
			logger.Log("Version endpoint returned configured version")
		})
	})
})
