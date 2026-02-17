//go:build test

package handlers

import (
	"context"
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

var _ = Describe("Feature Flags Admin Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelFeatureFlags), func() {
	var (
		httpUtils   *test_utils.HTTPTestUtils
		k8sUtils    *test_utils.K8sTestUtils
		fakeClients *test_utils.FakeClientSet
		testToken   string
	)

	BeforeEach(func() {
		logger.Log("Setting up Feature Flags Admin Handler test")

		// Use centralized K8s test setup with fake cluster
		k8sUtils = test_utils.NewK8sTestUtils(false, "test-project")
		SetupHandlerDependencies(k8sUtils)

		// Create fake clients that match the K8s utils setup
		fakeClients = &test_utils.FakeClientSet{
			K8sClient:     k8sUtils.K8sClient,
			DynamicClient: k8sUtils.DynamicClient,
		}

		httpUtils = test_utils.NewHTTPTestUtils()

		// Create namespace + role and mint a valid test token for this suite
		ctx := context.Background()
		_, err := k8sUtils.K8sClient.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: "test-project"},
		}, metav1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
		_, err = k8sUtils.CreateTestRole(ctx, "test-project", "test-full-access-role", []string{"get", "list", "create", "update", "delete", "patch"}, "*", "")
		Expect(err).NotTo(HaveOccurred())

		token, _, err := httpUtils.SetValidTestToken(
			k8sUtils,
			"test-project",
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
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "test-project", metav1.DeleteOptions{})
		}
	})

	Context("Authentication", func() {
		Describe("ListFeatureFlags", func() {
			It("Should require authentication", func() {
				// Arrange
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				// Don't set auth header

				// Act
				ListFeatureFlags(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User token required")

				logger.Log("ListFeatureFlags correctly requires authentication")
			})
		})

		Describe("EnableFeatureFlag", func() {
			It("Should require authentication", func() {
				// Arrange
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/enable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				// Don't set auth header

				// Act
				EnableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User token required")

				logger.Log("EnableFeatureFlag correctly requires authentication")
			})
		})

		Describe("DisableFeatureFlag", func() {
			It("Should require authentication", func() {
				// Arrange
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/disable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				// Don't set auth header

				// Act
				DisableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User token required")

				logger.Log("DisableFeatureFlag correctly requires authentication")
			})
		})

		Describe("DeleteFeatureFlagOverride", func() {
			It("Should require authentication", func() {
				// Arrange
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/feature-flags/my-flag/override", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				// Don't set auth header

				// Act
				DeleteFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User token required")

				logger.Log("DeleteFeatureFlagOverride correctly requires authentication")
			})
		})

		Describe("EvaluateFeatureFlag", func() {
			It("Should require authentication", func() {
				// Arrange
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/evaluate/my-flag", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				// Don't set auth header

				// Act
				EvaluateFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User token required")

				logger.Log("EvaluateFeatureFlag correctly requires authentication")
			})
		})

		Describe("GetFeatureFlag", func() {
			It("Should require authentication", func() {
				// Arrange
				restore := WithAuthCheckEnabled()
				defer restore()

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/my-flag", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				// Don't set auth header

				// Act
				GetFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User token required")

				logger.Log("GetFeatureFlag correctly requires authentication")
			})
		})

		Describe("SetFeatureFlagOverride", func() {
			It("Should require authentication", func() {
				// Arrange
				restore := WithAuthCheckEnabled()
				defer restore()

				requestBody := map[string]interface{}{
					"enabled": true,
				}
				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feature-flags/my-flag/override", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				// Don't set auth header

				// Act
				SetFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
				httpUtils.AssertErrorMessage("User token required")

				logger.Log("SetFeatureFlagOverride correctly requires authentication")
			})
		})
	})

	Context("ConfigMap Operations", func() {
		Describe("EnableFeatureFlag", func() {
			It("Should create ConfigMap when none exists and enable flag", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/enable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				EnableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["message"]).To(Equal("Feature enabled"))
				Expect(response["flag"]).To(Equal("my-flag"))
				Expect(response["enabled"]).To(Equal(true))
				Expect(response["source"]).To(Equal("workspace-override"))

				// Verify ConfigMap was created
				ctx := context.Background()
				cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
					ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(cm.Data["my-flag"]).To(Equal("true"))
				Expect(cm.Labels["app.kubernetes.io/managed-by"]).To(Equal("ambient-code"))
				Expect(cm.Labels["app.kubernetes.io/component"]).To(Equal("feature-flags"))

				logger.Log("Successfully created ConfigMap and enabled flag")
			})

			It("Should update existing ConfigMap when enabling flag", func() {
				// Arrange - create existing ConfigMap
				ctx := context.Background()
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FeatureFlagOverridesConfigMap,
						Namespace: "test-project",
					},
					Data: map[string]string{
						"other-flag": "true",
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
					ctx, existingCM, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/enable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				EnableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				// Verify both flags exist in ConfigMap
				cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
					ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(cm.Data["my-flag"]).To(Equal("true"))
				Expect(cm.Data["other-flag"]).To(Equal("true"))

				logger.Log("Successfully updated existing ConfigMap")
			})

			It("Should require flag name parameter", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags//enable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				EnableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Flag name is required")

				logger.Log("Correctly validated flag name requirement")
			})
		})

		Describe("DisableFeatureFlag", func() {
			It("Should create ConfigMap when none exists and disable flag", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/disable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				DisableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["message"]).To(Equal("Feature disabled"))
				Expect(response["flag"]).To(Equal("my-flag"))
				Expect(response["enabled"]).To(Equal(false))
				Expect(response["source"]).To(Equal("workspace-override"))

				// Verify ConfigMap was created with flag disabled
				ctx := context.Background()
				cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
					ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(cm.Data["my-flag"]).To(Equal("false"))

				logger.Log("Successfully created ConfigMap and disabled flag")
			})

			It("Should update existing ConfigMap when disabling flag", func() {
				// Arrange - create existing ConfigMap with flag enabled
				ctx := context.Background()
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FeatureFlagOverridesConfigMap,
						Namespace: "test-project",
					},
					Data: map[string]string{
						"my-flag": "true",
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
					ctx, existingCM, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/disable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				DisableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				// Verify flag is now disabled
				cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
					ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(cm.Data["my-flag"]).To(Equal("false"))

				logger.Log("Successfully updated existing ConfigMap to disable flag")
			})

			It("Should require flag name parameter", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags//disable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				DisableFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Flag name is required")

				logger.Log("Correctly validated flag name requirement")
			})
		})

		Describe("DeleteFeatureFlagOverride", func() {
			It("Should return success when no ConfigMap exists", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/feature-flags/my-flag/override", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				DeleteFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["message"]).To(Equal("No override to remove"))
				Expect(response["flag"]).To(Equal("my-flag"))

				logger.Log("Correctly handled missing ConfigMap")
			})

			It("Should remove flag from ConfigMap", func() {
				// Arrange - create ConfigMap with flag
				ctx := context.Background()
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FeatureFlagOverridesConfigMap,
						Namespace: "test-project",
					},
					Data: map[string]string{
						"my-flag":    "true",
						"other-flag": "false",
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
					ctx, existingCM, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginCtx := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/feature-flags/my-flag/override", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				DeleteFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["message"]).To(Equal("Override removed"))
				Expect(response["flag"]).To(Equal("my-flag"))
				Expect(response["source"]).To(Equal("unleash"))

				// Verify flag was removed but other flags remain
				cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
					ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(cm.Data).NotTo(HaveKey("my-flag"))
				Expect(cm.Data["other-flag"]).To(Equal("false"))

				logger.Log("Successfully removed flag override from ConfigMap")
			})

			It("Should require flag name parameter", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("DELETE", "/api/projects/test-project/feature-flags//override", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				DeleteFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Flag name is required")

				logger.Log("Correctly validated flag name requirement")
			})
		})

		Describe("SetFeatureFlagOverride", func() {
			It("Should create ConfigMap and set override when enabling", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"enabled": true,
				}
				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feature-flags/my-flag/override", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				SetFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["message"]).To(Equal("Override set"))
				Expect(response["flag"]).To(Equal("my-flag"))
				Expect(response["enabled"]).To(Equal(true))

				// Verify ConfigMap was created
				ctx := context.Background()
				cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
					ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(cm.Data["my-flag"]).To(Equal("true"))

				logger.Log("Successfully set override to enable")
			})

			It("Should update ConfigMap when disabling via override", func() {
				// Arrange - create existing ConfigMap with flag enabled
				ctx := context.Background()
				existingCM := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FeatureFlagOverridesConfigMap,
						Namespace: "test-project",
					},
					Data: map[string]string{
						"my-flag": "true",
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
					ctx, existingCM, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				requestBody := map[string]interface{}{
					"enabled": false,
				}
				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feature-flags/my-flag/override", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				SetFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["enabled"]).To(Equal(false))

				// Verify flag is now disabled
				cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
					ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(cm.Data["my-flag"]).To(Equal("false"))

				logger.Log("Successfully set override to disable")
			})

			It("Should require valid JSON body", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feature-flags/my-flag/override", "invalid-json")
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				SetFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Invalid request body")

				logger.Log("Correctly rejected invalid JSON body")
			})

			It("Should require flag name parameter", func() {
				// Arrange
				requestBody := map[string]interface{}{
					"enabled": true,
				}
				ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/feature-flags//override", requestBody)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				SetFeatureFlagOverride(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Flag name is required")

				logger.Log("Correctly validated flag name requirement")
			})
		})
	})

	Context("Unleash API Handling", func() {
		Describe("ListFeatureFlags", func() {
			It("Should return 503 when Unleash is not configured", func() {
				// Arrange - ensure no Unleash config exists (default state)
				// Note: In unit tests, Unleash env vars are not set
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				ListFeatureFlags(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusServiceUnavailable)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["error"]).To(Equal("Unleash Admin API not configured"))

				logger.Log("Correctly returned 503 when Unleash not configured")
			})

			It("Should return workspace overrides when Unleash not configured but overrides exist", func() {
				// Arrange - create ConfigMap with overrides
				ctx := context.Background()
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FeatureFlagOverridesConfigMap,
						Namespace: "test-project",
					},
					Data: map[string]string{
						"my-flag":      "true",
						"another-flag": "false",
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
					ctx, cm, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				ListFeatureFlags(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response).To(HaveKey("features"))

				features := response["features"].([]interface{})
				Expect(features).To(HaveLen(2))

				// Check that features are returned with correct source
				flagNames := make(map[string]bool)
				for _, f := range features {
					feature := f.(map[string]interface{})
					flagNames[feature["name"].(string)] = feature["enabled"].(bool)
					Expect(feature["source"]).To(Equal("workspace-override"))
				}
				Expect(flagNames["my-flag"]).To(BeTrue())
				Expect(flagNames["another-flag"]).To(BeFalse())

				logger.Log("Correctly returned workspace overrides when Unleash not configured")
			})
		})

		Describe("GetFeatureFlag", func() {
			It("Should return 503 when Unleash is not configured", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/my-flag", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				GetFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusServiceUnavailable)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["error"]).To(Equal("Unleash Admin API not configured"))

				logger.Log("Correctly returned 503 when Unleash not configured")
			})

			It("Should require flag name parameter", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				GetFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Flag name is required")

				logger.Log("Correctly validated flag name requirement")
			})
		})

		Describe("EvaluateFeatureFlag", func() {
			It("Should return workspace override when present", func() {
				// Arrange - create ConfigMap with override
				ctx := context.Background()
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FeatureFlagOverridesConfigMap,
						Namespace: "test-project",
					},
					Data: map[string]string{
						"my-flag": "true",
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
					ctx, cm, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/evaluate/my-flag", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				EvaluateFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["flag"]).To(Equal("my-flag"))
				Expect(response["enabled"]).To(Equal(true))
				Expect(response["source"]).To(Equal("workspace-override"))

				logger.Log("Correctly evaluated flag with workspace override")
			})

			It("Should return default disabled when no override and Unleash not configured", func() {
				// Arrange - no ConfigMap, no Unleash
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/evaluate/my-flag", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "my-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				EvaluateFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusOK)

				var response map[string]interface{}
				httpUtils.GetResponseJSON(&response)
				Expect(response["flag"]).To(Equal("my-flag"))
				Expect(response["enabled"]).To(Equal(false))
				Expect(response["source"]).To(Equal("default"))

				logger.Log("Correctly returned default disabled when nothing configured")
			})

			It("Should require flag name parameter", func() {
				// Arrange
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/evaluate/", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: ""},
				}
				httpUtils.SetAuthHeader(testToken)

				// Act
				EvaluateFeatureFlag(ginCtx)

				// Assert
				httpUtils.AssertHTTPStatus(http.StatusBadRequest)
				httpUtils.AssertErrorMessage("Flag name is required")

				logger.Log("Correctly validated flag name requirement")
			})
		})
	})

	Context("Override Precedence", func() {
		Describe("Workspace override takes precedence", func() {
			It("Should respect workspace override over Unleash default in evaluation", func() {
				// Arrange - create workspace override
				ctx := context.Background()
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      FeatureFlagOverridesConfigMap,
						Namespace: "test-project",
					},
					Data: map[string]string{
						"feature-a": "true",  // Override to enabled
						"feature-b": "false", // Override to disabled
					},
				}
				_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
					ctx, cm, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Test feature-a evaluation
				ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/evaluate/feature-a", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "feature-a"},
				}
				httpUtils.SetAuthHeader(testToken)

				EvaluateFeatureFlag(ginCtx)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				var responseA map[string]interface{}
				httpUtils.GetResponseJSON(&responseA)
				Expect(responseA["enabled"]).To(Equal(true))
				Expect(responseA["source"]).To(Equal("workspace-override"))

				// Test feature-b evaluation
				httpUtils = test_utils.NewHTTPTestUtils() // Reset
				ginCtx = httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/evaluate/feature-b", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "feature-b"},
				}
				httpUtils.SetAuthHeader(testToken)

				EvaluateFeatureFlag(ginCtx)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				var responseB map[string]interface{}
				httpUtils.GetResponseJSON(&responseB)
				Expect(responseB["enabled"]).To(Equal(false))
				Expect(responseB["source"]).To(Equal("workspace-override"))

				// Test feature-c (no override) - should use default
				httpUtils = test_utils.NewHTTPTestUtils() // Reset
				ginCtx = httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/feature-flags/evaluate/feature-c", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "feature-c"},
				}
				httpUtils.SetAuthHeader(testToken)

				EvaluateFeatureFlag(ginCtx)

				httpUtils.AssertHTTPStatus(http.StatusOK)
				var responseC map[string]interface{}
				httpUtils.GetResponseJSON(&responseC)
				Expect(responseC["enabled"]).To(Equal(false))
				Expect(responseC["source"]).To(Equal("default"))

				logger.Log("Successfully verified override precedence logic")
			})
		})
	})

	Context("Error Handling", func() {
		It("Should handle empty ConfigMap data gracefully in enable", func() {
			// Arrange - create ConfigMap with nil Data
			ctx := context.Background()
			existingCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FeatureFlagOverridesConfigMap,
					Namespace: "test-project",
				},
				// Data is nil
			}
			_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
				ctx, existingCM, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/enable", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
				{Key: "flagName", Value: "my-flag"},
			}
			httpUtils.SetAuthHeader(testToken)

			// Act
			EnableFeatureFlag(ginCtx)

			// Assert - should handle nil Data gracefully
			httpUtils.AssertHTTPStatus(http.StatusOK)

			cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
				ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Data["my-flag"]).To(Equal("true"))

			logger.Log("Handled nil ConfigMap data gracefully")
		})

		It("Should handle empty ConfigMap data gracefully in disable", func() {
			// Arrange - create ConfigMap with nil Data
			ctx := context.Background()
			existingCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FeatureFlagOverridesConfigMap,
					Namespace: "test-project",
				},
				// Data is nil
			}
			_, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Create(
				ctx, existingCM, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())

			ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/my-flag/disable", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
				{Key: "flagName", Value: "my-flag"},
			}
			httpUtils.SetAuthHeader(testToken)

			// Act
			DisableFeatureFlag(ginCtx)

			// Assert - should handle nil Data gracefully
			httpUtils.AssertHTTPStatus(http.StatusOK)

			cm, err := fakeClients.GetK8sClient().CoreV1().ConfigMaps("test-project").Get(
				ctx, FeatureFlagOverridesConfigMap, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Data["my-flag"]).To(Equal("false"))

			logger.Log("Handled nil ConfigMap data gracefully")
		})

		It("Should handle concurrent operations", func() {
			// Test that multiple requests don't cause issues
			for i := 0; i < 3; i++ {
				httpUtils = test_utils.NewHTTPTestUtils() // Reset for each test

				ginCtx := httpUtils.CreateTestGinContext("POST", "/api/projects/test-project/feature-flags/concurrent-flag/enable", nil)
				ginCtx.Params = gin.Params{
					{Key: "projectName", Value: "test-project"},
					{Key: "flagName", Value: "concurrent-flag"},
				}
				httpUtils.SetAuthHeader(testToken)

				EnableFeatureFlag(ginCtx)

				// Each request should be handled independently without errors
				status := httpUtils.GetResponseRecorder().Code
				Expect(status).To(BeElementOf(http.StatusOK, http.StatusInternalServerError))

				logger.Log("Concurrent request %d handled successfully", i+1)
			}
		})
	})
})
