# Git Authentication Setup

The Ambient Code Platform supports **two independent git authentication methods** that serve different purposes:

1. **GitHub App**: Backend OAuth login + Repository browser in UI
2. **Project-level Git Secrets**: Runner git operations (clone, commit, push)

You can use **either one or both** - the system gracefully handles all scenarios.

## Project-Level Git Authentication

This approach allows each project to have its own Git credentials, similar to how `ANTHROPIC_API_KEY` is configured.

### Setup: Using GitHub API Token

**1. Create a secret with a GitHub token:**

```bash
# Create secret with GitHub personal access token
oc create secret generic my-runner-secret \
  --from-literal=ANTHROPIC_API_KEY="your-anthropic-api-key" \
  --from-literal=GIT_USER_NAME="Your Name" \
  --from-literal=GIT_USER_EMAIL="your.email@example.com" \
  --from-literal=GIT_TOKEN="ghp_your_github_token" \
  -n your-project-namespace
```

**2. Reference the secret in your ProjectSettings:**

(Most users will access this from the frontend)

```yaml
apiVersion: vteam.ambient-code/v1
kind: ProjectSettings
metadata:
  name: my-project
  namespace: your-project-namespace
spec:
  runnerSecret: my-runner-secret
```

**3. Use HTTPS URLs in your AgenticSession:**

(Most users will access this from the frontend)

```yaml
spec:
  repos:
    - input:
        url: "https://github.com/your-org/your-repo.git"
        branch: "main"
    output:
      createPR: true
      prTitle: "feat: AI-generated changes"
```

The runner automatically uses credentials from the secret for git operations.

### Setup: Using SSH Keys

For SSH-based authentication:

```bash
# Create secret with SSH key
oc create secret generic my-runner-secret \
  --from-literal=ANTHROPIC_API_KEY="your-anthropic-api-key" \
  --from-literal=GIT_USER_NAME="Your Name" \
  --from-literal=GIT_USER_EMAIL="your.email@example.com" \
  --from-file=GIT_SSH_KEY=~/.ssh/id_rsa \
  --from-literal=GIT_SSH_KNOWN_HOSTS="$(ssh-keyscan github.com 2>/dev/null)" \
  -n your-project-namespace
```

Then use SSH URLs:
```yaml
repos:
  - input:
      url: "git@github.com:your-org/your-repo.git"
```

## GitHub App (Optional)

The GitHub App provides additional features:
- OAuth-based user login
- Repository browser in the UI
- Per-user GitHub integrations

**When to use:**
- You want users to log in with their GitHub accounts
- You want a repository browser in the UI
- You need per-user GitHub permissions

**Setup:** See [GitHub App Setup Guide](../GITHUB_APP_SETUP.md)

## GitLab Authentication

For GitLab repositories:

```bash
# Create secret with GitLab token
oc create secret generic my-runner-secret \
  --from-literal=ANTHROPIC_API_KEY="your-anthropic-api-key" \
  --from-literal=GIT_USER_NAME="Your Name" \
  --from-literal=GIT_USER_EMAIL="your.email@example.com" \
  --from-literal=GIT_TOKEN="glpat-your-gitlab-token" \
  -n your-project-namespace
```

**For self-hosted GitLab**, the URL format automatically detects the instance:
```yaml
repos:
  - input:
      url: "https://gitlab.company.com/org/repo.git"
```

See [GitLab Integration Guide](../gitlab-integration.md) for detailed setup.

## Security Best Practices

### Token Scopes

**GitHub Personal Access Token**:
- ✅ `repo` - Full repository access (required)
- ✅ `workflow` - If updating GitHub Actions workflows

**GitLab Personal Access Token**:
- ✅ `api` - Full API access
- ✅ `write_repository` - Push to repositories

### SSH Key Management

- Use **dedicated keys** for each environment (dev, staging, prod)
- **Never** use your personal SSH key
- Set **read-only** access where possible
- **Rotate keys** periodically

### Secret Lifecycle

- Create secrets **per project namespace**
- Secrets are **namespace-scoped** (isolated)
- Runners access secrets **only in their namespace**
- Delete secrets when projects are deleted

## Multiple Git Providers

Projects can use different git providers simultaneously:

```yaml
repos:
  - input:
      url: "https://github.com/org/frontend.git"
  - input:
      url: "https://gitlab.com/org/backend.git"
```

The runner automatically detects the provider and uses appropriate authentication.

## Troubleshooting

### "Authentication failed" errors

**HTTPS:**
- Verify `GIT_TOKEN` is set in the secret
- Check token has correct scopes
- Ensure token is not expired

**SSH:**
- Verify `GIT_SSH_KEY` is in the secret
- Check `GIT_SSH_KNOWN_HOSTS` includes the Git host
- Ensure SSH key is added to your Git account

### "Permission denied" errors

- Check token/key has **write access** to the repository
- Verify repository URL is correct
- Ensure you're not using a fork URL when you need the original

### Runner can't find credentials

- Verify `runnerSecret` is set in ProjectSettings
- Check secret exists: `oc get secret <name> -n <namespace>`
- Ensure secret has required keys (`GIT_TOKEN` or `GIT_SSH_KEY`)

## Related Documentation

- [GitHub App Setup](../GITHUB_APP_SETUP.md) - OAuth and repository browser
- [GitLab Integration](../gitlab-integration.md) - GitLab-specific configuration
- [GitLab Token Setup](../gitlab-token-setup.md) - Creating GitLab PATs
- [ProjectSettings Reference](../reference/project-settings.md) - Configuration schema (if exists)
