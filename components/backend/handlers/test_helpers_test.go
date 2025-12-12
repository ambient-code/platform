//go:build test

package handlers

import (
	"context"
	"strings"

	"ambient-code-backend/tests/logger"
	"ambient-code-backend/tests/test_utils"

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
	// Auth behavior is enforced by the -tags=test GetK8sClientsForRequest implementation:
	// it requires a token header and returns K8sClientMw/DynamicClient when present.
	restoreK8sClientsForRequestHook = nil

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
	// No-op: auth strictness is always enforced in the test build.
	return func() {}
}

// WithAuthCheckDisabled restores the default behavior for the duration of a test.
func WithAuthCheckDisabled() func() {
	// No-op for now: SetupHandlerDependencies already installs the default test hook.
	return func() {}
}
