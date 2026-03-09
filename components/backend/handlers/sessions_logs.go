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
	sessionName := SanitizeForLog(c.Param("sessionName"))

	k8sClt, _ := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
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

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

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
		log.Printf("GetSessionLogs: failed to get logs for pod %s: %v", podName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve logs"})
		return
	}
	defer logStream.Close()

	// Stream logs directly to the client with a size cap to prevent OOM
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, io.LimitReader(logStream, maxLogBytes)); err != nil {
		log.Printf("GetSessionLogs: error streaming logs for pod %s: %v", podName, err)
	}
}
