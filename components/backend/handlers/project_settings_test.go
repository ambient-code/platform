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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("ProjectSettings Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers, test_constants.LabelProjectSettings), func() {
	var (
		httpUtils *test_utils.HTTPTestUtils
		k8sUtils  *test_utils.K8sTestUtils
		testToken string
	)

	BeforeEach(func() {
		logger.Log("Setting up ProjectSettings Handler test")

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
		if k8sUtils != nil {
			_ = k8sUtils.K8sClient.CoreV1().Namespaces().Delete(context.Background(), "test-project", metav1.DeleteOptions{})
		}
	})

	// Helper to create a ProjectSettings CR in the fake cluster
	createProjectSettings := func(spec map[string]interface{}) {
		obj := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "vteam.ambient-code/v1alpha1",
				"kind":       "ProjectSettings",
				"metadata": map[string]interface{}{
					"name":      "projectsettings",
					"namespace": "test-project",
				},
				"spec": spec,
			},
		}
		k8sUtils.CreateCustomResource(context.Background(), projectSettingsGVR, "test-project", obj)
	}

	Context("GetProjectSettings", func() {
		It("Should return empty object when CR does not exist", func() {
			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/project-settings", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			GetProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			httpUtils.GetResponseJSON(&response)
			Expect(response).NotTo(HaveKey("defaultConfigRepo"), "Should not have defaultConfigRepo when CR does not exist")

			logger.Log("Correctly returned empty when ProjectSettings CR missing")
		})

		It("Should return defaultConfigRepo when set", func() {
			createProjectSettings(map[string]interface{}{
				"groupAccess": []interface{}{},
				"defaultConfigRepo": map[string]interface{}{
					"gitUrl": "https://github.com/org/session-config.git",
					"branch": "develop",
				},
			})

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/project-settings", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			GetProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			httpUtils.GetResponseJSON(&response)
			Expect(response).To(HaveKey("defaultConfigRepo"))

			configRepo := response["defaultConfigRepo"].(map[string]interface{})
			Expect(configRepo["gitUrl"]).To(Equal("https://github.com/org/session-config.git"))
			Expect(configRepo["branch"]).To(Equal("develop"))

			logger.Log("Successfully returned defaultConfigRepo")
		})

		It("Should return empty object when CR exists but no config repo set", func() {
			createProjectSettings(map[string]interface{}{
				"groupAccess": []interface{}{},
			})

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/project-settings", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			GetProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			httpUtils.GetResponseJSON(&response)
			Expect(response).NotTo(HaveKey("defaultConfigRepo"))

			logger.Log("Correctly returned empty when no config repo set")
		})

		It("Should require authentication", func() {
			restore := WithAuthCheckEnabled()
			defer restore()

			ginCtx := httpUtils.CreateTestGinContext("GET", "/api/projects/test-project/project-settings", nil)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}

			GetProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})
	})

	Context("UpdateProjectSettings", func() {
		It("Should set defaultConfigRepo on existing CR", func() {
			createProjectSettings(map[string]interface{}{
				"groupAccess": []interface{}{},
			})

			requestBody := map[string]interface{}{
				"defaultConfigRepo": map[string]interface{}{
					"gitUrl": "https://github.com/org/my-config.git",
					"branch": "main",
				},
			}

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/project-settings", requestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			httpUtils.GetResponseJSON(&response)
			Expect(response).To(HaveKey("defaultConfigRepo"))

			configRepo := response["defaultConfigRepo"].(map[string]interface{})
			Expect(configRepo["gitUrl"]).To(Equal("https://github.com/org/my-config.git"))
			Expect(configRepo["branch"]).To(Equal("main"))

			logger.Log("Successfully set defaultConfigRepo")
		})

		It("Should clear defaultConfigRepo when gitUrl is empty", func() {
			createProjectSettings(map[string]interface{}{
				"groupAccess": []interface{}{},
				"defaultConfigRepo": map[string]interface{}{
					"gitUrl": "https://github.com/org/old-config.git",
				},
			})

			requestBody := map[string]interface{}{
				"defaultConfigRepo": map[string]interface{}{
					"gitUrl": "",
				},
			}

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/project-settings", requestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)
			var response map[string]interface{}
			httpUtils.GetResponseJSON(&response)
			Expect(response).NotTo(HaveKey("defaultConfigRepo"))

			logger.Log("Successfully cleared defaultConfigRepo")
		})

		It("Should preserve existing spec fields like groupAccess", func() {
			createProjectSettings(map[string]interface{}{
				"groupAccess": []interface{}{
					map[string]interface{}{
						"groupName": "team-alpha",
						"role":      "edit",
					},
				},
				"runnerSecretsName": "my-secret",
			})

			requestBody := map[string]interface{}{
				"defaultConfigRepo": map[string]interface{}{
					"gitUrl": "https://github.com/org/config.git",
				},
			}

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/project-settings", requestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusOK)

			// Verify the CR still has groupAccess and runnerSecretsName
			ctx := context.Background()
			obj, err := k8sUtils.DynamicClient.Resource(projectSettingsGVR).Namespace("test-project").Get(ctx, "projectsettings", metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			groupAccess, found, _ := unstructured.NestedSlice(obj.Object, "spec", "groupAccess")
			Expect(found).To(BeTrue(), "groupAccess should be preserved")
			Expect(groupAccess).To(HaveLen(1))

			secretsName, found, _ := unstructured.NestedString(obj.Object, "spec", "runnerSecretsName")
			Expect(found).To(BeTrue(), "runnerSecretsName should be preserved")
			Expect(secretsName).To(Equal("my-secret"))

			logger.Log("Successfully preserved existing spec fields")
		})

		It("Should return 404 when CR does not exist", func() {
			requestBody := map[string]interface{}{
				"defaultConfigRepo": map[string]interface{}{
					"gitUrl": "https://github.com/org/config.git",
				},
			}

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/project-settings", requestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusNotFound)

			logger.Log("Correctly returned 404 when CR missing")
		})

		It("Should reject invalid JSON body", func() {
			createProjectSettings(map[string]interface{}{
				"groupAccess": []interface{}{},
			})

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/project-settings", "invalid-json")
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}
			httpUtils.SetAuthHeader(testToken)

			UpdateProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusBadRequest)

			logger.Log("Correctly rejected invalid JSON")
		})

		It("Should require authentication", func() {
			restore := WithAuthCheckEnabled()
			defer restore()

			requestBody := map[string]interface{}{
				"defaultConfigRepo": map[string]interface{}{
					"gitUrl": "https://github.com/org/config.git",
				},
			}

			ginCtx := httpUtils.CreateTestGinContext("PUT", "/api/projects/test-project/project-settings", requestBody)
			ginCtx.Params = gin.Params{
				{Key: "projectName", Value: "test-project"},
			}

			UpdateProjectSettings(ginCtx)

			httpUtils.AssertHTTPStatus(http.StatusUnauthorized)
			httpUtils.AssertErrorMessage("Invalid or missing token")
		})
	})
})
