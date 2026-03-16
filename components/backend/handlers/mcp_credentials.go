package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// MCPServerCredentials represents generic credentials for an MCP server
type MCPServerCredentials struct {
	UserID     string            `json:"userId"`
	ServerName string            `json:"serverName"`
	Fields     map[string]string `json:"fields"`
	UpdatedAt  time.Time         `json:"updatedAt"`
}

const mcpCredentialsSecretName = "mcp-server-credentials"

// validServerName matches lowercase alphanumeric with hyphens, max 63 chars
var validServerNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`)

func isValidServerName(name string) bool {
	if len(name) < 1 || len(name) > 63 {
		return false
	}
	// Allow single character names
	if len(name) == 1 {
		return name[0] >= 'a' && name[0] <= 'z' || name[0] >= '0' && name[0] <= '9'
	}
	return validServerNameRegex.MatchString(name)
}

func mcpSecretKey(serverName, userID string) string {
	return serverName + ":" + userID
}

// ConnectMCPServer handles POST /api/auth/mcp/:serverName/connect
func ConnectMCPServer(c *gin.Context) {
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
	if !isValidUserID(userID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user identifier"})
		return
	}

	serverName := c.Param("serverName")
	if !isValidServerName(serverName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server name: must be lowercase alphanumeric with hyphens, 1-63 chars"})
		return
	}

	var req struct {
		Fields map[string]string `json:"fields" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one credential field is required"})
		return
	}

	creds := &MCPServerCredentials{
		UserID:     userID,
		ServerName: serverName,
		Fields:     req.Fields,
		UpdatedAt:  time.Now(),
	}

	if err := storeMCPCredentials(c.Request.Context(), creds); err != nil {
		log.Printf("Failed to store MCP credentials for server %s, user %s: %v", serverName, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save MCP credentials"})
		return
	}

	log.Printf("✓ Stored MCP credentials for server %s, user %s", serverName, userID)
	c.JSON(http.StatusOK, gin.H{
		"message":    "MCP server credentials saved",
		"serverName": serverName,
	})
}

// GetMCPServerStatus handles GET /api/auth/mcp/:serverName/status
func GetMCPServerStatus(c *gin.Context) {
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

	serverName := c.Param("serverName")
	if !isValidServerName(serverName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server name"})
		return
	}

	creds, err := getMCPCredentials(c.Request.Context(), serverName, userID)
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"connected": false, "serverName": serverName})
			return
		}
		log.Printf("Failed to get MCP credentials for server %s, user %s: %v", serverName, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check MCP server status"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusOK, gin.H{"connected": false, "serverName": serverName})
		return
	}

	fieldNames := make([]string, 0, len(creds.Fields))
	for k := range creds.Fields {
		fieldNames = append(fieldNames, k)
	}
	sort.Strings(fieldNames)

	c.JSON(http.StatusOK, gin.H{
		"connected":  true,
		"serverName": serverName,
		"fieldNames": fieldNames,
		"updatedAt":  creds.UpdatedAt.Format(time.RFC3339),
	})
}

// DisconnectMCPServer handles DELETE /api/auth/mcp/:serverName/disconnect
func DisconnectMCPServer(c *gin.Context) {
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

	serverName := c.Param("serverName")
	if !isValidServerName(serverName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server name"})
		return
	}

	if err := deleteMCPCredentials(c.Request.Context(), serverName, userID); err != nil {
		log.Printf("Failed to delete MCP credentials for server %s, user %s: %v", serverName, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect MCP server"})
		return
	}

	log.Printf("✓ Deleted MCP credentials for server %s, user %s", serverName, userID)
	c.JSON(http.StatusOK, gin.H{"message": "MCP server disconnected"})
}

// GetMCPCredentialsForSession handles GET /api/projects/:project/agentic-sessions/:session/credentials/mcp/:serverName
func GetMCPCredentialsForSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	serverName := c.Param("serverName")

	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	if !isValidServerName(serverName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server name"})
		return
	}

	// Get userID from session CR
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s/%s: %v", project, session, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	userID, found, err := unstructured.NestedString(obj.Object, "spec", "userContext", "userId")
	if !found || err != nil || userID == "" {
		log.Printf("Failed to extract userID from session %s/%s: found=%v, err=%v", project, session, found, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in session"})
		return
	}

	// Verify authenticated user owns this session
	authenticatedUserID := c.GetString("userID")
	if authenticatedUserID != "" && authenticatedUserID != userID {
		log.Printf("RBAC violation: user %s attempted to access MCP credentials for session owned by %s", authenticatedUserID, userID)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: session belongs to different user"})
		return
	}

	creds, err := getMCPCredentials(c.Request.Context(), serverName, userID)
	if err != nil {
		log.Printf("Failed to get MCP credentials for server %s, user %s: %v", serverName, userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get MCP credentials"})
		return
	}

	if creds == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "MCP credentials not configured for server " + serverName})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"serverName": creds.ServerName,
		"fields":     creds.Fields,
	})
}

// storeMCPCredentials stores MCP server credentials in a cluster-level Secret
func storeMCPCredentials(ctx context.Context, creds *MCPServerCredentials) error {
	if creds == nil || creds.UserID == "" || creds.ServerName == "" {
		return fmt.Errorf("invalid credentials payload")
	}

	key := mcpSecretKey(creds.ServerName, creds.UserID)

	for i := 0; i < 3; i++ {
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, mcpCredentialsSecretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				secret = &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      mcpCredentialsSecretName,
						Namespace: Namespace,
						Labels: map[string]string{
							"app":                      "ambient-code",
							"ambient-code.io/provider": "mcp",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{},
				}
				if _, cerr := K8sClient.CoreV1().Secrets(Namespace).Create(ctx, secret, v1.CreateOptions{}); cerr != nil && !errors.IsAlreadyExists(cerr) {
					return fmt.Errorf("failed to create Secret: %w", cerr)
				}
				secret, err = K8sClient.CoreV1().Secrets(Namespace).Get(ctx, mcpCredentialsSecretName, v1.GetOptions{})
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
		secret.Data[key] = b

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// getMCPCredentials retrieves MCP server credentials for a user
func getMCPCredentials(ctx context.Context, serverName, userID string) (*MCPServerCredentials, error) {
	if userID == "" || serverName == "" {
		return nil, fmt.Errorf("serverName and userID are required")
	}

	key := mcpSecretKey(serverName, userID)

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, mcpCredentialsSecretName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if secret.Data == nil || len(secret.Data[key]) == 0 {
		return nil, nil
	}

	var creds MCPServerCredentials
	if err := json.Unmarshal(secret.Data[key], &creds); err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	return &creds, nil
}

// deleteMCPCredentials removes MCP server credentials for a user
func deleteMCPCredentials(ctx context.Context, serverName, userID string) error {
	if userID == "" || serverName == "" {
		return fmt.Errorf("serverName and userID are required")
	}

	key := mcpSecretKey(serverName, userID)

	for i := 0; i < 3; i++ {
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, mcpCredentialsSecretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get Secret: %w", err)
		}

		if secret.Data == nil || len(secret.Data[key]) == 0 {
			return nil
		}

		delete(secret.Data, key)

		if _, uerr := K8sClient.CoreV1().Secrets(Namespace).Update(ctx, secret, v1.UpdateOptions{}); uerr != nil {
			if errors.IsConflict(uerr) {
				continue
			}
			return fmt.Errorf("failed to update Secret: %w", uerr)
		}
		return nil
	}
	return fmt.Errorf("failed to update Secret after retries")
}

// getMCPServerStatusForUser returns status for all MCP servers a user has credentials for
func getMCPServerStatusForUser(ctx context.Context, userID string) gin.H {
	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(ctx, mcpCredentialsSecretName, v1.GetOptions{})
	if err != nil || secret.Data == nil {
		return gin.H{}
	}

	suffix := ":" + userID
	result := gin.H{}
	for key := range secret.Data {
		if strings.HasSuffix(key, suffix) {
			serverName := strings.TrimSuffix(key, suffix)
			result[serverName] = gin.H{
				"connected": true,
				"valid":     true,
			}
		}
	}
	return result
}
