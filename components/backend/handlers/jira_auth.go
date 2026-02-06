package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// JiraCredentials represents cluster-level Jira credentials for a user
type JiraCredentials struct {
	UserID    string    `json:"userId"`
	URL       string    `json:"url"`      // e.g., "https://company.atlassian.net"
	Email     string    `json:"email"`    // Jira account email
	APIToken  string    `json:"apiToken"` // Jira API token
	UpdatedAt time.Time `json:"updatedAt"`
}

// ConnectJira handles POST /api/auth/jira/connect
// Saves user's Jira credentials at cluster level
func ConnectJira(c *gin.Context) {
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
		URL      string `json:"url" binding:"required"`
		Email    string `json:"email" binding:"required"`
		APIToken string `json:"apiToken" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate URL format
	if req.URL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Jira URL is required"})
		return
	}

	// Store credentials
	creds := &JiraCredentials{
		UserID:    userID,
		URL:       req.URL,
		Email:     req.Email,
		APIToken:  req.APIToken,
		UpdatedAt: time.Now(),
	}

	if err := storeJiraCredentials(c.Request.Context(), creds); err != nil {
		log.Printf("Failed to store Jira credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save Jira credentials"})
		return
	}

	log.Printf("✓ Stored Jira credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Jira connected successfully",
		"url":     req.URL,
		"email":   req.Email,
	})
}

// GetJiraStatus handles GET /api/auth/jira/status
// Returns connection status for the authenticated user
func GetJiraStatus(c *gin.Context) {
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

	creds, err := GetJiraCredentials(c.Request.Context(), userID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"connected": false})
			return
		}
		log.Printf("Failed to get Jira credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check Jira status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"url":       creds.URL,
		"email":     creds.Email,
		"updatedAt": creds.UpdatedAt.Format(time.RFC3339),
	})
}

// DisconnectJira handles DELETE /api/auth/jira/disconnect
// Removes user's Jira credentials
func DisconnectJira(c *gin.Context) {
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

	if err := DeleteJiraCredentials(c.Request.Context(), userID); err != nil {
		log.Printf("Failed to delete Jira credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect Jira"})
		return
	}

	log.Printf("✓ Deleted Jira credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "Jira disconnected successfully"})
}

// storeJiraCredentials stores Jira credentials in cluster-level Secret
func storeJiraCredentials(ctx context.Context, creds *JiraCredentials) error {
	if creds == nil || creds.UserID == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	const secretName = "jira-credentials"

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
							"ambient-code.io/provider": "jira",
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

// GetJiraCredentials retrieves cluster-level Jira credentials for a user
func GetJiraCredentials(ctx context.Context, userID string) (*JiraCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	const secretName = "jira-credentials"

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[userID]) == 0 {
		return nil, nil // User hasn't connected Jira
	}

	var creds JiraCredentials
	if err := json.Unmarshal(secret.Data[userID], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// DeleteJiraCredentials removes Jira credentials for a user
func DeleteJiraCredentials(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID is required")
	}

	const secretName = "jira-credentials"

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
