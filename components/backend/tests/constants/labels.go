// Package constants provides constants shared across test suites
package constants

// Test Label Constants - used for organizing and categorizing Ginkgo tests
const (
	// Top-level test categories
	LabelUnit = "unit"

	// Package/area labels
	LabelHandlers = "handlers"
	LabelGit      = "git"
	LabelTypes    = "types"

	// Specific component labels for handlers
	LabelRepo        = "repo"
	LabelRepoSeed    = "repo_seed"
	LabelSecrets     = "secrets"
	LabelRepository  = "repository"
	LabelMiddleware  = "middleware"
	LabelPermissions = "permissions"
	LabelProjects    = "projects"
	LabelGitHubAuth  = "github-auth"
	LabelGitLabAuth  = "gitlab-auth"
	LabelSessions    = "sessions"
	LabelContent     = "content"
	LabelDisplayName = "display-name"
	LabelHealth      = "health"

	// Specific component labels for other areas
	LabelOperations = "operations" // for git operations
	LabelCommon     = "common"     // for common types
)
