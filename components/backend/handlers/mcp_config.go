package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const mcpConfigMapName = "ambient-mcp-config"
const mcpConfigKey = "mcp.json"
const httpToolsKey = "http-tools.json"

// getConfigMapKey reads a key from the ambient-mcp-config ConfigMap and returns
// its parsed JSON. If the ConfigMap or key does not exist, returns emptyValue.
func getConfigMapKey(c *gin.Context, key string, emptyValue interface{}) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	cm, err := k8sClient.CoreV1().ConfigMaps(projectName).Get(c.Request.Context(), mcpConfigMapName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, emptyValue)
			return
		}
		log.Printf("Failed to get ConfigMap %s/%s: %v", projectName, mcpConfigMapName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read config"})
		return
	}

	raw, ok := cm.Data[key]
	if !ok {
		c.JSON(http.StatusOK, emptyValue)
		return
	}

	var config map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		preview := raw
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		log.Printf("Failed to parse JSON for key %q from ConfigMap %s/%s: %v (raw: %s)", key, projectName, mcpConfigMapName, err, preview)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse config"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// updateConfigMapKey creates or updates a key in the ambient-mcp-config ConfigMap
// using the request body as the JSON value.
func updateConfigMapKey(c *gin.Context, key string) {
	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configJSON, err := json.Marshal(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to serialize config"})
		return
	}

	cm, err := k8sClient.CoreV1().ConfigMaps(projectName).Get(c.Request.Context(), mcpConfigMapName, v1.GetOptions{})
	if errors.IsNotFound(err) {
		newCM := &corev1.ConfigMap{
			ObjectMeta: v1.ObjectMeta{
				Name:      mcpConfigMapName,
				Namespace: projectName,
				Labels:    map[string]string{"app": "ambient-mcp-config"},
				Annotations: map[string]string{
					"ambient-code.io/mcp-config": "true",
				},
			},
			Data: map[string]string{
				key: string(configJSON),
			},
		}
		if _, err := k8sClient.CoreV1().ConfigMaps(projectName).Create(c.Request.Context(), newCM, v1.CreateOptions{}); err != nil {
			log.Printf("Failed to create ConfigMap %s/%s: %v", projectName, mcpConfigMapName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create config"})
			return
		}
	} else if err != nil {
		log.Printf("Failed to get ConfigMap %s/%s: %v", projectName, mcpConfigMapName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read config"})
		return
	} else {
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}
		cm.Data[key] = string(configJSON)
		if _, err := k8sClient.CoreV1().ConfigMaps(projectName).Update(c.Request.Context(), cm, v1.UpdateOptions{}); err != nil {
			log.Printf("Failed to update ConfigMap %s/%s: %v", projectName, mcpConfigMapName, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

// GetMcpConfig handles GET /api/projects/:projectName/mcp-config
func GetMcpConfig(c *gin.Context) {
	getConfigMapKey(c, mcpConfigKey, gin.H{"servers": map[string]interface{}{}})
}

// UpdateMcpConfig handles PUT /api/projects/:projectName/mcp-config
func UpdateMcpConfig(c *gin.Context) { updateConfigMapKey(c, mcpConfigKey) }

// GetHTTPTools handles GET /api/projects/:projectName/http-tools
func GetHTTPTools(c *gin.Context) {
	getConfigMapKey(c, httpToolsKey, gin.H{"tools": []interface{}{}})
}

// UpdateHTTPTools handles PUT /api/projects/:projectName/http-tools
func UpdateHTTPTools(c *gin.Context) { updateConfigMapKey(c, httpToolsKey) }
