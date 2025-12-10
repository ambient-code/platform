package handlers

import (
	"testing"

	"ambient-code-backend/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseSpec_ReposParsing verifies that parseSpec correctly parses repos
// in both legacy and new formats
func TestParseSpec_ReposParsing(t *testing.T) {
	tests := []struct {
		name     string
		spec     map[string]interface{}
		validate func(t *testing.T, result types.AgenticSessionSpec)
	}{
		{
			name: "new format repos with input/output/autoPush",
			spec: map[string]interface{}{
				"repos": []interface{}{
					map[string]interface{}{
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
				},
			},
			validate: func(t *testing.T, result types.AgenticSessionSpec) {
				require.Len(t, result.Repos, 1)

				repo := result.Repos[0]
				require.NotNil(t, repo.Input, "Input should be parsed")
				assert.Equal(t, "https://github.com/org/repo", repo.Input.URL)
				assert.Equal(t, "main", *repo.Input.Branch)

				require.NotNil(t, repo.Output, "Output should be parsed")
				assert.Equal(t, "https://github.com/user/fork", repo.Output.URL)
				assert.Equal(t, "feature", *repo.Output.Branch)

				require.NotNil(t, repo.AutoPush, "AutoPush should be parsed")
				assert.True(t, *repo.AutoPush)
			},
		},
		{
			name: "new format repos with input only",
			spec: map[string]interface{}{
				"repos": []interface{}{
					map[string]interface{}{
						"input": map[string]interface{}{
							"url":    "https://github.com/org/repo",
							"branch": "develop",
						},
						"autoPush": false,
					},
				},
			},
			validate: func(t *testing.T, result types.AgenticSessionSpec) {
				require.Len(t, result.Repos, 1)

				repo := result.Repos[0]
				require.NotNil(t, repo.Input)
				assert.Equal(t, "https://github.com/org/repo", repo.Input.URL)
				assert.Equal(t, "develop", *repo.Input.Branch)

				assert.Nil(t, repo.Output, "Output should be nil")

				require.NotNil(t, repo.AutoPush)
				assert.False(t, *repo.AutoPush)
			},
		},
		{
			name: "legacy format repos",
			spec: map[string]interface{}{
				"repos": []interface{}{
					map[string]interface{}{
						"url":    "https://github.com/org/legacy-repo",
						"branch": "master",
					},
				},
			},
			validate: func(t *testing.T, result types.AgenticSessionSpec) {
				require.Len(t, result.Repos, 1)

				repo := result.Repos[0]
				// Legacy format should be preserved
				assert.Equal(t, "https://github.com/org/legacy-repo", repo.URL)
				require.NotNil(t, repo.Branch)
				assert.Equal(t, "master", *repo.Branch)

				// New format fields should be nil
				assert.Nil(t, repo.Input)
				assert.Nil(t, repo.Output)
				assert.Nil(t, repo.AutoPush)
			},
		},
		{
			name: "mixed format repos (legacy and new)",
			spec: map[string]interface{}{
				"repos": []interface{}{
					// Legacy format
					map[string]interface{}{
						"url":    "https://github.com/org/legacy",
						"branch": "main",
					},
					// New format
					map[string]interface{}{
						"input": map[string]interface{}{
							"url":    "https://github.com/org/new",
							"branch": "develop",
						},
						"autoPush": true,
					},
				},
			},
			validate: func(t *testing.T, result types.AgenticSessionSpec) {
				require.Len(t, result.Repos, 2)

				// First repo (legacy)
				legacy := result.Repos[0]
				assert.Equal(t, "https://github.com/org/legacy", legacy.URL)
				assert.Equal(t, "main", *legacy.Branch)
				assert.Nil(t, legacy.Input)

				// Second repo (new)
				newRepo := result.Repos[1]
				require.NotNil(t, newRepo.Input)
				assert.Equal(t, "https://github.com/org/new", newRepo.Input.URL)
				assert.True(t, *newRepo.AutoPush)
			},
		},
		{
			name: "empty repos array",
			spec: map[string]interface{}{
				"repos": []interface{}{},
			},
			validate: func(t *testing.T, result types.AgenticSessionSpec) {
				assert.Empty(t, result.Repos)
			},
		},
		{
			name: "repos field missing",
			spec: map[string]interface{}{},
			validate: func(t *testing.T, result types.AgenticSessionSpec) {
				assert.Nil(t, result.Repos)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSpec(tt.spec)
			tt.validate(t, result)
		})
	}
}
