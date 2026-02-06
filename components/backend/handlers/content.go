package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"ambient-code-backend/git"
	"ambient-code-backend/pathutil"
	"ambient-code-backend/types"

	"github.com/gin-gonic/gin"
)

// StateBaseDir is the base directory for content storage
// Set by main during initialization
var StateBaseDir string

// Git operation functions - set by main package during initialization
// These are set to the actual implementations from git package
var (
	GitPushRepo           func(ctx context.Context, repoDir, commitMessage, outputRepoURL, branch, githubToken string) (string, error)
	GitAbandonRepo        func(ctx context.Context, repoDir string) error
	GitDiffRepo           func(ctx context.Context, repoDir string) (*git.DiffSummary, error)
	GitCheckMergeStatus   func(ctx context.Context, repoDir, branch, githubToken string) (*git.MergeStatus, error)
	GitPullRepo           func(ctx context.Context, repoDir, branch, githubToken string) error
	GitPushToRepo         func(ctx context.Context, repoDir, branch, commitMessage, githubToken string) error
	GitCreateBranch       func(ctx context.Context, repoDir, branchName string) error
	GitListRemoteBranches func(ctx context.Context, repoDir string) ([]string, error)
	GitSyncRepo           func(ctx context.Context, repoDir, commitMessage, branch, githubToken string) error
	// GetRemoteURL retrieves the origin remote URL - mockable for testing
	GetRemoteURL func(ctx context.Context, repoDir string) (string, error)
)

// getGitHubTokenFromContext extracts GitHub token from request header or environment
func getGitHubTokenFromContext(c *gin.Context) string {
	// Prefer header (passed by backend)
	if token := strings.TrimSpace(c.GetHeader("X-GitHub-Token")); token != "" {
		return token
	}
	// Fall back to env var (injected via EnvFrom)
	return strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
}

// getGitLabTokenFromContext extracts GitLab token from request header or environment
func getGitLabTokenFromContext(c *gin.Context) string {
	// Prefer header (passed by backend)
	if token := strings.TrimSpace(c.GetHeader("X-GitLab-Token")); token != "" {
		return token
	}
	// Fall back to env var (injected via EnvFrom)
	return strings.TrimSpace(os.Getenv("GITLAB_TOKEN"))
}

// getGitTokenForURL returns the appropriate token based on the repository URL
func getGitTokenForURL(c *gin.Context, repoURL string) string {
	provider := types.DetectProvider(repoURL)
	if provider == types.ProviderGitLab {
		return getGitLabTokenFromContext(c)
	}
	// Default to GitHub token for github.com or unknown providers
	return getGitHubTokenFromContext(c)
}

// defaultGetRemoteURL is the default implementation of GetRemoteURL
func defaultGetRemoteURL(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("no remote configured: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func init() {
	// Set default implementation if not already set (allows test mocking)
	if GetRemoteURL == nil {
		GetRemoteURL = defaultGetRemoteURL
	}
}

// ContentGitPush handles POST /content/github/push in CONTENT_SERVICE_MODE
func ContentGitPush(c *gin.Context) {
	var body struct {
		RepoPath      string `json:"repoPath"`
		CommitMessage string `json:"commitMessage"`
		OutputRepoURL string `json:"outputRepoUrl"`
		Branch        string `json:"branch"`
	}
	_ = c.BindJSON(&body)
	log.Printf("contentGitPush: request received repoPath=%q outputRepoUrl=%q branch=%q commitLen=%d", body.RepoPath, body.OutputRepoURL, body.Branch, len(strings.TrimSpace(body.CommitMessage)))

	// Require explicit output repo URL and branch from caller
	if strings.TrimSpace(body.OutputRepoURL) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing outputRepoUrl"})
		return
	}
	if strings.TrimSpace(body.Branch) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing branch"})
		return
	}

	repoDir := filepath.Clean(filepath.Join(StateBaseDir, body.RepoPath))
	if body.RepoPath == "" {
		repoDir = StateBaseDir
	}

	// Basic safety: repoDir must be under StateBaseDir
	if !pathutil.IsPathWithinBase(repoDir, StateBaseDir) && repoDir != StateBaseDir {
		log.Printf("contentGitPush: invalid repoPath resolved=%q stateBaseDir=%q", repoDir, StateBaseDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repoPath"})
		return
	}

	log.Printf("contentGitPush: using repoDir=%q (stateBaseDir=%q)", repoDir, StateBaseDir)

	// Get appropriate token based on repository URL
	gitToken := getGitTokenForURL(c, body.OutputRepoURL)
	log.Printf("contentGitPush: tokenHeaderPresent=%t url.host.redacted=%t branch=%q", gitToken != "", strings.HasPrefix(body.OutputRepoURL, "https://"), body.Branch)

	// Call refactored git push function
	out, err := GitPushRepo(c.Request.Context(), repoDir, body.CommitMessage, body.OutputRepoURL, body.Branch, gitToken)
	if err != nil {
		if out == "" {
			// No changes to commit
			c.JSON(http.StatusOK, gin.H{"ok": true, "message": "no changes"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "push failed", "stderr": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "stdout": out})
}

// ContentGitAbandon handles POST /content/github/abandon
func ContentGitAbandon(c *gin.Context) {
	var body struct {
		RepoPath string `json:"repoPath"`
	}
	_ = c.BindJSON(&body)
	log.Printf("contentGitAbandon: request repoPath=%q", body.RepoPath)

	repoDir := filepath.Clean(filepath.Join(StateBaseDir, body.RepoPath))
	if body.RepoPath == "" {
		repoDir = StateBaseDir
	}

	if !pathutil.IsPathWithinBase(repoDir, StateBaseDir) && repoDir != StateBaseDir {
		log.Printf("contentGitAbandon: invalid repoPath resolved=%q base=%q", repoDir, StateBaseDir)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repoPath"})
		return
	}

	log.Printf("contentGitAbandon: using repoDir=%q", repoDir)

	if err := GitAbandonRepo(c.Request.Context(), repoDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ContentGitDiff handles GET /content/github/diff
func ContentGitDiff(c *gin.Context) {
	repoPath := strings.TrimSpace(c.Query("repoPath"))
	if repoPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing repoPath"})
		return
	}

	repoDir := filepath.Clean(filepath.Join(StateBaseDir, repoPath))
	if !pathutil.IsPathWithinBase(repoDir, StateBaseDir) && repoDir != StateBaseDir {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid repoPath"})
		return
	}

	log.Printf("contentGitDiff: repoPath=%q repoDir=%q", repoPath, repoDir)

	summary, err := GitDiffRepo(c.Request.Context(), repoDir)
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

	c.JSON(http.StatusOK, gin.H{
		"files": gin.H{
			"added":   summary.FilesAdded,
			"removed": summary.FilesRemoved,
		},
		"total_added":   summary.TotalAdded,
		"total_removed": summary.TotalRemoved,
	})
}

// ContentGitStatus handles GET /content/git-status?path=
func ContentGitStatus(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	// Check if directory exists
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		c.JSON(http.StatusOK, gin.H{
			"initialized": false,
			"hasChanges":  false,
		})
		return
	}

	// Check if git repo exists
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"initialized": false,
			"hasChanges":  false,
		})
		return
	}

	// Get current branch
	branchCmd := exec.CommandContext(c.Request.Context(), "git", "rev-parse", "--abbrev-ref", "HEAD")
	branchCmd.Dir = abs
	branchOut, _ := branchCmd.Output()
	currentBranch := strings.TrimSpace(string(branchOut))

	// Check if remote is configured
	remoteCmd := exec.CommandContext(c.Request.Context(), "git", "remote", "get-url", "origin")
	remoteCmd.Dir = abs
	remoteOut, remoteErr := remoteCmd.Output()
	remoteURL := strings.TrimSpace(string(remoteOut))
	hasRemote := remoteErr == nil && remoteURL != ""

	// Get git status using existing git package
	summary, err := GitDiffRepo(c.Request.Context(), abs)
	if err != nil {
		log.Printf("ContentGitStatus: git diff failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"initialized": true,
			"hasChanges":  false,
			"branch":      currentBranch,
			"remoteUrl":   remoteURL,
			"hasRemote":   hasRemote,
		})
		return
	}

	hasChanges := summary.FilesAdded > 0 || summary.FilesRemoved > 0 || summary.TotalAdded > 0 || summary.TotalRemoved > 0

	c.JSON(http.StatusOK, gin.H{
		"initialized":      true,
		"hasChanges":       hasChanges,
		"branch":           currentBranch,
		"remoteUrl":        remoteURL,
		"hasRemote":        hasRemote,
		"filesAdded":       summary.FilesAdded,
		"filesRemoved":     summary.FilesRemoved,
		"uncommittedFiles": summary.FilesAdded + summary.FilesRemoved,
		"totalAdded":       summary.TotalAdded,
		"totalRemoved":     summary.TotalRemoved,
	})
}

// ContentGitConfigureRemote handles POST /content/git-configure-remote
// Body: { path: string, remoteURL: string, branch: string }
func ContentGitConfigureRemote(c *gin.Context) {
	var body struct {
		Path      string `json:"path"`
		RemoteURL string `json:"remoteUrl"`
		Branch    string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	// Check if directory exists
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "directory not found"})
		return
	}

	// Initialize git if not already
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		if err := git.InitRepo(c.Request.Context(), abs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to initialize git"})
			return
		}
		log.Printf("Initialized git repository at %s", abs)
	}

	// Get appropriate token and inject into URL for authentication
	// SECURITY: remoteURL may contain embedded token after injection - never log it
	remoteURL := body.RemoteURL
	token := getGitTokenForURL(c, body.RemoteURL)
	if token != "" {
		if authenticatedURL, err := git.InjectGitToken(remoteURL, token); err == nil {
			remoteURL = authenticatedURL
			log.Printf("ContentConfigureRemote: configured authentication for provider=%s tokenLen=%d", types.DetectProvider(body.RemoteURL), len(token))
		}
	}

	// Configure remote with authenticated URL
	if err := git.ConfigureRemote(c.Request.Context(), abs, "origin", remoteURL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to configure remote"})
		return
	}

	log.Printf("Configured remote for %s: %s", abs, body.RemoteURL)

	// Fetch from remote so merge status can be checked
	// This is best-effort - don't fail if fetch fails
	branch := body.Branch
	if branch == "" {
		branch = "main"
	}
	cmd := exec.CommandContext(c.Request.Context(), "git", "fetch", "origin", branch)
	cmd.Dir = abs
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("Initial fetch after configure remote failed (non-fatal): %v (output: %s)", err, string(out))
	} else {
		log.Printf("Fetched origin/%s after configuring remote", branch)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "remote configured",
		"remote":  body.RemoteURL,
		"branch":  body.Branch,
	})
}

// ContentGitSync handles POST /content/git-sync
// Body: { path: string, message: string, branch: string }
func ContentGitSync(c *gin.Context) {
	var body struct {
		Path    string `json:"path"`
		Message string `json:"message"`
		Branch  string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	// Check if git repo exists
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "git repository not initialized"})
		return
	}

	// Get remote URL to determine which token to use
	remoteURL, err := GetRemoteURL(c.Request.Context(), abs)
	if err != nil {
		log.Printf("ContentGitSync: failed to get remote URL for %s: %v", abs, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no remote configured"})
		return
	}

	// Perform git sync operations with authentication
	gitToken := getGitTokenForURL(c, remoteURL)
	if err := GitSyncRepo(c.Request.Context(), abs, body.Message, body.Branch, gitToken); err != nil {
		// Log actual error for debugging, but return generic message to avoid leaking internal details
		log.Printf("Internal server error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	log.Printf("Synchronized git repository at %s to branch %s", abs, body.Branch)
	c.JSON(http.StatusOK, gin.H{
		"message": "synchronized successfully",
		"branch":  body.Branch,
	})
}

// ContentWrite handles POST /content/write when running in CONTENT_SERVICE_MODE
func ContentWrite(c *gin.Context) {
	var req struct {
		Path     string `json:"path"`
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ContentWrite: bind JSON failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("ContentWrite: path=%q contentLen=%d encoding=%q StateBaseDir=%q", req.Path, len(req.Content), req.Encoding, StateBaseDir)

	path := filepath.Clean("/" + strings.TrimSpace(req.Path))
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		log.Printf("ContentWrite: path traversal attempt rejected: path=%q abs=%q", path, abs)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	log.Printf("ContentWrite: absolute path=%q", abs)

	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		log.Printf("ContentWrite: mkdir failed for %q: %v", filepath.Dir(abs), err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create directory"})
		return
	}
	var data []byte
	if strings.EqualFold(req.Encoding, "base64") {
		b, err := base64.StdEncoding.DecodeString(req.Content)
		if err != nil {
			log.Printf("ContentWrite: base64 decode failed: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 content"})
			return
		}
		data = b
	} else {
		data = []byte(req.Content)
	}
	if err := os.WriteFile(abs, data, 0644); err != nil {
		log.Printf("ContentWrite: write failed for %q: %v", abs, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}
	log.Printf("ContentWrite: successfully wrote %d bytes to %q", len(data), abs)
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ContentRead handles GET /content/file?path=
func ContentRead(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	log.Printf("ContentRead: requested path=%q StateBaseDir=%q", c.Query("path"), StateBaseDir)
	log.Printf("ContentRead: cleaned path=%q", path)

	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		log.Printf("ContentRead: path traversal attempt rejected: path=%q abs=%q", path, abs)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	log.Printf("ContentRead: absolute path=%q", abs)

	b, err := os.ReadFile(abs)
	if err != nil {
		log.Printf("ContentRead: read failed for %q: %v", abs, err)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "read failed"})
		}
		return
	}
	log.Printf("ContentRead: successfully read %d bytes from %q", len(b), abs)
	c.Data(http.StatusOK, "application/octet-stream", b)
}

// ContentList handles GET /content/list?path=
func ContentList(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	log.Printf("ContentList: requested path=%q", c.Query("path"))
	log.Printf("ContentList: cleaned path=%q", path)
	log.Printf("ContentList: StateBaseDir=%q", StateBaseDir)

	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		log.Printf("ContentList: path traversal attempt rejected: path=%q abs=%q", path, abs)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	log.Printf("ContentList: absolute path=%q", abs)

	info, err := os.Stat(abs)
	if err != nil {
		log.Printf("ContentList: stat failed for %q: %v", abs, err)
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "stat failed"})
		}
		return
	}
	if !info.IsDir() {
		// If it's a file, return single entry metadata
		c.JSON(http.StatusOK, gin.H{"items": []gin.H{{
			"name":       filepath.Base(abs),
			"path":       path,
			"isDir":      false,
			"size":       info.Size(),
			"modifiedAt": info.ModTime().UTC().Format(time.RFC3339),
		}}})
		return
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "readdir failed"})
		return
	}
	items := make([]gin.H, 0, len(entries))
	for _, e := range entries {
		info, _ := e.Info()
		items = append(items, gin.H{
			"name":       e.Name(),
			"path":       filepath.Join(path, e.Name()),
			"isDir":      e.IsDir(),
			"size":       info.Size(),
			"modifiedAt": info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	log.Printf("ContentList: returning %d items for path=%q", len(items), path)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// ContentWorkflowMetadata handles GET /content/workflow-metadata?session=
// Parses .claude/commands/*.md and .claude/agents/*.md files from active workflow
func ContentWorkflowMetadata(c *gin.Context) {
	sessionName := c.Query("session")
	if sessionName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing session parameter"})
		return
	}

	log.Printf("ContentWorkflowMetadata: session=%q", sessionName)

	// Find active workflow directory
	workflowDir := findActiveWorkflowDir(sessionName)
	if workflowDir == "" {
		log.Printf("ContentWorkflowMetadata: no active workflow found for session=%q", sessionName)
		c.JSON(http.StatusOK, gin.H{
			"commands": []interface{}{},
			"agents":   []interface{}{},
			"config":   gin.H{"artifactsDir": "artifacts"}, // Default platform folder when no workflow
		})
		return
	}

	log.Printf("ContentWorkflowMetadata: found workflow at %q", workflowDir)

	// Parse ambient.json configuration
	ambientConfig := parseAmbientConfig(workflowDir)

	// Parse commands from .claude/commands/*.md
	commandsDir := filepath.Join(workflowDir, ".claude", "commands")
	commands := []map[string]interface{}{}

	if files, err := os.ReadDir(commandsDir); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				filePath := filepath.Join(commandsDir, file.Name())
				metadata := parseFrontmatter(filePath)
				commandName := strings.TrimSuffix(file.Name(), ".md")

				displayName := metadata["displayName"]
				if displayName == "" {
					displayName = commandName
				}

				// Parse order field from frontmatter, default to MaxInt32 for unordered commands
				order := int(^uint(0) >> 1) // MaxInt
				if orderStr := metadata["order"]; orderStr != "" {
					if parsed, err := strconv.Atoi(orderStr); err == nil {
						order = parsed
					}
				}

				// Use full command name as slash command (e.g., /speckit.rfe.start)
				commands = append(commands, map[string]interface{}{
					"id":           commandName,
					"name":         displayName,
					"description":  metadata["description"],
					"slashCommand": "/" + commandName,
					"icon":         metadata["icon"],
					"order":        order,
				})
			}
		}

		// Sort commands by order field (ascending)
		sort.Slice(commands, func(i, j int) bool {
			iOrder, iOk := commands[i]["order"].(int)
			jOrder, jOk := commands[j]["order"].(int)
			if !iOk {
				iOrder = int(^uint(0) >> 1) // MaxInt
			}
			if !jOk {
				jOrder = int(^uint(0) >> 1) // MaxInt
			}
			// If orders are equal, sort alphabetically by id for consistent ordering
			if iOrder == jOrder {
				iID, _ := commands[i]["id"].(string)
				jID, _ := commands[j]["id"].(string)
				return iID < jID
			}
			return iOrder < jOrder
		})

		log.Printf("ContentWorkflowMetadata: found %d commands", len(commands))
	} else {
		log.Printf("ContentWorkflowMetadata: commands directory not found or unreadable: %v", err)
	}

	// Parse agents from .claude/agents/*.md
	agentsDir := filepath.Join(workflowDir, ".claude", "agents")
	agents := []map[string]interface{}{}

	if files, err := os.ReadDir(agentsDir); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				filePath := filepath.Join(agentsDir, file.Name())
				metadata := parseFrontmatter(filePath)
				agentID := strings.TrimSuffix(file.Name(), ".md")

				agents = append(agents, map[string]interface{}{
					"id":          agentID,
					"name":        metadata["name"],
					"description": metadata["description"],
					"tools":       metadata["tools"],
				})
			}
		}
		log.Printf("ContentWorkflowMetadata: found %d agents", len(agents))
	} else {
		log.Printf("ContentWorkflowMetadata: agents directory not found or unreadable: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"commands": commands,
		"agents":   agents,
		"config": gin.H{
			"name":         ambientConfig.Name,
			"description":  ambientConfig.Description,
			"systemPrompt": ambientConfig.SystemPrompt,
			"artifactsDir": ambientConfig.ArtifactsDir,
		},
	})
}

// parseFrontmatter extracts YAML frontmatter from a markdown file
func parseFrontmatter(filePath string) map[string]string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("parseFrontmatter: failed to read %q: %v", filePath, err)
		return map[string]string{}
	}

	str := string(content)
	if !strings.HasPrefix(str, "---\n") {
		return map[string]string{}
	}

	// Find end of frontmatter
	endIdx := strings.Index(str[4:], "\n---")
	if endIdx == -1 {
		return map[string]string{}
	}

	frontmatter := str[4 : 4+endIdx]
	result := map[string]string{}

	// Simple key: value parsing
	for _, line := range strings.Split(frontmatter, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
			result[key] = value
		}
	}

	return result
}

// AmbientConfig represents the ambient.json configuration
type AmbientConfig struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	SystemPrompt string `json:"systemPrompt"`
	ArtifactsDir string `json:"artifactsDir"`
}

// parseAmbientConfig reads and parses ambient.json from workflow directory
// Returns default config if file doesn't exist (not an error)
// For custom workflows without ambient.json, returns empty artifactsDir (root directory)
// allowing them to manage their own structure
func parseAmbientConfig(workflowDir string) *AmbientConfig {
	configPath := filepath.Join(workflowDir, ".ambient", "ambient.json")

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("parseAmbientConfig: no ambient.json found at %q, using defaults", configPath)
		return &AmbientConfig{
			ArtifactsDir: "", // Empty string means root (custom workflows manage their own structure)
		}
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("parseAmbientConfig: failed to read %q: %v", configPath, err)
		return &AmbientConfig{ArtifactsDir: ""}
	}

	// Parse JSON
	var config AmbientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		log.Printf("parseAmbientConfig: failed to parse JSON from %q: %v", configPath, err)
		return &AmbientConfig{ArtifactsDir: ""}
	}

	log.Printf("parseAmbientConfig: loaded config: name=%q artifactsDir=%q", config.Name, config.ArtifactsDir)
	return &config
}

// findActiveWorkflowDir finds the active workflow directory for a session
func findActiveWorkflowDir(sessionName string) string {
	// Workflows are stored at {StateBaseDir}/workflows/{workflow-name}
	// The runner clones workflows to /workspace/workflows/ at runtime
	workflowsBase := filepath.Join(StateBaseDir, "workflows")

	entries, err := os.ReadDir(workflowsBase)
	if err != nil {
		log.Printf("findActiveWorkflowDir: failed to read workflows directory %q: %v", workflowsBase, err)
		return ""
	}

	// Find first directory that has .claude subdirectory (excluding temp clones)
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "default" && !strings.HasSuffix(entry.Name(), "-clone-temp") {
			claudeDir := filepath.Join(workflowsBase, entry.Name(), ".claude")
			if stat, err := os.Stat(claudeDir); err == nil && stat.IsDir() {
				return filepath.Join(workflowsBase, entry.Name())
			}
		}
	}

	return ""
}

// ContentGitMergeStatus handles GET /content/git-merge-status?path=&branch=
func ContentGitMergeStatus(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))
	branch := strings.TrimSpace(c.Query("branch"))

	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if branch == "" {
		branch = "main"
	}

	// Check if git repo exists
	gitDir := filepath.Join(abs, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"canMergeClean":      true,
			"localChanges":       0,
			"remoteCommitsAhead": 0,
			"conflictingFiles":   []string{},
			"remoteBranchExists": false,
		})
		return
	}

	// Get remote URL to determine which token to use
	// Proceed without token if no remote configured - merge status check is read-only and may
	// be called before remote is configured (e.g., for local-only repositories)
	var gitToken string
	if remoteURL, err := GetRemoteURL(c.Request.Context(), abs); err == nil {
		gitToken = getGitTokenForURL(c, remoteURL)
	}

	status, err := GitCheckMergeStatus(c.Request.Context(), abs, branch, gitToken)
	if err != nil {
		log.Printf("ContentGitMergeStatus: check failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, status)
}

// ContentGitPull handles POST /content/git-pull
// Body: { path: string, branch: string }
func ContentGitPull(c *gin.Context) {
	var body struct {
		Path   string `json:"path"`
		Branch string `json:"branch"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if body.Branch == "" {
		body.Branch = "main"
	}

	// Get remote URL to determine which token to use
	remoteURL, err := GetRemoteURL(c.Request.Context(), abs)
	if err != nil {
		log.Printf("ContentGitPull: failed to get remote URL for %s: %v", abs, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no remote configured"})
		return
	}

	gitToken := getGitTokenForURL(c, remoteURL)
	if err := GitPullRepo(c.Request.Context(), abs, body.Branch, gitToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Pulled changes from origin/%s in %s", body.Branch, abs)
	c.JSON(http.StatusOK, gin.H{"message": "pulled successfully", "branch": body.Branch})
}

// ContentGitPushToBranch handles POST /content/git-push
// Body: { path: string, branch: string, message: string }
func ContentGitPushToBranch(c *gin.Context) {
	var body struct {
		Path    string `json:"path"`
		Branch  string `json:"branch"`
		Message string `json:"message"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if body.Branch == "" {
		body.Branch = "main"
	}

	if body.Message == "" {
		body.Message = "Session artifacts update"
	}

	// Get remote URL to determine which token to use
	remoteURL, err := GetRemoteURL(c.Request.Context(), abs)
	if err != nil {
		log.Printf("ContentGitPushToBranch: failed to get remote URL for %s: %v", abs, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no remote configured"})
		return
	}

	gitToken := getGitTokenForURL(c, remoteURL)
	if err := GitPushToRepo(c.Request.Context(), abs, body.Branch, body.Message, gitToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Pushed changes to origin/%s in %s", body.Branch, abs)
	c.JSON(http.StatusOK, gin.H{"message": "pushed successfully", "branch": body.Branch})
}

// ContentGitCreateBranch handles POST /content/git-create-branch
// Body: { path: string, branchName: string }
func ContentGitCreateBranch(c *gin.Context) {
	var body struct {
		Path       string `json:"path"`
		BranchName string `json:"branchName"`
	}

	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	path := filepath.Clean("/" + body.Path)
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	if body.BranchName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "branchName is required"})
		return
	}

	if err := GitCreateBranch(c.Request.Context(), abs, body.BranchName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("Created branch %s in %s", body.BranchName, abs)
	c.JSON(http.StatusOK, gin.H{"message": "branch created", "branchName": body.BranchName})
}

// ContentGitListBranches handles GET /content/git-list-branches?path=
func ContentGitListBranches(c *gin.Context) {
	path := filepath.Clean("/" + strings.TrimSpace(c.Query("path")))

	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}

	branches, err := GitListRemoteBranches(c.Request.Context(), abs)
	if err != nil {
		// Log actual error for debugging, but return generic message to avoid leaking internal details
		log.Printf("Internal server error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"branches": branches})
}

// ContentDelete handles DELETE /content/delete when running in CONTENT_SERVICE_MODE
func ContentDelete(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("ContentDelete: bind JSON failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("ContentDelete: path=%q StateBaseDir=%q", req.Path, StateBaseDir)

	path := filepath.Clean("/" + strings.TrimSpace(req.Path))
	abs := filepath.Join(StateBaseDir, path)
	// Verify abs is within StateBaseDir to prevent path traversal
	if !pathutil.IsPathWithinBase(abs, StateBaseDir) {
		log.Printf("ContentDelete: path traversal attempt rejected: path=%q abs=%q", path, abs)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path"})
		return
	}
	log.Printf("ContentDelete: absolute path=%q", abs)

	// Check if file exists
	if _, err := os.Stat(abs); os.IsNotExist(err) {
		log.Printf("ContentDelete: file not found: %q", abs)
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}

	// Delete the file
	if err := os.Remove(abs); err != nil {
		log.Printf("ContentDelete: delete failed for %q: %v", abs, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete file"})
		return
	}

	log.Printf("ContentDelete: successfully deleted %q", abs)
	c.JSON(http.StatusOK, gin.H{"message": "file deleted successfully"})
}
