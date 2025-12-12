//go:build !test

package handlers

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// GetK8sClientsForRequest returns K8s typed and dynamic clients using the caller's token when provided.
// It supports both Authorization: Bearer and X-Forwarded-Access-Token and NEVER falls back to the backend service account.
// Returns nil, nil if no valid user token is provided - all API operations require user authentication.
// Returns kubernetes.Interface (not *kubernetes.Clientset) to support both real and fake clients in tests.
//
// SECURITY: Production authentication path is immutable (no function-pointer indirection).
func GetK8sClientsForRequest(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
	return getK8sClientsDefault(c)
}
