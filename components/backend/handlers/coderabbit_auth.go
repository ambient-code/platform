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

// CodeRabbitCredentials represents cluster-level CodeRabbit credentials for a user
type CodeRabbitCredentials struct {
	UserID    string    `json:"userId"`
	APIKey    string    `json:"apiKey"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ConnectCodeRabbit handles POST /api/auth/coderabbit/connect
// Saves user's CodeRabbit credentials at cluster level
func ConnectCodeRabbit(c *gin.Context) {
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
		APIKey string `json:"apiKey" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate API key with CodeRabbit
	if err := ValidateCodeRabbitAPIKey(c.Request.Context(), req.APIKey); err != nil {
		log.Printf("Failed to validate CodeRabbit API key for user %s: %v", userID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CodeRabbit API key"})
		return
	}

	// Store credentials
	creds := &CodeRabbitCredentials{
		UserID:    userID,
		APIKey:    req.APIKey,
		UpdatedAt: time.Now(),
	}

	if err := storeCodeRabbitCredentials(c.Request.Context(), creds); err != nil {
		log.Printf("Failed to store CodeRabbit credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save CodeRabbit credentials"})
		return
	}

	log.Printf("✓ Stored CodeRabbit credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{
		"message": "CodeRabbit connected successfully",
	})
}

// GetCodeRabbitStatus handles GET /api/auth/coderabbit/status
// Returns connection status for the authenticated user
func GetCodeRabbitStatus(c *gin.Context) {
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

	creds, err := GetCodeRabbitCredentials(c.Request.Context(), userID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"connected": false})
			return
		}
		log.Printf("Failed to get CodeRabbit credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check CodeRabbit status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": true,
		"updatedAt": creds.UpdatedAt.Format(time.RFC3339),
	})
}

// DisconnectCodeRabbit handles DELETE /api/auth/coderabbit/disconnect
// Removes user's CodeRabbit credentials
func DisconnectCodeRabbit(c *gin.Context) {
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

	if err := DeleteCodeRabbitCredentials(c.Request.Context(), userID); err != nil {
		log.Printf("Failed to delete CodeRabbit credentials for user %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect CodeRabbit"})
		return
	}

	log.Printf("✓ Deleted CodeRabbit credentials for user %s", userID)
	c.JSON(http.StatusOK, gin.H{"message": "CodeRabbit disconnected successfully"})
}

// storeCodeRabbitCredentials stores CodeRabbit credentials in cluster-level Secret
func storeCodeRabbitCredentials(ctx context.Context, creds *CodeRabbitCredentials) error {
	if creds == nil || creds.UserID == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	const secretName = "coderabbit-credentials"

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
							"ambient-code.io/provider": "coderabbit",
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

// GetCodeRabbitCredentials retrieves cluster-level CodeRabbit credentials for a user
func GetCodeRabbitCredentials(ctx context.Context, userID string) (*CodeRabbitCredentials, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	const secretName = "coderabbit-credentials"

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, secretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[userID]) == 0 {
		return nil, nil // User hasn't connected CodeRabbit
	}

	var creds CodeRabbitCredentials
	if err := json.Unmarshal(secret.Data[userID], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// DeleteCodeRabbitCredentials removes CodeRabbit credentials for a user
func DeleteCodeRabbitCredentials(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("userID is required")
	}

	const secretName = "coderabbit-credentials"

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
