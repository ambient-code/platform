---
title: "Integrations"
---

Integrations connect external services to the Ambient Code Platform, giving the AI agent access to tools like repository hosting, issue trackers, and document storage. Integrations are **global** -- once configured, they work across all your workspaces.

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
2. Click **Connect with GitHub App**.
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

### Setup options

| Method | Best for |
|--------|---------|
| **OAuth 2.0** (recommended) | Self-managed and SaaS GitLab instances with SSO. |
| **Personal Access Token** | Quick setup or service accounts. |

### OAuth

1. Navigate to **Integrations > GitLab**.
2. Click **Connect with GitLab**.
3. Authorize the application in GitLab.
4. You will be redirected back to the platform once complete.

### Personal Access Token

1. In GitLab, go to **Preferences > Access Tokens**.
2. Create a token with `read_repository` and `write_repository` scopes.
3. Paste the token in **Integrations > GitLab** on the platform.

## Jira

Jira integration enables the agent to read issues, create tickets, and update statuses.

### Setup

1. Navigate to **Integrations > Jira**.
2. Click **Connect with Jira**.
3. Complete the OAuth 2.0 authorization flow in Atlassian.

Ensure your Atlassian account has the required project permissions.

## Google Workspace

Google Workspace integration allows the agent to access Google Drive, Calendar, and Gmail.

### Setup

1. Navigate to **Integrations > Google Workspace**.
2. Click **Connect with Google Workspace**.
3. Sign in with your Google account and grant the requested permissions.
4. You will be redirected back to the platform once authorization is complete.

## Troubleshooting

| Symptom | Likely cause | Fix |
|---------|-------------|-----|
| Status stays **Disconnected** after setup | OAuth callback did not complete | Retry the connection flow; check for pop-up blockers. |
| Token expired errors in sessions | PAT reached its expiry date | Generate a new token and update the integration. |
| "Insufficient permissions" in agent logs | Token scope is too narrow | Recreate the token with the required scopes. |
| Jira actions fail | Network or permission issue | Verify the Jira URL is reachable from the cluster and that your account has project access. |
