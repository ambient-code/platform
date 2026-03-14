//go:build test

package handlers

import (
	"context"
	"net/http"
	"time"

	"ambient-code-backend/models"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Drive Integration Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelDriveIntegration), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
		k8sUtils  *test_utils.K8sTestUtils
		storage   *DriveStorage
		handler   *DriveIntegrationHandler
	)

	BeforeEach(func() {
		logger.Log("Setting up Drive Integration Handler test")
		k8sUtils = test_utils.NewK8sTestUtils(false, "test-namespace")
		storage = NewDriveStorage(k8sUtils.K8sClient, "test-namespace")
		handler = NewDriveIntegrationHandler(
			storage,
			"test-client-id",
			"test-client-secret",
			[]byte("test-hmac-secret"),
			"test-api-key",
			"test-app-id",
		)
		httpUtils = test_utils.NewHTTPTestUtils()
	})

	Context("HandleDriveSetup", func() {
		It("Should return 200 with authUrl and state", func() {
			// Arrange
			body := models.SetupRequest{
				PermissionScope: models.PermissionScopeGranular,
				RedirectURI:     "http://localhost:3000/callback",
			}
			ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/integrations/google-drive/setup", body)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}

			// Act
			handler.HandleDriveSetup(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)
			httpUtils.AssertJSONStructure([]string{"authUrl", "state"})

			var resp map[string]interface{}
			httpUtils.GetResponseJSON(&resp)
			authURL := resp["authUrl"].(string)
			Expect(authURL).To(ContainSubstring("accounts.google.com"))
			Expect(authURL).To(ContainSubstring("test-client-id"))
			Expect(resp["state"]).NotTo(BeEmpty())

			logger.Log("HandleDriveSetup returned auth URL and state")
		})

		It("Should return 400 for missing redirectUri", func() {
			// Arrange — omit RedirectURI (required field)
			body := map[string]interface{}{
				"permissionScope": "granular",
			}
			ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/integrations/google-drive/setup", body)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}

			// Act
			handler.HandleDriveSetup(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("HandleDriveSetup correctly returned 400 for missing redirectUri")
		})

		It("Should default to granular scope when not specified", func() {
			// Arrange
			body := map[string]interface{}{
				"redirectUri": "http://localhost:3000/callback",
			}
			ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/integrations/google-drive/setup", body)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}

			// Act
			handler.HandleDriveSetup(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)
			httpUtils.AssertJSONStructure([]string{"authUrl", "state"})

			logger.Log("HandleDriveSetup defaulted to granular scope")
		})
	})

	Context("HandleGetDriveIntegration", func() {
		It("Should return 404 when no integration exists", func() {
			// Arrange
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			handler.HandleGetDriveIntegration(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusNotFound)

			logger.Log("HandleGetDriveIntegration returned 404 for missing integration")
		})

		It("Should return 200 with integration when it exists", func() {
			// Arrange — save an integration first
			integration := models.NewDriveIntegration("test-user", "test-project", models.PermissionScopeGranular)
			err := storage.SaveIntegration(context.Background(), integration)
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			handler.HandleGetDriveIntegration(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)
			httpUtils.AssertJSONStructure([]string{"id", "userId", "projectName", "provider", "status"})

			var resp map[string]interface{}
			httpUtils.GetResponseJSON(&resp)
			Expect(resp["id"]).To(Equal(integration.ID))
			Expect(resp["userId"]).To(Equal("test-user"))
			Expect(resp["projectName"]).To(Equal("test-project"))
			Expect(resp["provider"]).To(Equal("google"))
			Expect(resp["status"]).To(Equal("active"))

			logger.Log("HandleGetDriveIntegration returned existing integration")
		})

		It("Should use default-user when userID is not set", func() {
			// Arrange — save integration for default-user
			integration := models.NewDriveIntegration("default-user", "test-project", models.PermissionScopeGranular)
			err := storage.SaveIntegration(context.Background(), integration)
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			// Do not set userID — handler should fall back to "default-user"

			// Act
			handler.HandleGetDriveIntegration(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)

			logger.Log("HandleGetDriveIntegration fell back to default-user")
		})
	})

	Context("HandleDisconnectDriveIntegration", func() {
		It("Should return 404 when no integration exists", func() {
			// Arrange
			ginCtx := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/integrations/google-drive", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			handler.HandleDisconnectDriveIntegration(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusNotFound)

			logger.Log("HandleDisconnectDriveIntegration returned 404 for missing integration")
		})

		It("Should return 204 after disconnecting an existing integration", func() {
			// Arrange — save integration without tokens so the handler
			// skips the Google token revocation HTTP call in tests.
			integration := models.NewDriveIntegration("test-user", "test-project", models.PermissionScopeGranular)
			err := storage.SaveIntegration(context.Background(), integration)
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/integrations/google-drive", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			handler.HandleDisconnectDriveIntegration(ginCtx)

			// Assert — c.Status(204) sets the gin writer status but does not
			// flush to the underlying httptest.ResponseRecorder, so read the
			// status from the gin writer directly.
			Expect(ginCtx.Writer.Status()).To(Equal(http.StatusNoContent))

			// Verify integration was deleted
			retrieved, err := storage.GetIntegration(context.Background(), "test-project", "test-user")
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).To(BeNil())

			logger.Log("HandleDisconnectDriveIntegration successfully disconnected")
		})
	})

	Context("HandlePickerToken", func() {
		It("Should return 404 when no tokens exist", func() {
			// Arrange
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive/picker-token", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			handler.HandlePickerToken(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusNotFound)

			logger.Log("HandlePickerToken returned 404 for missing tokens")
		})

		It("Should return 200 with valid non-expired token", func() {
			// Arrange — save tokens that are still valid
			integration := models.NewDriveIntegration("test-user", "test-project", models.PermissionScopeGranular)
			expiresAt := time.Now().UTC().Add(1 * time.Hour)
			err := storage.SaveTokens(context.Background(), integration, "valid-access-token", "refresh-tok", expiresAt)
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive/picker-token", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			handler.HandlePickerToken(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)
			httpUtils.AssertJSONStructure([]string{"accessToken", "expiresIn"})

			var resp map[string]interface{}
			httpUtils.GetResponseJSON(&resp)
			Expect(resp["accessToken"]).To(Equal("valid-access-token"))
			expiresIn := resp["expiresIn"].(float64)
			Expect(expiresIn).To(BeNumerically(">", 0))

			logger.Log("HandlePickerToken returned valid token")
		})
	})
})
