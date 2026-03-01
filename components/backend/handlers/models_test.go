//go:build test

package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	test_constants "ambient-code-backend/tests/constants"
	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Models Handler", Label(test_constants.LabelUnit, test_constants.LabelHandlers), func() {
	var (
		httpTestUtils       *test_utils.HTTPTestUtils
		originalK8s         = K8sClient
		originalNs          = Namespace
		originalK8sClientMw = K8sClientMw
		originalDynClient   = DynamicClient
		validManifest       string
	)

	validManifestObj := types.ModelManifest{
		Version:      1,
		DefaultModel: "claude-sonnet-4-5",
		Models: []types.ModelEntry{
			{ID: "claude-sonnet-4-5", Label: "Claude Sonnet 4.5", VertexID: "claude-sonnet-4-5@20250929", Provider: "anthropic", Tier: "standard", Available: true},
			{ID: "claude-opus-4-6", Label: "Claude Opus 4.6", VertexID: "claude-opus-4-6@default", Provider: "anthropic", Tier: "premium", Available: true},
			{ID: "claude-opus-4-5", Label: "Claude Opus 4.5", VertexID: "claude-opus-4-5@20251101", Provider: "anthropic", Tier: "premium", Available: true},
			{ID: "claude-haiku-4-5", Label: "Claude Haiku 4.5", VertexID: "claude-haiku-4-5@20251001", Provider: "anthropic", Tier: "standard", Available: true},
		},
	}

	BeforeEach(func() {
		httpTestUtils = test_utils.NewHTTPTestUtils()
		manifestBytes, err := json.Marshal(validManifestObj)
		Expect(err).NotTo(HaveOccurred())
		validManifest = string(manifestBytes)
	})

	AfterEach(func() {
		K8sClient = originalK8s
		Namespace = originalNs
		K8sClientMw = originalK8sClientMw
		DynamicClient = originalDynClient
		cachedManifest.Store(nil)
		os.Unsetenv("MODELS_MANIFEST_PATH")
	})

	// writeManifestFile writes manifest JSON to a temp file and sets the
	// MODELS_MANIFEST_PATH env var so ManifestPath() returns it.
	writeManifestFile := func(data string) string {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "models.json")
		err := os.WriteFile(path, []byte(data), 0644)
		Expect(err).NotTo(HaveOccurred())
		os.Setenv("MODELS_MANIFEST_PATH", path)
		return path
	}

	// setupK8sWithOverrides creates a K8s fake client with optional override ConfigMaps.
	// This is still needed because getWorkspaceOverrides reads via the K8s API.
	setupK8sWithOverrides := func(overrideCMs ...*corev1.ConfigMap) kubernetes.Interface {
		var objs []runtime.Object
		for _, cm := range overrideCMs {
			objs = append(objs, cm)
		}
		fakeClient := k8sfake.NewSimpleClientset(objs...)
		K8sClient = fakeClient
		Namespace = "test-ns"
		return fakeClient
	}

	// setupAuth sets K8sClientMw and DynamicClient so the test-tag
	// GetK8sClientsForRequest returns non-nil when a token is present.
	setupAuth := func(k8s kubernetes.Interface) {
		K8sClientMw = k8s
		DynamicClient = dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	}

	// createAuthenticatedContext builds a Gin context with auth header and
	// projectName param, suitable for ListModelsForProject.
	createAuthenticatedContext := func(projectName string) *gin.Context {
		ginCtx := httpTestUtils.CreateTestGinContext("GET", "/api/projects/"+projectName+"/models", nil)
		httpTestUtils.SetAuthHeader("test-token")
		ginCtx.Params = gin.Params{{Key: "projectName", Value: projectName}}
		return ginCtx
	}

	Context("ListModelsForProject", func() {
		It("should return 401 when no auth token is provided", func() {
			logger.Log("Testing ListModelsForProject rejects unauthenticated requests")
			writeManifestFile(validManifest)

			ginCtx := httpTestUtils.CreateTestGinContext("GET", "/api/projects/test-project/models", nil)
			ginCtx.Params = gin.Params{{Key: "projectName", Value: "test-project"}}
			// No auth header set

			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusUnauthorized)
		})

		It("should return all models when no workspace overrides exist", func() {
			logger.Log("Testing ListModelsForProject with valid manifest and no overrides")
			writeManifestFile(validManifest)
			fakeClient := setupK8sWithOverrides()
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())
			// With no Unleash configured, IsModelEnabled returns true, so all 4 models pass
			Expect(resp.Models).To(HaveLen(4))
			Expect(resp.DefaultModel).To(Equal("claude-sonnet-4-5"))
		})

		It("should include model when workspace override is true", func() {
			logger.Log("Testing workspace override=true includes model regardless of Unleash")
			writeManifestFile(validManifest)
			overrideCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FeatureFlagOverridesConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"model.claude-opus-4-6.enabled": "true",
				},
			}
			fakeClient := setupK8sWithOverrides(overrideCM)
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			var found bool
			for _, m := range resp.Models {
				if m.ID == "claude-opus-4-6" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Workspace override=true should include the model")
		})

		It("should exclude model when workspace override is false", func() {
			logger.Log("Testing workspace override=false excludes model regardless of Unleash")
			writeManifestFile(validManifest)
			overrideCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FeatureFlagOverridesConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"model.claude-opus-4-6.enabled": "false",
				},
			}
			fakeClient := setupK8sWithOverrides(overrideCM)
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			for _, m := range resp.Models {
				Expect(m.ID).NotTo(Equal("claude-opus-4-6"),
					"Workspace override=false should exclude the model")
			}
		})

		It("should fall back to Unleash when no workspace override exists for a flag", func() {
			logger.Log("Testing fallback to Unleash when override is absent")
			writeManifestFile(validManifest)
			// Override only one flag; the others should use Unleash fallback
			overrideCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FeatureFlagOverridesConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"model.claude-opus-4-6.enabled": "false",
				},
			}
			fakeClient := setupK8sWithOverrides(overrideCM)
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			// opus-4-6 excluded by override; the other 3 should still be present
			// (default model + 2 non-default models via Unleash fallback which returns true when not configured)
			Expect(resp.Models).To(HaveLen(3))
			ids := make([]string, len(resp.Models))
			for i, m := range resp.Models {
				ids[i] = m.ID
			}
			Expect(ids).To(ContainElement("claude-sonnet-4-5"))
			Expect(ids).To(ContainElement("claude-opus-4-5"))
			Expect(ids).To(ContainElement("claude-haiku-4-5"))
			Expect(ids).NotTo(ContainElement("claude-opus-4-6"))
		})

		It("should always include the default model even with overrides", func() {
			logger.Log("Testing that default model is always included regardless of overrides")
			writeManifestFile(validManifest)
			fakeClient := setupK8sWithOverrides()
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())

			var foundDefault bool
			for _, m := range resp.Models {
				if m.ID == "claude-sonnet-4-5" && m.IsDefault {
					foundDefault = true
					break
				}
			}
			Expect(foundDefault).To(BeTrue(), "Default model should always be present")
		})

		It("should exclude models where available is false", func() {
			logger.Log("Testing that unavailable models are excluded")
			manifest := validManifestObj
			modelsCopy := make([]types.ModelEntry, len(manifest.Models))
			copy(modelsCopy, manifest.Models)
			manifest.Models = modelsCopy
			manifest.Models[1].Available = false // claude-opus-4-6

			manifestBytes, err := json.Marshal(manifest)
			Expect(err).NotTo(HaveOccurred())
			writeManifestFile(string(manifestBytes))
			fakeClient := setupK8sWithOverrides()
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err = json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Models).To(HaveLen(3))

			for _, m := range resp.Models {
				Expect(m.ID).NotTo(Equal("claude-opus-4-6"))
			}
		})

		It("should return hardcoded defaults when manifest file is missing and no cache", func() {
			logger.Log("Testing ListModelsForProject fallback when manifest file missing and no cache")
			os.Setenv("MODELS_MANIFEST_PATH", filepath.Join(GinkgoT().TempDir(), "nonexistent.json"))
			fakeClient := setupK8sWithOverrides()
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Models).To(HaveLen(5))
			Expect(resp.DefaultModel).To(Equal("claude-sonnet-4-5"))
		})

		It("should use cached manifest when file becomes unavailable", func() {
			logger.Log("Testing ListModelsForProject uses cached manifest on read error")
			// First call: load a valid manifest so it gets cached
			writeManifestFile(validManifest)
			fakeClient := setupK8sWithOverrides()
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)
			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			// Second call: point to a missing file — should use cached manifest
			os.Setenv("MODELS_MANIFEST_PATH", filepath.Join(GinkgoT().TempDir(), "gone.json"))
			httpTestUtils = test_utils.NewHTTPTestUtils()
			setupAuth(fakeClient)

			ginCtx = createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())
			// Cached manifest has 4 models (not 5 hardcoded defaults),
			// and they go through flag filtering
			Expect(resp.Models).To(HaveLen(4))
			Expect(resp.DefaultModel).To(Equal("claude-sonnet-4-5"))
		})

		It("should return hardcoded defaults when JSON is malformed and no cache", func() {
			logger.Log("Testing ListModelsForProject fallback with malformed JSON and no cache")
			writeManifestFile("{invalid json")
			fakeClient := setupK8sWithOverrides()
			setupAuth(fakeClient)

			ginCtx := createAuthenticatedContext("test-project")
			ListModelsForProject(ginCtx)

			httpTestUtils.AssertHTTPStatus(http.StatusOK)

			var resp types.ListModelsResponse
			err := json.Unmarshal(httpTestUtils.GetResponseRecorder().Body.Bytes(), &resp)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Models).To(HaveLen(5))
			Expect(resp.DefaultModel).To(Equal("claude-sonnet-4-5"))
		})
	})

	Context("isModelEnabledWithOverrides", func() {
		It("should return true when override is true", func() {
			overrides := map[string]string{"model.test.enabled": "true"}
			Expect(isModelEnabledWithOverrides("model.test.enabled", overrides)).To(BeTrue())
		})

		It("should return false when override is false", func() {
			overrides := map[string]string{"model.test.enabled": "false"}
			Expect(isModelEnabledWithOverrides("model.test.enabled", overrides)).To(BeFalse())
		})

		It("should fall back to Unleash when flag is not in overrides", func() {
			overrides := map[string]string{"model.other.enabled": "false"}
			// featureflags.IsModelEnabled returns true when Unleash is not configured
			Expect(isModelEnabledWithOverrides("model.test.enabled", overrides)).To(BeTrue())
		})

		It("should fall back to Unleash when overrides map is nil", func() {
			// featureflags.IsModelEnabled returns true when Unleash is not configured
			Expect(isModelEnabledWithOverrides("model.test.enabled", nil)).To(BeTrue())
		})

		It("should treat non-true string values as false", func() {
			overrides := map[string]string{"model.test.enabled": "yes"}
			Expect(isModelEnabledWithOverrides("model.test.enabled", overrides)).To(BeFalse())
		})
	})

	Context("LoadManifest", func() {
		It("should parse valid manifest JSON", func() {
			path := writeManifestFile(validManifest)

			manifest, err := LoadManifest(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(manifest.Version).To(Equal(1))
			Expect(manifest.DefaultModel).To(Equal("claude-sonnet-4-5"))
			Expect(manifest.Models).To(HaveLen(4))
		})

		It("should return error when file is missing", func() {
			path := filepath.Join(GinkgoT().TempDir(), "nonexistent.json")

			_, err := LoadManifest(path)
			Expect(err).To(HaveOccurred())
		})

		It("should return error when JSON is malformed", func() {
			path := writeManifestFile("{invalid json")

			_, err := LoadManifest(path)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("isModelAvailable", func() {
		It("should return true for empty model ID", func() {
			logger.Log("Testing isModelAvailable with empty model ID")
			result := isModelAvailable(context.Background(), K8sClient, "", "test-ns")
			Expect(result).To(BeTrue())
		})

		It("should return true for the default model", func() {
			logger.Log("Testing isModelAvailable with default model")
			writeManifestFile(validManifest)
			setupK8sWithOverrides()

			result := isModelAvailable(context.Background(), K8sClient, "claude-sonnet-4-5", "test-ns")
			Expect(result).To(BeTrue())
		})

		It("should return true for an available non-default model (no Unleash)", func() {
			logger.Log("Testing isModelAvailable with available model, Unleash not configured")
			writeManifestFile(validManifest)
			setupK8sWithOverrides()

			result := isModelAvailable(context.Background(), K8sClient, "claude-opus-4-6", "test-ns")
			Expect(result).To(BeTrue())
		})

		It("should return false for a model with available=false", func() {
			logger.Log("Testing isModelAvailable with unavailable model")
			manifest := validManifestObj
			modelsCopy := make([]types.ModelEntry, len(manifest.Models))
			copy(modelsCopy, manifest.Models)
			manifest.Models = modelsCopy
			manifest.Models[1].Available = false // claude-opus-4-6

			manifestBytes, err := json.Marshal(manifest)
			Expect(err).NotTo(HaveOccurred())
			writeManifestFile(string(manifestBytes))
			setupK8sWithOverrides()

			result := isModelAvailable(context.Background(), K8sClient, "claude-opus-4-6", "test-ns")
			Expect(result).To(BeFalse())
		})

		It("should reject when model is not found in manifest", func() {
			logger.Log("Testing isModelAvailable rejects unknown model")
			writeManifestFile(validManifest)
			setupK8sWithOverrides()

			result := isModelAvailable(context.Background(), K8sClient, "nonexistent-model", "test-ns")
			Expect(result).To(BeFalse())
		})

		It("should fail-open when manifest file is missing", func() {
			logger.Log("Testing isModelAvailable fail-open when manifest file missing")
			os.Setenv("MODELS_MANIFEST_PATH", filepath.Join(GinkgoT().TempDir(), "nonexistent.json"))

			result := isModelAvailable(context.Background(), K8sClient, "claude-opus-4-6", "test-ns")
			Expect(result).To(BeTrue())
		})

		It("should return false when workspace override disables the model", func() {
			logger.Log("Testing isModelAvailable respects workspace override=false")
			writeManifestFile(validManifest)
			overrideCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FeatureFlagOverridesConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"model.claude-opus-4-6.enabled": "false",
				},
			}
			setupK8sWithOverrides(overrideCM)

			result := isModelAvailable(context.Background(), K8sClient, "claude-opus-4-6", "test-project")
			Expect(result).To(BeFalse())
		})

		It("should return true when workspace override enables the model", func() {
			logger.Log("Testing isModelAvailable respects workspace override=true")
			writeManifestFile(validManifest)
			overrideCM := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      FeatureFlagOverridesConfigMap,
					Namespace: "test-project",
				},
				Data: map[string]string{
					"model.claude-opus-4-6.enabled": "true",
				},
			}
			setupK8sWithOverrides(overrideCM)

			result := isModelAvailable(context.Background(), K8sClient, "claude-opus-4-6", "test-project")
			Expect(result).To(BeTrue())
		})
	})
})
