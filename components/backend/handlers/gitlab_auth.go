package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"ambient-code-backend/gitlab"
)

// GitLabAuthHandler handles GitLab authentication endpoints
type GitLabAuthHandler struct {
	connectionManager *gitlab.ConnectionManager
}

// NewGitLabAuthHandler creates a new GitLab authentication handler
func NewGitLabAuthHandler(clientset kubernetes.Interface, namespace string) *GitLabAuthHandler {
	// Convert interface to concrete type for gitlab.NewConnectionManager
	var k8sClientset *kubernetes.Clientset
	if clientset != nil {
		if concrete, ok := clientset.(*kubernetes.Clientset); ok {
			k8sClientset = concrete
		}
		// For tests with fake clients, NewConnectionManager will handle nil gracefully
	}

	return &GitLabAuthHandler{
		connectionManager: gitlab.NewConnectionManager(k8sClientset, namespace),
	}
}

// ConnectGitLabRequest represents a request to connect a GitLab account
type ConnectGitLabRequest struct {
	PersonalAccessToken string `json:"personalAccessToken" binding:"required"`
	InstanceURL         string `json:"instanceUrl"`
}

// ConnectGitLabResponse represents the response from connecting a GitLab account
type ConnectGitLabResponse struct {
	UserID       string `json:"userId"`
	GitLabUserID string `json:"gitlabUserId"`
	Username     string `json:"username"`
	InstanceURL  string `json:"instanceUrl"`
	Connected    bool   `json:"connected"`
	Message      string `json:"message"`
}

// GitLabStatusResponse represents the GitLab connection status
type GitLabStatusResponse struct {
	Connected    bool   `json:"connected"`
	Username     string `json:"username,omitempty"`
	InstanceURL  string `json:"instanceUrl,omitempty"`
	GitLabUserID string `json:"gitlabUserId,omitempty"`
}

// validateGitLabInput validates GitLab connection request input
func validateGitLabInput(instanceURL, token string) error {
	// Validate instance URL
	if instanceURL != "" {
		parsedURL, err := url.Parse(instanceURL)
		if err != nil {
			return fmt.Errorf("invalid instance URL format")
		}

		// Require HTTPS for security
		if parsedURL.Scheme != "https" {
			return fmt.Errorf("instance URL must use HTTPS")
		}

		// Validate hostname is not empty
		if parsedURL.Host == "" {
			return fmt.Errorf("instance URL must have a valid hostname")
		}

		// Prevent common injection attempts - check both userinfo and hostname
		// url.Parse treats "user@host" as userinfo, so check both
		if parsedURL.User != nil && parsedURL.User.String() != "" {
			return fmt.Errorf("instance URL cannot contain user info (@ syntax)")
		}
		if strings.Contains(parsedURL.Host, "@") {
			return fmt.Errorf("instance URL hostname cannot contain '@'")
		}
	}

	// Validate token length (GitLab PATs are 20 chars, but allow for future changes)
	// Min: 20 chars, Max: 255 chars (reasonable upper bound)
	if len(token) < 20 {
		return fmt.Errorf("token must be at least 20 characters")
	}
	if len(token) > 255 {
		return fmt.Errorf("token must not exceed 255 characters")
	}

	// Validate token contains only valid characters (alphanumeric and some special chars)
	// GitLab tokens use: a-z, A-Z, 0-9, -, _, .
	for _, char := range token {
		if (char < 'a' || char > 'z') &&
			(char < 'A' || char > 'Z') &&
			(char < '0' || char > '9') &&
			char != '-' && char != '_' && char != '.' {
			return fmt.Errorf("token contains invalid characters")
		}
	}

	return nil
}

// ConnectGitLab handles POST /projects/:projectName/auth/gitlab/connect
func (h *GitLabAuthHandler) ConnectGitLab(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Project name is required",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	var req ConnectGitLabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Invalid request body",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Default to GitLab.com if no instance URL provided
	if req.InstanceURL == "" {
		req.InstanceURL = "https://gitlab.com"
	}

	// Validate input
	if err := validateGitLabInput(req.InstanceURL, req.PersonalAccessToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      fmt.Sprintf("Invalid input: %v", err),
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Get user ID from context (set by authentication middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "User not authenticated",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Invalid user ID format",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	// RBAC: Verify user can create/update secrets in this project
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "Invalid or missing token",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	ctx := c.Request.Context()
	if err := ValidateSecretAccess(ctx, reqK8s, project, "create"); err != nil {
		gitlab.LogError("RBAC check failed for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Insufficient permissions to manage GitLab credentials",
			"statusCode": http.StatusForbidden,
		})
		return
	}

	// Store GitLab connection (now project-scoped)
	connection, err := h.connectionManager.StoreGitLabConnection(ctx, userIDStr, req.PersonalAccessToken, req.InstanceURL)
	if err != nil {
		gitlab.LogError("Failed to store GitLab connection for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      err.Error(),
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, ConnectGitLabResponse{
		UserID:       connection.UserID,
		GitLabUserID: connection.GitLabUserID,
		Username:     connection.Username,
		InstanceURL:  connection.InstanceURL,
		Connected:    true,
		Message:      "GitLab account connected successfully to project " + project,
	})
}

// GetGitLabStatus handles GET /projects/:projectName/auth/gitlab/status
func (h *GitLabAuthHandler) GetGitLabStatus(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Project name is required",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "User not authenticated",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Invalid user ID format",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	// RBAC: Verify user can read secrets in this project
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "Invalid or missing token",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	ctx := c.Request.Context()
	if err := ValidateSecretAccess(ctx, reqK8s, project, "get"); err != nil {
		gitlab.LogError("RBAC check failed for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Insufficient permissions to read GitLab credentials",
			"statusCode": http.StatusForbidden,
		})
		return
	}

	// Get connection status (project-scoped)
	status, err := h.connectionManager.GetConnectionStatus(ctx, userIDStr)
	if err != nil {
		gitlab.LogError("Failed to get GitLab status for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to retrieve GitLab connection status",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	if !status.Connected {
		c.JSON(http.StatusOK, GitLabStatusResponse{
			Connected: false,
		})
		return
	}

	c.JSON(http.StatusOK, GitLabStatusResponse{
		Connected:    true,
		Username:     status.Username,
		InstanceURL:  status.InstanceURL,
		GitLabUserID: status.GitLabUserID,
	})
}

// DisconnectGitLab handles POST /projects/:projectName/auth/gitlab/disconnect
func (h *GitLabAuthHandler) DisconnectGitLab(c *gin.Context) {
	// Get project from URL parameter
	project := c.Param("projectName")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Project name is required",
			"statusCode": http.StatusBadRequest,
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "User not authenticated",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	userIDStr, ok := userID.(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Invalid user ID format",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	// RBAC: Verify user can update secrets in this project
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":      "Invalid or missing token",
			"statusCode": http.StatusUnauthorized,
		})
		return
	}

	ctx := c.Request.Context()
	if err := ValidateSecretAccess(ctx, reqK8s, project, "update"); err != nil {
		gitlab.LogError("RBAC check failed for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusForbidden, gin.H{
			"error":      "Insufficient permissions to manage GitLab credentials",
			"statusCode": http.StatusForbidden,
		})
		return
	}

	// Delete GitLab connection (project-scoped)
	if err := h.connectionManager.DeleteGitLabConnection(ctx, userIDStr); err != nil {
		gitlab.LogError("Failed to disconnect GitLab for user %s in project %s: %v", userIDStr, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":      "Failed to disconnect GitLab account",
			"statusCode": http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "GitLab account disconnected successfully from project " + project,
		"connected": false,
	})
}

// Cluster-level GitLab credential storage (user-scoped, not project-scoped)

type GitLabCredentials struct {
	UserID      string `json:"userId"`
	Token       string `json:"token"`
	InstanceURL string `json:"instanceUrl"`
	UpdatedAt   string `json:"updatedAt"`
}

// GetToken implements the interface for git package
func (g *GitLabCredentials) GetToken() string {
	return g.Token
}

// ConnectGitLabGlobal handles POST /api/auth/gitlab/connect
// Saves user's GitLab credentials at cluster level
func ConnectGitLabGlobal(c *gin.Context) {
	// Verify user has valid K8s token (follows RBAC pattern)
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	// Verify user is authenticated and userID is valid
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}
	if !isValidUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user identifier"})
		return
	}

	var req struct {
		PersonalAccessToken string `json:"personalAccessToken" binding:"required"`
		InstanceURL         string `json:"instanceUrl"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to GitLab.com if no instance URL provided
	if req.InstanceURL == "" {
		req.InstanceURL = "https://gitlab.com"
	}

	// Validate input
	if err := validateGitLabInput(req.InstanceURL, req.PersonalAccessToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid input: %v", err)})
		return
	}

	// Store credentials at cluster level
	creds := &GitLabCredentials{
		UserID:      userID,
		Token:       req.PersonalAccessToken,
		InstanceURL: req.InstanceURL,
		UpdatedAt:   fmt.Sprintf("%d", time.Now().Unix()),
	}

	if err := storeGitLabCredentials(c.Request.Context(), creds); err != nil {
		gitlab.LogError("Failed to store GitLab credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save GitLab credentials"})
		return
	}

	gitlab.LogInfo("✓ Stored GitLab credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{
		"message":     "GitLab connected successfully",
		"instanceUrl": req.InstanceURL,
	})
}

// GetGitLabStatusGlobal handles GET /api/auth/gitlab/status
// Returns connection status for the authenticated user
func GetGitLabStatusGlobal(c *gin.Context) {
	// Verify user has valid K8s token
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	creds, err := GetGitLabCredentials(c.Request.Context(), userID)
	if err != nil {
		gitlab.LogError("Failed to get GitLab credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check GitLab status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected":   true,
		"instanceUrl": creds.InstanceURL,
		"updatedAt":   creds.UpdatedAt,
	})
}

// DisconnectGitLabGlobal handles DELETE /api/auth/gitlab/disconnect
// Removes user's GitLab credentials
func DisconnectGitLabGlobal(c *gin.Context) {
	// Verify user has valid K8s token
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User authentication required"})
		return
	}

	if err := DeleteGitLabCredentials(c.Request.Context(), userID); err != nil {
		gitlab.LogError("Failed to delete GitLab credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect GitLab"})
		return
	}

	gitlab.LogInfo("✓ Deleted GitLab credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "GitLab disconnected successfully"})
}

// storeGitLabCredentials stores GitLab credentials in cluster-level Secret
func storeGitLabCredentials(ctx context.Context, creds *GitLabCredentials) error {
	if creds == nil || creds.UserID == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	const secretName = "gitlab-credentials"

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Create Secret
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
						Labels: map[string]string{
							"app":                      "ambient-code",
							"ambient-code.io/provider": "gitlab",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				// Fetch again to get resourceVersion
				secret, err = K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
				if err != nil {
					return fmt.Errorf("failed to fetch Secret after create: %w", err)
				}
			} else {
				return fmt.Errorf("failed to get Secret: %w", err)
			}
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		b, err := json.Marshal(creds)
		if err != nil {
			return fmt.Errorf("failed to marshal credentials: %w", err)
		}
		secret.Data[creds.UserID] = b

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// GetGitLabCredentials retrieves cluster-level GitLab credentials for a user
func GetGitLabCredentials(ctx context.Context, userID string) (*GitLabCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	const secretName = "gitlab-credentials"

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, nil // User hasn't connected GitLab
		}
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[userID]) == 0 {
		return nil, nil // User hasn't connected GitLab
	}

	var creds GitLabCredentials
	if err := json.Unmarshal(secret.Data[userID], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// DeleteGitLabCredentials removes GitLab credentials for a user
func DeleteGitLabCredentials(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID is required")
	}

	const secretName = "gitlab-credentials"

	for i := 0; i < 3; i++ { // retry on conflict
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil // Secret doesn't exist, nothing to delete
			}
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		if secret.Data == nil || len(secret.Data[userID]) == 0 {
			return nil // User's credentials don't exist
		}

		delete(secret.Data, userID)

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue // retry
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}
