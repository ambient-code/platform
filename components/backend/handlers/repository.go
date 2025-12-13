package handlers

import (
	"context"
	"fmt"

	"ambient-code-backend/gitlab"
	"ambient-code-backend/types"
)

// DetectRepositoryProvider determines the Git provider from a repository URL
// @Summary      Detect repository provider
// @Description  Analyzes a repository URL to determine the Git provider (GitHub, GitLab, etc.)
// @Tags         repositories
// @Produce      json
// @Param        repoURL  query     string  true  "Repository URL"
// @Success      200  {string}  string  "Provider type (github, gitlab, etc.)"
// @Router       /repositories/detect-provider [get]
func DetectRepositoryProvider(repoURL string) types.ProviderType {
	return types.DetectProvider(repoURL)
}

// ValidateGitLabRepository validates a GitLab repository URL and token access
// @Summary      Validate GitLab repository
// @Description  Validates a GitLab repository URL and verifies token has access to the repository
// @Tags         repositories
// @Security     BearerAuth
// @Param        repoURL  query  string  true  "GitLab repository URL"
// @Param        token    query  string  true  "GitLab access token"
// @Success      200  {object}  map[string]interface{}  "Repository validated successfully"
// @Failure      400  {object}  map[string]string       "Invalid repository URL or token"
// @Failure      401  {object}  map[string]string       "Unauthorized - token validation failed"
// @Failure      404  {object}  map[string]string       "Repository not found"
// @Router       /repositories/validate-gitlab [get]
func ValidateGitLabRepository(ctx context.Context, repoURL, token string) error {
	if token == "" {
		return fmt.Errorf("GitLab token is required for repository validation")
	}

	// Validate URL format
	if err := gitlab.ValidateGitLabURL(repoURL); err != nil {
		return fmt.Errorf("invalid GitLab repository URL: %w", err)
	}

	// Validate token and repository access
	result, err := gitlab.ValidateTokenAndRepository(ctx, token, repoURL)
	if err != nil {
		return fmt.Errorf("failed to validate GitLab repository: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("GitLab validation failed: %s", result.ErrorMessage)
	}

	return nil
}

// NormalizeRepositoryURL normalizes a repository URL based on its provider
// @Summary      Normalize repository URL
// @Description  Converts a repository URL to its canonical form based on the detected provider
// @Tags         repositories
// @Param        repoURL   query  string  true  "Repository URL"
// @Param        provider  query  string  true  "Provider type (github, gitlab, etc.)"
// @Success      200  {string}  string              "Normalized repository URL"
// @Failure      400  {object}  map[string]string   "Invalid URL or unsupported provider"
// @Router       /repositories/normalize [get]
func NormalizeRepositoryURL(repoURL string, provider types.ProviderType) (string, error) {
	switch provider {
	case types.ProviderGitLab:
		return gitlab.NormalizeGitLabURL(repoURL)
	case types.ProviderGitHub:
		// GitHub normalization would go here (if implemented)
		return repoURL, nil
	default:
		return repoURL, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// GetRepositoryInfo retrieves information about a repository
// @Summary      Get repository information
// @Description  Parses repository URL and returns detailed information (provider, owner, repo, host, API URL)
// @Tags         repositories
// @Param        repoURL  query  string  true  "Repository URL"
// @Success      200  {object}  RepositoryInfo      "Repository information"
// @Failure      400  {object}  map[string]string   "Invalid repository URL or unsupported provider"
// @Router       /repositories/info [get]
func GetRepositoryInfo(repoURL string) (*RepositoryInfo, error) {
	provider := DetectRepositoryProvider(repoURL)

	info := &RepositoryInfo{
		URL:      repoURL,
		Provider: provider,
	}

	switch provider {
	case types.ProviderGitLab:
		parsed, err := gitlab.ParseGitLabURL(repoURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse GitLab URL: %w", err)
		}
		info.Owner = parsed.Owner
		info.Repo = parsed.Repo
		info.Host = parsed.Host
		info.APIURL = parsed.APIURL
		info.IsGitLabSelfHosted = gitlab.IsGitLabSelfHosted(parsed.Host)

	case types.ProviderGitHub:
		// GitHub parsing would go here (if needed)
		info.Host = "github.com"

	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	return info, nil
}

// RepositoryInfo contains parsed information about a repository
type RepositoryInfo struct {
	URL                string             `json:"url"`
	Provider           types.ProviderType `json:"provider"`
	Owner              string             `json:"owner,omitempty"`
	Repo               string             `json:"repo,omitempty"`
	Host               string             `json:"host,omitempty"`
	APIURL             string             `json:"apiUrl,omitempty"`
	IsGitLabSelfHosted bool               `json:"isGitlabSelfHosted,omitempty"`
}

// ValidateProjectRepository validates a repository for use in a project
// @Summary      Validate project repository
// @Description  Validates repository access for a specific user and project (checks token permissions)
// @Tags         repositories
// @Security     BearerAuth
// @Param        repoURL  query  string  true  "Repository URL"
// @Param        userID   query  string  true  "User ID"
// @Success      200  {object}  RepositoryInfo      "Repository validated successfully"
// @Failure      400  {object}  map[string]string   "Invalid repository URL"
// @Failure      401  {object}  map[string]string   "Unauthorized - user token not found or invalid"
// @Failure      403  {object}  map[string]string   "Forbidden - user lacks repository access"
// @Router       /repositories/validate-project [get]
func ValidateProjectRepository(ctx context.Context, repoURL string, userID string) (*RepositoryInfo, error) {
	// Get repository info
	info, err := GetRepositoryInfo(repoURL)
	if err != nil {
		return nil, err
	}

	// For GitLab repositories, validate access if we have a token
	if info.Provider == types.ProviderGitLab {
		// Try to get GitLab token for this user
		// Use the handlers package K8sClient and Namespace globals
		connMgr := gitlab.NewConnectionManager(K8sClient, Namespace)
		_, token, err := connMgr.GetGitLabConnectionWithToken(ctx, userID)
		if err != nil {
			// If no token found, just return info without validation
			// The user will need to connect GitLab account first
			gitlab.LogWarning("No GitLab token found for user %s, skipping repository validation", userID)
			return info, nil
		}

		// Validate repository access with the token
		if err := ValidateGitLabRepository(ctx, repoURL, token); err != nil {
			return nil, fmt.Errorf("repository validation failed: %w", err)
		}

		gitlab.LogInfo("GitLab repository %s validated successfully for user %s", repoURL, userID)
	}

	return info, nil
}

// EnrichProjectSettingsWithProviders adds provider information to repositories in ProjectSettings
// @Summary      Enrich project repositories
// @Description  Adds provider type metadata to repository configurations in project settings
// @Tags         repositories
// @Accept       json
// @Produce      json
// @Param        repositories  body  []map[string]interface{}  true  "Repository configurations"
// @Success      200  {array}  map[string]interface{}  "Enriched repository configurations with provider metadata"
// @Router       /repositories/enrich [post]
func EnrichProjectSettingsWithProviders(repositories []map[string]interface{}) []map[string]interface{} {
	enriched := make([]map[string]interface{}, len(repositories))

	for i, repo := range repositories {
		enrichedRepo := make(map[string]interface{})

		// Copy existing fields
		for k, v := range repo {
			enrichedRepo[k] = v
		}

		// Add provider if not already present
		if _, hasProvider := repo["provider"]; !hasProvider {
			if url, hasURL := repo["url"].(string); hasURL {
				provider := DetectRepositoryProvider(url)
				if provider != "" {
					enrichedRepo["provider"] = string(provider)
				}
			}
		}

		enriched[i] = enrichedRepo
	}

	return enriched
}
