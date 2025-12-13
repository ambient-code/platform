package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Two-secret architecture (hardcoded secret names):
// 1. ambient-runner-secrets: ANTHROPIC_API_KEY only (ignored when Vertex enabled)
// 2. ambient-non-vertex-integrations: GITHUB_TOKEN, JIRA_*, custom keys (optional, injected if present)

// @Summary      List namespace secrets
// @Description  Returns all runner/session secrets in the project namespace (Opaque type with ambient-code.io/runner-secret annotation)
// @Tags         secrets
// @Produce      json
// @Param        projectName  path      string  true  "Project name (Kubernetes namespace)"
// @Success      200  {object}  map[string]interface{}  "List of secrets with name, createdAt, and type"
// @Failure      401  {object}  map[string]string       "Unauthorized - invalid or missing token"
// @Failure      500  {object}  map[string]string       "Internal server error"
// @Router       /projects/{projectName}/secrets [get]
func ListNamespaceSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)

	list, err := reqK8s.CoreV1().Secrets(projectName).List(c.Request.Context(), v1.ListOptions{})
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

// @Summary      List runner secrets
// @Description  Returns runner secrets (ANTHROPIC_API_KEY) from the ambient-runner-secrets Secret. Returns empty map if secret doesn't exist.
// @Tags         secrets
// @Produce      json
// @Param        projectName  path      string  true  "Project name (Kubernetes namespace)"
// @Success      200  {object}  map[string]interface{}  "Runner secrets data as key-value pairs"
// @Failure      401  {object}  map[string]string       "Unauthorized - invalid or missing token"
// @Failure      500  {object}  map[string]string       "Internal server error"
// @Router       /projects/{projectName}/runner-secrets [get]
func ListRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	const secretName = "ambient-runner-secrets"

	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
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

// @Summary      Update runner secrets
// @Description  Creates or updates the ambient-runner-secrets Secret. Only ANTHROPIC_API_KEY is allowed. Used when Vertex is disabled.
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Param        projectName  path      string                 true  "Project name (Kubernetes namespace)"
// @Param        secrets      body      map[string]interface{}  true  "Secret data (only ANTHROPIC_API_KEY allowed)"
// @Success      200  {object}  map[string]string  "Runner secrets updated successfully"
// @Failure      400  {object}  map[string]string  "Invalid request body or disallowed keys"
// @Failure      401  {object}  map[string]string  "Unauthorized - invalid or missing token"
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /projects/{projectName}/runner-secrets [put]
func UpdateRunnerSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
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

	// Validate that only allowed keys are present in runner secrets
	allowedKeys := map[string]bool{
		"ANTHROPIC_API_KEY": true,
		// Future: "GEMINI_KEY": true, etc.
	}
	for key := range req.Data {
		if !allowedKeys[key] {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid key '%s' for ambient-runner-secrets. Only ANTHROPIC_API_KEY is allowed.", key),
			})
			return
		}
	}

	const secretName = "ambient-runner-secrets"

	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
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
		if _, err := reqK8s.CoreV1().Secrets(projectName).Create(c.Request.Context(), newSec, v1.CreateOptions{}); err != nil {
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
		if _, err := reqK8s.CoreV1().Secrets(projectName).Update(c.Request.Context(), sec, v1.UpdateOptions{}); err != nil {
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

// @Summary      List integration secrets
// @Description  Returns integration secrets (GITHUB_TOKEN, JIRA_*, custom keys) from the ambient-non-vertex-integrations Secret. Returns empty map if secret doesn't exist.
// @Tags         secrets
// @Produce      json
// @Param        projectName  path      string  true  "Project name (Kubernetes namespace)"
// @Success      200  {object}  map[string]interface{}  "Integration secrets data as key-value pairs"
// @Failure      401  {object}  map[string]string       "Unauthorized - invalid or missing token"
// @Failure      500  {object}  map[string]string       "Internal server error"
// @Router       /projects/{projectName}/integration-secrets [get]
func ListIntegrationSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	const secretName = "ambient-non-vertex-integrations"

	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
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

// @Summary      Update integration secrets
// @Description  Creates or updates the ambient-non-vertex-integrations Secret. Accepts GITHUB_TOKEN, JIRA_*, and custom integration keys. Injected regardless of Vertex setting.
// @Tags         secrets
// @Accept       json
// @Produce      json
// @Param        projectName  path      string                 true  "Project name (Kubernetes namespace)"
// @Param        secrets      body      map[string]interface{}  true  "Secret data (GITHUB_TOKEN, JIRA_*, custom keys)"
// @Success      200  {object}  map[string]string  "Integration secrets updated successfully"
// @Failure      400  {object}  map[string]string  "Invalid request body"
// @Failure      401  {object}  map[string]string  "Unauthorized - invalid or missing token"
// @Failure      500  {object}  map[string]string  "Internal server error"
// @Router       /projects/{projectName}/integration-secrets [put]
func UpdateIntegrationSecrets(c *gin.Context) {
	projectName := c.Param("projectName")
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
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

	sec, err := reqK8s.CoreV1().Secrets(projectName).Get(c.Request.Context(), secretName, v1.GetOptions{})
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
		if _, err := reqK8s.CoreV1().Secrets(projectName).Create(c.Request.Context(), newSec, v1.CreateOptions{}); err != nil {
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
		if _, err := reqK8s.CoreV1().Secrets(projectName).Update(c.Request.Context(), sec, v1.UpdateOptions{}); err != nil {
			log.Printf("Failed to update Secret %s/%s: %v", projectName, secretName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update integration secrets"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "integration secrets updated"})
}
