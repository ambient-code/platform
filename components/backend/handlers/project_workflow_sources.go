package handlers

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowSource represents a custom workflow source repository.
type WorkflowSource struct {
	Name   string `json:"name"`
	GitURL string `json:"gitUrl"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// WorkflowSourcesConfig is the request/response envelope for workflow sources.
type WorkflowSourcesConfig struct {
	Sources []WorkflowSource `json:"sources"`
}

// GetProjectWorkflowSources returns the custom workflow sources from the ProjectSettings CR.
// GET /api/projects/:projectName/workflow-sources
func GetProjectWorkflowSources(c *gin.Context) {
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
			// No project settings yet, return empty sources
			c.JSON(http.StatusOK, WorkflowSourcesConfig{Sources: []WorkflowSource{}})
			return
		}
		log.Printf("Failed to get project settings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get project settings"})
		return
	}

	spec, ok := ps.Object["spec"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusOK, WorkflowSourcesConfig{Sources: []WorkflowSource{}})
		return
	}

	rawSources, ok := spec["workflowSources"].([]interface{})
	if !ok {
		c.JSON(http.StatusOK, WorkflowSourcesConfig{Sources: []WorkflowSource{}})
		return
	}

	sources := make([]WorkflowSource, 0, len(rawSources))
	for _, raw := range rawSources {
		srcMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		src := WorkflowSource{}
		if name, ok := srcMap["name"].(string); ok {
			src.Name = name
		}
		if gitURL, ok := srcMap["gitUrl"].(string); ok {
			src.GitURL = gitURL
		}
		if branch, ok := srcMap["branch"].(string); ok {
			src.Branch = branch
		}
		if path, ok := srcMap["path"].(string); ok {
			src.Path = path
		}
		sources = append(sources, src)
	}

	c.JSON(http.StatusOK, WorkflowSourcesConfig{Sources: sources})
}

// UpdateProjectWorkflowSources updates the custom workflow sources in the ProjectSettings CR.
// PUT /api/projects/:projectName/workflow-sources
func UpdateProjectWorkflowSources(c *gin.Context) {
	project := c.GetString("project")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	var req WorkflowSourcesConfig
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

	// Build the workflowSources array
	if len(req.Sources) > 0 {
		sourcesArr := make([]interface{}, len(req.Sources))
		for i, src := range req.Sources {
			srcMap := map[string]interface{}{
				"name":   src.Name,
				"gitUrl": src.GitURL,
			}
			if src.Branch != "" {
				srcMap["branch"] = src.Branch
			}
			if src.Path != "" {
				srcMap["path"] = src.Path
			}
			sourcesArr[i] = srcMap
		}
		spec["workflowSources"] = sourcesArr
	} else {
		delete(spec, "workflowSources")
	}

	_, err = k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), ps, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update project settings workflow sources for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workflow sources configuration"})
		return
	}

	c.JSON(http.StatusOK, req)
}
