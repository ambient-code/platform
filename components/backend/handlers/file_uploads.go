package handlers

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"ambient-code-backend/pathutil"
	"ambient-code-backend/storage"

	"github.com/gin-gonic/gin"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// maxPreUploadSize is the maximum file size for pre-uploads (10MB).
const maxPreUploadSize = 10 * 1024 * 1024

// S3Storage is the shared S3 client used for pre-upload file operations.
// Initialized at startup in main.go. If nil, pre-upload endpoints return 503.
var S3Storage *storage.S3Client

// fileUploadS3Key returns the S3 object key for a pre-uploaded file.
// Layout: {namespace}/{sessionName}/file-uploads/{path}
// This matches the path that hydrate.sh downloads from, ensuring files
// uploaded before session start are available when the pod initializes.
func fileUploadS3Key(namespace, sessionName, filePath string) string {
	return fmt.Sprintf("%s/%s/file-uploads/%s", namespace, sessionName, filePath)
}

// fileUploadS3Prefix returns the S3 prefix for listing pre-uploaded files.
func fileUploadS3Prefix(namespace, sessionName string) string {
	return fmt.Sprintf("%s/%s/file-uploads/", namespace, sessionName)
}

// PreUploadFile uploads a file directly to S3 for a session that may not be running yet.
// The file is stored at the same S3 path that hydrate.sh downloads from, so it will
// be available in the session pod's workspace when it initializes.
func PreUploadFile(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	if S3Storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "S3 storage not configured"})
		return
	}

	// Get user-scoped K8s clients for auth
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
		c.Abort()
		return
	}

	// Validate and sanitize path
	sub := strings.TrimPrefix(c.Param("path"), "/")
	if sub == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File path required"})
		return
	}
	workspaceBase := "/workspace/file-uploads"
	validationPath := filepath.Join(workspaceBase, sub)
	if !pathutil.IsPathWithinBase(validationPath, workspaceBase) {
		log.Printf("PreUploadFile: path traversal attempt detected - path=%q", sub)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: must be within file-uploads directory"})
		return
	}
	filePath := filepath.ToSlash(sub)

	// RBAC check: verify user has update permission on agenticsessions
	ssar := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "update",
				Namespace: project,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
	if err != nil {
		log.Printf("RBAC check failed for pre-upload in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return
	}
	if !res.Status.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to upload files to session"})
		return
	}

	// Verify session exists (CR must exist, but pod doesn't need to be running)
	gvr := GetAgenticSessionV1Alpha1Resource()
	_, err = reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Read request body with size limit
	limitedReader := io.LimitReader(c.Request.Body, maxPreUploadSize+1)
	payload, err := io.ReadAll(limitedReader)
	if err != nil {
		log.Printf("PreUploadFile: failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file data"})
		return
	}
	if len(payload) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty file"})
		return
	}
	if len(payload) > maxPreUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "File exceeds 10MB limit"})
		return
	}

	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(payload)
	}

	// Upload to S3
	key := fileUploadS3Key(project, session, filePath)
	if err := S3Storage.PutObject(c.Request.Context(), key, bytes.NewReader(payload), int64(len(payload)), contentType); err != nil {
		log.Printf("PreUploadFile: S3 upload failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload file to storage"})
		return
	}

	log.Printf("PreUploadFile: uploaded %s for session %s/%s (%d bytes)", filePath, project, session, len(payload))
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"filename": filePath,
		"size":     len(payload),
	})
}

// ListPreUploadedFiles lists files that have been pre-uploaded to S3 for a session.
func ListPreUploadedFiles(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	if S3Storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "S3 storage not configured"})
		return
	}

	// Get user-scoped K8s clients for auth
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
		c.Abort()
		return
	}

	// RBAC check: verify user has get permission on agenticsessions
	ssar := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "get",
				Namespace: project,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
	if err != nil {
		log.Printf("RBAC check failed for listing pre-uploads in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return
	}
	if !res.Status.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to list session files"})
		return
	}

	// Verify session exists
	gvr := GetAgenticSessionV1Alpha1Resource()
	_, err = reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	prefix := fileUploadS3Prefix(project, session)
	files, err := S3Storage.ListObjects(c.Request.Context(), prefix)
	if err != nil {
		log.Printf("ListPreUploadedFiles: S3 list failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list files"})
		return
	}

	if files == nil {
		files = []storage.S3FileInfo{}
	}

	c.JSON(http.StatusOK, gin.H{"files": files})
}

// DeletePreUploadedFile deletes a pre-uploaded file from S3.
func DeletePreUploadedFile(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	if S3Storage == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "S3 storage not configured"})
		return
	}

	// Get user-scoped K8s clients for auth
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil || reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
		c.Abort()
		return
	}

	// Validate and sanitize path
	sub := strings.TrimPrefix(c.Param("path"), "/")
	if sub == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File path required"})
		return
	}
	workspaceBase := "/workspace/file-uploads"
	validationPath := filepath.Join(workspaceBase, sub)
	if !pathutil.IsPathWithinBase(validationPath, workspaceBase) {
		log.Printf("DeletePreUploadedFile: path traversal attempt detected - path=%q", sub)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"})
		return
	}
	filePath := filepath.ToSlash(sub)

	// RBAC check
	ssar := &authzv1.SelfSubjectAccessReview{
		Spec: authzv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authzv1.ResourceAttributes{
				Group:     "vteam.ambient-code",
				Resource:  "agenticsessions",
				Verb:      "update",
				Namespace: project,
			},
		},
	}
	res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
	if err != nil {
		log.Printf("RBAC check failed for deleting pre-upload in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return
	}
	if !res.Status.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to delete session files"})
		return
	}

	// Verify session exists
	gvr := GetAgenticSessionV1Alpha1Resource()
	_, err = reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	key := fileUploadS3Key(project, session, filePath)
	if err := S3Storage.DeleteObject(c.Request.Context(), key); err != nil {
		log.Printf("DeletePreUploadedFile: S3 delete failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
