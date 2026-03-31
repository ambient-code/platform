package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Two-secret architecture (hardcoded secret names):
// 1. ambient-runner-secrets: ANTHROPIC_API_KEY only (ignored when Vertex enabled)
// 2. ambient-non-vertex-integrations: GITHUB_TOKEN, JIRA_*, custom keys (optional, injected if present)

// ListNamespaceSecrets handles GET /api/projects/:projectName/secrets -> { items: [{name, createdAt}] }
func ListNamespaceSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	list, err := k8sClient.CoreV1().Secrets(projectName).List(c.Request.Context(), v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list secrets in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list secrets"})
		return
	}

	type Item struct {
		Name      string `json:"name"`
		CreatedAt string `json:"createdAt,omitempty"`
		Type      string `json:"type"`
	}
	items := []Item{}
	for _, s := range list.Items {
		// Only include runner/session secrets: Opaque + annotated
		if s.Type != corev1.SecretTypeOpaque {
			continue
		}
		if s.Annotations == nil || s.Annotations["ambient-code.io/runner-secret"] != "true" {
			continue
		}
		it := Item{Name: s.Name, Type: string(s.Type)}
		if !s.CreationTimestamp.IsZero() {
			it.CreatedAt = s.CreationTimestamp.Format(time.RFC3339)
		}
		items = append(items, it)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// Runner secrets (ANTHROPIC_API_KEY only)
// Hardcoded secret name: "ambient-runner-secrets"
// Only injected when Vertex is disabled

// ListRunnerSecrets handles GET /api/projects/:projectName/runner-secrets -> { data: { key: value } }
func ListRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	const secretName = "ambient-runner-secrets"

	sec, err := k8sClient.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
			return
		}
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets"})
		return
	}

	out := map[string]string{}
	for k, v := range sec.Data {
		out[k] = string(v)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// UpdateRunnerSecrets handles PUT /api/projects/:projectName/runner-secrets { data: { key: value } }
func UpdateRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var req struct {
		Data map[string]string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that only allowed keys are present in runner secrets.
	allowedKeys := map[string]bool{
		"ANTHROPIC_API_KEY": true,
		"GOOGLE_API_KEY":    true,
		"GEMINI_API_KEY":    true,
	}
	for key := range req.Data {
		if !allowedKeys[key] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid key '%s' for ambient-runner-secrets. Allowed keys: %v", key, allowedKeys),
			})
			return
		}
	}

	const secretName = "ambient-runner-secrets"

	sec, err := k8sClient.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		// Create new Secret
		newSec := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      secretName,
				Namespace: projectName,
				Labels:    map[string]string{"app": "ambient-runner-secrets"},
				Annotations: map[string]string{
					"ambient-code.io/runner-secret": "true",
				},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: req.Data,
		}
		if _, err := k8sClient.CoreV1().Secrets(projectName).Create(c.Request.Context(), newSec, v1.CreateOptions{}); err != nil {
			log.Printf("Failed to create Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create runner secrets"})
			return
		}
	} else if err != nil {
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read runner secrets"})
		return
	} else {
		// Update existing - replace Data
		sec.Type = corev1.SecretTypeOpaque
		sec.Data = map[string][]byte{}
		for k, v := range req.Data {
			sec.Data[k] = []byte(v)
		}
		if _, err := k8sClient.CoreV1().Secrets(projectName).Update(c.Request.Context(), sec, v1.UpdateOptions{}); err != nil {
			log.Printf("Failed to update Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update runner secrets"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "runner secrets updated"})
}

// Integration secrets (GITHUB_TOKEN, JIRA_*, custom keys)
// Hardcoded secret name: "ambient-non-vertex-integrations"
// Injected as env vars if present (optional), regardless of Vertex setting

// ListIntegrationSecrets handles GET /api/projects/:projectName/integration-secrets -> { data: { key: value } }
func ListIntegrationSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	const secretName = "ambient-non-vertex-integrations"

	sec, err := k8sClient.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
			return
		}
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read integration secrets"})
		return
	}

	out := map[string]string{}
	for k, v := range sec.Data {
		out[k] = string(v)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// UpdateIntegrationSecrets handles PUT /api/projects/:projectName/integration-secrets { data: { key: value } }
func UpdateIntegrationSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var req struct {
		Data map[string]string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const secretName = "ambient-non-vertex-integrations"

	sec, err := k8sClient.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		newSec := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      secretName,
				Namespace: projectName,
				Labels:    map[string]string{"app": "ambient-integration-secrets"},
				Annotations: map[string]string{
					"ambient-code.io/runner-secret": "true",
				},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: req.Data,
		}
		if _, err := k8sClient.CoreV1().Secrets(projectName).Create(c.Request.Context(), newSec, v1.CreateOptions{}); err != nil {
			log.Printf("Failed to create Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create integration secrets"})
			return
		}
	} else if err != nil {
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read integration secrets"})
		return
	} else {
		sec.Type = corev1.SecretTypeOpaque
		sec.Data = map[string][]byte{}
		for k, v := range req.Data {
			sec.Data[k] = []byte(v)
		}
		if _, err := k8sClient.CoreV1().Secrets(projectName).Update(c.Request.Context(), sec, v1.UpdateOptions{}); err != nil {
			log.Printf("Failed to update Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update integration secrets"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "integration secrets updated"})
}

// Generic secrets (workspace-level arbitrary credentials)
// Hardcoded secret name: "ambient-generic-secrets"
// Injected as env vars if present (optional), available for sessions, schedules, and webhooks

// ListGenericSecrets handles GET /api/projects/:projectName/generic-secrets -> { data: { key: value } }
func ListGenericSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	const secretName = "ambient-generic-secrets"

	sec, err := k8sClient.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
			return
		}
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read generic secrets"})
		return
	}

	out := map[string]string{}
	for k, v := range sec.Data {
		out[k] = string(v)
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// UpdateGenericSecrets handles PUT /api/projects/:projectName/generic-secrets { data: { key: value } }
func UpdateGenericSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var req struct {
		Data map[string]string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const secretName = "ambient-generic-secrets"

	sec, err := k8sClient.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		newSec := &corev1.Secret{
			ObjectMeta: v1.ObjectMeta{
				Name:      secretName,
				Namespace: projectName,
				Labels:    map[string]string{"app": "ambient-generic-secrets"},
				Annotations: map[string]string{
					"ambient-code.io/runner-secret": "true",
				},
			},
			Type:       corev1.SecretTypeOpaque,
			StringData: req.Data,
		}
		if _, err := k8sClient.CoreV1().Secrets(projectName).Create(c.Request.Context(), newSec, v1.CreateOptions{}); err != nil {
			log.Printf("Failed to create Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create generic secrets"})
			return
		}
	} else if err != nil {
		log.Printf("Failed to get Secret %s/%s: %v", projectName, secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read generic secrets"})
		return
	} else {
		sec.Type = corev1.SecretTypeOpaque
		sec.Data = map[string][]byte{}
		for k, v := range req.Data {
			sec.Data[k] = []byte(v)
		}
		if _, err := k8sClient.CoreV1().Secrets(projectName).Update(c.Request.Context(), sec, v1.UpdateOptions{}); err != nil {
			log.Printf("Failed to update Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update generic secrets"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "generic secrets updated"})
}

// User-level generic secrets (arbitrary credentials per user)
// Stored in cluster-level secret "user-generic-secrets" in backend namespace
// Each user's secrets are stored as a JSON-encoded map under their sanitized userID key

// ListUserGenericSecrets handles GET /api/auth/generic-secrets -> { data: { key: value } }
func ListUserGenericSecrets(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing user token"})
		return
	}

	const secretName = "user-generic-secrets"
	secretKey := sanitizeSecretKey(userID)

	secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(c.Request.Context(), secretName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
			return
		}
		log.Printf("Failed to get Secret %s: %v", secretName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user generic secrets"})
		return
	}

	if secret.Data == nil || len(secret.Data[secretKey]) == 0 {
		c.JSON(http.StatusOK, gin.H{"data": map[string]string{}})
		return
	}

	var userSecrets map[string]string
	if err := json.Unmarshal(secret.Data[secretKey], &userSecrets); err != nil {
		log.Printf("Failed to unmarshal user secrets for %s: %v", userID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse user secrets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": userSecrets})
}

// UpdateUserGenericSecrets handles PUT /api/auth/generic-secrets { data: { key: value } }
func UpdateUserGenericSecrets(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing user token"})
		return
	}

	var req struct {
		Data map[string]string `json:"data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	const secretName = "user-generic-secrets"
	secretKey := sanitizeSecretKey(userID)

	// Retry loop for handling conflicts with exponential backoff
	for i := 0; i < 3; i++ {
		if i > 0 {
			time.Sleep(time.Duration(50*(1<<uint(i-1))) * time.Millisecond)
		}
		secret, err := K8sClient.CoreV1().Secrets(Namespace).Get(c.Request.Context(), secretName, v1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				// Create new secret
				b, jsonErr := json.Marshal(req.Data)
				if jsonErr != nil {
					log.Printf("Failed to marshal user secrets: %v", jsonErr)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode secrets"})
					return
				}

				newSecret := &corev1.Secret{
					ObjectMeta: v1.ObjectMeta{
						Name:      secretName,
						Namespace: Namespace,
						Labels: map[string]string{
							"app":                             "ambient-code",
							"ambient-code.io/user-secrets":    "true",
							"ambient-code.io/secrets-type":    "generic",
						},
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						secretKey: b,
					},
				}

				if _, err := K8sClient.CoreV1().Secrets(Namespace).Create(c.Request.Context(), newSecret, v1.CreateOptions{}); err != nil {
					log.Printf("Failed to create Secret %s: %v", secretName, err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user secrets"})
					return
				}

				log.Printf("Created user generic secrets for user %s", userID)
				c.JSON(http.StatusOK, gin.H{"message": "user generic secrets created"})
				return
			}

			log.Printf("Failed to get Secret %s: %v", secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read user secrets"})
			return
		}

		// Update existing secret
		b, jsonErr := json.Marshal(req.Data)
		if jsonErr != nil {
			log.Printf("Failed to marshal user secrets: %v", jsonErr)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode secrets"})
			return
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[secretKey] = b

		if _, err := K8sClient.CoreV1().Secrets(Namespace).Update(c.Request.Context(), secret, v1.UpdateOptions{}); err != nil {
			if errors.IsConflict(err) {
				// Retry on conflict
				continue
			}
			log.Printf("Failed to update Secret %s: %v", secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user secrets"})
			return
		}

		log.Printf("Updated user generic secrets for user %s", userID)
		c.JSON(http.StatusOK, gin.H{"message": "user generic secrets updated"})
		return
	}

	// Failed after retries
	log.Printf("Failed to update user secrets after retries for user %s", userID)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user secrets after retries"})
}

// sanitizeSecretKey converts a userID to a valid Kubernetes secret key.
// IMPORTANT: This duplicates the function in oauth.go - keep implementations in sync.
// TODO: Extract to shared helpers.go to avoid duplication.
// Kubernetes secret keys must match: [-._a-zA-Z0-9]+
func sanitizeSecretKey(userID string) string {
	sanitized := strings.ReplaceAll(userID, ":", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	return sanitized
}
