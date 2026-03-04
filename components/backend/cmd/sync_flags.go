// Package cmd implements CLI subcommands for the backend binary.
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"ambient-code-backend/types"
)

const (
	defaultManifestPath = "/config/models/models.json"
	defaultFlagsConfig  = "/config/flags/flags.json"
	maxRetries          = 3
	retryDelay          = 10 * time.Second
)

var errConflict = errors.New("flag already exists (conflict)")

// FlagTag represents a tag to attach to an Unleash feature flag.
type FlagTag struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// FlagSpec describes a feature flag to sync to Unleash.
// All flags are created disabled with type "release" and a flexibleRollout
// strategy at 0%. Tags are optional and per-flag.
type FlagSpec struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tags        []FlagTag `json:"tags,omitempty"`
}

// FlagsConfig is the JSON structure for the generic flags config file.
type FlagsConfig struct {
	Flags []FlagSpec `json:"flags"`
}

// FlagsFromManifest converts a model manifest into FlagSpecs.
// Skips default models (global and per-provider) and unavailable models.
func FlagsFromManifest(manifest *types.ModelManifest) []FlagSpec {
	// Build set of all default model IDs (global + per-provider)
	defaults := map[string]bool{manifest.DefaultModel: true}
	for _, id := range manifest.ProviderDefaults {
		defaults[id] = true
	}

	var specs []FlagSpec
	for _, model := range manifest.Models {
		if defaults[model.ID] {
			continue
		}
		if !model.Available {
			continue
		}
		specs = append(specs, FlagSpec{
			Name:        sanitizeLogString(fmt.Sprintf("model.%s.enabled", model.ID)),
			Description: sanitizeLogString(fmt.Sprintf("Enable %s (%s) for users", model.Label, model.ID)),
			Tags:        []FlagTag{{Type: "scope", Value: "workspace"}},
		})
	}
	return specs
}

// FlagsConfigPath returns the filesystem path to the generic flags config.
// Defaults to defaultFlagsConfig; override via FLAGS_CONFIG_PATH env var.
func FlagsConfigPath() string {
	if p := os.Getenv("FLAGS_CONFIG_PATH"); p != "" {
		return p
	}
	return defaultFlagsConfig
}

// FlagsFromConfig loads generic flag definitions from a JSON file.
// Returns nil if the file does not exist (flags config is optional).
func FlagsFromConfig(path string) ([]FlagSpec, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading flags config %s: %w", path, err)
	}

	var cfg FlagsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing flags config: %w", err)
	}

	// Sanitize flag names and descriptions to prevent log injection.
	// Model-derived names are constrained (model.<id>.enabled) but
	// config-file names are user-defined and unconstrained.
	for i := range cfg.Flags {
		cfg.Flags[i].Name = sanitizeLogString(cfg.Flags[i].Name)
		cfg.Flags[i].Description = sanitizeLogString(cfg.Flags[i].Description)
		for j := range cfg.Flags[i].Tags {
			cfg.Flags[i].Tags[j].Type = sanitizeLogString(cfg.Flags[i].Tags[j].Type)
			cfg.Flags[i].Tags[j].Value = sanitizeLogString(cfg.Flags[i].Tags[j].Value)
		}
	}

	return cfg.Flags, nil
}

// SyncModelFlagsFromFile reads a model manifest from disk and syncs flags.
// Used by the sync-model-flags subcommand.
func SyncModelFlagsFromFile(manifestPath string) error {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest %s: %w", manifestPath, err)
	}

	var manifest types.ModelManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("parsing manifest: %w", err)
	}

	return SyncFlags(context.Background(), FlagsFromManifest(&manifest))
}

// SyncFlagsAsync runs SyncFlags in a background goroutine with retries.
// Intended for use at server startup — does not block the caller.
// Cancel the context to abort retries (e.g. on SIGTERM).
func SyncFlagsAsync(ctx context.Context, flags []FlagSpec) {
	go func() {
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := SyncFlags(ctx, flags)
			if err == nil {
				return
			}
			log.Printf("sync-flags: attempt %d/%d failed: %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					log.Printf("sync-flags: cancelled, stopping retries")
					return
				case <-time.After(retryDelay):
				}
			}
		}
		log.Printf("sync-flags: all %d attempts failed, giving up", maxRetries)
	}()
}

// SyncFlags ensures every FlagSpec has a corresponding Unleash feature flag.
// Flags are created disabled with type "release" and a flexibleRollout strategy.
//
// Required env vars: UNLEASH_ADMIN_URL, UNLEASH_ADMIN_TOKEN
// Optional env var:  UNLEASH_PROJECT (default: "default")
func SyncFlags(ctx context.Context, flags []FlagSpec) error {
	adminURL := strings.TrimSuffix(strings.TrimSpace(os.Getenv("UNLEASH_ADMIN_URL")), "/")
	adminToken := strings.TrimSpace(os.Getenv("UNLEASH_ADMIN_TOKEN"))
	project := strings.TrimSpace(os.Getenv("UNLEASH_PROJECT"))
	if project == "" {
		project = "default"
	}

	environment := strings.TrimSpace(os.Getenv("UNLEASH_ENVIRONMENT"))
	if environment == "" {
		environment = "development"
	}

	if adminURL == "" || adminToken == "" {
		log.Printf("sync-flags: UNLEASH_ADMIN_URL or UNLEASH_ADMIN_TOKEN not set, skipping")
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Ensure all required tag types exist
	tagTypes := collectTagTypes(flags)
	for _, tt := range tagTypes {
		if err := ensureTagType(ctx, client, adminURL, tt, fmt.Sprintf("Tag type: %s", tt), adminToken); err != nil {
			return fmt.Errorf("ensuring tag type %q: %w", tt, err)
		}
	}

	var created, skipped, errCount int
	log.Printf("Syncing %d Unleash flag(s)...", len(flags))

	for _, flag := range flags {
		exists, err := flagExists(ctx, client, adminURL, project, flag.Name, adminToken)
		if err != nil {
			log.Printf("  ERROR checking %s: %v", flag.Name, err)
			errCount++
			continue
		}

		if exists {
			log.Printf("  %s: already exists, skipping", flag.Name)
			skipped++
			continue
		}

		if err := createFlag(ctx, client, adminURL, project, flag.Name, flag.Description, adminToken); err != nil {
			if errors.Is(err, errConflict) {
				log.Printf("  %s: created by another instance, skipping", flag.Name)
				skipped++
				continue
			}
			log.Printf("  ERROR creating %s: %v", flag.Name, err)
			errCount++
			continue
		}

		for _, tag := range flag.Tags {
			if err := addFlagTag(ctx, client, adminURL, flag.Name, tag, adminToken); err != nil {
				log.Printf("  WARNING: created %s but failed to add tag %s:%s: %v", flag.Name, tag.Type, tag.Value, err)
			}
		}

		if err := addRolloutStrategy(ctx, client, adminURL, project, environment, flag.Name, adminToken); err != nil {
			log.Printf("  WARNING: created %s but failed to add rollout strategy: %v", flag.Name, err)
		}

		log.Printf("  %s: created (disabled, 0%% rollout)", flag.Name)
		created++
	}

	log.Printf("Summary: %d created, %d skipped, %d errors", created, skipped, errCount)

	if errCount > 0 {
		return fmt.Errorf("%d errors occurred during sync", errCount)
	}
	return nil
}

// sanitizeLogString strips newlines and carriage returns from strings
// that will be interpolated into log messages, preventing log injection.
func sanitizeLogString(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", ""), "\r", "")
}

// collectTagTypes returns the unique set of tag types across all flags.
func collectTagTypes(flags []FlagSpec) []string {
	seen := map[string]bool{}
	var result []string
	for _, f := range flags {
		for _, t := range f.Tags {
			if !seen[t.Type] {
				seen[t.Type] = true
				result = append(result, t.Type)
			}
		}
	}
	return result
}

// ParseManifestPath extracts --manifest-path from args, returning the path
// and whether it was found. Falls back to defaultManifestPath.
func ParseManifestPath(args []string) string {
	for i, arg := range args {
		if arg == "--manifest-path" && i+1 < len(args) {
			return args[i+1]
		}
		if v, ok := strings.CutPrefix(arg, "--manifest-path="); ok {
			return v
		}
	}
	return defaultManifestPath
}

func ensureTagType(ctx context.Context, client *http.Client, adminURL, name, description, token string) error {
	reqURL := fmt.Sprintf("%s/api/admin/tag-types/%s", adminURL, url.PathEscape(name))
	resp, err := doRequest(ctx, client, "GET", reqURL, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusOK {
		log.Printf("Tag type %q already exists", name)
		return nil
	}

	createURL := fmt.Sprintf("%s/api/admin/tag-types", adminURL)
	body, err := json.Marshal(map[string]string{
		"name":        name,
		"description": description,
	})
	if err != nil {
		return fmt.Errorf("marshaling tag type request: %w", err)
	}
	resp2, err := doRequest(ctx, client, "POST", createURL, token, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp2.Body.Close()
	respBody, _ := io.ReadAll(resp2.Body)

	switch resp2.StatusCode {
	case http.StatusOK, http.StatusCreated:
		log.Printf("Tag type %q created", name)
		return nil
	case http.StatusConflict:
		log.Printf("Tag type %q created by another instance", name)
		return nil
	default:
		return fmt.Errorf("creating tag type %q: HTTP %d: %s", name, resp2.StatusCode, string(respBody))
	}
}

func flagExists(ctx context.Context, client *http.Client, adminURL, project, flagName, token string) (bool, error) {
	reqURL := fmt.Sprintf("%s/api/admin/projects/%s/features/%s", adminURL, url.PathEscape(project), url.PathEscape(flagName))
	resp, err := doRequest(ctx, client, "GET", reqURL, token, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}
	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
}

func createFlag(ctx context.Context, client *http.Client, adminURL, project, flagName, description, token string) error {
	reqURL := fmt.Sprintf("%s/api/admin/projects/%s/features", adminURL, url.PathEscape(project))
	body, err := json.Marshal(map[string]any{
		"name":           flagName,
		"description":    description,
		"type":           "release",
		"enabled":        false,
		"impressionData": true,
	})
	if err != nil {
		return fmt.Errorf("marshaling flag request: %w", err)
	}

	resp, err := doRequest(ctx, client, "POST", reqURL, token, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusConflict:
		return errConflict
	default:
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
}

func addFlagTag(ctx context.Context, client *http.Client, adminURL, flagName string, tag FlagTag, token string) error {
	reqURL := fmt.Sprintf("%s/api/admin/features/%s/tags", adminURL, url.PathEscape(flagName))
	body, err := json.Marshal(map[string]string{
		"type":  tag.Type,
		"value": tag.Value,
	})
	if err != nil {
		return fmt.Errorf("marshaling tag request: %w", err)
	}

	resp, err := doRequest(ctx, client, "POST", reqURL, token, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func addRolloutStrategy(ctx context.Context, client *http.Client, adminURL, project, environment, flagName, token string) error {
	reqURL := fmt.Sprintf("%s/api/admin/projects/%s/features/%s/environments/%s/strategies",
		adminURL, url.PathEscape(project), url.PathEscape(flagName), url.PathEscape(environment))
	body, err := json.Marshal(map[string]any{
		"name": "flexibleRollout",
		"parameters": map[string]string{
			"rollout":    "0",
			"stickiness": "default",
			"groupId":    flagName,
		},
	})
	if err != nil {
		return fmt.Errorf("marshaling strategy request: %w", err)
	}

	resp, err := doRequest(ctx, client, "POST", reqURL, token, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func doRequest(ctx context.Context, client *http.Client, method, reqURL, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}
