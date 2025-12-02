package types

// GetProviderSpecificGuidance returns remediation guidance for provider-specific errors
func GetProviderSpecificGuidance(provider ProviderType, errorType string) string {
	switch provider {
	case ProviderGitHub:
		switch errorType {
		case "auth":
			return "Ensure your GitHub App is installed and has access to the repository, or configure a GitHub PAT in the project runner secret"
		case "permissions":
			return "Ensure the GitHub App or PAT has write access to the repository"
		case "not_found":
			return "Verify the repository URL is correct and you have access to it on GitHub"
		default:
			return "Check your GitHub repository configuration and try again"
		}
	case ProviderGitLab:
		switch errorType {
		case "auth":
			return "Connect your GitLab account with a valid Personal Access Token via /auth/gitlab/connect"
		case "permissions":
			return "Ensure your GitLab PAT has 'api', 'read_repository', and 'write_repository' scopes"
		case "not_found":
			return "Verify the repository URL is correct and you have access to it on GitLab"
		default:
			return "Check your GitLab repository configuration and try again"
		}
	default:
		return "Check your repository configuration and try again"
	}
}
