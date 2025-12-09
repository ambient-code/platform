package handlers

import (
	"context"
	"strings"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var restoreK8sClientsForRequestHook func()

// SetupHandlerDependencies sets up package-level variables that handlers depend on for unit tests.
// Tests are now in the handlers package, so this avoids import cycles while keeping a single setup path.
func SetupHandlerDependencies(k8sUtils *test_utils.K8sTestUtils) {
	// Core clients used by handlers
	DynamicClient = k8sUtils.DynamicClient
	K8sClientProjects = k8sUtils.K8sClient
	DynamicClientProjects = k8sUtils.DynamicClient
	K8sClientMw = k8sUtils.K8sClient
	K8sClient = k8sUtils.K8sClient

	// Common GVR helpers used by sessions handlers
	GetAgenticSessionV1Alpha1Resource = func() schema.GroupVersionResource {
		return schema.GroupVersionResource{
			Group:    "vteam.ambient-code",
			Version:  "v1alpha1",
			Resource: "agenticsessions",
		}
	}

	// Default: require auth header and return fake clients.
	// Individual tests can loosen or tighten behavior by overriding the test-only hook.
	fakeClientset := k8sUtils.K8sClient
	if restoreK8sClientsForRequestHook != nil {
		restoreK8sClientsForRequestHook()
		restoreK8sClientsForRequestHook = nil
	}
	original := getK8sClientsForRequest
	getK8sClientsForRequest = func(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			auth = c.GetHeader("X-Forwarded-Access-Token")
		}
		if strings.TrimSpace(auth) == "" {
			return nil, nil
		}
		// Simulate invalid token scenarios deterministically in unit tests
		if strings.TrimSpace(auth) == "invalid-token" || strings.Contains(strings.TrimSpace(auth), "invalid-token") {
			return nil, nil
		}
		return fakeClientset, k8sUtils.DynamicClient
	}
	restoreK8sClientsForRequestHook = func() { getK8sClientsForRequest = original }

	// Other handler dependencies with safe defaults for unit tests
	GetGitHubToken = func(ctx context.Context, k8sClient kubernetes.Interface, dynClient dynamic.Interface, namespace, userID string) (string, error) {
		return "fake-github-token", nil
	}
	DeriveRepoFolderFromURL = func(url string) string {
		parts := strings.Split(url, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return "repo"
	}
	SendMessageToSession = func(sessionID, userID string, message map[string]interface{}) {
		// no-op in unit tests
	}

	logger.Log("Handler dependencies set up with fake clients")
}

// WithAuthCheckEnabled temporarily forces auth checks by returning nil clients when no auth header is present.
func WithAuthCheckEnabled() func() {
	original := getK8sClientsForRequest
	getK8sClientsForRequest = func(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
		auth := c.GetHeader("Authorization")
		if auth == "" {
			auth = c.GetHeader("X-Forwarded-Access-Token")
		}
		if strings.TrimSpace(auth) == "" {
			return nil, nil
		}
		return original(c)
	}
	return func() { getK8sClientsForRequest = original }
}

// WithAuthCheckDisabled restores the default behavior for the duration of a test.
func WithAuthCheckDisabled() func() {
	// No-op for now: SetupHandlerDependencies already installs the default test hook.
	return func() {}
}
