package handlers

import (
	"context"
	"log"
	"net/http"

	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetProjectMCPServers returns the MCP server configuration from the ProjectSettings CR.
// GET /api/projects/:projectName/mcp-servers
func GetProjectMCPServers(c *gin.Context) {
	project := c.GetString("project")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	gvr := GetProjectSettingsResource()
	ps, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// No project settings yet, return empty config
			c.JSON(http.StatusOK, types.MCPServersConfig{})
			return
		}
		log.Printf("Failed to get project settings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project settings"})
		return
	}

	spec, ok := ps.Object["spec"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusOK, types.MCPServersConfig{})
		return
	}

	mcpServers, ok := spec["mcpServers"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusOK, types.MCPServersConfig{})
		return
	}

	result := types.MCPServersConfig{}
	if custom, ok := mcpServers["custom"].(map[string]interface{}); ok {
		result.Custom = make(map[string]map[string]interface{}, len(custom))
		for name, cfg := range custom {
			if cfgMap, ok := cfg.(map[string]interface{}); ok {
				result.Custom[name] = cfgMap
			}
		}
	}
	if disabled, ok := mcpServers["disabled"].([]interface{}); ok {
		for _, d := range disabled {
			if s, ok := d.(string); ok {
				result.Disabled = append(result.Disabled, s)
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// UpdateProjectMCPServers updates the MCP server configuration in the ProjectSettings CR.
// PUT /api/projects/:projectName/mcp-servers
func UpdateProjectMCPServers(c *gin.Context) {
	project := c.GetString("project")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	var req types.MCPServersConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	gvr := GetProjectSettingsResource()
	ps, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project settings not found"})
			return
		}
		log.Printf("Failed to get project settings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project settings"})
		return
	}

	spec, ok := ps.Object["spec"].(map[string]interface{})
	if !ok {
		spec = map[string]interface{}{}
		ps.Object["spec"] = spec
	}

	// Build the mcpServers map
	mcpMap := map[string]interface{}{}
	if len(req.Custom) > 0 {
		customMap := make(map[string]interface{}, len(req.Custom))
		for name, cfg := range req.Custom {
			customMap[name] = cfg
		}
		mcpMap["custom"] = customMap
	}
	if len(req.Disabled) > 0 {
		disabledArr := make([]interface{}, len(req.Disabled))
		for i, d := range req.Disabled {
			disabledArr[i] = d
		}
		mcpMap["disabled"] = disabledArr
	}

	if len(mcpMap) > 0 {
		spec["mcpServers"] = mcpMap
	} else {
		delete(spec, "mcpServers")
	}

	_, err = k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), ps, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update project settings MCP for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update MCP configuration"})
		return
	}

	c.JSON(http.StatusOK, req)
}
