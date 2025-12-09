//go:build test

package handlers

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// GetK8sClientsForRequest is the test-build implementation.
//
// SECURITY NOTE:
//   - There is NO function-pointer override hook (to avoid leaking behavior across tests).
//   - Tests provide fake clients via package-level dependency setup (e.g. SetupHandlerDependencies),
//     which sets K8sClientMw and DynamicClient to fake clients.
//   - We still enforce "token present" semantics: missing/invalid tokens return nil clients.
func GetK8sClientsForRequest(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
	token, _, _, _ := extractRequestToken(c)

	// Enforce "token required" semantics in tests too (same as production behavior).
	if token == "" {
		return nil, nil
	}
	// Deterministic invalid-token sentinel used by unit tests.
	if token == "invalid-token" {
		return nil, nil
	}

	// Return the fake clients set up by unit tests.
	if K8sClientMw == nil || DynamicClient == nil {
		// If a test didn't set up fake clients (or is intentionally exercising the real auth path),
		// fall back to the normal implementation.
		return getK8sClientsDefault(c)
	}
	return K8sClientMw, DynamicClient
}
