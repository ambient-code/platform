//go:build test

package handlers

import (
	"context"
	"time"

	"ambient-code-backend/models"
	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Drive Storage", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelDriveIntegration), func() {
	var (
		k8sUtils *test_utils.K8sTestUtils
		storage  *DriveStorage
		testCtx  context.Context
	)

	BeforeEach(func() {
		logger.Log("Setting up Drive Storage test")
		k8sUtils = test_utils.NewK8sTestUtils(false, "test-namespace")
		storage = NewDriveStorage(k8sUtils.K8sClient, "test-namespace")
		testCtx = context.Background()
	})

	Context("NewDriveStorage", func() {
		It("Should create a DriveStorage with correct fields", func() {
			// Arrange & Act
			s := NewDriveStorage(k8sUtils.K8sClient, "my-namespace")

			// Assert
			Expect(s).NotTo(BeNil())
			Expect(s.clientset).NotTo(BeNil())
			Expect(s.namespace).To(Equal("my-namespace"))

			logger.Log("DriveStorage created successfully")
		})
	})

	Context("Integration CRUD", func() {
		It("Should round-trip SaveIntegration and GetIntegration", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)

			// Act
			err := storage.SaveIntegration(testCtx, integration)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := storage.GetIntegration(testCtx, "project-1", "user-1")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).NotTo(BeNil())
			Expect(retrieved.ID).To(Equal(integration.ID))
			Expect(retrieved.UserID).To(Equal("user-1"))
			Expect(retrieved.ProjectName).To(Equal("project-1"))
			Expect(retrieved.Provider).To(Equal("google"))
			Expect(retrieved.PermissionScope).To(Equal(models.PermissionScopeGranular))
			Expect(retrieved.Status).To(Equal(models.IntegrationStatusActive))

			logger.Log("Integration round-trip successful")
		})

		It("Should return nil when ConfigMap does not exist", func() {
			// Act
			retrieved, err := storage.GetIntegration(testCtx, "nonexistent-project", "nonexistent-user")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).To(BeNil())

			logger.Log("GetIntegration correctly returned nil for non-existent ConfigMap")
		})

		It("Should update an existing integration on re-save", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)
			err := storage.SaveIntegration(testCtx, integration)
			Expect(err).NotTo(HaveOccurred())

			// Modify and re-save
			integration.Disconnect()
			err = storage.SaveIntegration(testCtx, integration)
			Expect(err).NotTo(HaveOccurred())

			// Act
			retrieved, err := storage.GetIntegration(testCtx, "project-1", "user-1")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).NotTo(BeNil())
			Expect(retrieved.Status).To(Equal(models.IntegrationStatusDisconnected))

			logger.Log("Integration update successful")
		})

		It("Should delete an integration", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)
			err := storage.SaveIntegration(testCtx, integration)
			Expect(err).NotTo(HaveOccurred())

			// Act
			err = storage.DeleteIntegration(testCtx, "project-1", "user-1")
			Expect(err).NotTo(HaveOccurred())

			// Assert
			retrieved, err := storage.GetIntegration(testCtx, "project-1", "user-1")
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).To(BeNil())

			logger.Log("Integration deletion successful")
		})

		It("Should not error when deleting a non-existent integration", func() {
			// Act
			err := storage.DeleteIntegration(testCtx, "nonexistent", "nonexistent")

			// Assert
			Expect(err).NotTo(HaveOccurred())

			logger.Log("Delete of non-existent integration succeeded without error")
		})

		It("Should create ConfigMap with correct labels", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)

			// Act
			err := storage.SaveIntegration(testCtx, integration)
			Expect(err).NotTo(HaveOccurred())

			// Assert - verify ConfigMap labels
			cmName := configMapName("project-1", "user-1")
			cm, err := k8sUtils.K8sClient.CoreV1().ConfigMaps("test-namespace").Get(testCtx, cmName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Labels["app.kubernetes.io/managed-by"]).To(Equal("platform-backend"))
			Expect(cm.Labels["app.kubernetes.io/component"]).To(Equal("drive-integration"))
			Expect(cm.Labels["platform/project"]).To(Equal("project-1"))
			Expect(cm.Labels["platform/user"]).To(Equal("user-1"))

			logger.Log("ConfigMap labels verified")
		})
	})

	Context("FileGrant operations", func() {
		var integration *models.DriveIntegration

		BeforeEach(func() {
			integration = models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)
			err := storage.SaveIntegration(testCtx, integration)
			Expect(err).NotTo(HaveOccurred())
		})

		It("Should round-trip UpdateFileGrants and ListFileGrants", func() {
			// Arrange
			grants := []models.FileGrant{
				{
					ID:            "grant-1",
					IntegrationID: integration.ID,
					GoogleFileID:  "gf-1",
					FileName:      "file1.txt",
					MimeType:      "text/plain",
					Status:        models.FileGrantStatusActive,
					GrantedAt:     time.Now().UTC(),
				},
				{
					ID:            "grant-2",
					IntegrationID: integration.ID,
					GoogleFileID:  "gf-2",
					FileName:      "file2.pdf",
					MimeType:      "application/pdf",
					Status:        models.FileGrantStatusActive,
					GrantedAt:     time.Now().UTC(),
				},
			}

			// Act
			err := storage.UpdateFileGrants(testCtx, integration.ID, grants)
			Expect(err).NotTo(HaveOccurred())

			retrieved, err := storage.ListFileGrants(testCtx, integration.ID)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).To(HaveLen(2))
			Expect(retrieved[0].GoogleFileID).To(Equal("gf-1"))
			Expect(retrieved[1].GoogleFileID).To(Equal("gf-2"))

			logger.Log("FileGrants round-trip successful")
		})

		It("Should return empty list when no file grants exist", func() {
			// Act
			grants, err := storage.ListFileGrants(testCtx, integration.ID)

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(grants).To(BeEmpty())

			logger.Log("Empty file grants list returned correctly")
		})

		It("Should replace file grants on update", func() {
			// Arrange - save initial grants
			initialGrants := []models.FileGrant{
				{
					ID:            "grant-1",
					IntegrationID: integration.ID,
					GoogleFileID:  "gf-1",
					FileName:      "old.txt",
					MimeType:      "text/plain",
					Status:        models.FileGrantStatusActive,
					GrantedAt:     time.Now().UTC(),
				},
			}
			err := storage.UpdateFileGrants(testCtx, integration.ID, initialGrants)
			Expect(err).NotTo(HaveOccurred())

			// Act - replace with new grants
			newGrants := []models.FileGrant{
				{
					ID:            "grant-new",
					IntegrationID: integration.ID,
					GoogleFileID:  "gf-new",
					FileName:      "new.txt",
					MimeType:      "text/plain",
					Status:        models.FileGrantStatusActive,
					GrantedAt:     time.Now().UTC(),
				},
			}
			err = storage.UpdateFileGrants(testCtx, integration.ID, newGrants)
			Expect(err).NotTo(HaveOccurred())

			// Assert
			retrieved, err := storage.ListFileGrants(testCtx, integration.ID)
			Expect(err).NotTo(HaveOccurred())
			Expect(retrieved).To(HaveLen(1))
			Expect(retrieved[0].GoogleFileID).To(Equal("gf-new"))

			logger.Log("FileGrants replacement successful")
		})
	})

	Context("Token operations", func() {
		It("Should round-trip SaveTokens and GetTokens", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)
			expiresAt := time.Now().UTC().Add(1 * time.Hour).Truncate(time.Second)

			// Act
			err := storage.SaveTokens(testCtx, integration, "access-tok", "refresh-tok", expiresAt)
			Expect(err).NotTo(HaveOccurred())

			accessToken, refreshToken, retrievedExpiry, err := storage.GetTokens(testCtx, "project-1", "user-1")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(accessToken).To(Equal("access-tok"))
			Expect(refreshToken).To(Equal("refresh-tok"))
			Expect(retrievedExpiry.Unix()).To(Equal(expiresAt.Unix()))

			logger.Log("Token round-trip successful")
		})

		It("Should return error when tokens do not exist", func() {
			// Act
			_, _, _, err := storage.GetTokens(testCtx, "nonexistent", "nonexistent")

			// Assert
			Expect(err).To(HaveOccurred())

			logger.Log("GetTokens correctly returned error for non-existent Secret")
		})

		It("Should update existing tokens on re-save", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)
			expiresAt := time.Now().UTC().Add(1 * time.Hour).Truncate(time.Second)
			err := storage.SaveTokens(testCtx, integration, "old-access", "old-refresh", expiresAt)
			Expect(err).NotTo(HaveOccurred())

			// Act
			newExpiry := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
			err = storage.SaveTokens(testCtx, integration, "new-access", "new-refresh", newExpiry)
			Expect(err).NotTo(HaveOccurred())

			accessToken, refreshToken, retrievedExpiry, err := storage.GetTokens(testCtx, "project-1", "user-1")

			// Assert
			Expect(err).NotTo(HaveOccurred())
			Expect(accessToken).To(Equal("new-access"))
			Expect(refreshToken).To(Equal("new-refresh"))
			Expect(retrievedExpiry.Unix()).To(Equal(newExpiry.Unix()))

			logger.Log("Token update successful")
		})

		It("Should delete tokens", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)
			expiresAt := time.Now().UTC().Add(1 * time.Hour)
			err := storage.SaveTokens(testCtx, integration, "access-tok", "refresh-tok", expiresAt)
			Expect(err).NotTo(HaveOccurred())

			// Act
			err = storage.DeleteTokens(testCtx, "project-1", "user-1")
			Expect(err).NotTo(HaveOccurred())

			// Assert
			_, _, _, err = storage.GetTokens(testCtx, "project-1", "user-1")
			Expect(err).To(HaveOccurred())

			logger.Log("Token deletion successful")
		})

		It("Should not error when deleting non-existent tokens", func() {
			// Act
			err := storage.DeleteTokens(testCtx, "nonexistent", "nonexistent")

			// Assert
			Expect(err).NotTo(HaveOccurred())

			logger.Log("Delete of non-existent tokens succeeded without error")
		})

		It("Should create Secret with correct labels", func() {
			// Arrange
			integration := models.NewDriveIntegration("user-1", "project-1", models.PermissionScopeGranular)
			expiresAt := time.Now().UTC().Add(1 * time.Hour)

			// Act
			err := storage.SaveTokens(testCtx, integration, "access-tok", "refresh-tok", expiresAt)
			Expect(err).NotTo(HaveOccurred())

			// Assert
			sName := secretName("project-1", "user-1")
			secret, err := k8sUtils.K8sClient.CoreV1().Secrets("test-namespace").Get(testCtx, sName, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(secret.Labels["app.kubernetes.io/managed-by"]).To(Equal("platform-backend"))
			Expect(secret.Labels["app.kubernetes.io/component"]).To(Equal("drive-tokens"))
			Expect(secret.Labels["platform/project"]).To(Equal("project-1"))
			Expect(secret.Labels["platform/user"]).To(Equal("user-1"))

			logger.Log("Secret labels verified")
		})
	})

})
