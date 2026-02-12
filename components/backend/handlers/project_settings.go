package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var projectSettingsGVR = schema.GroupVersionResource{
	Group:    "vteam.ambient-code",
	Version:  "v1alpha1",
	Resource: "projectsettings",
}

// GetProjectSettings handles GET /api/projects/:projectName/project-settings
func GetProjectSettings(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	obj, err := reqDyn.Resource(projectSettingsGVR).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{})
			return
		}
		log.Printf("Failed to get ProjectSettings in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read project settings"})
		return
	}

	result := gin.H{}
	if configRepo, found, _ := unstructured.NestedMap(obj.Object, "spec", "defaultConfigRepo"); found {
		result["defaultConfigRepo"] = configRepo
	}

	c.JSON(http.StatusOK, result)
}

// UpdateProjectSettings handles PUT /api/projects/:projectName/project-settings
func UpdateProjectSettings(c *gin.Context) {
	projectName := c.Param("projectName")
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var req struct {
		DefaultConfigRepo *struct {
			GitURL string `json:"gitUrl"`
			Branch string `json:"branch,omitempty"`
		} `json:"defaultConfigRepo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get current ProjectSettings CR
	obj, err := reqDyn.Resource(projectSettingsGVR).Namespace(projectName).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Project settings not found"})
			return
		}
		log.Printf("Failed to get ProjectSettings in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read project settings"})
		return
	}

	// Merge defaultConfigRepo into spec
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec == nil {
		spec = map[string]interface{}{}
	}

	if req.DefaultConfigRepo != nil && req.DefaultConfigRepo.GitURL != "" {
		configRepo := map[string]interface{}{
			"gitUrl": req.DefaultConfigRepo.GitURL,
		}
		if req.DefaultConfigRepo.Branch != "" {
			configRepo["branch"] = req.DefaultConfigRepo.Branch
		}
		spec["defaultConfigRepo"] = configRepo
	} else {
		delete(spec, "defaultConfigRepo")
	}

	if err := unstructured.SetNestedMap(obj.Object, spec, "spec"); err != nil {
		log.Printf("Failed to set spec on ProjectSettings in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project settings"})
		return
	}

	_, err = reqDyn.Resource(projectSettingsGVR).Namespace(projectName).Update(c.Request.Context(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update ProjectSettings in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project settings"})
		return
	}

	// Return updated settings
	result := gin.H{}
	if req.DefaultConfigRepo != nil && req.DefaultConfigRepo.GitURL != "" {
		configRepo := map[string]interface{}{
			"gitUrl": req.DefaultConfigRepo.GitURL,
		}
		if req.DefaultConfigRepo.Branch != "" {
			configRepo["branch"] = req.DefaultConfigRepo.Branch
		}
		result["defaultConfigRepo"] = configRepo
	}

	c.JSON(http.StatusOK, result)
}
