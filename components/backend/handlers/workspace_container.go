// Package handlers implements HTTP handlers for the backend API.
package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// WorkspaceContainerSettings represents workspace container customization.
// Workspace container mode is always enabled (ADR-0006); these settings allow optional customization.
type WorkspaceContainerSettings struct {
	Image     string                            `json:"image,omitempty"`
	Resources *WorkspaceContainerResourceLimits `json:"resources,omitempty"`
}

// WorkspaceContainerResourceLimits represents resource limits for workspace containers
type WorkspaceContainerResourceLimits struct {
	CPURequest    string `json:"cpuRequest,omitempty"`
	CPULimit      string `json:"cpuLimit,omitempty"`
	MemoryRequest string `json:"memoryRequest,omitempty"`
	MemoryLimit   string `json:"memoryLimit,omitempty"`
}

// GetWorkspaceContainerSettings returns the workspace container settings for a project
func GetWorkspaceContainerSettings(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project name required"})
		return
	}

	// Get user-scoped dynamic client
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
		return
	}

	ctx := context.Background()
	gvr := GetProjectSettingsResource()

	// Get the ProjectSettings CR (singleton per namespace)
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(ctx, "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// No ProjectSettings CR exists, return empty settings (uses platform defaults)
			c.JSON(http.StatusOK, WorkspaceContainerSettings{})
			return
		}
		log.Printf("Failed to get ProjectSettings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project settings"})
		return
	}

	// Extract workspaceContainer from spec
	spec, specFound, err := unstructured.NestedMap(obj.Object, "spec")
	if err != nil || !specFound {
		// No spec or error reading it, return empty settings
		c.JSON(http.StatusOK, WorkspaceContainerSettings{})
		return
	}
	wcMap, found, err := unstructured.NestedMap(spec, "workspaceContainer")
	if err != nil || !found {
		// No custom settings, uses platform defaults
		c.JSON(http.StatusOK, WorkspaceContainerSettings{})
		return
	}

	// Build response with optional customizations
	settings := WorkspaceContainerSettings{}
	if image, ok := wcMap["image"].(string); ok {
		settings.Image = image
	}

	// Extract resources if present
	if resources, found, err := unstructured.NestedMap(wcMap, "resources"); err == nil && found {
		settings.Resources = &WorkspaceContainerResourceLimits{}
		if v, ok := resources["cpuRequest"].(string); ok {
			settings.Resources.CPURequest = v
		}
		if v, ok := resources["cpuLimit"].(string); ok {
			settings.Resources.CPULimit = v
		}
		if v, ok := resources["memoryRequest"].(string); ok {
			settings.Resources.MemoryRequest = v
		}
		if v, ok := resources["memoryLimit"].(string); ok {
			settings.Resources.MemoryLimit = v
		}
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateWorkspaceContainerSettings updates the workspace container settings for a project
func UpdateWorkspaceContainerSettings(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project name required"})
		return
	}

	var req WorkspaceContainerSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user-scoped dynamic client
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
		return
	}

	ctx := context.Background()
	gvr := GetProjectSettingsResource()

	// Get or create the ProjectSettings CR
	obj, err := reqDyn.Resource(gvr).Namespace(project).Get(ctx, "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ProjectSettings with workspaceContainer
			obj = &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "vteam.ambient-code/v1alpha1",
					"kind":       "ProjectSettings",
					"metadata": map[string]interface{}{
						"name":      "projectsettings",
						"namespace": project,
					},
					"spec": map[string]interface{}{
						"groupAccess": []interface{}{}, // Required field
					},
				},
			}
		} else {
			log.Printf("Failed to get ProjectSettings for %s: %v", project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project settings"})
			return
		}
	}

	// Build workspaceContainer map with optional customizations
	wcMap := map[string]interface{}{}
	if req.Image != "" {
		wcMap["image"] = req.Image
	}
	if req.Resources != nil {
		resources := map[string]interface{}{}
		if req.Resources.CPURequest != "" {
			resources["cpuRequest"] = req.Resources.CPURequest
		}
		if req.Resources.CPULimit != "" {
			resources["cpuLimit"] = req.Resources.CPULimit
		}
		if req.Resources.MemoryRequest != "" {
			resources["memoryRequest"] = req.Resources.MemoryRequest
		}
		if req.Resources.MemoryLimit != "" {
			resources["memoryLimit"] = req.Resources.MemoryLimit
		}
		if len(resources) > 0 {
			wcMap["resources"] = resources
		}
	}

	// Set workspaceContainer in spec
	if err := unstructured.SetNestedMap(obj.Object, wcMap, "spec", "workspaceContainer"); err != nil {
		log.Printf("Failed to set workspaceContainer in spec: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	// Create or update the ProjectSettings CR
	if obj.GetResourceVersion() == "" {
		// Create new
		_, err = reqDyn.Resource(gvr).Namespace(project).Create(ctx, obj, v1.CreateOptions{})
		if err != nil {
			log.Printf("Failed to create ProjectSettings for %s: %v", project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project settings"})
			return
		}
		log.Printf("Created ProjectSettings with workspaceContainer for project %s", project)
	} else {
		// Update existing
		_, err = reqDyn.Resource(gvr).Namespace(project).Update(ctx, obj, v1.UpdateOptions{})
		if err != nil {
			log.Printf("Failed to update ProjectSettings for %s: %v", project, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project settings"})
			return
		}
		log.Printf("Updated workspaceContainer settings for project %s", project)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Workspace container settings updated",
		"image":   req.Image,
	})
}
