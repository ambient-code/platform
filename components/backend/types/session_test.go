package types_test

import (
	"testing"

	"ambient-code-backend/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNormalizeRepo_LegacyFormat verifies that legacy format repos are converted to new format
func TestNormalizeRepo_LegacyFormat(t *testing.T) {
	tests := []struct {
		name           string
		repo           types.SimpleRepo
		sessionDefault bool
		wantInputURL   string
		wantInputBr    *string
		wantAutoPush   bool
	}{
		{
			name: "legacy with branch, session autoPush=false",
			repo: types.SimpleRepo{
				URL:    "https://github.com/org/repo",
				Branch: types.StringPtr("main"),
			},
			sessionDefault: false,
			wantInputURL:   "https://github.com/org/repo",
			wantInputBr:    types.StringPtr("main"),
			wantAutoPush:   false,
		},
		{
			name: "legacy with branch, session autoPush=true",
			repo: types.SimpleRepo{
				URL:    "https://github.com/org/repo",
				Branch: types.StringPtr("develop"),
			},
			sessionDefault: true,
			wantInputURL:   "https://github.com/org/repo",
			wantInputBr:    types.StringPtr("develop"),
			wantAutoPush:   true,
		},
		{
			name: "legacy without branch",
			repo: types.SimpleRepo{
				URL: "https://github.com/org/repo",
			},
			sessionDefault: false,
			wantInputURL:   "https://github.com/org/repo",
			wantInputBr:    nil,
			wantAutoPush:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := tt.repo.NormalizeRepo(tt.sessionDefault)
			require.NoError(t, err, "NormalizeRepo should not return error for valid repos")

			// Verify input structure was created
			require.NotNil(t, normalized.Input, "Input should not be nil")
			assert.Equal(t, tt.wantInputURL, normalized.Input.URL)
			if tt.wantInputBr != nil {
				require.NotNil(t, normalized.Input.Branch)
				assert.Equal(t, *tt.wantInputBr, *normalized.Input.Branch)
			} else {
				assert.Nil(t, normalized.Input.Branch)
			}

			// Verify autoPush was set from session default
			require.NotNil(t, normalized.AutoPush)
			assert.Equal(t, tt.wantAutoPush, *normalized.AutoPush)

			// Verify output is nil (legacy repos don't specify output)
			assert.Nil(t, normalized.Output)

			// Verify normalized struct uses new format fields only (not legacy fields)
			assert.Empty(t, normalized.URL, "normalized struct should not use legacy URL field")
			assert.Nil(t, normalized.Branch, "normalized struct should not use legacy Branch field")
		})
	}
}

// TestNormalizeRepo_NewFormat verifies that new format repos are returned unchanged
func TestNormalizeRepo_NewFormat(t *testing.T) {
	tests := []struct {
		name string
		repo types.SimpleRepo
	}{
		{
			name: "new format with input only",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL:    "https://github.com/org/repo",
					Branch: types.StringPtr("main"),
				},
				AutoPush: types.BoolPtr(true),
			},
		},
		{
			name: "new format with input and output",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL:    "https://github.com/org/repo",
					Branch: types.StringPtr("main"),
				},
				Output: &types.RepoLocation{
					URL:    "https://github.com/user/fork",
					Branch: types.StringPtr("feature"),
				},
				AutoPush: types.BoolPtr(false),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Session default should be ignored for repos already in new format
			normalized, err := tt.repo.NormalizeRepo(true)
			require.NoError(t, err, "NormalizeRepo should not return error for valid repos")

			// Should be identical to original
			assert.Equal(t, tt.repo.Input, normalized.Input)
			assert.Equal(t, tt.repo.Output, normalized.Output)
			assert.Equal(t, tt.repo.AutoPush, normalized.AutoPush)
		})
	}
}

// TestToMapForCR_NewFormat verifies conversion to CR map for new format repos
func TestToMapForCR_NewFormat(t *testing.T) {
	tests := []struct {
		name     string
		repo     types.SimpleRepo
		validate func(t *testing.T, m map[string]interface{})
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
			validate: func(t *testing.T, m map[string]interface{}) {
				input := m["input"].(map[string]interface{})
				assert.Equal(t, "https://github.com/org/repo", input["url"])
				assert.Equal(t, "main", input["branch"])

				output := m["output"].(map[string]interface{})
				assert.Equal(t, "https://github.com/user/fork", output["url"])
				assert.Equal(t, "feature", output["branch"])

				assert.Equal(t, true, m["autoPush"])
			},
		},
		{
			name: "new format input only, no branches",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL: "https://github.com/org/repo",
				},
				AutoPush: types.BoolPtr(false),
			},
			validate: func(t *testing.T, m map[string]interface{}) {
				input := m["input"].(map[string]interface{})
				assert.Equal(t, "https://github.com/org/repo", input["url"])
				_, hasBranch := input["branch"]
				assert.False(t, hasBranch, "branch should not be present when nil")

				_, hasOutput := m["output"]
				assert.False(t, hasOutput, "output should not be present when nil")

				assert.Equal(t, false, m["autoPush"])
			},
		},
		{
			name: "new format with nil autoPush",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL: "https://github.com/org/repo",
				},
			},
			validate: func(t *testing.T, m map[string]interface{}) {
				_, hasAutoPush := m["autoPush"]
				assert.False(t, hasAutoPush, "autoPush should not be present when nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.repo.ToMapForCR()
			tt.validate(t, m)
		})
	}
}

// TestToMapForCR_LegacyFormat verifies conversion preserves legacy format
func TestToMapForCR_LegacyFormat(t *testing.T) {
	repo := types.SimpleRepo{
		URL:    "https://github.com/org/repo",
		Branch: types.StringPtr("main"),
	}

	m := repo.ToMapForCR()

	assert.Equal(t, "https://github.com/org/repo", m["url"])
	assert.Equal(t, "main", m["branch"])

	// New format fields should not be present
	_, hasInput := m["input"]
	assert.False(t, hasInput, "input should not be present for legacy format")
}

// TestNormalizeAndConvert_RoundTrip verifies normalize + convert workflow
func TestNormalizeAndConvert_RoundTrip(t *testing.T) {
	// Start with legacy format
	legacy := types.SimpleRepo{
		URL:    "https://github.com/org/repo",
		Branch: types.StringPtr("main"),
	}

	// Normalize with session autoPush=true
	normalized, err := legacy.NormalizeRepo(true)
	require.NoError(t, err, "NormalizeRepo should not return error for valid repo")

	// Convert to CR map
	m := normalized.ToMapForCR()

	// Verify map has new format structure
	input := m["input"].(map[string]interface{})
	assert.Equal(t, "https://github.com/org/repo", input["url"])
	assert.Equal(t, "main", input["branch"])
	assert.Equal(t, true, m["autoPush"])

	// Legacy fields should not be in the map
	_, hasURL := m["url"]
	assert.False(t, hasURL, "legacy url should not be present")
}

// TestBackwardCompatibility_LegacyRepoFields ensures legacy fields still accessible
func TestBackwardCompatibility_LegacyRepoFields(t *testing.T) {
	repo := types.SimpleRepo{
		URL:    "https://github.com/org/repo",
		Branch: types.StringPtr("develop"),
	}

	// Legacy fields should still be accessible
	assert.Equal(t, "https://github.com/org/repo", repo.URL)
	require.NotNil(t, repo.Branch)
	assert.Equal(t, "develop", *repo.Branch)

	// New fields should be nil for legacy repos
	assert.Nil(t, repo.Input)
	assert.Nil(t, repo.Output)
	assert.Nil(t, repo.AutoPush)
}

// TestNormalizeRepo_ErrorCases verifies that NormalizeRepo returns errors for invalid repos
func TestNormalizeRepo_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		repo          types.SimpleRepo
		expectedError string
	}{
		{
			name: "legacy format with empty URL",
			repo: types.SimpleRepo{
				URL: "",
			},
			expectedError: "cannot normalize repo with empty url",
		},
		{
			name: "legacy format with whitespace-only URL",
			repo: types.SimpleRepo{
				URL: "   ",
			},
			expectedError: "cannot normalize repo with empty url",
		},
		{
			name: "new format with empty input URL",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL: "",
				},
			},
			expectedError: "cannot normalize repo with empty input.url",
		},
		{
			name: "new format with whitespace-only input URL",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL: "   ",
				},
			},
			expectedError: "cannot normalize repo with empty input.url",
		},
		{
			name: "output same as input (same URL and branch)",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL:    "https://github.com/org/repo",
					Branch: types.StringPtr("main"),
				},
				Output: &types.RepoLocation{
					URL:    "https://github.com/org/repo",
					Branch: types.StringPtr("main"),
				},
			},
			expectedError: "output repository must differ from input",
		},
		{
			name: "output same URL as input (both nil branches)",
			repo: types.SimpleRepo{
				Input: &types.RepoLocation{
					URL: "https://github.com/org/repo",
				},
				Output: &types.RepoLocation{
					URL: "https://github.com/org/repo",
				},
			},
			expectedError: "output repository must differ from input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.repo.NormalizeRepo(false)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
