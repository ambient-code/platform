//go:build test

package handlers

import (
	"context"
	"net/http"

	"ambient-code-backend/models"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var _ = Describe("Drive File Grants Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelDriveIntegration), func() {
	var (
		httpUtils     *test_utils.HTTPTestUtils
		k8sUtils      *test_utils.K8sTestUtils
		storage       *DriveStorage
		grantsHandler *DriveFileGrantsHandler
	)

	BeforeEach(func() {
		logger.Log("Setting up Drive File Grants Handler test")
		k8sUtils = test_utils.NewK8sTestUtils(false, "test-namespace")
		storage = NewDriveStorage(k8sUtils.K8sClient, "test-namespace")

		oauthCfg := &oauth2.Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			Endpoint:     google.Endpoint,
		}
		grantsHandler = NewDriveFileGrantsHandler(storage, oauthCfg)

		httpUtils = test_utils.NewHTTPTestUtils()
	})

	Context("HandleUpdateFileGrants", func() {
		It("Should return 400 for empty files array", func() {
			// Arrange
			body := map[string]interface{}{
				"files": []interface{}{},
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integrations/google-drive/files", body)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleUpdateFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("HandleUpdateFileGrants returned 400 for empty files")
		})

		It("Should return 404 when no integration exists", func() {
			// Arrange
			body := models.UpdateFileGrantsRequest{
				Files: []models.PickerFile{
					{
						ID:       "file-1",
						Name:     "doc.txt",
						MimeType: "text/plain",
					},
				},
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integrations/google-drive/files", body)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleUpdateFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusNotFound)

			logger.Log("HandleUpdateFileGrants returned 404 for missing integration")
		})

		It("Should return 200 with correct added/removed counts", func() {
			// Arrange — create integration
			integration := models.NewDriveIntegration("test-user", "test-project", models.PermissionScopeGranular)
			err := storage.SaveIntegration(context.Background(), integration)
			Expect(err).NotTo(HaveOccurred())

			body := models.UpdateFileGrantsRequest{
				Files: []models.PickerFile{
					{
						ID:       "file-1",
						Name:     "doc.txt",
						MimeType: "text/plain",
					},
					{
						ID:       "file-2",
						Name:     "sheet.xlsx",
						MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
					},
				},
			}
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integrations/google-drive/files", body)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleUpdateFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp models.UpdateFileGrantsResponse
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.Added).To(Equal(2))
			Expect(resp.Removed).To(Equal(0))
			Expect(resp.Files).To(HaveLen(2))

			logger.Log("HandleUpdateFileGrants returned correct counts")
		})

		It("Should compute correct removed count when replacing files", func() {
			// Arrange — create integration and initial file grants
			integration := models.NewDriveIntegration("test-user", "test-project", models.PermissionScopeGranular)
			err := storage.SaveIntegration(context.Background(), integration)
			Expect(err).NotTo(HaveOccurred())

			initialGrants := []models.FileGrant{
				{
					ID:            "g-1",
					IntegrationID: integration.ID,
					GoogleFileID:  "old-file-1",
					FileName:      "old1.txt",
					MimeType:      "text/plain",
					Status:        models.FileGrantStatusActive,
				},
				{
					ID:            "g-2",
					IntegrationID: integration.ID,
					GoogleFileID:  "old-file-2",
					FileName:      "old2.txt",
					MimeType:      "text/plain",
					Status:        models.FileGrantStatusActive,
				},
			}
			err = storage.UpdateFileGrants(context.Background(), integration.ID, initialGrants)
			Expect(err).NotTo(HaveOccurred())

			// Replace with one new file (removing both old ones)
			body := models.UpdateFileGrantsRequest{
				Files: []models.PickerFile{
					{
						ID:       "new-file-1",
						Name:     "new.txt",
						MimeType: "text/plain",
					},
				},
			}

			httpUtils = test_utils.NewHTTPTestUtils()
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integrations/google-drive/files", body)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleUpdateFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp models.UpdateFileGrantsResponse
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.Added).To(Equal(1))
			Expect(resp.Removed).To(Equal(2))
			Expect(resp.Files).To(HaveLen(1))

			logger.Log("HandleUpdateFileGrants computed correct removed count")
		})

		It("Should return 400 for invalid request body", func() {
			// Arrange
			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/integrations/google-drive/files", "invalid-json")
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleUpdateFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("HandleUpdateFileGrants returned 400 for invalid body")
		})
	})

	Context("HandleListFileGrants", func() {
		It("Should return 404 when no integration exists", func() {
			// Arrange
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive/files", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleListFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusNotFound)

			logger.Log("HandleListFileGrants returned 404 for missing integration")
		})

		It("Should return 200 with empty file list when no grants exist", func() {
			// Arrange — create integration without file grants
			integration := models.NewDriveIntegration("test-user", "test-project", models.PermissionScopeGranular)
			err := storage.SaveIntegration(context.Background(), integration)
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive/files", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleListFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp models.ListFileGrantsResponse
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.TotalCount).To(Equal(0))
			Expect(resp.Files).To(BeEmpty())

			logger.Log("HandleListFileGrants returned empty list")
		})

		It("Should return 200 with file list when grants exist", func() {
			// Arrange — create integration and file grants
			integration := models.NewDriveIntegration("test-user", "test-project", models.PermissionScopeGranular)
			err := storage.SaveIntegration(context.Background(), integration)
			Expect(err).NotTo(HaveOccurred())

			grants := []models.FileGrant{
				{
					ID:            "g-1",
					IntegrationID: integration.ID,
					GoogleFileID:  "gf-1",
					FileName:      "file1.txt",
					MimeType:      "text/plain",
					Status:        models.FileGrantStatusActive,
				},
				{
					ID:            "g-2",
					IntegrationID: integration.ID,
					GoogleFileID:  "gf-2",
					FileName:      "file2.pdf",
					MimeType:      "application/pdf",
					Status:        models.FileGrantStatusActive,
				},
			}
			err = storage.UpdateFileGrants(context.Background(), integration.ID, grants)
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/integrations/google-drive/files", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			ginCtx.Set("userID", "test-user")

			// Act
			grantsHandler.HandleListFileGrants(ginCtx)

			// Assert
			httpUtils.AssertHTTPStatus(http.StatusOK)

			var resp models.ListFileGrantsResponse
			httpUtils.GetResponseJSON(&resp)
			Expect(resp.TotalCount).To(Equal(2))
			Expect(resp.Files).To(HaveLen(2))

			logger.Log("HandleListFileGrants returned correct file list")
		})
	})
})
