package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSessionMetrics returns usage metrics extracted from the session CR status.
// GET /api/projects/:projectName/agentic-sessions/:sessionName/metrics
func GetSessionMetrics(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	sessionName := c.Param("sessionName")

	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(ctx, sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("GetSessionMetrics: failed to get session %s: %v", sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	metrics := gin.H{
		"sessionId": sessionName,
	}

	// Extract timing info from status
	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			metrics["phase"] = phase
		}
		if startTime, ok := status["startTime"].(string); ok {
			metrics["startTime"] = startTime

			// Calculate duration if possible
			start, err := time.Parse(time.RFC3339, startTime)
			if err == nil {
				var end time.Time
				if completionTime, ok := status["completionTime"].(string); ok && completionTime != "" {
					end, err = time.Parse(time.RFC3339, completionTime)
					if err != nil {
						end = time.Now()
					}
					metrics["completionTime"] = completionTime
				} else {
					end = time.Now()
				}
				metrics["durationSeconds"] = int(end.Sub(start).Seconds())
			}
		}
		if sdkRestartCount, ok := status["sdkRestartCount"].(float64); ok {
			metrics["restartCount"] = int(sdkRestartCount)
		}
	}

	// Extract timeout from spec
	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		if timeout, ok := spec["timeout"].(float64); ok {
			metrics["timeoutSeconds"] = int(timeout)
		}
	}

	// Extract any usage annotations (token counts, tool calls, etc.)
	annotations := item.GetAnnotations()
	usage := gin.H{}
	for k, v := range annotations {
		// Look for usage-related annotations
		switch k {
		case "ambient-code.io/input-tokens":
			usage["inputTokens"] = v
		case "ambient-code.io/output-tokens":
			usage["outputTokens"] = v
		case "ambient-code.io/total-cost":
			usage["totalCost"] = v
		case "ambient-code.io/tool-calls":
			usage["toolCalls"] = v
		}
	}
	if len(usage) > 0 {
		metrics["usage"] = usage
	}

	c.JSON(http.StatusOK, metrics)
}
