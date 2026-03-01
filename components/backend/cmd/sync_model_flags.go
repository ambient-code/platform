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
	defaultManifestPath = "/config/models.json"
	maxRetries          = 3
	retryDelay          = 10 * time.Second
)

var errConflict = errors.New("flag already exists (conflict)")

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

	return SyncModelFlags(context.Background(), &manifest)
}

// SyncModelFlagsAsync runs SyncModelFlags in a background goroutine with
// retries. Intended for use at server startup — does not block the caller.
// Cancel the context to abort retries (e.g. on SIGTERM).
func SyncModelFlagsAsync(ctx context.Context, manifest *types.ModelManifest) {
	go func() {
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := SyncModelFlags(ctx, manifest)
			if err == nil {
				return
			}
			log.Printf("sync-model-flags: attempt %d/%d failed: %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				select {
				case <-ctx.Done():
					log.Printf("sync-model-flags: cancelled, stopping retries")
					return
				case <-time.After(retryDelay):
				}
			}
		}
		log.Printf("sync-model-flags: all %d attempts failed, giving up", maxRetries)
	}()
}

// SyncModelFlags ensures every model in the manifest has a corresponding
// Unleash feature flag. Flags are created disabled with type "release"
// and tagged scope:workspace so they appear in the admin UI.
//
// Required env vars: UNLEASH_ADMIN_URL, UNLEASH_ADMIN_TOKEN
// Optional env var:  UNLEASH_PROJECT (default: "default")
func SyncModelFlags(ctx context.Context, manifest *types.ModelManifest) error {
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
		log.Printf("sync-model-flags: UNLEASH_ADMIN_URL or UNLEASH_ADMIN_TOKEN not set, skipping")
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}

	// Ensure the "scope" tag type exists before creating flags
	if err := ensureTagType(ctx, client, adminURL, "scope", "Controls flag visibility scope", adminToken); err != nil {
		return fmt.Errorf("ensuring scope tag type: %w", err)
	}

	var created, skipped, excluded, errCount int
	log.Printf("Syncing Unleash flags for %d models...", len(manifest.Models))

	for _, model := range manifest.Models {
		if model.ID == manifest.DefaultModel {
			log.Printf("  %s: default model, no flag needed", model.ID)
			excluded++
			continue
		}

		if !model.Available {
			log.Printf("  %s: not available, skipping flag creation", model.ID)
			excluded++
			continue
		}

		flagName := fmt.Sprintf("model.%s.enabled", model.ID)

		exists, err := flagExists(ctx, client, adminURL, project, flagName, adminToken)
		if err != nil {
			log.Printf("  ERROR checking %s: %v", flagName, err)
			errCount++
			continue
		}

		if exists {
			log.Printf("  %s: already exists, skipping", flagName)
			skipped++
			continue
		}

		description := fmt.Sprintf("Enable %s (%s) for users", model.Label, model.ID)
		if err := createFlag(ctx, client, adminURL, project, flagName, description, adminToken); err != nil {
			if errors.Is(err, errConflict) {
				log.Printf("  %s: created by another instance, skipping", flagName)
				skipped++
				continue
			}
			log.Printf("  ERROR creating %s: %v", flagName, err)
			errCount++
			continue
		}

		if err := addTag(ctx, client, adminURL, flagName, adminToken); err != nil {
			log.Printf("  WARNING: created %s but failed to add tag: %v", flagName, err)
		}

		if err := addRolloutStrategy(ctx, client, adminURL, project, environment, flagName, adminToken); err != nil {
			log.Printf("  WARNING: created %s but failed to add rollout strategy: %v", flagName, err)
		}

		log.Printf("  %s: created (disabled, 0%% rollout)", flagName)
		created++
	}

	log.Printf("Summary: %d created, %d skipped, %d excluded, %d errors", created, skipped, excluded, errCount)

	if errCount > 0 {
		return fmt.Errorf("%d errors occurred during sync", errCount)
	}
	return nil
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
	// Check if tag type exists
	url := fmt.Sprintf("%s/api/admin/tag-types/%s", adminURL, url.PathEscape(name))
	resp, err := doRequest(ctx, client, "GET", url, token, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == http.StatusOK {
		log.Printf("Tag type %q already exists", name)
		return nil
	}

	// Create it
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
	url := fmt.Sprintf("%s/api/admin/projects/%s/features/%s", adminURL, url.PathEscape(project), url.PathEscape(flagName))
	resp, err := doRequest(ctx, client, "GET", url, token, nil)
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
	url := fmt.Sprintf("%s/api/admin/projects/%s/features", adminURL, url.PathEscape(project))
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

	resp, err := doRequest(ctx, client, "POST", url, token, bytes.NewReader(body))
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

func addTag(ctx context.Context, client *http.Client, adminURL, flagName, token string) error {
	url := fmt.Sprintf("%s/api/admin/features/%s/tags", adminURL, url.PathEscape(flagName))
	body, err := json.Marshal(map[string]string{
		"type":  "scope",
		"value": "workspace",
	})
	if err != nil {
		return fmt.Errorf("marshaling tag request: %w", err)
	}

	resp, err := doRequest(ctx, client, "POST", url, token, bytes.NewReader(body))
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
	url := fmt.Sprintf("%s/api/admin/projects/%s/features/%s/environments/%s/strategies",
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

	resp, err := doRequest(ctx, client, "POST", url, token, bytes.NewReader(body))
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

func doRequest(ctx context.Context, client *http.Client, method, url, token string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}
