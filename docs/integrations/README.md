# Platform Integrations

Documentation for integrating the Ambient Code Platform with external services.

## 🔌 Available Integrations

### Git Providers

**[GitHub Integration](../GITHUB_APP_SETUP.md)**
- GitHub App authentication
- Repository browsing
- PR creation
- OAuth flow for users

**[GitLab Integration](../gitlab-integration.md)**
- GitLab.com and self-hosted support
- Personal Access Token authentication
- Clone, commit, and push operations
- Multi-provider projects (mix GitHub and GitLab)

**Getting Started:**
- [GitHub Setup Guide](../GITHUB_APP_SETUP.md)
- [GitLab Token Setup](../gitlab-token-setup.md)
- [GitLab Self-Hosted Configuration](../gitlab-self-hosted.md)

---

### Google Workspace

**[Google Workspace Integration](google-workspace.md)**
- Google Drive file access
- Read and write capabilities
- Search functionality
- Session-scoped credentials

**Use Cases:**
- Read documents from Drive during sessions
- Create/update Drive files from agents
- Search Drive for relevant content

**Setup:** [Google Workspace Guide](google-workspace.md)

---

## 🔐 Authentication Patterns

### GitHub
- **GitHub App** - Recommended for organizations
- **Personal Access Tokens** - Fallback option
- **OAuth Flow** - User authorization

### GitLab
- **Personal Access Tokens** - Primary method
- **Instance URL** - Support for self-hosted

### Google Workspace
- **OAuth 2.0** - User authorization
- **Session-scoped** - Credentials auto-removed after session

## 🛠️ Configuration

All integrations are configured per-project via:
- **Web UI:** Project Settings → Integrations
- **API:** REST endpoints for connection management
- **Secrets:** Kubernetes Secrets for credential storage

## 📚 Integration Documentation

### GitHub
- [GitHub App Setup](../GITHUB_APP_SETUP.md) - Complete setup guide
- [API Endpoints](../api/github-endpoints.md) - GitHub API reference (if exists)

### GitLab
- [GitLab Integration](../gitlab-integration.md) - User guide
- [GitLab Token Setup](../gitlab-token-setup.md) - PAT creation
- [Self-Hosted GitLab](../gitlab-self-hosted.md) - Enterprise setup
- [GitLab Testing](../gitlab-testing-procedures.md) - Test procedures
- [GitLab API Endpoints](../api/gitlab-endpoints.md) - API reference

### Google Workspace
- [Google Workspace Integration](google-workspace.md) - Setup and usage

## 🔮 Future Integrations

Planned or potential integrations:
- **Jira** - Issue tracking and project management
- **Slack** - Notifications and chat integration
- **Azure DevOps** - Repository and pipeline integration
- **Bitbucket** - Alternative Git provider

## 🤝 Adding New Integrations

To add a new integration:

1. **Design:** Create integration proposal with security review
2. **Implement:** Add backend handlers and frontend UI
3. **Test:** Add contract and E2E tests
4. **Document:** Create integration guide in this directory
5. **Example:** Provide example usage and configuration

See [Contributing Guide](../../CONTRIBUTING.md) for development workflow.

---

**Questions?** Open a [GitHub Discussion](https://github.com/ambient-code/vTeam/discussions)
