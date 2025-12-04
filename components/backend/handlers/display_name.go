package handlers

import (
	"context"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/vertex"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// runnerSecretsName is the hardcoded secret name for API keys
	runnerSecretsName = "ambient-runner-secrets"
	// anthropicAPIKeyField is the secret field containing the Anthropic API key
	anthropicAPIKeyField = "ANTHROPIC_API_KEY"
	// haiku model for quick, cheap name generation (standard API)
	haiku3Model = "claude-3-5-haiku-20241022"
	// haiku model for Vertex AI (requires version suffix)
	haiku3ModelVertex = "claude-3-5-haiku-v2@20241022"
	// Maximum display name length
	maxDisplayNameLength = 50
	// Timeout for API call
	displayNameAPITimeout = 10 * time.Second
)

// SessionContext contains relevant session information for display name generation
type SessionContext struct {
	Repos          []map[string]interface{}
	ActiveWorkflow map[string]interface{}
	InitialPrompt  string
}

// displayNameValidationRegex matches characters that could be problematic for logging or K8s
var displayNameValidationRegex = regexp.MustCompile(`[\x00-\x1F\x7F]`)

// sanitizeDisplayName removes control characters and trims whitespace
func sanitizeDisplayName(name string) string {
	// Remove control characters that could cause log injection
	name = displayNameValidationRegex.ReplaceAllString(name, "")
	// Trim whitespace
	name = strings.TrimSpace(name)
	// Remove quotes that might have slipped through
	name = strings.Trim(name, "\"'`")
	return name
}

// ValidateDisplayName validates a display name for the HTTP handler
// Returns an error message if invalid, empty string if valid
func ValidateDisplayName(name string) string {
	if strings.TrimSpace(name) == "" {
		return "display name cannot be empty"
	}
	if utf8.RuneCountInString(name) > maxDisplayNameLength {
		return fmt.Sprintf("display name cannot exceed %d characters", maxDisplayNameLength)
	}
	// Check for control characters
	if displayNameValidationRegex.MatchString(name) {
		return "display name contains invalid characters"
	}
	return ""
}

// GenerateDisplayNameAsync asynchronously generates a display name for a session
// based on the user's first message and session context. Runs in a goroutine
// and fails silently on error.
//
// Goroutine Lifecycle:
// - Bounded by displayNameAPITimeout (10s max) preventing indefinite hangs
// - Gracefully handles session deletion during generation (checks IsNotFound)
// - No cancellation mechanism exists; goroutine runs to completion or timeout
// - Safe for backend restarts: orphaned goroutines will timeout naturally
func GenerateDisplayNameAsync(projectName, sessionName, userMessage string, sessionCtx SessionContext) {
	go func() {
		if err := generateAndUpdateDisplayName(projectName, sessionName, userMessage, sessionCtx); err != nil {
			log.Printf("DisplayNameGen: Failed to generate display name for %s/%s: %v", projectName, sessionName, err)
		}
	}()
}

// generateAndUpdateDisplayName generates a display name using Claude Haiku and updates the CR
func generateAndUpdateDisplayName(projectName, sessionName, userMessage string, sessionCtx SessionContext) error {
	ctx, cancel := context.WithTimeout(context.Background(), displayNameAPITimeout)
	defer cancel()

	// Get Anthropic client (Vertex or API key)
	client, isVertex, err := getAnthropicClient(ctx, projectName)
	if err != nil {
		return fmt.Errorf("failed to get Anthropic client: %w", err)
	}

	// Build prompt with context
	prompt := buildDisplayNamePrompt(userMessage, sessionCtx)

	// Call Claude Haiku with appropriate model name
	modelName := haiku3Model
	if isVertex {
		modelName = haiku3ModelVertex
	}
	displayName, err := callClaudeForDisplayName(ctx, client, prompt, modelName)
	if err != nil {
		return fmt.Errorf("failed to call Claude: %w", err)
	}

	// Sanitize and validate display name
	displayName = sanitizeDisplayName(displayName)

	// Truncate if too long (using runes for proper Unicode handling)
	if utf8.RuneCountInString(displayName) > maxDisplayNameLength {
		runes := []rune(displayName)
		displayName = string(runes[:maxDisplayNameLength-3]) + "..."
	}

	// Update the session CR
	if err := updateSessionDisplayNameInternal(projectName, sessionName, displayName); err != nil {
		return fmt.Errorf("failed to update session display name: %w", err)
	}

	log.Printf("DisplayNameGen: Successfully generated display name for %s/%s: %q", projectName, sessionName, displayName)
	return nil
}

// getAnthropicClient creates an Anthropic client using either Vertex AI or API key
// Returns the client and a boolean indicating if Vertex AI is being used
func getAnthropicClient(ctx context.Context, projectName string) (anthropic.Client, bool, error) {
	// Check if Vertex AI is enabled (cluster-wide setting)
	if os.Getenv("CLAUDE_CODE_USE_VERTEX") == "1" {
		// For Vertex AI, use the vertex package with Google Application Default Credentials
		// Required env vars: GOOGLE_APPLICATION_CREDENTIALS, ANTHROPIC_VERTEX_PROJECT_ID, CLOUD_ML_REGION
		region := os.Getenv("CLOUD_ML_REGION")
		gcpProjectID := os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")

		if region == "" {
			region = "us-central1" // Default region
		}
		if gcpProjectID == "" {
			return anthropic.Client{}, false, fmt.Errorf("ANTHROPIC_VERTEX_PROJECT_ID is required when CLAUDE_CODE_USE_VERTEX=1 (check backend deployment env vars)")
		}

		log.Printf("DisplayNameGen: Using Vertex AI for %s (region: %s, project: %s)", projectName, region, gcpProjectID)
		client := anthropic.NewClient(
			vertex.WithGoogleAuth(ctx, region, gcpProjectID),
		)
		return client, true, nil
	}

	// Vertex not enabled - use API key from project secret
	apiKey, err := getAPIKeyFromSecret(ctx, projectName)
	if err != nil {
		return anthropic.Client{}, false, fmt.Errorf("failed to get API key: %w", err)
	}

	if apiKey == "" {
		return anthropic.Client{}, false, fmt.Errorf("no API key configured for project %s", projectName)
	}

	// Note: Intentionally not logging API key details to minimize information disclosure
	return anthropic.NewClient(option.WithAPIKey(apiKey)), false, nil
}

// getAPIKeyFromSecret retrieves the Anthropic API key from the project's runner secrets
func getAPIKeyFromSecret(ctx context.Context, projectName string) (string, error) {
	if K8sClientProjects == nil {
		return "", fmt.Errorf("K8s client not initialized")
	}

	secret, err := K8sClientProjects.CoreV1().Secrets(projectName).Get(ctx, runnerSecretsName, v1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", projectName, runnerSecretsName, err)
	}

	apiKey, ok := secret.Data[anthropicAPIKeyField]
	if !ok {
		return "", fmt.Errorf("secret %s/%s does not contain %s", projectName, runnerSecretsName, anthropicAPIKeyField)
	}

	return string(apiKey), nil
}

// buildDisplayNamePrompt constructs the prompt for display name generation
func buildDisplayNamePrompt(userMessage string, sessionCtx SessionContext) string {
	var contextParts []string

	// Add repo information
	if len(sessionCtx.Repos) > 0 {
		var repoNames []string
		for _, repo := range sessionCtx.Repos {
			if url, ok := repo["url"].(string); ok {
				// Extract repo name from URL
				parts := strings.Split(url, "/")
				if len(parts) > 0 {
					name := parts[len(parts)-1]
					name = strings.TrimSuffix(name, ".git")
					repoNames = append(repoNames, name)
				}
			}
		}
		if len(repoNames) > 0 {
			contextParts = append(contextParts, fmt.Sprintf("Repositories: %s", strings.Join(repoNames, ", ")))
		}
	}

	// Add workflow information
	if sessionCtx.ActiveWorkflow != nil {
		if gitURL, ok := sessionCtx.ActiveWorkflow["gitUrl"].(string); ok {
			parts := strings.Split(gitURL, "/")
			if len(parts) > 0 {
				workflowName := parts[len(parts)-1]
				workflowName = strings.TrimSuffix(workflowName, ".git")
				contextParts = append(contextParts, fmt.Sprintf("Workflow: %s", workflowName))
			}
		}
	}

	contextStr := ""
	if len(contextParts) > 0 {
		contextStr = "\nContext: " + strings.Join(contextParts, "; ")
	}

	return fmt.Sprintf(`Generate a short, descriptive display name (max %d characters) for this AI coding session.
User's message: "%s"%s

Return ONLY the display name, no quotes, no explanation, no punctuation at the end.
Examples of good names: "Debug auth middleware", "Add user dashboard", "Refactor API routes"`, maxDisplayNameLength, userMessage, contextStr)
}

// callClaudeForDisplayName calls the Claude API to generate a display name
func callClaudeForDisplayName(ctx context.Context, client anthropic.Client, prompt string, modelName string) (string, error) {
	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(modelName),
		MaxTokens: 100,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}

	if len(message.Content) == 0 {
		return "", fmt.Errorf("empty response from Claude")
	}

	// Extract text from response
	for _, block := range message.Content {
		if block.Type == "text" {
			text := strings.TrimSpace(block.Text)
			// Remove surrounding quotes if present
			text = strings.Trim(text, "\"'")
			return text, nil
		}
	}

	return "", fmt.Errorf("no text content in response")
}

// updateSessionDisplayNameInternal updates the session CR's displayName field
// Uses the backend service account since this is an internal operation
func updateSessionDisplayNameInternal(projectName, sessionName, displayName string) error {
	if DynamicClient == nil {
		return fmt.Errorf("dynamic client not initialized")
	}

	gvr := GetAgenticSessionV1Alpha1Resource()
	ctx := context.Background()

	// Get current session - check if it still exists (prevents goroutine leak)
	item, err := DynamicClient.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, v1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Session was deleted, this is not an error for async generation
			log.Printf("DisplayNameGen: Session %s/%s no longer exists, skipping update", projectName, sessionName)
			return nil
		}
		return fmt.Errorf("failed to get session: %w", err)
	}

	// Use unstructured helper for safe type access (per CLAUDE.md guidelines)
	spec, found, err := unstructured.NestedMap(item.Object, "spec")
	if err != nil {
		return fmt.Errorf("failed to get spec from session: %w", err)
	}
	if !found {
		spec = make(map[string]interface{})
	}

	// Check if displayName was already set (race condition mitigation)
	existingName, _, _ := unstructured.NestedString(spec, "displayName")
	if existingName != "" {
		log.Printf("DisplayNameGen: Session %s/%s already has display name %q, skipping", projectName, sessionName, existingName)
		return nil
	}

	spec["displayName"] = displayName

	// Set the updated spec back
	if err := unstructured.SetNestedMap(item.Object, spec, "spec"); err != nil {
		return fmt.Errorf("failed to set spec: %w", err)
	}

	// Persist the change
	_, err = DynamicClient.Resource(gvr).Namespace(projectName).Update(ctx, item, v1.UpdateOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Session was deleted during update
			log.Printf("DisplayNameGen: Session %s/%s deleted during update, skipping", projectName, sessionName)
			return nil
		}
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// ShouldGenerateDisplayName checks if display name generation should be triggered
// Returns true if displayName is empty/unset
func ShouldGenerateDisplayName(spec map[string]interface{}) bool {
	displayName, ok := spec["displayName"].(string)
	if !ok {
		return true
	}
	return strings.TrimSpace(displayName) == ""
}

// ExtractSessionContext extracts context from session spec for display name generation
// Uses unstructured helpers for safe type access per CLAUDE.md guidelines
func ExtractSessionContext(spec map[string]interface{}) SessionContext {
	ctx := SessionContext{}

	// Extract repos using unstructured helper
	repos, found, err := unstructured.NestedSlice(spec, "repos")
	if err == nil && found {
		for _, r := range repos {
			if repo, ok := r.(map[string]interface{}); ok {
				ctx.Repos = append(ctx.Repos, repo)
			}
		}
	}

	// Extract active workflow using unstructured helper
	workflow, found, err := unstructured.NestedMap(spec, "activeWorkflow")
	if err == nil && found {
		ctx.ActiveWorkflow = workflow
	}

	// Extract initial prompt using unstructured helper
	prompt, found, err := unstructured.NestedString(spec, "initialPrompt")
	if err == nil && found {
		ctx.InitialPrompt = prompt
	}

	return ctx
}
