package handlers_test

import (
	"testing"

	"ambient-code-backend/handlers"
	"ambient-code-backend/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseRepoMap_NewFormat verifies parsing of new format repos from CR maps
func TestParseRepoMap_NewFormat(t *testing.T) {
	tests := []struct {
		name        string
		repoMap     map[string]interface{}
		expectError bool
		validate    func(t *testing.T, repo types.SimpleRepo)
	}{
		{
			name: "new format with input and output",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url":    "https://github.com/org/repo",
					"branch": "main",
				},
				"output": map[string]interface{}{
					"url":    "https://github.com/user/fork",
					"branch": "feature",
				},
				"autoPush": true,
			},
			expectError: false,
			validate: func(t *testing.T, repo types.SimpleRepo) {
				require.NotNil(t, repo.Input)
				assert.Equal(t, "https://github.com/org/repo", repo.Input.URL)
				assert.Equal(t, "main", *repo.Input.Branch)

				require.NotNil(t, repo.Output)
				assert.Equal(t, "https://github.com/user/fork", repo.Output.URL)
				assert.Equal(t, "feature", *repo.Output.Branch)

				require.NotNil(t, repo.AutoPush)
				assert.True(t, *repo.AutoPush)
			},
		},
		{
			name: "new format input only",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url":    "https://github.com/org/repo",
					"branch": "develop",
				},
				"autoPush": false,
			},
			expectError: false,
			validate: func(t *testing.T, repo types.SimpleRepo) {
				require.NotNil(t, repo.Input)
				assert.Equal(t, "https://github.com/org/repo", repo.Input.URL)
				assert.Equal(t, "develop", *repo.Input.Branch)

				assert.Nil(t, repo.Output)

				require.NotNil(t, repo.AutoPush)
				assert.False(t, *repo.AutoPush)
			},
		},
		{
			name: "new format without branches",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url": "https://github.com/org/repo",
				},
				"autoPush": true,
			},
			expectError: false,
			validate: func(t *testing.T, repo types.SimpleRepo) {
				require.NotNil(t, repo.Input)
				assert.Equal(t, "https://github.com/org/repo", repo.Input.URL)
				assert.Nil(t, repo.Input.Branch)
			},
		},
		{
			name: "new format without autoPush field",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url": "https://github.com/org/repo",
				},
			},
			expectError: false,
			validate: func(t *testing.T, repo types.SimpleRepo) {
				require.NotNil(t, repo.Input)
				assert.Nil(t, repo.AutoPush, "AutoPush should be nil when not specified")
			},
		},
		{
			name: "new format with empty input URL",
			repoMap: map[string]interface{}{
				"input": map[string]interface{}{
					"url": "",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := handlers.ParseRepoMap(tt.repoMap)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, repo)
			}
		})
	}
}

// TestParseRepoMap_LegacyFormat verifies parsing of legacy format repos from CR maps
func TestParseRepoMap_LegacyFormat(t *testing.T) {
	tests := []struct {
		name        string
		repoMap     map[string]interface{}
		expectError bool
		validate    func(t *testing.T, repo types.SimpleRepo)
	}{
		{
			name: "legacy format with branch",
			repoMap: map[string]interface{}{
				"url":    "https://github.com/org/repo",
				"branch": "main",
			},
			expectError: false,
			validate: func(t *testing.T, repo types.SimpleRepo) {
				assert.Equal(t, "https://github.com/org/repo", repo.URL)
				require.NotNil(t, repo.Branch)
				assert.Equal(t, "main", *repo.Branch)

				// New format fields should be nil
				assert.Nil(t, repo.Input)
				assert.Nil(t, repo.Output)
				assert.Nil(t, repo.AutoPush)
			},
		},
		{
			name: "legacy format without branch",
			repoMap: map[string]interface{}{
				"url": "https://github.com/org/repo",
			},
			expectError: false,
			validate: func(t *testing.T, repo types.SimpleRepo) {
				assert.Equal(t, "https://github.com/org/repo", repo.URL)
				assert.Nil(t, repo.Branch)

				// New format fields should be nil
				assert.Nil(t, repo.Input)
			},
		},
		{
			name: "legacy format with empty URL",
			repoMap: map[string]interface{}{
				"url": "",
			},
			expectError: true,
		},
		{
			name: "legacy format with whitespace-only branch",
			repoMap: map[string]interface{}{
				"url":    "https://github.com/org/repo",
				"branch": "   ",
			},
			expectError: false,
			validate: func(t *testing.T, repo types.SimpleRepo) {
				assert.Equal(t, "https://github.com/org/repo", repo.URL)
				assert.Nil(t, repo.Branch, "Whitespace-only branch should result in nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := handlers.ParseRepoMap(tt.repoMap)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validate(t, repo)
			}
		})
	}
}

// TestParseRepoMap_RoundTrip verifies ToMapForCR and ParseRepoMap are inverses
func TestParseRepoMap_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		repo types.SimpleRepo
	}{
		{
			name: "new format with all fields",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL:    "https://github.com/org/repo",
					Branch: types.StringPtr("main"),
				},
				Output: &types.RepoLocation{
					URL:    "https://github.com/user/fork",
					Branch: types.StringPtr("feature"),
				},
				AutoPush: types.BoolPtr(true),
			},
		},
		{
			name: "new format input only",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL:    "https://github.com/org/repo",
					Branch: types.StringPtr("develop"),
				},
				AutoPush: types.BoolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to map
			m := tt.repo.ToMapForCR()

			// Parse back
			parsed, err := handlers.ParseRepoMap(m)
			require.NoError(t, err)

			// Verify they match
			assert.Equal(t, tt.repo.Input, parsed.Input)
			assert.Equal(t, tt.repo.Output, parsed.Output)
			assert.Equal(t, tt.repo.AutoPush, parsed.AutoPush)
		})
	}
}
