// Package handlers implements HTTP request handlers for the vTeam backend API.
package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"ambient-code-backend/git"
	"ambient-code-backend/pathutil"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
	authnv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

// Package-level variables for session handlers (set from main package)
var (
	GetAgenticSessionV1Alpha1Resource func() schema.GroupVersionResource
	DynamicClient                     dynamic.Interface
	GetGitHubToken                    func(context.Context, kubernetes.Interface, dynamic.Interface, string, string) (string, error)
	DeriveRepoFolderFromURL           func(string) string
	// LEGACY: SendMessageToSession removed - AG-UI server uses HTTP/SSE instead of WebSocket
)

// ootbWorkflowsCache provides in-memory caching for OOTB workflows to avoid GitHub API rate limits.
// The cache stores workflows by repo URL key and expires after ootbCacheTTL.
type ootbWorkflowsCache struct {
	mu        sync.RWMutex
	workflows []OOTBWorkflow
	cachedAt  time.Time
	cacheKey  string // repo+branch+path combination
}

var (
	ootbCache    = &ootbWorkflowsCache{}
	ootbCacheTTL = 5 * time.Minute // Cache OOTB workflows for 5 minutes
)

// isBinaryContentType checks if a MIME type represents binary content that should be base64 encoded.
// This includes images, archives, documents, executables, and other non-text formats.
func isBinaryContentType(contentType string) bool {
	// Comprehensive list of binary MIME type prefixes and exact matches
	binaryPrefixes := []string{
		"image/",                   // All image formats (jpeg, png, gif, webp, etc.)
		"audio/",                   // All audio formats (mp3, wav, ogg, etc.)
		"video/",                   // All video formats (mp4, webm, avi, etc.)
		"font/",                    // Font files (woff, woff2, ttf, etc.)
		"application/octet-stream", // Generic binary
		"application/pdf",          // PDF documents
		"application/zip",          // ZIP archives
		"application/x-",           // Many binary formats (x-7z-compressed, x-tar, x-gzip, etc.)
		"application/vnd.",         // Vendor-specific formats (MS Office, etc.)
	}

	// Check exact matches for common binary types not covered by prefixes
	binaryExact := []string{
		"application/gzip",
		"application/x-bzip2",
		"application/java-archive", // JAR files
		"application/msword",       // Legacy .doc
		"application/rtf",
	}

	for _, prefix := range binaryPrefixes {
		if strings.HasPrefix(contentType, prefix) {
			return true
		}
	}

	for _, exact := range binaryExact {
		if contentType == exact {
			return true
		}
	}

	return false
}

// parseSpec parses AgenticSessionSpec with v1alpha1 fields
func parseSpec(spec map[string]interface{}) types.AgenticSessionSpec {
	result := types.AgenticSessionSpec{}

	if prompt, ok := spec["initialPrompt"].(string); ok {
		result.InitialPrompt = prompt
	}

	if interactive, ok := spec["interactive"].(bool); ok {
		result.Interactive = interactive
	}

	if displayName, ok := spec["displayName"].(string); ok {
		result.DisplayName = displayName
	}

	if project, ok := spec["project"].(string); ok {
		result.Project = project
	}

	if timeout, ok := spec["timeout"].(float64); ok {
		result.Timeout = int(timeout)
	}

	if llmSettings, ok := spec["llmSettings"].(map[string]interface{}); ok {
		if model, ok := llmSettings["model"].(string); ok {
			result.LLMSettings.Model = model
		}
		if temperature, ok := llmSettings["temperature"].(float64); ok {
			result.LLMSettings.Temperature = temperature
		}
		if maxTokens, ok := llmSettings["maxTokens"].(float64); ok {
			result.LLMSettings.MaxTokens = int(maxTokens)
		}
	}

	// environmentVariables passthrough
	if env, ok := spec["environmentVariables"].(map[string]interface{}); ok {
		resultEnv := make(map[string]string, len(env))
		for k, v := range env {
			if s, ok := v.(string); ok {
				resultEnv[k] = s
			}
		}
		if len(resultEnv) > 0 {
			result.EnvironmentVariables = resultEnv
		}
	}

	if userContext, ok := spec["userContext"].(map[string]interface{}); ok {
		uc := &types.UserContext{}
		if userID, ok := userContext["userId"].(string); ok {
			uc.UserID = userID
		}
		if displayName, ok := userContext["displayName"].(string); ok {
			uc.DisplayName = displayName
		}
		if groups, ok := userContext["groups"].([]interface{}); ok {
			for _, group := range groups {
				if groupStr, ok := group.(string); ok {
					uc.Groups = append(uc.Groups, groupStr)
				}
			}
		}
		result.UserContext = uc
	}

	// Multi-repo parsing (simplified format)
	if arr, ok := spec["repos"].([]interface{}); ok {
		repos := make([]types.SimpleRepo, 0, len(arr))
		for _, it := range arr {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			r := types.SimpleRepo{}
			if url, ok := m["url"].(string); ok {
				r.URL = url
			}
			if branch, ok := m["branch"].(string); ok && strings.TrimSpace(branch) != "" {
				r.Branch = types.StringPtr(branch)
			}
			// Parse autoPush as optional boolean. Preserve nil to allow CRD default.
			// nil = use default (false), false = explicit no-push, true = explicit push
			if autoPush, ok := m["autoPush"].(bool); ok {
				r.AutoPush = types.BoolPtr(autoPush)
			}
			if strings.TrimSpace(r.URL) != "" {
				repos = append(repos, r)
			}
		}
		result.Repos = repos
	}

	// Parse activeWorkflow
	if workflow, ok := spec["activeWorkflow"].(map[string]interface{}); ok {
		ws := &types.WorkflowSelection{}
		if gitURL, ok := workflow["gitUrl"].(string); ok {
			ws.GitURL = gitURL
		}
		if branch, ok := workflow["branch"].(string); ok {
			ws.Branch = branch
		}
		if path, ok := workflow["path"].(string); ok {
			ws.Path = path
		}
		result.ActiveWorkflow = ws
	}

	return result
}

// parseStatus parses AgenticSessionStatus with detailed reconciliation fields
func parseStatus(status map[string]interface{}) *types.AgenticSessionStatus {
	if status == nil {
		return nil
	}

	result := &types.AgenticSessionStatus{}

	if og, ok := status["observedGeneration"]; ok {
		switch v := og.(type) {
		case int64:
			result.ObservedGeneration = v
		case int32:
			result.ObservedGeneration = int64(v)
		case float64:
			result.ObservedGeneration = int64(v)
		case json.Number:
			if parsed, err := v.Int64(); err == nil {
				result.ObservedGeneration = parsed
			}
		}
	}

	if phase, ok := status["phase"].(string); ok {
		result.Phase = phase
	}

	if startTime, ok := status["startTime"].(string); ok && strings.TrimSpace(startTime) != "" {
		result.StartTime = types.StringPtr(startTime)
	}

	if completionTime, ok := status["completionTime"].(string); ok && strings.TrimSpace(completionTime) != "" {
		result.CompletionTime = types.StringPtr(completionTime)
	}

	// jobName and runnerPodName removed - they go stale on restarts
	// Use GET /k8s-resources endpoint for live job/pod information

	if sdkSessionID, ok := status["sdkSessionId"].(string); ok {
		result.SDKSessionID = sdkSessionID
	}

	if restarts, ok := status["sdkRestartCount"]; ok {
		switch v := restarts.(type) {
		case int64:
			result.SDKRestartCount = int(v)
		case int32:
			result.SDKRestartCount = int(v)
		case float64:
			result.SDKRestartCount = int(v)
		case json.Number:
			if parsed, err := v.Int64(); err == nil {
				result.SDKRestartCount = int(parsed)
			}
		}
	}

	if repos, ok := status["reconciledRepos"].([]interface{}); ok && len(repos) > 0 {
		result.ReconciledRepos = make([]types.ReconciledRepo, 0, len(repos))
		for _, entry := range repos {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			repo := types.ReconciledRepo{}
			if url, ok := m["url"].(string); ok {
				repo.URL = url
			}
			if branch, ok := m["branch"].(string); ok {
				repo.Branch = branch
			}
			if name, ok := m["name"].(string); ok {
				repo.Name = name
			}
			if statusVal, ok := m["status"].(string); ok {
				repo.Status = statusVal
			}
			if clonedAt, ok := m["clonedAt"].(string); ok && strings.TrimSpace(clonedAt) != "" {
				repo.ClonedAt = types.StringPtr(clonedAt)
			}
			result.ReconciledRepos = append(result.ReconciledRepos, repo)
		}
	}

	if wf, ok := status["reconciledWorkflow"].(map[string]interface{}); ok && len(wf) > 0 {
		reconciled := &types.ReconciledWorkflow{}
		if gitURL, ok := wf["gitUrl"].(string); ok {
			reconciled.GitURL = gitURL
		}
		if branch, ok := wf["branch"].(string); ok {
			reconciled.Branch = branch
		}
		if state, ok := wf["status"].(string); ok {
			reconciled.Status = state
		}
		if appliedAt, ok := wf["appliedAt"].(string); ok && strings.TrimSpace(appliedAt) != "" {
			reconciled.AppliedAt = types.StringPtr(appliedAt)
		}
		result.ReconciledWorkflow = reconciled
	}

	if conds, ok := status["conditions"].([]interface{}); ok && len(conds) > 0 {
		result.Conditions = make([]types.Condition, 0, len(conds))
		for _, entry := range conds {
			m, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			cond := types.Condition{}
			if t, ok := m["type"].(string); ok {
				cond.Type = t
			}
			if s, ok := m["status"].(string); ok {
				cond.Status = s
			}
			if reason, ok := m["reason"].(string); ok {
				cond.Reason = reason
			}
			if message, ok := m["message"].(string); ok {
				cond.Message = message
			}
			if ts, ok := m["lastTransitionTime"].(string); ok {
				cond.LastTransitionTime = ts
			}
			if og, ok := m["observedGeneration"]; ok {
				switch v := og.(type) {
				case int64:
					cond.ObservedGeneration = v
				case int32:
					cond.ObservedGeneration = int64(v)
				case float64:
					cond.ObservedGeneration = int64(v)
				case json.Number:
					if parsed, err := v.Int64(); err == nil {
						cond.ObservedGeneration = parsed
					}
				}
			}
			result.Conditions = append(result.Conditions, cond)
		}
	}

	return result
}

// V2 API Handlers - Multi-tenant session management

func ListSessions(c *gin.Context) {
	project := c.GetString("project")

	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	gvr := GetAgenticSessionV1Alpha1Resource()

	// Parse pagination parameters
	var params types.PaginationParams
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pagination parameters"})
		return
	}
	types.NormalizePaginationParams(&params)

	// Build list options with pagination
	// Note: Kubernetes List with Limit returns a continue token for server-side pagination
	// We use offset-based pagination on top of fetching all items for search/sort flexibility
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	list, err := k8sDyn.Resource(gvr).Namespace(project).List(ctx, v1.ListOptions{})
	if err != nil {
		log.Printf("Failed to list agentic sessions in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list agentic sessions"})
		return
	}

	var sessions []types.AgenticSession
	for _, item := range list.Items {
		meta, _, err := unstructured.NestedMap(item.Object, "metadata")
		if err != nil {
			log.Printf("ListSessions: failed to read metadata for %s/%s: %v", project, item.GetName(), err)
			meta = map[string]interface{}{}
		}
		session := types.AgenticSession{
			APIVersion: item.GetAPIVersion(),
			Kind:       item.GetKind(),
			Metadata:   meta,
		}

		if spec, found, err := unstructured.NestedMap(item.Object, "spec"); err == nil && found {
			session.Spec = parseSpec(spec)
		}

		if status, found, err := unstructured.NestedMap(item.Object, "status"); err == nil && found {
			session.Status = parseStatus(status)
		}

		session.AutoBranch = ComputeAutoBranch(item.GetName())

		sessions = append(sessions, session)
	}

	// Apply search filter if provided
	if params.Search != "" {
		sessions = filterSessionsBySearch(sessions, params.Search)
	}

	// Sort by creation timestamp (newest first)
	sortSessionsByCreationTime(sessions)

	// Apply pagination
	totalCount := len(sessions)
	paginatedSessions, hasMore, nextOffset := paginateSessions(sessions, params.Offset, params.Limit)

	response := types.PaginatedResponse{
		Items:      paginatedSessions,
		TotalCount: totalCount,
		Limit:      params.Limit,
		Offset:     params.Offset,
		HasMore:    hasMore,
	}
	if hasMore {
		response.NextOffset = &nextOffset
	}

	c.JSON(http.StatusOK, response)
}

// filterSessionsBySearch filters sessions by search term (name or displayName)
func filterSessionsBySearch(sessions []types.AgenticSession, search string) []types.AgenticSession {
	if search == "" {
		return sessions
	}

	searchLower := strings.ToLower(search)
	filtered := make([]types.AgenticSession, 0, len(sessions))

	for _, session := range sessions {
		// Match against name
		if name, ok := session.Metadata["name"].(string); ok {
			if strings.Contains(strings.ToLower(name), searchLower) {
				filtered = append(filtered, session)
				continue
			}
		}

		// Match against displayName in spec
		if strings.Contains(strings.ToLower(session.Spec.DisplayName), searchLower) {
			filtered = append(filtered, session)
			continue
		}

		// Match against initialPrompt
		if strings.Contains(strings.ToLower(session.Spec.InitialPrompt), searchLower) {
			filtered = append(filtered, session)
			continue
		}
	}

	return filtered
}

// sortSessionsByCreationTime sorts sessions by creation timestamp (newest first)
func sortSessionsByCreationTime(sessions []types.AgenticSession) {
	// Use sort.Slice for O(n log n) performance
	sort.Slice(sessions, func(i, j int) bool {
		ts1 := getSessionCreationTimestamp(sessions[i])
		ts2 := getSessionCreationTimestamp(sessions[j])
		// Sort descending (newest first) - RFC3339 timestamps sort lexicographically
		return ts1 > ts2
	})
}

// getSessionCreationTimestamp extracts the creation timestamp from session metadata
func getSessionCreationTimestamp(session types.AgenticSession) string {
	if ts, ok := session.Metadata["creationTimestamp"].(string); ok {
		return ts
	}
	return ""
}

// paginateSessions applies offset/limit pagination to the session list
func paginateSessions(sessions []types.AgenticSession, offset, limit int) ([]types.AgenticSession, bool, int) {
	total := len(sessions)

	// Handle offset beyond available items
	if offset >= total {
		return []types.AgenticSession{}, false, 0
	}

	// Calculate end index
	end := offset + limit
	if end > total {
		end = total
	}

	// Determine if there are more items
	hasMore := end < total
	nextOffset := end

	return sessions[offset:end], hasMore, nextOffset
}

func CreateSession(c *gin.Context) {
	project := c.GetString("project")

	reqK8s, k8sDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil || k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User token required"})
		c.Abort()
		return
	}
	var req types.CreateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validation for multi-repo can be added here if needed

	// Set defaults for LLM settings if not provided
	llmSettings := types.LLMSettings{
		Model:       "sonnet",
		Temperature: 0.7,
		MaxTokens:   4000,
	}
	if req.LLMSettings != nil {
		if req.LLMSettings.Model != "" {
			llmSettings.Model = req.LLMSettings.Model
		}
		if req.LLMSettings.Temperature != 0 {
			llmSettings.Temperature = req.LLMSettings.Temperature
		}
		if req.LLMSettings.MaxTokens != 0 {
			llmSettings.MaxTokens = req.LLMSettings.MaxTokens
		}
	}

	timeout := 300
	if req.Timeout != nil {
		timeout = *req.Timeout
	}

	// Generate unique name (timestamp-based)
	// Note: Runner will create branch as "ambient/{session-name}"
	timestamp := time.Now().Unix()
	name := fmt.Sprintf("session-%d", timestamp)

	// Create the custom resource
	// Metadata
	metadata := map[string]interface{}{
		"name":      name,
		"namespace": project,
	}
	if len(req.Labels) > 0 {
		labels := map[string]interface{}{}
		for k, v := range req.Labels {
			labels[k] = v
		}
		metadata["labels"] = labels
	}
	if len(req.Annotations) > 0 {
		annotations := map[string]interface{}{}
		for k, v := range req.Annotations {
			annotations[k] = v
		}
		metadata["annotations"] = annotations
	}

	spec := map[string]interface{}{
		"displayName": req.DisplayName,
		"project":     project,
		"llmSettings": map[string]interface{}{
			"model":       llmSettings.Model,
			"temperature": llmSettings.Temperature,
			"maxTokens":   llmSettings.MaxTokens,
		},
		"timeout": timeout,
	}
	if strings.TrimSpace(req.InitialPrompt) != "" {
		spec["initialPrompt"] = req.InitialPrompt
	}

	session := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata":   metadata,
		"spec":       spec,
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	}

	// Optional environment variables passthrough (always, independent of git config presence)
	envVars := make(map[string]string)
	for k, v := range req.EnvironmentVariables {
		envVars[k] = v
	}

	// Handle session continuation
	if req.ParentSessionID != "" {
		envVars["PARENT_SESSION_ID"] = req.ParentSessionID
		// Add annotation to track continuation lineage
		if metadata["annotations"] == nil {
			metadata["annotations"] = make(map[string]interface{})
		}
		annotations := metadata["annotations"].(map[string]interface{})
		annotations["vteam.ambient-code/parent-session-id"] = req.ParentSessionID
		log.Printf("Creating continuation session from parent %s (operator will handle temp pod cleanup)", req.ParentSessionID)
		// Note: Operator will delete temp pod when session starts (desired-phase=Running)
	}

	if len(envVars) > 0 {
		spec := session["spec"].(map[string]interface{})
		spec["environmentVariables"] = envVars
	}

	// Interactive flag
	if req.Interactive != nil {
		session["spec"].(map[string]interface{})["interactive"] = *req.Interactive
	}

	// Set multi-repo configuration on spec (simplified format)
	{
		spec := session["spec"].(map[string]interface{})
		if len(req.Repos) > 0 {
			arr := make([]map[string]interface{}, 0, len(req.Repos))
			for _, r := range req.Repos {
				m := map[string]interface{}{"url": r.URL}
				// Fill in branch if not provided (auto-generate from session name)
				if r.Branch != nil && strings.TrimSpace(*r.Branch) != "" {
					m["branch"] = *r.Branch
				} else {
					m["branch"] = ComputeAutoBranch(name)
				}
				if r.AutoPush != nil {
					m["autoPush"] = *r.AutoPush
				}
				arr = append(arr, m)
			}
			spec["repos"] = arr
		}
	}

	// Add userContext derived from authenticated caller; ignore client-supplied userId
	{
		uidVal, _ := c.Get("userID")
		uid, _ := uidVal.(string)
		uid = strings.TrimSpace(uid)
		if uid != "" {
			displayName := ""
			if v, ok := c.Get("userName"); ok {
				if s, ok2 := v.(string); ok2 {
					displayName = s
				}
			}
			groups := []string{}
			if v, ok := c.Get("userGroups"); ok {
				if gg, ok2 := v.([]string); ok2 {
					groups = gg
				}
			}
			// Fallbacks for non-identity fields only
			if displayName == "" && req.UserContext != nil {
				displayName = req.UserContext.DisplayName
			}
			if len(groups) == 0 && req.UserContext != nil {
				groups = req.UserContext.Groups
			}
			session["spec"].(map[string]interface{})["userContext"] = map[string]interface{}{
				"userId":      uid,
				"displayName": displayName,
				"groups":      groups,
			}
		}
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	obj := &unstructured.Unstructured{Object: session}

	// Create AgenticSession using user token (enforces user RBAC permissions)
	created, err := k8sDyn.Resource(gvr).Namespace(project).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create agentic session in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agentic session"})
		return
	}

	// Best-effort prefill of agent markdown into PVC workspace for immediate UI availability
	// Uses AGENT_PERSONAS or AGENT_PERSONA if provided in request environment variables
	func() {
		defer func() { _ = recover() }()
		personasCsv := ""
		if v, ok := req.EnvironmentVariables["AGENT_PERSONAS"]; ok && strings.TrimSpace(v) != "" {
			personasCsv = v
		} else if v, ok := req.EnvironmentVariables["AGENT_PERSONA"]; ok && strings.TrimSpace(v) != "" {
			personasCsv = v
		}
		if strings.TrimSpace(personasCsv) == "" {
			return
		}
		// content service removed; skip workspace path handling
		// Write each agent markdown
		for _, p := range strings.Split(personasCsv, ",") {
			persona := strings.TrimSpace(p)
			if persona == "" {
				continue
			}
			// ambient-content removed: skip agent prefill writes
		}
	}()

	// Runner token provisioning is handled by the operator when creating the pod.
	// This ensures consistent behavior whether sessions are created via API or kubectl.

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Agentic session created successfully",
		"name":       name,
		"uid":        created.GetUID(),
		"autoBranch": ComputeAutoBranch(name),
	})
}

func GetSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")

	reqK8s, k8sDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil || k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	gvr := GetAgenticSessionV1Alpha1Resource()

	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Safely extract metadata using type-safe pattern
	metadata, ok := item.Object["metadata"].(map[string]interface{})
	if !ok {
		log.Printf("GetSession: invalid metadata for session %s", sessionName)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid session metadata"})
		return
	}

	session := types.AgenticSession{
		APIVersion: item.GetAPIVersion(),
		Kind:       item.GetKind(),
		Metadata:   metadata,
	}

	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	session.AutoBranch = ComputeAutoBranch(sessionName)

	c.JSON(http.StatusOK, session)
}

// MintSessionGitHubToken validates the token via TokenReview, ensures SA matches CR annotation, and returns a short-lived GitHub token.
// POST /api/projects/:projectName/agentic-sessions/:sessionName/github/token
// Auth: Authorization: Bearer <BOT_TOKEN> (K8s SA token with audience "ambient-backend")
func MintSessionGitHubToken(c *gin.Context) {
	project := c.Param("projectName")
	sessionName := c.Param("sessionName")

	rawAuth := strings.TrimSpace(c.GetHeader("Authorization"))
	if rawAuth == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
		return
	}
	parts := strings.SplitN(rawAuth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header"})
		return
	}
	token := strings.TrimSpace(parts[1])
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "empty token"})
		return
	}

	// TokenReview using default audience (works with standard SA tokens)
	tr := &authnv1.TokenReview{Spec: authnv1.TokenReviewSpec{Token: token}}
	rv, err := K8sClient.AuthenticationV1().TokenReviews().Create(c.Request.Context(), tr, v1.CreateOptions{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token review failed"})
		return
	}
	if rv.Status.Error != "" || !rv.Status.Authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}
	subj := strings.TrimSpace(rv.Status.User.Username)
	const pfx = "system:serviceaccount:"
	if !strings.HasPrefix(subj, pfx) {
		c.JSON(http.StatusForbidden, gin.H{"error": "subject is not a service account"})
		return
	}
	rest := strings.TrimPrefix(subj, pfx)
	segs := strings.SplitN(rest, ":", 2)
	if len(segs) != 2 {
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid service account subject"})
		return
	}
	nsFromToken, saFromToken := segs[0], segs[1]
	if nsFromToken != project {
		c.JSON(http.StatusForbidden, gin.H{"error": "namespace mismatch"})
		return
	}

	// Load session and verify SA matches annotation
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := DynamicClient.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read session"})
		return
	}
	meta, _ := obj.Object["metadata"].(map[string]interface{})
	anns, _ := meta["annotations"].(map[string]interface{})
	expectedSA := ""
	if anns != nil {
		if v, ok := anns["ambient-code.io/runner-sa"].(string); ok {
			expectedSA = strings.TrimSpace(v)
		}
	}
	if expectedSA == "" || expectedSA != saFromToken {
		c.JSON(http.StatusForbidden, gin.H{"error": "service account not authorized for session"})
		return
	}

	// Read authoritative userId from spec.userContext.userId
	spec, _ := obj.Object["spec"].(map[string]interface{})
	userID := ""
	if spec != nil {
		if uc, ok := spec["userContext"].(map[string]interface{}); ok {
			if v, ok := uc["userId"].(string); ok {
				userID = strings.TrimSpace(v)
			}
		}
	}
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "session missing user context"})
		return
	}

	// Get GitHub token (GitHub App or PAT fallback via project runner secret)
	tokenStr, err := GetGitHubToken(c.Request.Context(), K8sClient, DynamicClient, project, userID)
	if err != nil {
		log.Printf("Failed to get GitHub token for project %s: %v", project, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to retrieve GitHub token"})
		return
	}
	// Note: PATs don't have expiration, so we omit expiresAt for simplicity
	// Runners should treat all tokens as short-lived and request new ones as needed
	c.JSON(http.StatusOK, gin.H{"token": tokenStr})
}

func PatchSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var patch map[string]interface{}
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource
	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	// Apply patch to metadata annotations
	if metaPatch, ok := patch["metadata"].(map[string]interface{}); ok {
		if annsPatch, ok := metaPatch["annotations"].(map[string]interface{}); ok {
			metadata, found, err := unstructured.NestedMap(item.Object, "metadata")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to patch session"})
				return
			}
			if !found || metadata == nil {
				metadata = map[string]interface{}{}
			}
			anns, found, err := unstructured.NestedMap(metadata, "annotations")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to patch session"})
				return
			}
			if !found || anns == nil {
				anns = map[string]interface{}{}
			}
			for k, v := range annsPatch {
				anns[k] = v
			}
			_ = unstructured.SetNestedMap(metadata, anns, "annotations")
			_ = unstructured.SetNestedMap(item.Object, metadata, "metadata")
		}
	}

	// Update the resource
	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to patch agentic session %s: %v", sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to patch session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session patched successfully", "annotations": updated.GetAnnotations()})
}

func UpdateSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	var req types.UpdateAgenticSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body for UpdateSession (project=%s session=%s): %v", project, sessionName, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get current resource with brief retry to avoid race on creation
	var item *unstructured.Unstructured
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		item, err = k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
		if err == nil {
			break
		}
		if errors.IsNotFound(err) {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Prevent spec changes while session is running or being created
	if status, ok := item.Object["status"].(map[string]interface{}); ok {
		if phase, ok := status["phase"].(string); ok {
			if strings.EqualFold(phase, "Running") || strings.EqualFold(phase, "Creating") {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Cannot modify session specification while the session is running",
					"phase": phase,
				})
				return
			}
		}
	}

	// Update spec
	spec := item.Object["spec"].(map[string]interface{})
	if req.InitialPrompt != nil {
		spec["initialPrompt"] = *req.InitialPrompt
	}
	if req.DisplayName != nil {
		spec["displayName"] = *req.DisplayName
	}

	if req.LLMSettings != nil {
		llmSettings := make(map[string]interface{})
		if req.LLMSettings.Model != "" {
			llmSettings["model"] = req.LLMSettings.Model
		}
		if req.LLMSettings.Temperature != 0 {
			llmSettings["temperature"] = req.LLMSettings.Temperature
		}
		if req.LLMSettings.MaxTokens != 0 {
			llmSettings["maxTokens"] = req.LLMSettings.MaxTokens
		}
		spec["llmSettings"] = llmSettings
	}

	if req.Timeout != nil {
		spec["timeout"] = *req.Timeout
	}

	// Update the resource
	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agentic session"})
		return
	}

	// Parse and return updated session
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusOK, session)
}

// UpdateSessionDisplayName updates only the spec.displayName field on the AgenticSession.
// PUT /api/projects/:projectName/agentic-sessions/:sessionName/displayname
func UpdateSessionDisplayName(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil || k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// RBAC check: verify user has update permission on agenticsessions in this namespace
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
	res, err := k8sClt.AuthorizationV1().SelfSubjectAccessReviews().Create(c.Request.Context(), ssar, v1.CreateOptions{})
	if err != nil {
		log.Printf("RBAC check failed for update session display name in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return
	}
	if !res.Status.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to update session in this project"})
		return
	}

	var req struct {
		DisplayName string `json:"displayName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate display name (length, sanitization)
	if validationErr := ValidateDisplayName(req.DisplayName); validationErr != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": validationErr})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Retrieve current resource
	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Use unstructured helper for safe type access (per CLAUDE.md guidelines)
	spec, found, err := unstructured.NestedMap(item.Object, "spec")
	if err != nil {
		log.Printf("Failed to get spec from session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse session spec"})
		return
	}
	if !found {
		spec = make(map[string]interface{})
	}
	spec["displayName"] = req.DisplayName

	// Set the updated spec back using unstructured helper
	if err := unstructured.SetNestedMap(item.Object, spec, "spec"); err != nil {
		log.Printf("Failed to set spec for session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session spec"})
		return
	}

	// Persist the change
	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update display name for agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update display name"})
		return
	}

	// Respond with updated session summary using safe type access
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
	}
	if meta, found, _ := unstructured.NestedMap(updated.Object, "metadata"); found {
		session.Metadata = meta
	}
	if s, found, _ := unstructured.NestedMap(updated.Object, "spec"); found {
		session.Spec = parseSpec(s)
	}
	if st, found, _ := unstructured.NestedMap(updated.Object, "status"); found {
		session.Status = parseStatus(st)
	}

	c.JSON(http.StatusOK, session)
}

// SelectWorkflow sets the active workflow for a session
// POST /api/projects/:projectName/agentic-sessions/:sessionName/workflow
func SelectWorkflow(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var req types.WorkflowSelection
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Retrieve current resource
	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	if err := ensureRuntimeMutationAllowed(item); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Build workflow config
	branch := req.Branch
	if branch == "" {
		branch = "main"
	}

	// Update activeWorkflow in spec
	spec, ok := item.Object["spec"].(map[string]interface{})
	if !ok {
		spec = make(map[string]interface{})
		item.Object["spec"] = spec
	}

	// Set activeWorkflow
	workflowMap := map[string]interface{}{
		"gitUrl": req.GitURL,
		"branch": branch,
	}
	if req.Path != "" {
		workflowMap["path"] = req.Path
	}
	spec["activeWorkflow"] = workflowMap

	// Persist the change
	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update workflow for agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update workflow"})
		return
	}

	log.Printf("Workflow updated for session %s: %s@%s", sessionName, req.GitURL, branch)

	// Respond with updated session summary
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if s, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(s)
	}
	if st, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(st)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Workflow updated successfully",
		"session": session,
	})
}

// AddRepo adds a new repository to a running session
// POST /api/projects/:projectName/agentic-sessions/:sessionName/repos
func AddRepo(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var req struct {
		URL      string `json:"url" binding:"required"`
		Branch   string `json:"branch"`
		AutoPush *bool  `json:"autoPush,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Branch == "" {
		req.Branch = "main"
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	if err := ensureRuntimeMutationAllowed(item); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Derive repo name from URL
	repoName := req.URL
	if idx := strings.LastIndex(req.URL, "/"); idx != -1 {
		repoName = req.URL[idx+1:]
	}
	repoName = strings.TrimSuffix(repoName, ".git")

	// Call runner to clone the repository (if session is running)
	status, _ := item.Object["status"].(map[string]interface{})
	phase, _ := status["phase"].(string)
	if phase == "Running" {
		runnerURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/repos/add", sessionName, project)
		runnerReq := map[string]string{
			"url":    req.URL,
			"branch": req.Branch,
			"name":   repoName,
		}
		reqBody, _ := json.Marshal(runnerReq)

		log.Printf("Calling runner to clone repo: %s -> %s", req.URL, runnerURL)
		httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", runnerURL, bytes.NewReader(reqBody))
		if err != nil {
			log.Printf("Failed to create runner request: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create runner request"})
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 120 * time.Second} // Allow time for clone
		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("Failed to call runner to clone repo: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clone repository (runner not reachable)"})
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			log.Printf("Runner failed to clone repo (status %d): %s", resp.StatusCode, string(body))
			c.JSON(resp.StatusCode, gin.H{"error": fmt.Sprintf("Failed to clone repository: %s", string(body))})
			return
		}
		log.Printf("Runner successfully cloned repo %s for session %s", repoName, sessionName)
	}

	// Update spec.repos
	spec, ok := item.Object["spec"].(map[string]interface{})
	if !ok {
		spec = make(map[string]interface{})
		item.Object["spec"] = spec
	}
	repos, _ := spec["repos"].([]interface{})
	if repos == nil {
		repos = []interface{}{}
	}

	newRepo := map[string]interface{}{
		"url":    req.URL,
		"branch": req.Branch,
	}
	if req.AutoPush != nil {
		newRepo["autoPush"] = *req.AutoPush
	}
	repos = append(repos, newRepo)
	spec["repos"] = repos

	// Persist change
	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if specMap, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(specMap)
	}
	if statusMap, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(statusMap)
	}

	log.Printf("Added repository %s to session %s in project %s", req.URL, sessionName, project)
	c.JSON(http.StatusOK, gin.H{"message": "Repository added", "name": repoName, "session": session})
}

// RemoveRepo removes a repository from a running session
// DELETE /api/projects/:projectName/agentic-sessions/:sessionName/repos/:repoName
func RemoveRepo(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	repoName := c.Param("repoName")
	_, reqDyn := GetK8sClientsForRequest(c)
	if reqDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	gvr := GetAgenticSessionV1Alpha1Resource()
	item, err := reqDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get session"})
		return
	}

	if err := ensureRuntimeMutationAllowed(item); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Update spec.repos
	spec, ok := item.Object["spec"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session has no spec"})
		return
	}
	repos, _ := spec["repos"].([]interface{})

	filteredRepos := []interface{}{}
	foundInSpec := false
	for _, r := range repos {
		rm, _ := r.(map[string]interface{})
		url, _ := rm["url"].(string)
		if DeriveRepoFolderFromURL(url) != repoName {
			filteredRepos = append(filteredRepos, r)
		} else {
			foundInSpec = true
		}
	}

	// Also check status.reconciledRepos for repos added directly to runner
	// Note: status map is read-only here, not persisted back to CR
	status, found, err := unstructured.NestedMap(item.Object, "status")
	if !found || err != nil {
		log.Printf("Failed to get status: %v", err)
		status = make(map[string]interface{}) // Local empty map for safe reads
	}

	reconciledRepos, found, err := unstructured.NestedSlice(status, "reconciledRepos")
	if !found || err != nil {
		log.Printf("Failed to get reconciledRepos: %v", err)
		reconciledRepos = []interface{}{}
	}

	foundInReconciled := false
	for _, r := range reconciledRepos {
		rm, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		name, found, err := unstructured.NestedString(rm, "name")
		if found && err == nil && name == repoName {
			foundInReconciled = true
			break
		}

		// Also try matching by URL
		url, found, err := unstructured.NestedString(rm, "url")
		if found && err == nil && DeriveRepoFolderFromURL(url) == repoName {
			foundInReconciled = true
			break
		}
	}

	// Always call runner to remove from filesystem (if session is running)
	// Do this BEFORE checking if repo exists in CR, because it might only be on filesystem
	phase, _, _ := unstructured.NestedString(status, "phase")
	runnerRemoved := false
	if phase == "Running" {
		runnerURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/repos/remove", sessionName, project)
		runnerReq := map[string]string{"name": repoName}
		reqBody, _ := json.Marshal(runnerReq)
		resp, err := http.Post(runnerURL, "application/json", bytes.NewReader(reqBody))
		if err != nil {
			log.Printf("Warning: failed to call runner /repos/remove: %v", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				runnerRemoved = true
				log.Printf("Runner successfully removed repo %s from filesystem", repoName)
			} else {
				body, _ := io.ReadAll(resp.Body)
				log.Printf("Runner failed to remove repo %s (status %d): %s", repoName, resp.StatusCode, string(body))
			}
		}
	}

	// Allow delete if repo is in CR OR was successfully removed from runner
	if !foundInSpec && !foundInReconciled && !runnerRemoved {
		c.JSON(http.StatusNotFound, gin.H{"error": "Repository not found in session or runner"})
		return
	}

	spec["repos"] = filteredRepos

	// Persist change
	updated, err := reqDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if specMap, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(specMap)
	}
	if statusMap, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(statusMap)
	}

	log.Printf("Removed repository %s from session %s in project %s", repoName, sessionName, project)
	c.JSON(http.StatusOK, gin.H{"message": "Repository removed", "session": session})
}

// GetWorkflowMetadata retrieves commands and agents metadata from the active workflow
// getContentServiceName returns the ambient-content service name for a session
// Temp-content pods are deprecated - sessions must be running to access workspace
func getContentServiceName(session string) string {
	return fmt.Sprintf("ambient-content-%s", session)
}

// GetWorkflowMetadata retrieves the workflow metadata for an agentic session
// GET /api/projects/:projectName/agentic-sessions/:sessionName/workflow/metadata
func GetWorkflowMetadata(c *gin.Context) {
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	sessionName := c.Param("sessionName")

	if project == "" {
		log.Printf("GetWorkflowMetadata: project is empty, session=%s", sessionName)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	// Validate user authentication and authorization
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Get authorization token
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Use ambient-content service (per-session content service)
	serviceName := fmt.Sprintf("ambient-content-%s", sessionName)

	// Build URL to content service
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/workflow-metadata?session=%s", endpoint, sessionName)

	log.Printf("GetWorkflowMetadata: project=%s session=%s endpoint=%s", project, sessionName, endpoint)

	// Create and send request to content pod
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetWorkflowMetadata: content service request failed: %v", err)
		// Return empty metadata on error
		c.JSON(http.StatusOK, gin.H{"commands": []interface{}{}, "agents": []interface{}{}})
		return
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GetWorkflowMetadata: failed to read response body: %v", err)
		c.JSON(http.StatusOK, gin.H{"commands": []interface{}{}, "agents": []interface{}{}})
		return
	}

	// Log if content service returned an error
	if resp.StatusCode >= 400 {
		log.Printf("GetWorkflowMetadata: content service returned error status %d: %s", resp.StatusCode, string(b))
	}

	c.Data(resp.StatusCode, "application/json", b)
}

// fetchGitHubFileContent fetches a file from GitHub via API
// token is optional - works for public repos without authentication (but has rate limits)
func fetchGitHubFileContent(ctx context.Context, owner, repo, ref, path, token string) ([]byte, error) {
	api := "https://api.github.com"
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Only set Authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github.raw")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("file not found")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// fetchGitHubDirectoryListing lists files/folders in a GitHub directory
// token is optional - works for public repos without authentication (but has rate limits)
func fetchGitHubDirectoryListing(ctx context.Context, owner, repo, ref, path, token string) ([]map[string]interface{}, error) {
	api := "https://api.github.com"
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s?ref=%s", api, owner, repo, path, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	// Only set Authorization header if token is provided
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var entries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// OOTBWorkflow represents an out-of-the-box workflow
type OOTBWorkflow struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	GitURL      string `json:"gitUrl"`
	Branch      string `json:"branch"`
	Path        string `json:"path,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// ListOOTBWorkflows returns the list of out-of-the-box workflows dynamically discovered from GitHub
// Uses in-memory caching (5 min TTL) to avoid GitHub API rate limits.
// Attempts to use user's GitHub token for better rate limits when cache miss occurs.
// GET /api/workflows/ootb?project=<projectName>
func ListOOTBWorkflows(c *gin.Context) {
	// Read OOTB repo configuration from environment
	ootbRepo := strings.TrimSpace(os.Getenv("OOTB_WORKFLOWS_REPO"))
	if ootbRepo == "" {
		ootbRepo = "https://github.com/ambient-code/ootb-ambient-workflows.git"
	}

	ootbBranch := strings.TrimSpace(os.Getenv("OOTB_WORKFLOWS_BRANCH"))
	if ootbBranch == "" {
		ootbBranch = "main"
	}

	ootbWorkflowsPath := strings.TrimSpace(os.Getenv("OOTB_WORKFLOWS_PATH"))
	if ootbWorkflowsPath == "" {
		ootbWorkflowsPath = "workflows"
	}

	// Build cache key from repo configuration
	cacheKey := fmt.Sprintf("%s|%s|%s", ootbRepo, ootbBranch, ootbWorkflowsPath)

	// Check cache first (read lock)
	ootbCache.mu.RLock()
	if ootbCache.cacheKey == cacheKey && time.Since(ootbCache.cachedAt) < ootbCacheTTL && len(ootbCache.workflows) > 0 {
		workflows := ootbCache.workflows
		ootbCache.mu.RUnlock()
		log.Printf("ListOOTBWorkflows: returning %d cached workflows (age: %v)", len(workflows), time.Since(ootbCache.cachedAt).Round(time.Second))
		c.JSON(http.StatusOK, gin.H{"workflows": workflows})
		return
	}
	ootbCache.mu.RUnlock()

	// Cache miss - need to fetch from GitHub
	// Try to get user's GitHub token (best effort - not required)
	// This gives better rate limits (5000/hr vs 60/hr) and supports private repos
	token := ""
	project := c.Query("project") // Optional query parameter
	if project != "" {
		usrID, _ := c.Get("userID")
		k8sClt, sessDyn := GetK8sClientsForRequest(c)
		if k8sClt != nil && sessDyn != nil {
			if userIDStr, ok := usrID.(string); ok && userIDStr != "" {
				if githubToken, err := GetGitHubToken(c.Request.Context(), k8sClt, sessDyn, project, userIDStr); err == nil {
					token = githubToken
					log.Printf("ListOOTBWorkflows: using user's GitHub token for project %s (better rate limits)", project)
				} else {
					log.Printf("ListOOTBWorkflows: failed to get GitHub token for project %s: %v", project, err)
				}
			}
		}
	}
	if token == "" {
		log.Printf("ListOOTBWorkflows: proceeding without GitHub token (public repo, lower rate limits)")
	}

	// Parse GitHub URL
	owner, repoName, err := git.ParseGitHubURL(ootbRepo)
	if err != nil {
		log.Printf("ListOOTBWorkflows: invalid repo URL: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid OOTB repo URL"})
		return
	}

	// List workflow directories
	entries, err := fetchGitHubDirectoryListing(c.Request.Context(), owner, repoName, ootbBranch, ootbWorkflowsPath, token)
	if err != nil {
		log.Printf("ListOOTBWorkflows: failed to list workflows directory: %v", err)
		// On error, try to return stale cache if available
		ootbCache.mu.RLock()
		if len(ootbCache.workflows) > 0 && ootbCache.cacheKey == cacheKey {
			workflows := ootbCache.workflows
			ootbCache.mu.RUnlock()
			log.Printf("ListOOTBWorkflows: returning stale cached workflows due to GitHub error")
			c.JSON(http.StatusOK, gin.H{"workflows": workflows})
			return
		}
		ootbCache.mu.RUnlock()
		// Include more context in error message for debugging
		errMsg := "Failed to discover OOTB workflows"
		if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "rate limit") {
			errMsg = "Failed to discover OOTB workflows: GitHub rate limit exceeded. Try again later or configure a GitHub token in project settings."
		} else if strings.Contains(err.Error(), "404") {
			errMsg = "Failed to discover OOTB workflows: Repository or path not found"
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": errMsg})
		return
	}

	// Scan each subdirectory for ambient.json
	workflows := []OOTBWorkflow{}
	for _, entry := range entries {
		entryType, _ := entry["type"].(string)
		entryName, _ := entry["name"].(string)

		if entryType != "dir" {
			continue
		}

		// Try to fetch ambient.json from this workflow directory
		ambientPath := fmt.Sprintf("%s/%s/.ambient/ambient.json", ootbWorkflowsPath, entryName)
		ambientData, err := fetchGitHubFileContent(c.Request.Context(), owner, repoName, ootbBranch, ambientPath, token)

		var ambientConfig struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err == nil {
			// Parse ambient.json if found
			if parseErr := json.Unmarshal(ambientData, &ambientConfig); parseErr != nil {
				log.Printf("ListOOTBWorkflows: failed to parse ambient.json for %s: %v", entryName, parseErr)
			}
		}

		// Use ambient.json values or fallback to directory name
		workflowName := ambientConfig.Name
		if workflowName == "" {
			workflowName = strings.ReplaceAll(entryName, "-", " ")
			workflowName = strings.Title(workflowName)
		}

		workflows = append(workflows, OOTBWorkflow{
			ID:          entryName,
			Name:        workflowName,
			Description: ambientConfig.Description,
			GitURL:      ootbRepo,
			Branch:      ootbBranch,
			Path:        fmt.Sprintf("%s/%s", ootbWorkflowsPath, entryName),
			Enabled:     true,
		})
	}

	// Update cache (write lock)
	ootbCache.mu.Lock()
	ootbCache.workflows = workflows
	ootbCache.cachedAt = time.Now()
	ootbCache.cacheKey = cacheKey
	ootbCache.mu.Unlock()

	log.Printf("ListOOTBWorkflows: discovered %d workflows from %s (cached for %v)", len(workflows), ootbRepo, ootbCacheTTL)
	c.JSON(http.StatusOK, gin.H{"workflows": workflows})
}

func DeleteSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	gvr := GetAgenticSessionV1Alpha1Resource()

	err := k8sDyn.Resource(gvr).Namespace(project).Delete(context.TODO(), sessionName, v1.DeleteOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to delete agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agentic session"})
		return
	}

	c.Status(http.StatusNoContent)
}

func CloneSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	var req types.CloneSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	gvr := GetAgenticSessionV1Alpha1Resource()

	// Get source session
	sourceItem, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source session not found"})
			return
		}
		log.Printf("Failed to get source agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get source agentic session"})
		return
	}

	// Validate target project exists and is managed by Ambient via OpenShift Project
	projGvr := GetOpenShiftProjectResource()
	projObj, err := k8sDyn.Resource(projGvr).Get(context.TODO(), req.TargetProject, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target project not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate target project"})
		return
	}

	isAmbient := false
	if meta, ok := projObj.Object["metadata"].(map[string]interface{}); ok {
		if raw, ok := meta["labels"].(map[string]interface{}); ok {
			if v, ok := raw["ambient-code.io/managed"].(string); ok && v == "true" {
				isAmbient = true
			}
		}
	}
	if !isAmbient {
		c.JSON(http.StatusForbidden, gin.H{"error": "Target project is not managed by Ambient"})
		return
	}

	// Ensure unique target session name in target namespace; if exists, append "-duplicate" (and numeric suffix)
	newName := strings.TrimSpace(req.NewSessionName)
	if newName == "" {
		newName = sessionName
	}
	finalName := newName
	conflicted := false
	for i := 0; i < 50; i++ {
		_, getErr := k8sDyn.Resource(gvr).Namespace(req.TargetProject).Get(context.TODO(), finalName, v1.GetOptions{})
		if errors.IsNotFound(getErr) {
			break
		}
		if getErr != nil && !errors.IsNotFound(getErr) {
			// On unexpected error, still attempt to proceed with a duplicate suffix to reduce collision chance
			log.Printf("cloneSession: name check encountered error for %s/%s: %v", req.TargetProject, finalName, getErr)
		}
		conflicted = true
		if i == 0 {
			finalName = fmt.Sprintf("%s-duplicate", newName)
		} else {
			finalName = fmt.Sprintf("%s-duplicate-%d", newName, i+1)
		}
	}

	// Create cloned session
	clonedSession := map[string]interface{}{
		"apiVersion": "vteam.ambient-code/v1alpha1",
		"kind":       "AgenticSession",
		"metadata": map[string]interface{}{
			"name":      finalName,
			"namespace": req.TargetProject,
		},
		"spec": sourceItem.Object["spec"],
		"status": map[string]interface{}{
			"phase": "Pending",
		},
	}

	// Update project in spec
	clonedSpec := clonedSession["spec"].(map[string]interface{})
	clonedSpec["project"] = req.TargetProject
	if conflicted {
		if dn, ok := clonedSpec["displayName"].(string); ok && strings.TrimSpace(dn) != "" {
			clonedSpec["displayName"] = fmt.Sprintf("%s (Duplicate)", dn)
		} else {
			clonedSpec["displayName"] = fmt.Sprintf("%s (Duplicate)", finalName)
		}
	}

	obj := &unstructured.Unstructured{Object: clonedSession}

	created, err := k8sDyn.Resource(gvr).Namespace(req.TargetProject).Create(context.TODO(), obj, v1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create cloned agentic session in project %s: %v", req.TargetProject, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create cloned agentic session"})
		return
	}

	// Parse and return created session
	session := types.AgenticSession{
		APIVersion: created.GetAPIVersion(),
		Kind:       created.GetKind(),
		Metadata:   created.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := created.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)
	}

	if status, ok := created.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusCreated, session)
}

func StartSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	gvr := GetAgenticSessionV1Alpha1Resource()

	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Get current resource
	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Log current phase for debugging
	if currentStatus, ok := item.Object["status"].(map[string]interface{}); ok {
		if phase, ok := currentStatus["phase"].(string); ok {
			log.Printf("StartSession: Current phase is %s", phase)
		}
	}

	// Set annotations to signal desired state to operator
	annotations := item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Signal start/restart request to operator
	annotations["ambient-code.io/desired-phase"] = "Running"
	annotations["ambient-code.io/start-requested-at"] = time.Now().Format(time.RFC3339)

	// Clean up self-referential parent-session-id annotations.
	// Old code used to set parent-session-id to the session's own name for PVC reuse,
	// but this caused the runner to skip INITIAL_PROMPT thinking it was a continuation.
	// With S3 storage, we don't need this anymore. Session state persists via S3 sync.
	// Keep legitimate parent-session-id annotations (pointing to a DIFFERENT session).
	if existingParent, ok := annotations["vteam.ambient-code/parent-session-id"]; ok {
		if existingParent == sessionName {
			log.Printf("StartSession: Clearing self-referential parent-session-id annotation")
			delete(annotations, "vteam.ambient-code/parent-session-id")
		}
	}

	item.SetAnnotations(annotations)

	// For headless sessions being continued, force interactive mode
	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		if interactive, ok := spec["interactive"].(bool); !ok || !interactive {
			spec["interactive"] = true
			log.Printf("StartSession: Converting headless session to interactive for continuation")
		}
	}

	// Update spec and annotations (operator will observe and handle job lifecycle)
	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		log.Printf("Failed to update agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	log.Printf("StartSession: Set desired-phase=Running annotation (operator will reconcile)")

	// Parse and return updated session
	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}

	if spec, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(spec)

		// NOTE: INITIAL_PROMPT auto-execution handled by runner on startup
		// Runner POSTs to /agui/run when ready, events flow through backend
		// This works for both UI and headless/API usage
	}

	if status, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(status)
	}

	c.JSON(http.StatusAccepted, session)
}

func ensureRuntimeMutationAllowed(item *unstructured.Unstructured) error {
	if item == nil {
		return fmt.Errorf("session not loaded")
	}

	spec, _ := item.Object["spec"].(map[string]interface{})
	interactive := false
	if spec != nil {
		if v, ok := spec["interactive"].(bool); ok {
			interactive = v
		}
	}
	if !interactive {
		return fmt.Errorf("session is not interactive")
	}

	status, _ := item.Object["status"].(map[string]interface{})
	phase := ""
	if status != nil {
		if p, ok := status["phase"].(string); ok {
			phase = strings.TrimSpace(strings.ToLower(p))
		}
	}

	if phase != "running" {
		displayPhase := "unknown"
		if status != nil {
			if original, ok := status["phase"].(string); ok && strings.TrimSpace(original) != "" {
				displayPhase = original
			}
		}
		return fmt.Errorf("session must be Running to mutate spec (current phase: %s)", displayPhase)
	}

	return nil
}

func StopSession(c *gin.Context) {
	project := c.GetString("project")
	sessionName := c.Param("sessionName")
	gvr := GetAgenticSessionV1Alpha1Resource()

	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	item, err := k8sDyn.Resource(gvr).Namespace(project).Get(context.TODO(), sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("Failed to get agentic session %s in project %s: %v", sessionName, project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get agentic session"})
		return
	}

	// Set annotations to signal desired state to operator
	annotations := item.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	// Signal stop request to operator
	annotations["ambient-code.io/desired-phase"] = "Stopped"
	annotations["ambient-code.io/stop-requested-at"] = time.Now().Format(time.RFC3339)
	item.SetAnnotations(annotations)

	// Force interactive mode so session can be restarted later
	if spec, ok := item.Object["spec"].(map[string]interface{}); ok {
		if interactive, ok := spec["interactive"].(bool); !ok || !interactive {
			spec["interactive"] = true
			log.Printf("StopSession: Converting headless session to interactive for future restart capability")
		}
	}

	// Update spec and annotations (operator will observe and handle job cleanup)
	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(context.TODO(), item, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"message": "Session no longer exists (already deleted)"})
			return
		}
		log.Printf("Failed to update agentic session %s: %v", sessionName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	log.Printf("StopSession: Set desired-phase=Stopped annotation (operator will reconcile)")

	session := types.AgenticSession{
		APIVersion: updated.GetAPIVersion(),
		Kind:       updated.GetKind(),
		Metadata:   updated.Object["metadata"].(map[string]interface{}),
	}
	if specMap, ok := updated.Object["spec"].(map[string]interface{}); ok {
		session.Spec = parseSpec(specMap)
	}
	if statusMap, ok := updated.Object["status"].(map[string]interface{}); ok {
		session.Status = parseStatus(statusMap)
	}

	c.JSON(http.StatusAccepted, session)
}

// GetSessionK8sResources returns job, pod, and PVC information for a session
// GET /api/projects/:projectName/agentic-sessions/:sessionName/k8s-resources
func GetSessionK8sResources(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	sessionName := c.Param("sessionName")

	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Get session to find job name
	gvr := GetAgenticSessionV1Alpha1Resource()
	session, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	status, _ := session.Object["status"].(map[string]interface{})
	jobName, _ := status["jobName"].(string)
	if jobName == "" {
		jobName = fmt.Sprintf("%s-job", sessionName)
	}

	result := map[string]interface{}{}

	// Get Job status
	job, err := k8sClt.BatchV1().Jobs(project).Get(c.Request.Context(), jobName, v1.GetOptions{})
	jobExists := err == nil

	if jobExists {
		result["jobName"] = jobName
		jobStatus := "Unknown"
		if job.Status.Active > 0 {
			jobStatus = "Active"
		} else if job.Status.Succeeded > 0 {
			jobStatus = "Succeeded"
		} else if job.Status.Failed > 0 {
			jobStatus = "Failed"
		}
		result["jobStatus"] = jobStatus
		result["jobConditions"] = job.Status.Conditions
	} else if errors.IsNotFound(err) {
		// Job not found - don't return job info at all
		log.Printf("GetSessionK8sResources: Job %s not found, omitting from response", jobName)
		// Don't include jobName or jobStatus in result
	} else {
		// Other error - still show job name but with error status
		result["jobName"] = jobName
		result["jobStatus"] = "Error"
		log.Printf("GetSessionK8sResources: Error getting job %s: %v", jobName, err)
	}

	// Get Pods for this job (only if job exists)
	podInfos := []map[string]interface{}{}
	if jobExists {
		pods, err := k8sClt.CoreV1().Pods(project).List(c.Request.Context(), v1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", jobName),
		})
		if err == nil {
			for _, pod := range pods.Items {
				// Check if pod is terminating (has DeletionTimestamp)
				podPhase := string(pod.Status.Phase)
				if pod.DeletionTimestamp != nil {
					podPhase = "Terminating"
				}

				containerInfos := []map[string]interface{}{}
				for _, cs := range pod.Status.ContainerStatuses {
					state := "Unknown"
					var exitCode *int32
					var reason string
					if cs.State.Running != nil {
						state = "Running"
						// If pod is terminating but container still shows running, mark it as terminating
						if pod.DeletionTimestamp != nil {
							state = "Terminating"
						}
					} else if cs.State.Terminated != nil {
						state = "Terminated"
						exitCode = &cs.State.Terminated.ExitCode
						reason = cs.State.Terminated.Reason
					} else if cs.State.Waiting != nil {
						state = "Waiting"
						reason = cs.State.Waiting.Reason
					}
					containerInfos = append(containerInfos, map[string]interface{}{
						"name":     cs.Name,
						"state":    state,
						"exitCode": exitCode,
						"reason":   reason,
					})
				}
				podInfos = append(podInfos, map[string]interface{}{
					"name":       pod.Name,
					"phase":      podPhase,
					"containers": containerInfos,
				})
			}
		}
	}

	result["pods"] = podInfos

	// PVCs deprecated - sessions now use EmptyDir with S3 state persistence
	result["pvcExists"] = false
	result["pvcName"] = "N/A (using EmptyDir + S3)"
	result["storageMode"] = "EmptyDir + S3"

	c.JSON(http.StatusOK, result)
}

// setRepoStatus removed - status.repos no longer in CRD (status simplified to phase, message, is_error)

// ListSessionWorkspace proxies to per-job content service for directory listing.
func ListSessionWorkspace(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("ListSessionWorkspace: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	// Validate user authentication and authorization
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	rel := strings.TrimSpace(c.Query("path"))
	// Path is relative to content service's StateBaseDir (which is /workspace)
	// Content service handles the base path, so we just pass the relative path
	absPath := ""
	if rel != "" {
		absPath = rel
	}

	// Call per-job service or temp service for completed sessions
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Use ambient-content service (per-session content service)
	serviceName := fmt.Sprintf("ambient-content-%s", session)

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/list?path=%s", endpoint, url.QueryEscape(absPath))
	log.Printf("ListSessionWorkspace: project=%s session=%s endpoint=%s", project, session, endpoint)
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if err != nil {
		log.Printf("ListSessionWorkspace: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("ListSessionWorkspace: content service request failed: %v", err)
		// Soften error to 200 with empty list so UI doesn't spam
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ListSessionWorkspace: failed to read response body: %v", err)
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}

	// Log if content service returned an error (other than 404 which is handled below)
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		log.Printf("ListSessionWorkspace: content service returned error status %d: %s", resp.StatusCode, string(b))
	}

	// If content service returns 404, check if it's because workspace doesn't exist yet
	if resp.StatusCode == http.StatusNotFound {
		log.Printf("ListSessionWorkspace: workspace not found (may not be created yet by runner)")
		// Return empty list instead of error for better UX during session startup
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// GetSessionWorkspaceFile reads a file via content service.
func GetSessionWorkspaceFile(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("GetSessionWorkspaceFile: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	// Validate user authentication and authorization
	reqK8s, _ := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	sub := strings.TrimPrefix(c.Param("path"), "/")
	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := sub
	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// Use ambient-content service (per-session content service)
	serviceName := fmt.Sprintf("ambient-content-%s", session)

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	u := fmt.Sprintf("%s/content/file?path=%s", endpoint, url.QueryEscape(absPath))
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, u, nil)
	if err != nil {
		log.Printf("GetSessionWorkspaceFile: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GetSessionWorkspaceFile: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file from content service"})
		return
	}

	// Log if content service returned an error
	if resp.StatusCode >= 400 {
		log.Printf("GetSessionWorkspaceFile: content service returned error status %d for path %s", resp.StatusCode, sub)
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), b)
}

// PutSessionWorkspaceFile writes a file via content service.
func PutSessionWorkspaceFile(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("PutSessionWorkspaceFile: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	// Get user-scoped K8s clients and validate authentication IMMEDIATELY
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
		c.Abort()
		return
	}

	// Validate and sanitize path to prevent directory traversal
	// Use robust path validation that works across platforms
	sub := strings.TrimPrefix(c.Param("path"), "/")
	workspaceBase := "/workspace"

	// Construct absolute path using filepath.Join for path validation
	validationPath := filepath.Join(workspaceBase, sub)

	// Use robust path validation from pathutil package
	// This is more secure than manual string checks and works across platforms
	if !pathutil.IsPathWithinBase(validationPath, workspaceBase) {
		log.Printf("PutSessionWorkspaceFile: path traversal attempt detected - path=%q escapes workspace=%q", validationPath, workspaceBase)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: must be within workspace directory"})
		return
	}

	// Use relative path for content service (it has its own StateBaseDir=/workspace)
	// Convert to forward slashes for content service (expects POSIX paths)
	absPath := filepath.ToSlash(sub)

	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// RBAC check: verify user has update permission on agenticsessions (file operations modify session state)
	// IMPORTANT: RBAC check MUST happen BEFORE checking session existence to prevent enumeration attacks
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
		log.Printf("RBAC check failed for file upload in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return
	}
	if !res.Status.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to modify session workspace"})
		return
	}

	// Verify session exists using reqDyn AFTER RBAC check
	// This prevents enumeration attacks - unauthorized users get same "Forbidden" response
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

	// Check if ambient-content service exists (session must be running)
	serviceName := fmt.Sprintf("ambient-content-%s", session)
	if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
		// Service doesn't exist - session is not running
		log.Printf("PutSessionWorkspaceFile: Content service not found for session %s (session not running)", session)
		c.JSON(http.StatusConflict, gin.H{
			"error": "Session is not running. Start the session to upload files.",
			"hint":  "File uploads require an active session. Start the session and try again.",
		})
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("PutSessionWorkspaceFile: using service %s for session %s", serviceName, session)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Printf("PutSessionWorkspaceFile: failed to read request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file data"})
		return
	}

	// Detect if content is binary and encode accordingly
	encoding := "utf8"
	var content string
	contentType := c.GetHeader("Content-Type")

	// If no Content-Type header, detect from payload
	if contentType == "" {
		contentType = http.DetectContentType(payload)
	}

	// Use base64 for binary content types or if content isn't valid UTF-8
	// Check comprehensive list of binary MIME types and UTF-8 validity
	// IMPORTANT: Validate UTF-8 BEFORE converting to string
	isBinary := isBinaryContentType(contentType) || !utf8.Valid(payload)

	if isBinary {
		encoding = "base64"
		content = base64.StdEncoding.EncodeToString(payload)
		// Don't log user-controlled strings (contentType header) to prevent log injection
		log.Printf("PutSessionWorkspaceFile: detected binary content, using base64 encoding (size=%d, contentTypeLen=%d)", len(payload), len(contentType))
	} else {
		// Only convert to string after validating UTF-8
		content = string(payload)
	}

	wreq := struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}{Path: absPath, Content: content, Encoding: encoding}
	b, err := json.Marshal(wreq)
	if err != nil {
		log.Printf("PutSessionWorkspaceFile: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/write", strings.NewReader(string(b)))
	if err != nil {
		log.Printf("PutSessionWorkspaceFile: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("PutSessionWorkspaceFile: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}

	// Log if content service returned an error
	if resp.StatusCode >= 400 {
		log.Printf("PutSessionWorkspaceFile: content service returned error status %d for path %s: %s", resp.StatusCode, sub, string(rb))
	}

	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), rb)
}

// DeleteSessionWorkspaceFile deletes a file via content service.
func DeleteSessionWorkspaceFile(c *gin.Context) {
	// Get project from context (set by middleware) or param
	project := c.GetString("project")
	if project == "" {
		project = c.Param("projectName")
	}
	session := c.Param("sessionName")

	if project == "" {
		log.Printf("DeleteSessionWorkspaceFile: project is empty, session=%s", session)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Project namespace required"})
		return
	}

	// Get user-scoped K8s clients and validate authentication IMMEDIATELY
	reqK8s, reqDyn := GetK8sClientsForRequest(c)
	if reqK8s == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing authentication token"})
		c.Abort()
		return
	}

	// Validate and sanitize path to prevent directory traversal
	// Use robust path validation that works across platforms
	sub := strings.TrimPrefix(c.Param("path"), "/")
	workspaceBase := "/workspace"

	// Construct absolute path using filepath.Join for path validation
	validationPath := filepath.Join(workspaceBase, sub)

	// Use robust path validation from pathutil package
	// This is more secure than manual string checks and works across platforms
	if !pathutil.IsPathWithinBase(validationPath, workspaceBase) {
		log.Printf("DeleteSessionWorkspaceFile: path traversal attempt detected - path=%q escapes workspace=%q", validationPath, workspaceBase)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: must be within workspace directory"})
		return
	}

	// Use relative path for content service (it has its own StateBaseDir=/workspace)
	// Convert to forward slashes for content service (expects POSIX paths)
	absPath := filepath.ToSlash(sub)

	token := c.GetHeader("Authorization")
	if strings.TrimSpace(token) == "" {
		token = c.GetHeader("X-Forwarded-Access-Token")
	}

	// RBAC check: verify user has update permission on agenticsessions (file operations modify session state)
	// IMPORTANT: RBAC check MUST happen BEFORE checking session existence to prevent enumeration attacks
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
		log.Printf("RBAC check failed for file deletion in project %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify permissions"})
		return
	}
	if !res.Status.Allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to modify session workspace"})
		return
	}

	// Verify session exists using reqDyn AFTER RBAC check
	// This prevents enumeration attacks - unauthorized users get same "Forbidden" response
	gvr := GetAgenticSessionV1Alpha1Resource()
	if _, err := reqDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}
		log.Printf("DeleteSessionWorkspaceFile: Failed to verify session existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify session"})
		return
	}

	// Check if content service exists (session must be running)
	serviceName := getContentServiceName(session)
	if _, err := reqK8s.CoreV1().Services(project).Get(c.Request.Context(), serviceName, v1.GetOptions{}); err != nil {
		log.Printf("DeleteSessionWorkspaceFile: Content service not found for session %s (session not running)", session)
		c.JSON(http.StatusConflict, gin.H{"error": "Session is not running. Start the session to access files."})
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("DeleteSessionWorkspaceFile: using service %s for session %s, path=%s", serviceName, session, absPath)

	// Use DELETE request with path in body
	wreq := struct {
		Path string `json:"path"`
	}{Path: absPath}
	b, err := json.Marshal(wreq)
	if err != nil {
		log.Printf("DeleteSessionWorkspaceFile: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodDelete, endpoint+"/content/delete", strings.NewReader(string(b)))
	if err != nil {
		log.Printf("DeleteSessionWorkspaceFile: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", token)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// Always return JSON for consistency with frontend expectations
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
	} else {
		rb, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("DeleteSessionWorkspaceFile: failed to read error response: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
			return
		}
		// Try to parse error from content service, otherwise use generic message
		var errResp map[string]interface{}
		if err := json.Unmarshal(rb, &errResp); err == nil {
			c.JSON(resp.StatusCode, errResp)
		} else {
			c.JSON(resp.StatusCode, gin.H{"error": "Failed to delete file"})
		}
	}
}

// PushSessionRepo proxies a push request for a given session repo to the per-job content service.
// POST /api/projects/:projectName/agentic-sessions/:sessionName/github/push
// Body: { repoIndex: number, commitMessage?: string, branch?: string }
func PushSessionRepo(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		RepoIndex     int    `json:"repoIndex"`
		CommitMessage string `json:"commitMessage"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}
	log.Printf("pushSessionRepo: request project=%s session=%s repoIndex=%d commitLen=%d", project, session, body.RepoIndex, len(strings.TrimSpace(body.CommitMessage)))

	// Try temp service first (for completed sessions), then regular service
	serviceName := getContentServiceName(session)
	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil || k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("pushSessionRepo: using service %s", serviceName)

	// Simplified: 1) get session; 2) compute repoPath from INPUT repo folder; 3) get output url/branch; 4) proxy
	resolvedRepoPath := ""
	// default branch when not defined on output
	resolvedBranch := fmt.Sprintf("sessions/%s", session)
	resolvedOutputURL := ""
	gvr := GetAgenticSessionV1Alpha1Resource()
	obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read session"})
		return
	}
	spec, _ := obj.Object["spec"].(map[string]interface{})
	repos, _ := spec["repos"].([]interface{})
	if body.RepoIndex < 0 || body.RepoIndex >= len(repos) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repo index"})
		return
	}
	rm, _ := repos[body.RepoIndex].(map[string]interface{})
	// Derive repoPath from input URL folder name
	// Paths are relative to content service's StateBaseDir (which is /workspace)
	if in, ok := rm["input"].(map[string]interface{}); ok {
		if urlv, ok2 := in["url"].(string); ok2 && strings.TrimSpace(urlv) != "" {
			folder := DeriveRepoFolderFromURL(strings.TrimSpace(urlv))
			if folder != "" {
				resolvedRepoPath = folder
			}
		}
	}
	if out, ok := rm["output"].(map[string]interface{}); ok {
		if urlv, ok2 := out["url"].(string); ok2 && strings.TrimSpace(urlv) != "" {
			resolvedOutputURL = strings.TrimSpace(urlv)
		}
		if bs, ok2 := out["branch"].(string); ok2 && strings.TrimSpace(bs) != "" {
			resolvedBranch = strings.TrimSpace(bs)
		} else if bv, ok2 := out["branch"].(*string); ok2 && bv != nil && strings.TrimSpace(*bv) != "" {
			resolvedBranch = strings.TrimSpace(*bv)
		}
	}
	// If input URL missing or unparsable, fall back to numeric index path (last resort)
	if strings.TrimSpace(resolvedRepoPath) == "" {
		if body.RepoIndex >= 0 {
			resolvedRepoPath = fmt.Sprintf("%d", body.RepoIndex)
		} else {
			resolvedRepoPath = ""
		}
	}
	if strings.TrimSpace(resolvedOutputURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing output repo url"})
		return
	}
	log.Printf("pushSessionRepo: resolved repoPath=%q outputUrl=%q branch=%q", resolvedRepoPath, resolvedOutputURL, resolvedBranch)

	payload := map[string]interface{}{
		"repoPath":      resolvedRepoPath,
		"commitMessage": body.CommitMessage,
		"branch":        resolvedBranch,
		"outputRepoUrl": resolvedOutputURL,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("pushSessionRepo: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/github/push", strings.NewReader(string(b)))
	if err != nil {
		log.Printf("pushSessionRepo: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}
	if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
		req.Header.Set("X-Forwarded-Access-Token", v)
	}
	req.Header.Set("Content-Type", "application/json")
	k8sClt, k8sDyn = GetK8sClientsForRequest(c)
	if k8sClt == nil || k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Attach short-lived GitHub token for one-shot authenticated push
	// Load session to get authoritative userId
	gvr = GetAgenticSessionV1Alpha1Resource()
	obj, err = k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{})
	if err == nil {
		spec, _ := obj.Object["spec"].(map[string]interface{})
		userID := ""
		if spec != nil {
			if uc, ok := spec["userContext"].(map[string]interface{}); ok {
				if v, ok := uc["userId"].(string); ok {
					userID = strings.TrimSpace(v)
				}
			}
		}
		if userID != "" {
			if tokenStr, err := GetGitHubToken(c.Request.Context(), k8sClt, k8sDyn, project, userID); err == nil && strings.TrimSpace(tokenStr) != "" {
				req.Header.Set("X-GitHub-Token", tokenStr)
				log.Printf("pushSessionRepo: attached short-lived GitHub token for project=%s session=%s", project, session)
			} else if err != nil {
				log.Printf("pushSessionRepo: failed to resolve GitHub token: %v", err)
			}
		} else {
			log.Printf("pushSessionRepo: session %s/%s missing userContext.userId; proceeding without token", project, session)
		}
	} else {
		log.Printf("pushSessionRepo: failed to read session for token attach: %v", err)
	}

	log.Printf("pushSessionRepo: proxy push project=%s session=%s repoIndex=%d repoPath=%s endpoint=%s", project, session, body.RepoIndex, resolvedRepoPath, endpoint+"/content/github/push")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Log actual error for debugging, but return generic message to avoid leaking internal details
		log.Printf("Bad gateway error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Service temporarily unavailable"})
		return
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("pushSessionRepo: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("pushSessionRepo: content returned status=%d body.snip=%q", resp.StatusCode, func() string {
			s := string(bodyBytes)
			if len(s) > 1500 {
				return s[:1500] + "..."
			}
			return s
		}())
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}
	// Note: status.repos removed from CRD - no longer tracking per-repo status
	log.Printf("pushSessionRepo: content push succeeded status=%d body.len=%d", resp.StatusCode, len(bodyBytes))
	c.Data(http.StatusOK, "application/json", bodyBytes)
}

// AbandonSessionRepo instructs sidecar to discard local changes for a repo.
func AbandonSessionRepo(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	var body struct {
		RepoIndex int    `json:"repoIndex"`
		RepoPath  string `json:"repoPath"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body"})
		return
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := getContentServiceName(session)
	k8sClt, _ := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("AbandonSessionRepo: using service %s", serviceName)
	repoPath := strings.TrimSpace(body.RepoPath)
	if repoPath == "" {
		if body.RepoIndex >= 0 {
			repoPath = fmt.Sprintf("%d", body.RepoIndex)
		} else {
			repoPath = ""
		}
	}
	payload := map[string]interface{}{
		"repoPath": repoPath,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("abandonSessionRepo: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint+"/content/github/abandon", strings.NewReader(string(b)))
	if err != nil {
		log.Printf("abandonSessionRepo: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}
	if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
		req.Header.Set("X-Forwarded-Access-Token", v)
	}
	req.Header.Set("Content-Type", "application/json")
	log.Printf("abandonSessionRepo: proxy abandon project=%s session=%s repoIndex=%d repoPath=%s", project, session, body.RepoIndex, repoPath)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Log actual error for debugging, but return generic message to avoid leaking internal details
		log.Printf("Bad gateway error: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Service temporarily unavailable"})
		return
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("abandonSessionRepo: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("abandonSessionRepo: content returned status=%d body=%s", resp.StatusCode, string(bodyBytes))
		c.Data(resp.StatusCode, "application/json", bodyBytes)
		return
	}
	// Note: status.repos removed from CRD - no longer tracking per-repo status
	c.Data(http.StatusOK, "application/json", bodyBytes)
}

// DiffSessionRepo proxies diff counts for a given session repo to the content sidecar.
// GET /api/projects/:projectName/agentic-sessions/:sessionName/github/diff?repoIndex=0&repoPath=...
func DiffSessionRepo(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	repoIndexStr := strings.TrimSpace(c.Query("repoIndex"))
	repoPath := strings.TrimSpace(c.Query("repoPath"))
	// Paths are relative to content service's StateBaseDir (which is /workspace)
	if repoPath == "" && repoIndexStr != "" {
		repoPath = repoIndexStr
	}
	if repoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing repoPath/repoIndex"})
		return
	}

	// Try temp service first (for completed sessions), then regular service
	serviceName := getContentServiceName(session)
	k8sClt, _ := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}
	endpoint := fmt.Sprintf("http://%s.%s.svc:8080", serviceName, project)
	log.Printf("DiffSessionRepo: using service %s", serviceName)
	url := fmt.Sprintf("%s/content/github/diff?repoPath=%s", endpoint, url.QueryEscape(repoPath))
	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}
	if v := c.GetHeader("X-Forwarded-Access-Token"); v != "" {
		req.Header.Set("X-Forwarded-Access-Token", v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"files": gin.H{
				"added":   0,
				"removed": 0,
			},
			"total_added":   0,
			"total_removed": 0,
		})
		return
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("DiffSessionRepo: failed to read response body: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"files": gin.H{
				"added":   0,
				"removed": 0,
			},
			"total_added":   0,
			"total_removed": 0,
		})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GetReposStatus returns current status of all repositories (branches, current branch, etc.)
// GET /api/projects/:projectName/agentic-sessions/:sessionName/repos/status
func GetReposStatus(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	k8sClt, dynClt := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	// Verify user has access to the session using user-scoped K8s client
	// This ensures RBAC is enforced before we call the runner
	gvr := GetAgenticSessionV1Alpha1Resource()
	_, err := dynClt.Resource(gvr).Namespace(project).Get(context.TODO(), session, v1.GetOptions{})
	if errors.IsNotFound(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}
	if err != nil {
		log.Printf("GetReposStatus: failed to verify session access: %v", err)
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Call runner's /repos/status endpoint directly
	// Authentication flow:
	// 1. Backend validated user has access to session (above)
	// 2. Backend calls runner as trusted internal service (no auth header forwarding)
	// 3. Runner trusts backend's validation
	// Port 8001 matches AG-UI Service defined in operator (sessions.go:1384)
	// If changing this port, also update: operator containerPort, Service port, and AGUI_PORT env
	runnerURL := fmt.Sprintf("http://session-%s.%s.svc.cluster.local:8001/repos/status", session, project)

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, runnerURL, nil)
	if err != nil {
		log.Printf("GetReposStatus: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	// NOTE: Do NOT forward Authorization header to runner (matches pattern of AddWorkflow, AddRepository, RemoveRepo)
	// Runner is treated as a trusted backend service; RBAC enforcement happens in backend

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GetReposStatus: runner not reachable: %v", err)
		// Return empty repos list instead of error for better UX
		c.JSON(http.StatusOK, gin.H{"repos": []interface{}{}})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GetReposStatus: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from runner"})
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("GetReposStatus: runner returned status %d", resp.StatusCode)
		c.JSON(http.StatusOK, gin.H{"repos": []interface{}{}})
		return
	}

	c.Data(http.StatusOK, "application/json", bodyBytes)
}

// GetGitStatus returns git status for a directory in the workspace
// GET /api/projects/:projectName/agentic-sessions/:sessionName/git/status?path=artifacts
func GetGitStatus(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	relativePath := strings.TrimSpace(c.Query("path"))

	if relativePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path parameter required"})
		return
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := relativePath

	// Get content service endpoint
	serviceName := getContentServiceName(session)
	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-status?path=%s", serviceName, project, url.QueryEscape(absPath))

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		log.Printf("GetGitStatus: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	// Attach short-lived GitHub token for authenticated git status
	gvr := GetAgenticSessionV1Alpha1Resource()
	if obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{}); err == nil {
		if spec, _, _ := unstructured.NestedMap(obj.Object, "spec"); spec != nil {
			if uc, ok := spec["userContext"].(map[string]interface{}); ok {
				if userID, ok := uc["userId"].(string); ok && strings.TrimSpace(userID) != "" {
					if tokenStr, err := GetGitHubToken(c.Request.Context(), k8sClt, k8sDyn, project, userID); err == nil && strings.TrimSpace(tokenStr) != "" {
						req.Header.Set("X-GitHub-Token", tokenStr)
						log.Printf("GetGitStatus: attached GitHub token for project=%s session=%s", project, session)
					}
				}
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GetGitStatus: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// ConfigureGitRemote initializes git and configures remote for a workspace directory
// Body: { path: string, remoteURL: string, branch: string }
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/configure-remote
func ConfigureGitRemote(c *gin.Context) {
	project := c.Param("projectName")
	sessionName := c.Param("sessionName")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	var body struct {
		Path      string `json:"path" binding:"required"`
		RemoteURL string `json:"remoteUrl" binding:"required"`
		Branch    string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Branch == "" {
		body.Branch = "main"
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := body.Path

	// Get content service endpoint
	serviceName := getContentServiceName(sessionName)
	k8sClt, _ := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-configure-remote", serviceName, project)

	reqBody, err := json.Marshal(map[string]interface{}{
		"path":      absPath,
		"remoteUrl": body.RemoteURL,
		"branch":    body.Branch,
	})
	if err != nil {
		log.Printf("ConfigureGitRemote: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Printf("ConfigureGitRemote: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	// Get and forward GitHub token for authenticated remote URL
	if GetGitHubToken != nil {
		if token, err := GetGitHubToken(c.Request.Context(), k8sClt, k8sDyn, project, ""); err == nil && token != "" {
			req.Header.Set("X-GitHub-Token", token)
			log.Printf("Forwarding GitHub token for remote configuration")
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	// If successful, persist remote config to session annotations for persistence
	if resp.StatusCode == http.StatusOK {
		// Persist remote config in annotations (supports multiple directories)
		gvr := GetAgenticSessionV1Alpha1Resource()
		item, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), sessionName, v1.GetOptions{})
		if err == nil {
			metadata, _, err := unstructured.NestedMap(item.Object, "metadata")
			if err != nil || metadata == nil {
				metadata = map[string]interface{}{}
			}
			anns, _, err := unstructured.NestedMap(metadata, "annotations")
			if err != nil || anns == nil {
				anns = map[string]interface{}{}
			}

			// Derive safe annotation key from path (use :: as separator to avoid conflicts with hyphens in path)
			annotationKey := strings.ReplaceAll(body.Path, "/", "::")
			anns[fmt.Sprintf("ambient-code.io/remote-%s-url", annotationKey)] = body.RemoteURL
			anns[fmt.Sprintf("ambient-code.io/remote-%s-branch", annotationKey)] = body.Branch
			_ = unstructured.SetNestedMap(metadata, anns, "annotations")
			_ = unstructured.SetNestedMap(item.Object, metadata, "metadata")

			_, err = k8sDyn.Resource(gvr).Namespace(project).Update(c.Request.Context(), item, v1.UpdateOptions{})
			if err != nil {
				log.Printf("Warning: Failed to persist remote config to annotations: %v", err)
			} else {
				log.Printf("Persisted remote config for %s to session annotations: %s@%s", body.Path, body.RemoteURL, body.Branch)
			}
		}
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("ConfigureGitRemote: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// SynchronizeGit commits, pulls, and pushes changes for a workspace directory
// Body: { path: string, message?: string, branch?: string }
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/synchronize
func SynchronizeGit(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path    string `json:"path" binding:"required"`
		Message string `json:"message"`
		Branch  string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Auto-generate commit message if not provided
	if body.Message == "" {
		body.Message = fmt.Sprintf("Session %s - %s", session, time.Now().Format(time.RFC3339))
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := body.Path

	// Get content service endpoint
	serviceName := getContentServiceName(session)
	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-sync", serviceName, project)

	reqBody, err := json.Marshal(map[string]interface{}{
		"path":    absPath,
		"message": body.Message,
		"branch":  body.Branch,
	})
	if err != nil {
		log.Printf("SynchronizeGit: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Printf("SynchronizeGit: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	// Attach short-lived GitHub token for authenticated sync
	gvr := GetAgenticSessionV1Alpha1Resource()
	if obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{}); err == nil {
		if spec, _, _ := unstructured.NestedMap(obj.Object, "spec"); spec != nil {
			if uc, ok := spec["userContext"].(map[string]interface{}); ok {
				if userID, ok := uc["userId"].(string); ok && strings.TrimSpace(userID) != "" {
					if tokenStr, err := GetGitHubToken(c.Request.Context(), k8sClt, k8sDyn, project, userID); err == nil && strings.TrimSpace(tokenStr) != "" {
						req.Header.Set("X-GitHub-Token", tokenStr)
						log.Printf("SynchronizeGit: attached GitHub token for project=%s session=%s", project, session)
					}
				}
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("SynchronizeGit: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GetGitMergeStatus checks if local and remote can merge cleanly
// GET /api/projects/:projectName/agentic-sessions/:sessionName/git/merge-status?path=&branch=
func GetGitMergeStatus(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	relativePath := strings.TrimSpace(c.Query("path"))
	branch := strings.TrimSpace(c.Query("branch"))

	if relativePath == "" {
		relativePath = "artifacts"
	}
	if branch == "" {
		branch = "main"
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := relativePath

	serviceName := getContentServiceName(session)
	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-merge-status?path=%s&branch=%s",
		serviceName, project, url.QueryEscape(absPath), url.QueryEscape(branch))

	req, _ := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint, nil)
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	// Attach short-lived GitHub token for authenticated fetch
	gvr := GetAgenticSessionV1Alpha1Resource()
	if obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{}); err == nil {
		if spec, _, _ := unstructured.NestedMap(obj.Object, "spec"); spec != nil {
			if uc, ok := spec["userContext"].(map[string]interface{}); ok {
				if userID, ok := uc["userId"].(string); ok && strings.TrimSpace(userID) != "" {
					if tokenStr, err := GetGitHubToken(c.Request.Context(), k8sClt, k8sDyn, project, userID); err == nil && strings.TrimSpace(tokenStr) != "" {
						req.Header.Set("X-GitHub-Token", tokenStr)
						log.Printf("GetGitMergeStatus: attached GitHub token for project=%s session=%s", project, session)
					}
				}
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GetGitMergeStatus: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitPullSession pulls changes from remote
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/pull
func GitPullSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Path == "" {
		body.Path = "artifacts"
	}
	if body.Branch == "" {
		body.Branch = "main"
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := body.Path

	serviceName := getContentServiceName(session)
	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-pull", serviceName, project)

	reqBody, err := json.Marshal(map[string]interface{}{
		"path":   absPath,
		"branch": body.Branch,
	})
	if err != nil {
		log.Printf("GitPullSession: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Printf("GitPullSession: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	// Attach short-lived GitHub token for authenticated pull
	gvr := GetAgenticSessionV1Alpha1Resource()
	if obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{}); err == nil {
		if spec, _, _ := unstructured.NestedMap(obj.Object, "spec"); spec != nil {
			if uc, ok := spec["userContext"].(map[string]interface{}); ok {
				if userID, ok := uc["userId"].(string); ok && strings.TrimSpace(userID) != "" {
					if tokenStr, err := GetGitHubToken(c.Request.Context(), k8sClt, k8sDyn, project, userID); err == nil && strings.TrimSpace(tokenStr) != "" {
						req.Header.Set("X-GitHub-Token", tokenStr)
						log.Printf("GitPullSession: attached GitHub token for project=%s session=%s", project, session)
					}
				}
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GitPullSession: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitPushSession pushes changes to remote branch
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/push
func GitPushSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path    string `json:"path"`
		Branch  string `json:"branch"`
		Message string `json:"message"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Path == "" {
		body.Path = "artifacts"
	}
	if body.Branch == "" {
		body.Branch = "main"
	}
	if body.Message == "" {
		body.Message = fmt.Sprintf("Session %s artifacts", session)
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := body.Path

	serviceName := getContentServiceName(session)
	k8sClt, k8sDyn := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-push", serviceName, project)

	reqBody, err := json.Marshal(map[string]interface{}{
		"path":    absPath,
		"branch":  body.Branch,
		"message": body.Message,
	})
	if err != nil {
		log.Printf("GitPushSession: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Printf("GitPushSession: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	// Attach short-lived GitHub token for authenticated push
	gvr := GetAgenticSessionV1Alpha1Resource()
	if obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), session, v1.GetOptions{}); err == nil {
		if spec, _, _ := unstructured.NestedMap(obj.Object, "spec"); spec != nil {
			if uc, ok := spec["userContext"].(map[string]interface{}); ok {
				if userID, ok := uc["userId"].(string); ok && strings.TrimSpace(userID) != "" {
					if tokenStr, err := GetGitHubToken(c.Request.Context(), k8sClt, k8sDyn, project, userID); err == nil && strings.TrimSpace(tokenStr) != "" {
						req.Header.Set("X-GitHub-Token", tokenStr)
						log.Printf("GitPushSession: attached GitHub token for project=%s session=%s", project, session)
					}
				}
			}
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GitPushSession: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitCreateBranchSession creates a new git branch
// POST /api/projects/:projectName/agentic-sessions/:sessionName/git/create-branch
func GitCreateBranchSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")

	var body struct {
		Path       string `json:"path"`
		BranchName string `json:"branchName" binding:"required"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if body.Path == "" {
		body.Path = "artifacts"
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := body.Path

	serviceName := getContentServiceName(session)
	k8sClt, _ := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-create-branch", serviceName, project)

	reqBody, err := json.Marshal(map[string]interface{}{
		"path":       absPath,
		"branchName": body.BranchName,
	})
	if err != nil {
		log.Printf("GitCreateBranchSession: failed to marshal request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare request"})
		return
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		log.Printf("GitCreateBranchSession: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GitCreateBranchSession: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// GitListBranchesSession lists all remote branches
// GET /api/projects/:projectName/agentic-sessions/:sessionName/git/list-branches?path=
func GitListBranchesSession(c *gin.Context) {
	project := c.Param("projectName")
	session := c.Param("sessionName")
	relativePath := strings.TrimSpace(c.Query("path"))

	if relativePath == "" {
		relativePath = "artifacts"
	}

	// Path is relative to content service's StateBaseDir (which is /workspace)
	absPath := relativePath

	serviceName := getContentServiceName(session)
	k8sClt, _ := GetK8sClientsForRequest(c)
	if k8sClt == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc:8080/content/git-list-branches?path=%s",
		serviceName, project, url.QueryEscape(absPath))

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, endpoint, nil)
	if err != nil {
		log.Printf("GitListBranchesSession: failed to create HTTP request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
		return
	}
	if v := c.GetHeader("Authorization"); v != "" {
		req.Header.Set("Authorization", v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content service unavailable"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("GitListBranchesSession: failed to read response body: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read response from content service"})
		return
	}
	c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), bodyBytes)
}

// NOTE: autoTriggerInitialPrompt removed - runner handles INITIAL_PROMPT auto-execution
// Runner POSTs to backend's /agui/run when ready, events flow through middleware
// See: components/runners/claude-code-runner/main.py auto_execute_initial_prompt()
