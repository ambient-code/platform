package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultTailLines = int64(1000)
	maxTailLines     = int64(10000)
	maxLogBytes      = 10 * 1024 * 1024 // 10MB cap on log response size
)

// GetSessionLogs returns container logs for the session's runner pod.
// GET /api/projects/:projectName/agentic-sessions/:sessionName/logs
//
// Query params:
//   - tailLines: number of lines from the end (default 1000, max 10000)
//   - container: specific container name (optional)
func GetSessionLogs(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	// SanitizeForLog strips control characters for log-injection safety.
	// Safe to reuse as K8s lookup key — K8s names cannot contain control characters.
	sessionName := SanitizeForLog(c.Param("sessionName"))

	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Verify the session CR exists before attempting pod log retrieval
	gvr := GetAgenticSessionV1Alpha1Resource()
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	_, err := k8sDyn.Resource(gvr).Namespace(project).Get(ctx, sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		if errors.IsForbidden(err) {
			log.Printf("GetSessionLogs: access denied for session %s/%s", project, sessionName)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		log.Printf("GetSessionLogs: failed to verify session %s/%s: %v", project, sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session"})
		return
	}

	// Parse tailLines query param
	tailLines := defaultTailLines
	if tl := c.Query("tailLines"); tl != "" {
		parsed, err := strconv.ParseInt(tl, 10, 64)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tailLines must be a positive integer"})
			return
		}
		if parsed > maxTailLines {
			parsed = maxTailLines
		}
		tailLines = parsed
	}

	container := c.Query("container")

	// Pod naming convention: {sessionName}-runner
	// Must match operator pod creation in internal/controller/reconcile_phases.go
	podName := fmt.Sprintf("%s-runner", sessionName)

	logOpts := &corev1.PodLogOptions{
		TailLines: &tailLines,
	}
	if container != "" {
		logOpts.Container = container
	}

	logReq := k8sClt.CoreV1().Pods(project).GetLogs(podName, logOpts)
	logStream, err := logReq.Stream(ctx)
	if err != nil {
		if errors.IsNotFound(err) {
			// Pod doesn't exist (not yet created or already cleaned up) — return empty 200
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(""))
			return
		}
		if errors.IsForbidden(err) {
			log.Printf("GetSessionLogs: access denied for pod %s in project %s", podName, project)
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}
		log.Printf("GetSessionLogs: failed to get logs for pod %s in project %s: %v", podName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve logs"})
		return
	}
	defer logStream.Close()

	// Stream logs directly to the client with a size cap to prevent OOM
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, io.LimitReader(logStream, maxLogBytes)); err != nil {
		log.Printf("GetSessionLogs: error streaming logs for pod %s in project %s: %v", podName, project, err)
	}
}
