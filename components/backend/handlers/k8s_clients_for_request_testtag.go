//go:build test

package handlers

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"strings"
)

// GetK8sClientsForRequest is the test-build implementation.
//
// SECURITY NOTE:
//   - There is NO function-pointer override hook (to avoid leaking behavior across tests).
//   - Tests provide fake clients via package-level dependency setup (e.g. SetupHandlerDependencies),
//     which sets K8sClientMw and DynamicClient to fake clients.
//   - We still enforce "token present" semantics: missing/invalid tokens return nil clients.
func GetK8sClientsForRequest(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
	// Mirror the production parsing behavior at a high level:
	// - accept Authorization header (Bearer <token> or raw token)
	// - accept X-Forwarded-Access-Token
	rawAuth := c.GetHeader("Authorization")
	rawFwd := c.GetHeader("X-Forwarded-Access-Token")
	token := rawAuth

	if token != "" {
		// Best-effort Bearer parsing (same behavior as prod impl)
		parts := strings.SplitN(token, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			token = strings.TrimSpace(parts[1])
		} else {
			token = strings.TrimSpace(token)
		}
	}
	if token == "" {
		token = strings.TrimSpace(rawFwd)
	}

	// Enforce "token required" semantics in tests too.
	if strings.TrimSpace(token) == "" {
		return nil, nil
	}
	if strings.TrimSpace(token) == "invalid-token" {
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
