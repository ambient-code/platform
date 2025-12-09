package k8sclient_test

import (
	"net/http/httptest"
	"os"
	"testing"

	"ambient-code-backend/handlers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func buildTestKubeConfig(t *testing.T) *rest.Config {
	t.Helper()

	// Try in-cluster config first, then fall back to kubeconfig on disk.
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg
	}

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		require.NoError(t, err)
		kubeconfig = home + "/.kube/config"
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	require.NoError(t, err)
	return cfg
}

func TestGetK8sClientsForRequest_ReturnsClientsetInRealCluster(t *testing.T) {
	// Match existing integration-test pattern in this repo.
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run")
	}

	token := os.Getenv("K8S_TEST_TOKEN")
	if token == "" {
		t.Skip("Skipping: K8S_TEST_TOKEN not set")
	}

	namespace := os.Getenv("K8S_TEST_NAMESPACE")
	if namespace == "" {
		t.Skip("Skipping: K8S_TEST_NAMESPACE not set")
	}

	cfg := buildTestKubeConfig(t)
	// Keep defaults small; this test should be cheap.
	cfg.QPS = 10
	cfg.Burst = 20

	originalBase := handlers.BaseKubeConfig
	handlers.BaseKubeConfig = cfg
	t.Cleanup(func() { handlers.BaseKubeConfig = originalBase })

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	typed, dyn := handlers.GetK8sClientsForRequest(c)
	require.NotNil(t, typed, "expected typed client")
	require.NotNil(t, dyn, "expected dynamic client")

	// Proof that production behavior remains a real clientset (no functional change),
	// even though the return type is kubernetes.Interface.
	_, ok := typed.(*kubernetes.Clientset)
	require.True(t, ok, "expected kubernetes.Interface to be backed by *kubernetes.Clientset")

	// Exercise a namespaced API call via kubernetes.Interface to prove callers work with the interface.
	// Note: permission requirements depend on the provided token.
	_, err := typed.CoreV1().ConfigMaps(namespace).List(c.Request.Context(), metav1.ListOptions{Limit: 1})
	require.NoError(t, err, "expected to be able to list ConfigMaps in the test namespace via kubernetes.Interface")
}

func TestGetK8sClientsForRequest_NoAuthHeader_ReturnsNil(t *testing.T) {
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TESTS=true to run")
	}

	cfg := buildTestKubeConfig(t)
	originalBase := handlers.BaseKubeConfig
	handlers.BaseKubeConfig = cfg
	t.Cleanup(func() { handlers.BaseKubeConfig = originalBase })

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest("GET", "/test", nil)

	typed, dyn := handlers.GetK8sClientsForRequest(c)
	require.Nil(t, typed)
	require.Nil(t, dyn)
}
