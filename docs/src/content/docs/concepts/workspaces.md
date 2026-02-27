---
title: "Workspaces"
---

A **workspace** is the top-level container in the Ambient Code Platform. It groups sessions, secrets, and shared settings so that teams can collaborate within a well-defined boundary.

:::note
In the API and codebase, workspaces are referred to as **projects**. The terms are interchangeable -- the UI says "workspace" while API endpoints and Kubernetes resources use "project."
:::

## Creating a workspace

Open the platform UI and click **New Workspace**. You will be prompted for:

| Field | Purpose |
|-------|---------|
| **Display name** | A human-readable label shown across the UI. |
| **Description** | Optional notes describing what the workspace is for. |

After creation you land on the workspace dashboard, where you can start sessions, configure integrations, and invite collaborators.

## Workspace settings

Each workspace carries its own configuration. Open **Settings** from the workspace sidebar to adjust the following.

### General

- **Display name and description** -- Update at any time.

### Storage

By default, session data is stored in a shared storage backend. If your organization requires data isolation you can configure a **custom S3-compatible bucket** per workspace:

- Bucket name, endpoint, region
- Access key and secret key

### Secrets

Workspaces manage two categories of secrets:

- **Runner secrets** -- The `ANTHROPIC_API_KEY` that powers the AI agent. Every workspace needs one.
- **Integration secrets** -- Tokens for external services such as GitHub, GitLab, Jira, and custom environment variables that should be available inside sessions.

Secrets are stored securely and are only injected into session pods at runtime.

## Sharing and permissions

You can share a workspace with individual users or groups. Each share is assigned one of three roles:

| Role | Capabilities |
|------|-------------|
| **View** | See sessions, read chat history and artifacts. Cannot create or modify. |
| **Edit** | Everything in View, plus create sessions, send messages, and manage repos. |
| **Admin** | Everything in Edit, plus manage workspace settings, secrets, and sharing. |

To invite someone, go to **Settings > Sharing**, search for a user or group, and pick a role.

## API keys

For programmatic access -- CI/CD pipelines, scripts, or external tooling -- you can create **workspace-scoped API keys**.

1. Navigate to **Settings > API Keys**.
2. Click **Create API Key**.
3. Copy the key immediately; it will not be shown again.

Each API key is assigned an explicit role -- **Admin**, **Edit**, or **View** (defaults to Edit) -- that controls what the key can do. Keys can be revoked at any time.

## Best practices

- **One workspace per team or project** -- keeps secrets, integrations, and permissions cleanly separated.
- **Set a session timeout** -- prevents forgotten sessions from running indefinitely.
- **Use descriptive names** -- other users will see the workspace name when it is shared with them.
- **Rotate API keys regularly** -- treat them like any other credential.
