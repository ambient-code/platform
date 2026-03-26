package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Marketplace types

type MarketplaceSource struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Branch      string `json:"branch"`
	Path        string `json:"path,omitempty"`
	CatalogURL  string `json:"catalogUrl,omitempty"`
	Description string `json:"description,omitempty"`
}

type ScanRequest struct {
	URL    string `json:"url"`
	Branch string `json:"branch"`
	Path   string `json:"path,omitempty"`
}

type ScanResult struct {
	Items        []DiscoveredItem `json:"items"`
	IsWorkflow   bool             `json:"isWorkflow"`
	HasClaudeMd  bool             `json:"hasClaudeMd"`
	WorkflowName string           `json:"workflowName,omitempty"`
	WorkflowDesc string           `json:"workflowDescription,omitempty"`
}

type DiscoveredItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	FilePath    string `json:"filePath"`
}

type InstalledItem struct {
	SourceURL    string `json:"sourceUrl"`
	SourceBranch string `json:"sourceBranch,omitempty"`
	SourcePath   string `json:"sourcePath,omitempty"`
	ItemID       string `json:"itemId"`
	ItemType     string `json:"itemType"`
	ItemName     string `json:"itemName,omitempty"`
	FilePath     string `json:"filePath,omitempty"`
}

// Catalog cache (follows ootbWorkflowsCache pattern)
type catalogCache struct {
	mu       sync.RWMutex
	data     json.RawMessage
	cachedAt time.Time
	cacheKey string
}

var (
	mktCatalogCache    = &catalogCache{}
	mktCatalogCacheTTL = 5 * time.Minute
)

// ListMarketplaceSources reads marketplace sources from the marketplace-sources ConfigMap.
// GET /api/marketplace/sources
func ListMarketplaceSources(c *gin.Context) {
	if K8sClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "K8s client not initialized"})
		return
	}

	cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(c.Request.Context(), "marketplace-sources", v1.GetOptions{})
	if err != nil {
		log.Printf("ListMarketplaceSources: failed to read ConfigMap: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read marketplace sources"})
		return
	}

	sourcesJSON, ok := cm.Data["sources.json"]
	if !ok {
		c.JSON(http.StatusOK, gin.H{"sources": []MarketplaceSource{}})
		return
	}

	var sources []MarketplaceSource
	if err := json.Unmarshal([]byte(sourcesJSON), &sources); err != nil {
		log.Printf("ListMarketplaceSources: failed to parse sources.json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse marketplace sources"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"sources": sources})
}

// GetMarketplaceCatalog fetches and caches a remote catalog for a given source index.
// GET /api/marketplace/sources/:idx/catalog
func GetMarketplaceCatalog(c *gin.Context) {
	if K8sClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "K8s client not initialized"})
		return
	}

	idxStr := c.Param("idx")
	idx, err := strconv.Atoi(idxStr)
	if err != nil || idx < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source index"})
		return
	}

	// Read sources from ConfigMap
	cm, err := K8sClient.CoreV1().ConfigMaps(Namespace).Get(c.Request.Context(), "marketplace-sources", v1.GetOptions{})
	if err != nil {
		log.Printf("GetMarketplaceCatalog: failed to read ConfigMap: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read marketplace sources"})
		return
	}

	var sources []MarketplaceSource
	if err := json.Unmarshal([]byte(cm.Data["sources.json"]), &sources); err != nil {
		log.Printf("GetMarketplaceCatalog: failed to parse sources.json: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse marketplace sources"})
		return
	}

	if idx >= len(sources) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Source index out of range"})
		return
	}

	source := sources[idx]
	if source.CatalogURL == "" {
		c.JSON(http.StatusOK, gin.H{"items": []interface{}{}})
		return
	}

	cacheKey := source.CatalogURL

	// Check cache (read lock)
	mktCatalogCache.mu.RLock()
	if mktCatalogCache.cacheKey == cacheKey && time.Since(mktCatalogCache.cachedAt) < mktCatalogCacheTTL && mktCatalogCache.data != nil {
		data := mktCatalogCache.data
		mktCatalogCache.mu.RUnlock()
		log.Printf("GetMarketplaceCatalog: returning cached catalog (age: %v)", time.Since(mktCatalogCache.cachedAt).Round(time.Second))
		c.Data(http.StatusOK, "application/json", data)
		return
	}
	mktCatalogCache.mu.RUnlock()

	// Fetch remote catalog
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.CatalogURL, nil)
	if err != nil {
		log.Printf("GetMarketplaceCatalog: failed to create request: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch catalog"})
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("GetMarketplaceCatalog: failed to fetch catalog from %s: %v", source.CatalogURL, err)
		// Return stale cache if available
		mktCatalogCache.mu.RLock()
		if mktCatalogCache.data != nil && mktCatalogCache.cacheKey == cacheKey {
			data := mktCatalogCache.data
			mktCatalogCache.mu.RUnlock()
			log.Printf("GetMarketplaceCatalog: returning stale cached catalog due to fetch error")
			c.Data(http.StatusOK, "application/json", data)
			return
		}
		mktCatalogCache.mu.RUnlock()
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch catalog"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("GetMarketplaceCatalog: catalog returned status %d", resp.StatusCode)
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("Catalog returned status %d", resp.StatusCode)})
		return
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB limit
	if err != nil {
		log.Printf("GetMarketplaceCatalog: failed to read catalog body: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read catalog response"})
		return
	}

	// Normalize catalog: ai-helpers uses {tools: {skills: [...], commands: [...], agents: [...]}}
	// We flatten into a single items array with a "category" field.
	normalized, err := normalizeCatalog(body)
	if err != nil {
		log.Printf("GetMarketplaceCatalog: failed to normalize catalog: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to parse catalog"})
		return
	}

	// Update cache with normalized data
	mktCatalogCache.mu.Lock()
	mktCatalogCache.data = normalized
	mktCatalogCache.cachedAt = time.Now()
	mktCatalogCache.cacheKey = cacheKey
	mktCatalogCache.mu.Unlock()

	log.Printf("GetMarketplaceCatalog: fetched catalog from %s (cached for %v)", source.CatalogURL, mktCatalogCacheTTL)
	c.Data(http.StatusOK, "application/json", normalized)
}

// normalizeCatalog converts various catalog formats into a consistent {"items": [...]} response.
// Supports the ai-helpers format: {"tools": {"skills": [...], "commands": [...], "agents": [...]}}
// and a flat format: {"items": [...]} or just [...].
func normalizeCatalog(raw []byte) (json.RawMessage, error) {
	// Try ai-helpers format: {tools: {skills: [], commands: [], agents: []}}
	var aiHelpers struct {
		Tools map[string][]json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(raw, &aiHelpers); err == nil && aiHelpers.Tools != nil {
		var items []map[string]interface{}
		categoryMap := map[string]string{
			"skills":   "skill",
			"commands": "command",
			"agents":   "agent",
		}
		for toolType, entries := range aiHelpers.Tools {
			category, ok := categoryMap[toolType]
			if !ok {
				continue // skip unknown types like "gemini"
			}
			for _, entry := range entries {
				var item map[string]interface{}
				if err := json.Unmarshal(entry, &item); err == nil {
					item["category"] = category
					// Ensure id field exists (skills/agents have it, commands may not)
					if _, hasID := item["id"]; !hasID {
						if name, ok := item["name"].(string); ok {
							item["id"] = name
						}
					}
					items = append(items, item)
				}
			}
		}
		result, err := json.Marshal(gin.H{"items": items})
		return result, err
	}

	// Try already-normalized format: {"items": [...]}
	var wrapped struct {
		Items json.RawMessage `json:"items"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil && wrapped.Items != nil {
		return raw, nil
	}

	// Try flat array format: [...]
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err == nil {
		result, err := json.Marshal(gin.H{"items": arr})
		return result, err
	}

	return nil, fmt.Errorf("unrecognized catalog format")
}

// ScanGitSource clones a git repo and scans for skills, commands, agents, and workflows.
// POST /api/marketplace/scan
func ScanGitSource(c *gin.Context) {
	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if req.URL == "" || req.Branch == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url and branch are required"})
		return
	}

	// Validate URL scheme (only allow https:// and git:// to prevent SSRF with file://, ftp://, etc.)
	if !strings.HasPrefix(req.URL, "https://") && !strings.HasPrefix(req.URL, "git@") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only HTTPS and SSH Git URLs are allowed"})
		return
	}

	// Validate path doesn't escape the clone directory
	if req.Path != "" && (strings.Contains(req.Path, "..") || filepath.IsAbs(req.Path)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path: must be relative without parent directory references"})
		return
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "marketplace-scan-*")
	if err != nil {
		log.Printf("ScanGitSource: failed to create temp dir: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temp directory"})
		return
	}
	defer os.RemoveAll(tmpDir)

	// Clone the repo
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "-b", req.Branch, req.URL, tmpDir)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("ScanGitSource: git clone failed: %v, output: %s", err, string(output))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to clone repository"})
		return
	}

	// Determine scan root
	scanRoot := tmpDir
	if req.Path != "" {
		scanRoot = filepath.Join(tmpDir, req.Path)
	}

	result := ScanResult{
		Items: []DiscoveredItem{},
	}

	// Check for CLAUDE.md
	if _, err := os.Stat(filepath.Join(scanRoot, "CLAUDE.md")); err == nil {
		result.HasClaudeMd = true
	}

	// Check for .ambient/ambient.json (workflow)
	ambientJSONPath := filepath.Join(scanRoot, ".ambient", "ambient.json")
	if data, err := os.ReadFile(ambientJSONPath); err == nil {
		result.IsWorkflow = true
		var ambientConfig struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(data, &ambientConfig); err == nil {
			result.WorkflowName = ambientConfig.Name
			result.WorkflowDesc = ambientConfig.Description
		}
	}

	// Track seen IDs to deduplicate across .claude/ and root-level patterns
	seenIDs := make(map[string]bool)

	// Scan for skills in both {scanRoot}/.claude/skills/ and {scanRoot}/skills/
	for _, skillsDir := range []string{
		filepath.Join(scanRoot, ".claude", "skills"),
		filepath.Join(scanRoot, "skills"),
	} {
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			id := entry.Name()
			if seenIDs["skill:"+id] {
				continue
			}
			skillMDPath := filepath.Join(skillsDir, id, "SKILL.md")
			if name, desc := parseFrontmatter(skillMDPath); name != "" {
				relPath, _ := filepath.Rel(tmpDir, skillMDPath)
				result.Items = append(result.Items, DiscoveredItem{
					ID:          id,
					Name:        name,
					Description: desc,
					Type:        "skill",
					FilePath:    relPath,
				})
				seenIDs["skill:"+id] = true
			}
		}
	}

	// Scan for commands in both {scanRoot}/.claude/commands/ and {scanRoot}/commands/
	for _, commandsDir := range []string{
		filepath.Join(scanRoot, ".claude", "commands"),
		filepath.Join(scanRoot, "commands"),
	} {
		entries, err := os.ReadDir(commandsDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".md")
			if seenIDs["command:"+id] {
				continue
			}
			cmdPath := filepath.Join(commandsDir, entry.Name())
			name, desc := parseFrontmatter(cmdPath)
			if name == "" {
				name = id
			}
			relPath, _ := filepath.Rel(tmpDir, cmdPath)
			result.Items = append(result.Items, DiscoveredItem{
				ID:          id,
				Name:        name,
				Description: desc,
				Type:        "command",
				FilePath:    relPath,
			})
			seenIDs["command:"+id] = true
		}
	}

	// Scan for agents in both {scanRoot}/.claude/agents/ and {scanRoot}/agents/
	for _, agentsDir := range []string{
		filepath.Join(scanRoot, ".claude", "agents"),
		filepath.Join(scanRoot, "agents"),
	} {
		entries, err := os.ReadDir(agentsDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			id := strings.TrimSuffix(entry.Name(), ".md")
			if seenIDs["agent:"+id] {
				continue
			}
			agentPath := filepath.Join(agentsDir, entry.Name())
			name, desc := parseFrontmatter(agentPath)
			if name == "" {
				name = id
			}
			relPath, _ := filepath.Rel(tmpDir, agentPath)
			result.Items = append(result.Items, DiscoveredItem{
				ID:          id,
				Name:        name,
				Description: desc,
				Type:        "agent",
				FilePath:    relPath,
			})
			seenIDs["agent:"+id] = true
		}
	}

	c.JSON(http.StatusOK, result)
}

// parseFrontmatter reads YAML frontmatter from a markdown file and extracts name and description.
func parseFrontmatter(path string) (name, description string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inFrontmatter {
			if trimmed == "---" {
				inFrontmatter = true
				continue
			}
			return "", "" // no frontmatter
		}

		if trimmed == "---" {
			break // end of frontmatter
		}

		if strings.HasPrefix(trimmed, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, "name:"))
			name = strings.Trim(name, "\"'")
		} else if strings.HasPrefix(trimmed, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(trimmed, "description:"))
			description = strings.Trim(description, "\"'")
		}
	}

	return name, description
}

// ListInstalledItems returns installed marketplace items from the ProjectSettings CR.
// GET /api/projects/:projectName/marketplace/items
func ListInstalledItems(c *gin.Context) {
	project := c.GetString("project")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	gvr := GetProjectSettingsResource()
	obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil {
		log.Printf("ListInstalledItems: failed to get ProjectSettings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read project settings"})
		return
	}

	items, _, _ := unstructured.NestedSlice(obj.Object, "spec", "installedItems")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// InstallItems adds marketplace items to the ProjectSettings CR.
// POST /api/projects/:projectName/marketplace/items
func InstallItems(c *gin.Context) {
	project := c.GetString("project")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	var newItems []InstalledItem
	if err := c.ShouldBindJSON(&newItems); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	gvr := GetProjectSettingsResource()
	obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil {
		log.Printf("InstallItems: failed to get ProjectSettings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read project settings"})
		return
	}

	// Get existing items
	existingItems, _, _ := unstructured.NestedSlice(obj.Object, "spec", "installedItems")

	// Build a set of existing (sourceUrl, itemId) pairs for dedup
	existingKeys := make(map[string]bool)
	for _, item := range existingItems {
		if m, ok := item.(map[string]interface{}); ok {
			src, _ := m["sourceUrl"].(string)
			id, _ := m["itemId"].(string)
			existingKeys[src+"\x00"+id] = true
		}
	}

	// Convert new items to unstructured and append (skip duplicates)
	for _, item := range newItems {
		key := item.SourceURL + "\x00" + item.ItemID
		if existingKeys[key] {
			continue
		}
		entry := map[string]interface{}{
			"sourceUrl": item.SourceURL,
			"itemId":    item.ItemID,
			"itemType":  item.ItemType,
		}
		if item.SourceBranch != "" {
			entry["sourceBranch"] = item.SourceBranch
		}
		if item.SourcePath != "" {
			entry["sourcePath"] = item.SourcePath
		}
		if item.ItemName != "" {
			entry["itemName"] = item.ItemName
		}
		if item.FilePath != "" {
			entry["filePath"] = item.FilePath
		}
		existingItems = append(existingItems, entry)
		existingKeys[key] = true
	}

	if err := unstructured.SetNestedSlice(obj.Object, existingItems, "spec", "installedItems"); err != nil {
		log.Printf("InstallItems: failed to set installedItems: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project settings"})
		return
	}

	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(c.Request.Context(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("InstallItems: failed to update ProjectSettings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project settings"})
		return
	}

	items, _, _ := unstructured.NestedSlice(updated.Object, "spec", "installedItems")
	c.JSON(http.StatusOK, gin.H{"items": items})
}

// UninstallItem removes a marketplace item from the ProjectSettings CR.
// DELETE /api/projects/:projectName/marketplace/items/:itemId
func UninstallItem(c *gin.Context) {
	project := c.GetString("project")
	itemID := c.Param("itemId")
	_, k8sDyn := GetK8sClientsForRequest(c)
	if k8sDyn == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		return
	}

	gvr := GetProjectSettingsResource()
	obj, err := k8sDyn.Resource(gvr).Namespace(project).Get(c.Request.Context(), "projectsettings", v1.GetOptions{})
	if err != nil {
		log.Printf("UninstallItem: failed to get ProjectSettings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read project settings"})
		return
	}

	existingItems, _, _ := unstructured.NestedSlice(obj.Object, "spec", "installedItems")

	sourceURL := c.Query("sourceUrl")

	// Filter out the item with matching itemId (and sourceUrl if provided)
	var filtered []interface{}
	for _, item := range existingItems {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, _ := itemMap["itemId"].(string)
		src, _ := itemMap["sourceUrl"].(string)
		match := id == itemID
		if sourceURL != "" {
			match = match && src == sourceURL
		}
		if !match {
			filtered = append(filtered, item)
		}
	}

	if filtered == nil {
		filtered = []interface{}{}
	}

	if err := unstructured.SetNestedSlice(obj.Object, filtered, "spec", "installedItems"); err != nil {
		log.Printf("UninstallItem: failed to set installedItems: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project settings"})
		return
	}

	updated, err := k8sDyn.Resource(gvr).Namespace(project).Update(c.Request.Context(), obj, v1.UpdateOptions{})
	if err != nil {
		log.Printf("UninstallItem: failed to update ProjectSettings for %s: %v", project, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project settings"})
		return
	}

	items, _, _ := unstructured.NestedSlice(updated.Object, "spec", "installedItems")
	c.JSON(http.StatusOK, gin.H{"items": items})
}
