// Package handlers: post-session insight extraction.
//
// When a session's run finishes (RUN_FINISHED or RUN_ERROR event), the
// backend optionally runs a lightweight LLM extraction pass against the
// session transcript and writes candidate insights as markdown files on
// a new branch, opening a draft PR for human review.
//
// Gated behind the "learning-agent-loop" feature flag.
// Configuration is read from .ambient/config.json in the workspace repo.
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
	"regexp"
	"sort"
	"strings"
	"time"

	"ambient-code-backend/types"

	"github.com/anthropics/anthropic-sdk-go"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// LoadEventsForExtraction loads AG-UI events for a session.
// Set by the websocket package at init to avoid circular imports.
var LoadEventsForExtraction func(sessionName string) []map[string]interface{}

// slugRegex is compiled once for use in slugify().
var slugRegex = regexp.MustCompile(`[^a-z0-9]+`)

// ─── Extraction status constants ────────────────────────────────────

const (
	ExtractionStatusPending        = "pending"
	ExtractionStatusRunning        = "running"
	ExtractionStatusCompleted      = "completed"
	ExtractionStatusSkipped        = "skipped"
	ExtractionStatusFailed         = "failed"
	ExtractionStatusPartialFailure = "partial-failure"
)

// ─── Default configuration ──────────────────────────────────────────

const (
	defaultExtractionModel       = "claude-haiku-4-20250414"
	defaultExtractionModelVertex = "claude-haiku-4@20250414"
	defaultMaxMemoriesPerSession = 5
	defaultMinTurnThreshold      = 5
	extractionAPITimeout         = 30 * time.Second
	maxTranscriptChars           = 50000
)

// ─── Configuration types ────────────────────────────────────────────

// ExtractionConfig holds the extraction settings from .ambient/config.json.
type ExtractionConfig struct {
	Enabled               bool   `json:"enabled"`
	Model                 string `json:"model"`
	MaxMemoriesPerSession int    `json:"maxMemoriesPerSession"`
	MinTurnThreshold      int    `json:"minTurnThreshold"`
}

// LearningConfig holds the top-level learning settings.
type LearningConfig struct {
	Enabled    bool              `json:"enabled"`
	Extraction *ExtractionConfig `json:"extraction"`
}

// AmbientConfig represents the .ambient/config.json file.
type AmbientConfig struct {
	Learning *LearningConfig `json:"learning"`
}

// InsightCandidate represents a single extracted insight from the LLM.
type InsightCandidate struct {
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Type       string  `json:"type"`       // "correction" or "pattern"
	Confidence float64 `json:"confidence"` // 0.0 - 1.0
}

// ─── Config parsing ─────────────────────────────────────────────────

// parseExtractionConfig parses extraction config from raw JSON bytes.
// Returns nil if extraction is not enabled or not configured.
func parseExtractionConfig(data []byte) *ExtractionConfig {
	var cfg AmbientConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("Extraction: failed to parse .ambient/config.json: %v", err)
		return nil
	}
	if cfg.Learning == nil || !cfg.Learning.Enabled {
		return nil
	}
	if cfg.Learning.Extraction == nil || !cfg.Learning.Extraction.Enabled {
		return nil
	}
	ext := cfg.Learning.Extraction

	// Apply defaults
	if ext.Model == "" {
		ext.Model = defaultExtractionModel
	}
	if ext.MaxMemoriesPerSession <= 0 {
		ext.MaxMemoriesPerSession = defaultMaxMemoriesPerSession
	}
	if ext.MinTurnThreshold <= 0 {
		ext.MinTurnThreshold = defaultMinTurnThreshold
	}
	return ext
}

// ─── Transcript helpers ─────────────────────────────────────────────

// countUserTurns counts the number of user messages in the event log.
// Prefers MESSAGES_SNAPSHOT (compacted sessions), falls back to
// counting TEXT_MESSAGE_START events with role=user.
func countUserTurns(events []map[string]interface{}) int {
	// Try MESSAGES_SNAPSHOT first (last one wins — most recent state)
	for i := len(events) - 1; i >= 0; i-- {
		evt := events[i]
		eventType, _ := evt["type"].(string)
		if eventType == types.EventTypeMessagesSnapshot {
			messages, ok := evt["messages"].([]interface{})
			if !ok {
				continue
			}
			count := 0
			for _, msg := range messages {
				m, ok := msg.(map[string]interface{})
				if !ok {
					continue
				}
				if role, _ := m["role"].(string); role == types.RoleUser {
					count++
				}
			}
			return count
		}
	}

	// Fallback: count streaming TEXT_MESSAGE_START with role=user
	count := 0
	for _, evt := range events {
		eventType, _ := evt["type"].(string)
		if eventType == types.EventTypeTextMessageStart {
			if role, _ := evt["role"].(string); role == types.RoleUser {
				count++
			}
		}
	}
	return count
}

// buildTranscriptText extracts a compact text transcript from AG-UI events.
// Prefers MESSAGES_SNAPSHOT for compacted sessions. Truncates to maxTranscriptChars.
func buildTranscriptText(events []map[string]interface{}) string {
	var sb strings.Builder

	for i := len(events) - 1; i >= 0; i-- {
		evt := events[i]
		eventType, _ := evt["type"].(string)
		if eventType == types.EventTypeMessagesSnapshot {
			messages, ok := evt["messages"].([]interface{})
			if !ok {
				continue
			}
			for _, msg := range messages {
				m, ok := msg.(map[string]interface{})
				if !ok {
					continue
				}
				role, _ := m["role"].(string)
				content, _ := m["content"].(string)
				if role == "" || content == "" {
					continue
				}
				// Skip system/developer messages (not useful for extraction)
				if role == types.RoleSystem || role == types.RoleDeveloper {
					continue
				}
				fmt.Fprintf(&sb, "[%s]: %s\n\n", role, content)
			}
			break
		}
	}

	text := sb.String()
	if len(text) > maxTranscriptChars {
		text = text[:maxTranscriptChars] + "\n\n[transcript truncated]"
	}
	return text
}

// ─── LLM extraction ─────────────────────────────────────────────────

const extractionPrompt = `You are an expert at identifying reusable engineering knowledge from coding session transcripts.

Analyze the following transcript from an AI-assisted coding session. Extract reusable knowledge that would help future sessions avoid mistakes or follow better patterns.

Focus on:
- CORRECTIONS: Mistakes that were made and corrected. Things to avoid in the future.
- PATTERNS: Conventions, idioms, or approaches that worked well and should be repeated.

Ignore:
- Session-specific details (file names, variable names, specific bugs)
- Obvious or trivial knowledge
- Anything that wouldn't generalize to other sessions

Return a JSON array of candidates. Each candidate must have:
- "title": A short descriptive title (max 80 chars)
- "content": The reusable knowledge as markdown (2-5 sentences)
- "type": Either "correction" or "pattern"
- "confidence": A float from 0.0 to 1.0 indicating how reusable this knowledge is

Return ONLY the JSON array, no markdown fences, no explanation. If nothing is worth extracting, return an empty array [].`

// callExtractionModel sends the transcript to the extraction LLM and returns parsed candidates.
func callExtractionModel(ctx context.Context, client anthropic.Client, transcript, modelName string) ([]InsightCandidate, error) {
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(modelName),
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{Text: extractionPrompt},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(transcript)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("LLM API call failed: %w", err)
	}
	if len(message.Content) == 0 {
		return nil, fmt.Errorf("empty response from extraction model")
	}
	var responseText string
	for _, block := range message.Content {
		if block.Type == "text" {
			responseText = strings.TrimSpace(block.Text)
			break
		}
	}
	if responseText == "" {
		return nil, fmt.Errorf("no text content in extraction response")
	}
	return parseExtractionResponse(responseText)
}

// parseExtractionResponse parses the LLM JSON response into InsightCandidate structs.
func parseExtractionResponse(responseText string) ([]InsightCandidate, error) {
	responseText = strings.TrimSpace(responseText)
	// Strip markdown code fences if present
	if strings.HasPrefix(responseText, "```") {
		lines := strings.Split(responseText, "\n")
		if len(lines) >= 3 {
			responseText = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}

	var candidates []InsightCandidate
	if err := json.Unmarshal([]byte(responseText), &candidates); err != nil {
		return nil, fmt.Errorf("failed to parse extraction JSON: %w (response: %.200s)", err, responseText)
	}

	// Validate and filter candidates
	var valid []InsightCandidate
	for _, c := range candidates {
		if c.Title == "" || c.Content == "" || c.Type == "" {
			continue
		}
		if c.Type != "correction" && c.Type != "pattern" {
			continue
		}
		if c.Confidence < 0 {
			c.Confidence = 0
		}
		if c.Confidence > 1 {
			c.Confidence = 1
		}
		valid = append(valid, c)
	}
	return valid, nil
}

// rankAndCap sorts candidates by confidence (descending) and truncates to maxCount.
func rankAndCap(candidates []InsightCandidate, maxCount int) []InsightCandidate {
	if len(candidates) == 0 {
		return candidates
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})
	if len(candidates) > maxCount {
		candidates = candidates[:maxCount]
	}
	return candidates
}

// ─── Markdown + file path generation ────────────────────────────────

// slugify converts a title into a URL-safe slug.
func slugify(title string) string {
	s := strings.ToLower(title)
	s = slugRegex.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}
	if s == "" {
		s = "insight"
	}
	return s
}

// formatInsightMarkdown formats an insight candidate as a markdown file.
func formatInsightMarkdown(c InsightCandidate, sessionName, projectName string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", c.Title)
	fmt.Fprintf(&sb, "**Type:** %s  \n", c.Type)
	fmt.Fprintf(&sb, "**Confidence:** %.2f  \n", c.Confidence)
	sb.WriteString("**Source:** insight-extraction  \n")
	fmt.Fprintf(&sb, "**Session:** %s/%s  \n", projectName, sessionName)
	fmt.Fprintf(&sb, "**Extracted:** %s  \n\n", time.Now().UTC().Format(time.RFC3339))
	sb.WriteString("---\n\n")
	sb.WriteString(c.Content)
	sb.WriteString("\n")
	return sb.String()
}

// insightFilePath returns the path for an insight file in the docs/learned/ directory.
func insightFilePath(c InsightCandidate) string {
	date := time.Now().UTC().Format("2006-01-02")
	slug := slugify(c.Title)
	typeDir := c.Type + "s" // "corrections" or "patterns"
	return fmt.Sprintf("docs/learned/%s/%s-%s.md", typeDir, date, slug)
}

// ─── GitHub API helpers ─────────────────────────────────────────────

type gitHubFileContent struct {
	Path    string
	Content string
}

// parseGitHubOwnerRepo extracts owner and repo from a GitHub URL.
func parseGitHubOwnerRepo(repoURL string) (string, string, error) {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	if strings.Contains(repoURL, "github.com") {
		parts := strings.Split(repoURL, "github.com")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub URL: %s", repoURL)
		}
		path := strings.Trim(parts[1], "/:")
		pathParts := strings.Split(path, "/")
		if len(pathParts) < 2 {
			return "", "", fmt.Errorf("invalid GitHub URL path: %s", repoURL)
		}
		return pathParts[0], pathParts[1], nil
	}
	return "", "", fmt.Errorf("not a GitHub URL: %s", repoURL)
}

// githubAPIRequest is a helper for making GitHub API requests.
func githubAPIRequest(ctx context.Context, method, url, token string, body interface{}) ([]byte, int, error) {
	var reqBody io.Reader
	if body != nil {
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

func getDefaultBranchSHA(ctx context.Context, apiBase, owner, repo, token string) (string, string, error) {
	respBody, status, err := githubAPIRequest(ctx, "GET",
		fmt.Sprintf("%s/repos/%s/%s", apiBase, owner, repo), token, nil)
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API error %d: %s", status, string(respBody))
	}
	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(respBody, &repoInfo); err != nil {
		return "", "", err
	}

	refBody, refStatus, err := githubAPIRequest(ctx, "GET",
		fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", apiBase, owner, repo, repoInfo.DefaultBranch), token, nil)
	if err != nil {
		return "", "", err
	}
	if refStatus != http.StatusOK {
		return "", "", fmt.Errorf("GitHub ref API error %d: %s", refStatus, string(refBody))
	}
	var refInfo struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.Unmarshal(refBody, &refInfo); err != nil {
		return "", "", err
	}
	return repoInfo.DefaultBranch, refInfo.Object.SHA, nil
}

func createGitRef(ctx context.Context, apiBase, owner, repo, token, branchName, sha string) error {
	body := map[string]string{
		"ref": fmt.Sprintf("refs/heads/%s", branchName),
		"sha": sha,
	}
	respBody, status, err := githubAPIRequest(ctx, "POST",
		fmt.Sprintf("%s/repos/%s/%s/git/refs", apiBase, owner, repo), token, body)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create ref failed %d: %s", status, string(respBody))
	}
	return nil
}

func createFileOnBranch(ctx context.Context, apiBase, owner, repo, token, branch, path, content, commitMsg string) error {
	body := map[string]string{
		"message": commitMsg,
		"content": base64.StdEncoding.EncodeToString([]byte(content)),
		"branch":  branch,
	}
	respBody, status, err := githubAPIRequest(ctx, "PUT",
		fmt.Sprintf("%s/repos/%s/%s/contents/%s", apiBase, owner, repo, path), token, body)
	if err != nil {
		return err
	}
	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("create file failed %d: %s", status, string(respBody))
	}
	return nil
}

func createDraftPR(ctx context.Context, apiBase, owner, repo, token, head, baseBranch, title, prBodyText string) (int, error) {
	body := map[string]interface{}{
		"title": title,
		"head":  head,
		"base":  baseBranch,
		"body":  prBodyText,
		"draft": true,
	}
	respBody, status, err := githubAPIRequest(ctx, "POST",
		fmt.Sprintf("%s/repos/%s/%s/pulls", apiBase, owner, repo), token, body)
	if err != nil {
		return 0, err
	}
	if status != http.StatusCreated {
		return 0, fmt.Errorf("create PR failed %d: %s", status, string(respBody))
	}
	var prResult struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(respBody, &prResult); err != nil {
		return 0, fmt.Errorf("failed to parse PR response: %w", err)
	}
	return prResult.Number, nil
}

// addLabelToPR adds the continuous-learning label to a PR. Best-effort.
func addLabelToPR(ctx context.Context, apiBase, owner, repo, token string, prNumber int) {
	body := map[string][]string{
		"labels": {"continuous-learning"},
	}
	_, status, err := githubAPIRequest(ctx, "POST",
		fmt.Sprintf("%s/repos/%s/%s/issues/%d/labels", apiBase, owner, repo, prNumber), token, body)
	if err != nil {
		log.Printf("Extraction: failed to add label: %v", err)
		return
	}
	if status != http.StatusOK {
		log.Printf("Extraction: add label returned status %d", status)
	}
}

// createExtractionPR creates a branch with insight files and opens a draft PR.
func createExtractionPR(ctx context.Context, repoURL, token, sessionName, projectName string, files []gitHubFileContent) error {
	owner, repo, err := parseGitHubOwnerRepo(repoURL)
	if err != nil {
		return fmt.Errorf("failed to parse repo URL: %w", err)
	}

	apiBase := "https://api.github.com"
	defaultBranch, baseSHA, err := getDefaultBranchSHA(ctx, apiBase, owner, repo, token)
	if err != nil {
		return fmt.Errorf("failed to get default branch: %w", err)
	}

	branchName := fmt.Sprintf("learned/%s/%s", sessionName, time.Now().UTC().Format("20060102-150405"))

	if err := createGitRef(ctx, apiBase, owner, repo, token, branchName, baseSHA); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	for _, f := range files {
		commitMsg := fmt.Sprintf("learned: add insight from %s", sessionName)
		if err := createFileOnBranch(ctx, apiBase, owner, repo, token, branchName, f.Path, f.Content, commitMsg); err != nil {
			return fmt.Errorf("failed to create file %s: %w", f.Path, err)
		}
	}

	prTitle := fmt.Sprintf("learned: insights from %s", sessionName)
	prBody := fmt.Sprintf("## Extracted Insights\n\n"+
		"**source:** insight-extraction  \n"+
		"**session:** %s/%s  \n"+
		"**files:** %d insight(s)  \n\n"+
		"These insights were automatically extracted from a completed agentic session. "+
		"Review the changes, edit as needed, and merge to include in future sessions.\n\n"+
		"---\n_Generated by the Ambient Code Platform continuous learning pipeline._",
		projectName, sessionName, len(files))

	prNumber, err := createDraftPR(ctx, apiBase, owner, repo, token, branchName, defaultBranch, prTitle, prBody)
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	addLabelToPR(ctx, apiBase, owner, repo, token, prNumber)
	log.Printf("Extraction: created draft PR #%d for %s/%s with %d files", prNumber, projectName, sessionName, len(files))
	return nil
}

// ─── Status update helper ───────────────────────────────────────────

func updateExtractionStatus(projectName, sessionName, status string) error {
	if DynamicClient == nil {
		return fmt.Errorf("dynamic client not initialized")
	}
	gvr := GetAgenticSessionV1Alpha1Resource()
	ctx := context.Background()

	item, err := DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	statusMap, _, _ := unstructured.NestedMap(item.Object, "status")
	if statusMap == nil {
		statusMap = make(map[string]interface{})
	}
	statusMap["extractionStatus"] = status
	if err := unstructured.SetNestedMap(item.Object, statusMap, "status"); err != nil {
		return fmt.Errorf("failed to set status: %w", err)
	}

	_, err = DynamicClient.Resource(gvr).Namespace(projectName).UpdateStatus(ctx, item, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update session status: %w", err)
	}
	return nil
}

// ─── Main extraction orchestrator ───────────────────────────────────

// TriggerExtractionAsync is the entry point called when a session run
// finishes (RUN_FINISHED or RUN_ERROR). It runs the entire extraction
// pipeline in a background goroutine and does not block the caller.
func TriggerExtractionAsync(projectName, sessionName string) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Extraction: recovered from panic for %s/%s: %v", projectName, sessionName, r)
			}
		}()
		if err := runExtraction(projectName, sessionName); err != nil {
			log.Printf("Extraction: failed for %s/%s: %v", projectName, sessionName, err)
		}
	}()
}

func runExtraction(projectName, sessionName string) error {
	// 1. Check feature flag
	if !FeatureEnabled(correctionsFeatureFlag) {
		log.Printf("Extraction: learning-agent-loop flag disabled, skipping %s/%s", projectName, sessionName)
		return nil
	}

	// 2. Set extraction status to pending (at-most-once guard)
	if err := claimExtraction(projectName, sessionName); err != nil {
		log.Printf("Extraction: at-most-once guard for %s/%s: %v", projectName, sessionName, err)
		return nil // Not an error — already claimed
	}

	// 3. Update status to running
	if err := updateExtractionStatus(projectName, sessionName, ExtractionStatusRunning); err != nil {
		log.Printf("Extraction: failed to set running status for %s/%s: %v", projectName, sessionName, err)
	}

	// 4. Get repo URL from session spec
	repoURL, err := getSessionRepoURL(projectName, sessionName)
	if err != nil || repoURL == "" {
		log.Printf("Extraction: no repo URL for %s/%s, skipping", projectName, sessionName)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}

	// 5. Get GitHub token
	ctx, cancel := context.WithTimeout(context.Background(), extractionAPITimeout)
	defer cancel()

	token, err := getExtractionGitHubToken(ctx, projectName)
	if err != nil {
		log.Printf("Extraction: no GitHub token for %s/%s: %v", projectName, sessionName, err)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}

	// 6. Fetch .ambient/config.json from the repo
	configData, err := fetchAmbientConfig(ctx, repoURL, token)
	if err != nil {
		log.Printf("Extraction: config fetch failed for %s/%s: %v", projectName, sessionName, err)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}

	cfg := parseExtractionConfig(configData)
	if cfg == nil {
		log.Printf("Extraction: not enabled for %s/%s", projectName, sessionName)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}

	// 7. Load transcript events
	if LoadEventsForExtraction == nil {
		log.Printf("Extraction: LoadEventsForExtraction not initialized for %s/%s", projectName, sessionName)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}
	events := LoadEventsForExtraction(sessionName)
	if len(events) == 0 {
		log.Printf("Extraction: empty transcript for %s/%s", projectName, sessionName)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}

	// 8. Check minimum turn threshold
	turns := countUserTurns(events)
	if turns < cfg.MinTurnThreshold {
		log.Printf("Extraction: session %s/%s below minimum turn threshold (%d < %d)",
			projectName, sessionName, turns, cfg.MinTurnThreshold)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}

	// 9. Build transcript text
	transcript := buildTranscriptText(events)
	if strings.TrimSpace(transcript) == "" {
		log.Printf("Extraction: empty transcript text for %s/%s", projectName, sessionName)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusSkipped)
		return nil
	}

	// 10. Get Anthropic client and call extraction model
	llmCtx, llmCancel := context.WithTimeout(context.Background(), extractionAPITimeout)
	defer llmCancel()

	client, isVertex, err := getAnthropicClient(llmCtx, projectName)
	if err != nil {
		log.Printf("Extraction: Anthropic client error for %s/%s: %v", projectName, sessionName, err)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusFailed)
		return fmt.Errorf("failed to get Anthropic client: %w", err)
	}

	modelName := cfg.Model
	if isVertex && !strings.Contains(modelName, "@") {
		modelName = defaultExtractionModelVertex
	}

	candidates, err := callExtractionModel(llmCtx, client, transcript, modelName)
	if err != nil {
		log.Printf("Extraction: LLM call failed for %s/%s: %v", projectName, sessionName, err)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusFailed)
		return fmt.Errorf("LLM extraction failed: %w", err)
	}

	if len(candidates) == 0 {
		log.Printf("Extraction: no candidates for %s/%s", projectName, sessionName)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusCompleted)
		return nil
	}

	// 11. Rank and cap
	candidates = rankAndCap(candidates, cfg.MaxMemoriesPerSession)

	// 12. Build file list
	var files []gitHubFileContent
	for _, c := range candidates {
		files = append(files, gitHubFileContent{
			Path:    insightFilePath(c),
			Content: formatInsightMarkdown(c, sessionName, projectName),
		})
	}

	// 13. Create PR
	prCtx, prCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer prCancel()

	if err := createExtractionPR(prCtx, repoURL, token, sessionName, projectName, files); err != nil {
		log.Printf("Extraction: PR creation failed for %s/%s: %v", projectName, sessionName, err)
		_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusPartialFailure)
		return fmt.Errorf("PR creation failed: %w", err)
	}

	_ = updateExtractionStatus(projectName, sessionName, ExtractionStatusCompleted)
	log.Printf("Extraction: completed for %s/%s (%d insights)", projectName, sessionName, len(candidates))
	return nil
}

// claimExtraction atomically sets extractionStatus to "pending" only if it
// is currently unset. Returns an error if extraction was already claimed.
// The K8s optimistic concurrency (resourceVersion) provides the at-most-once
// guarantee: if two goroutines race, only one update will succeed.
func claimExtraction(projectName, sessionName string) error {
	if DynamicClient == nil {
		return fmt.Errorf("dynamic client not initialized")
	}
	gvr := GetAgenticSessionV1Alpha1Resource()
	ctx := context.Background()

	item, err := DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Check if already claimed
	currentStatus, _, _ := unstructured.NestedString(item.Object, "status", "extractionStatus")
	if currentStatus != "" {
		return fmt.Errorf("extraction already claimed (status=%s)", currentStatus)
	}

	// Claim it
	statusMap, _, _ := unstructured.NestedMap(item.Object, "status")
	if statusMap == nil {
		statusMap = make(map[string]interface{})
	}
	statusMap["extractionStatus"] = ExtractionStatusPending
	if err := unstructured.SetNestedMap(item.Object, statusMap, "status"); err != nil {
		return fmt.Errorf("failed to set status: %w", err)
	}

	_, err = DynamicClient.Resource(gvr).Namespace(projectName).UpdateStatus(ctx, item, v1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update status (conflict expected on race): %w", err)
	}
	return nil
}

// getSessionRepoURL returns the URL of the first repo from the session spec.
func getSessionRepoURL(projectName, sessionName string) (string, error) {
	if DynamicClient == nil {
		return "", fmt.Errorf("dynamic client not initialized")
	}
	gvr := GetAgenticSessionV1Alpha1Resource()
	ctx := context.Background()

	item, err := DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}

	repos, found, err := unstructured.NestedSlice(item.Object, "spec", "repos")
	if err != nil || !found || len(repos) == 0 {
		return "", nil
	}

	firstRepo, ok := repos[0].(map[string]interface{})
	if !ok {
		return "", nil
	}

	repoURL, _ := firstRepo["url"].(string)
	return repoURL, nil
}

// getExtractionGitHubToken gets a GitHub token for the extraction pipeline.
func getExtractionGitHubToken(ctx context.Context, projectName string) (string, error) {
	if GetGitHubToken == nil {
		return "", fmt.Errorf("GetGitHubToken not initialized")
	}
	// Use the backend service account for extraction (internal operation).
	// Pass empty userID — the token function will fall back to project-level credentials.
	token, err := GetGitHubToken(ctx, nil, DynamicClient, projectName, "")
	if err != nil {
		return "", err
	}
	return token, nil
}

// fetchAmbientConfig fetches .ambient/config.json from the default branch of the repo.
func fetchAmbientConfig(ctx context.Context, repoURL, token string) ([]byte, error) {
	owner, repo, err := parseGitHubOwnerRepo(repoURL)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/.ambient/config.json", owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github.v3.raw")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf(".ambient/config.json not found in repo")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}
