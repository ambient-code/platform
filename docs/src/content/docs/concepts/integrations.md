---
title: "Integrations"
---

Integrations connect external services to the Ambient Code Platform, giving the AI agent access to tools like repository hosting, issue trackers, and document storage. Integrations are **user-scoped** -- they are tied to your SSO identity and stored at the cluster level, so once configured, they are available across all your workspaces. MCP tools and runner secrets are workspace-scoped.

## Overview

Each integration appears on the **Integrations** page with a connection status indicator:

- **Connected** -- credentials are valid and the service is reachable.
- **Disconnected** -- no credentials configured, or the existing ones have expired.

You can connect and disconnect integrations at any time without affecting running sessions (changes take effect on the next session start).

## GitHub

GitHub integration lets the agent clone repositories, read pull requests, create branches, and push commits.

### Setup options

| Method | Best for |
|--------|---------|
| **GitHub App** (recommended) | Organizations that want fine-grained permission control and automatic token refresh. |
| **Personal Access Token (PAT)** | Individual users or quick setups where installing an app is not practical. |

### GitHub App

1. Navigate to **Integrations > GitHub**.
2. Click **Connect to GitHub App**.
3. You will be redirected to GitHub to authorize the Ambient Code Platform app.
4. Select the organization and repositories you want to grant access to.
5. Complete the OAuth flow -- you will be redirected back to the platform.

The app handles token refresh automatically. You can adjust repository access at any time from your GitHub organization settings.

### Personal Access Token

1. In GitHub, go to **Settings > Developer settings > Personal access tokens > Fine-grained tokens**.
2. Create a token with the `repo` scope (or specific repository access).
3. Back in the platform, go to **Integrations > GitHub** and paste the token.

PATs do not auto-refresh. You will need to rotate them before they expire.

## GitLab

GitLab integration provides the same repository access capabilities as GitHub.

### Setup

1. Navigate to **Integrations > GitLab**.
2. Optionally enter your **GitLab instance URL** if you use a self-managed instance (defaults to `https://gitlab.com`).
3. In GitLab, go to **Preferences > Access Tokens** and create a token with `read_repository` and `write_repository` scopes.
4. Paste the token and click **Connect**.

## Jira

Jira integration enables the agent to read issues, create tickets, and update statuses.

### Setup

1. Navigate to **Integrations > Jira**.
2. Provide your **Jira instance URL** (e.g., `https://yourcompany.atlassian.net`).
3. Enter the **email address** associated with your Jira account.
4. Generate an [API token](https://id.atlassian.com/manage-profile/security/api-tokens) from your Atlassian account and paste it into the **API Token** field.
5. Click **Connect**.

Ensure your Atlassian account has the required project permissions.

## Google Drive

Google Drive integration allows the agent to access files stored in your Google Drive.

### Setup

1. Navigate to **Integrations > Google Drive**.
2. Click **Connect Google Drive**.
3. Sign in with your Google account and grant the requested Drive permissions.
4. You will be redirected back to the platform once authorization is complete.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Status stays **Disconnected** after setup | Authorization callback did not complete | Retry the connection flow; check for pop-up blockers. |
| Token expired errors in sessions | PAT reached its expiry date | Generate a new token and update the integration. |
| "Insufficient permissions" in agent logs | Token scope is too narrow | Recreate the token with the required scopes. |
| Jira actions fail | Network or permission issue | Verify the Jira URL is reachable from the cluster and that your account has project access. |
