package handlers

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Client selection functions - IMMUTABLE in production
// These functions always call GetK8sClientsForRequest which enforces authentication.
// They cannot be overridden - all authentication logic is centralized in GetK8sClientsForRequest.
// For testing, override GetK8sClientsFunc in middleware.go instead of these functions.

// GetK8sClientForPermissions returns K8s client for permissions operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetK8sClientForPermissions(c *gin.Context) kubernetes.Interface {
	reqK8s, _ := GetK8sClientsForRequest(c)
	return reqK8s // May be nil - callers must handle
}

// GetK8sClientForRepo returns K8s client for repository operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetK8sClientForRepo(c *gin.Context) kubernetes.Interface {
	reqK8s, _ := GetK8sClientsForRequest(c)
	return reqK8s // May be nil - callers must handle
}

// GetK8sClientForGitLab returns K8s client for GitLab operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetK8sClientForGitLab(c *gin.Context) kubernetes.Interface {
	reqK8s, _ := GetK8sClientsForRequest(c)
	return reqK8s // May be nil - callers must handle
}

// GetK8sClientForSessions returns K8s client for sessions operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetK8sClientForSessions(c *gin.Context) kubernetes.Interface {
	reqK8s, _ := GetK8sClientsForRequest(c)
	return reqK8s // May be nil - callers must handle
}

// GetDynamicClientForSessions returns dynamic client for sessions operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetDynamicClientForSessions(c *gin.Context) dynamic.Interface {
	_, reqDyn := GetK8sClientsForRequest(c)
	return reqDyn // May be nil - callers must handle
}

// GetK8sClientForProjects returns K8s client for projects operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetK8sClientForProjects(c *gin.Context) kubernetes.Interface {
	reqK8s, _ := GetK8sClientsForRequest(c)
	return reqK8s // May be nil - callers must handle
}

// GetDynamicClientForProjects returns dynamic client for projects operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetDynamicClientForProjects(c *gin.Context) dynamic.Interface {
	_, reqDyn := GetK8sClientsForRequest(c)
	return reqDyn // May be nil - callers must handle
}

// GetDynamicClientForRepo returns dynamic client for repository operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetDynamicClientForRepo(c *gin.Context) dynamic.Interface {
	_, reqDyn := GetK8sClientsForRequest(c)
	return reqDyn // May be nil - callers must handle
}

// GetK8sClientsForRequestRepo returns both K8s clients for repository operations
// Always calls GetK8sClientsForRequest which enforces user token authentication.
func GetK8sClientsForRequestRepo(c *gin.Context) (kubernetes.Interface, dynamic.Interface) {
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	return reqK8s, reqDyn // May be nil - callers must handle
}
