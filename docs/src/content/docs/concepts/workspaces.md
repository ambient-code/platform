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

Session state is stored using an S3-compatible backend (such as MinIO), typically configured cluster-wide by the platform administrator.

:::note
By default, workspaces use the cluster-level storage configuration. Per-project storage overrides are supported via project secrets, with operator defaults as fallback.
:::

### Secrets

Workspaces manage two categories of secrets:

- **Runner secrets** -- The `ANTHROPIC_API_KEY` that powers the AI agent. Every workspace needs one.
- **Integration secrets** -- Tokens for external services such as GitHub, GitLab, Jira, and custom environment variables that should be available inside sessions.

Secrets are stored securely and are only injected into session pods at runtime.

## Sharing and permissions

You can share a workspace with individual **users** or **groups**.
When you grant a permission, the platform creates a Kubernetes `RoleBinding` in the workspace namespace that maps the subject to one of three ClusterRoles.

The user who creates a workspace is automatically assigned the **Admin** role.

### Roles

Each role is cumulative -- higher roles include all capabilities of lower roles.

| Role | Capabilities |
|------|-------------|
| **View** | See sessions and their status. Read chat history and browse artifacts. View workspace settings, integrations, and storage (read-only). |
| **Edit** | Everything in View, plus create, stop, and delete sessions. Send messages to running sessions. Add repositories and manage workflows. |
| **Admin** | Everything in Edit, plus manage workspace settings, secrets, sharing, and API keys. Grant and revoke permissions for other users and groups. |

### Granting access

1. Open **Settings > Sharing** in the workspace sidebar.
2. Click **Grant Permission**.
3. Choose whether you are sharing with a **User** or a **Group**.
4. Enter the subject name and select a role.
5. Click **Grant Permission** to confirm.

Only users with the Admin role can grant permissions.

### Revoking access

To remove access, click the delete icon next to any entry in the sharing table.
The user or group immediately loses access to the workspace.

## API keys

For programmatic access -- CI/CD pipelines, scripts, or external tooling -- you can create **workspace-scoped API keys**. Each key is backed by a dedicated Kubernetes ServiceAccount and bound to a role, so it has the same permission model as a human user.

### Creating an API key

1. Navigate to **Settings > API Keys**.
2. Click **Create Key**.
3. Fill in the required fields:

| Field | Required | Description |
|-------|----------|-------------|
| **Name** | Yes | A label to identify the key (for example, `my-ci-key`). Maximum 64 characters. |
| **Description** | No | Optional context such as "Used by CI pipelines". Maximum 200 characters. |
| **Role** | Yes | The permission level for the key. Defaults to **Edit**. |
| **Token Lifetime** | Yes | How long the key remains valid. Defaults to **90 days**. |

The available roles match the workspace sharing roles:

| Role | Capabilities |
|------|-------------|
| **View** | Read sessions, chat history, and artifacts. |
| **Edit** | Everything in View, plus create sessions, send messages, and manage repos. |
| **Admin** | Everything in Edit, plus manage workspace settings, secrets, and sharing. |

Choose a token lifetime based on your use case:

| Lifetime | Seconds | Typical use case |
|----------|---------|------------------|
| 1 day | 86,400 | Short-lived CI jobs or one-time scripts |
| 7 days | 604,800 | Weekly pipeline runs |
| 30 days | 2,592,000 | Monthly rotation schedules |
| 90 days | 7,776,000 | General-purpose automation (default) |
| 1 year | 31,536,000 | Long-running integrations |
| No expiration | -- | Persistent service keys that you rotate manually |

4. Click **Create Key**.

### Copying the key

After creation, the platform displays the full key **exactly once**. Copy it immediately and store it in a secure location such as a secrets manager. You cannot retrieve the key value after you close the dialog.

### Using the key

Pass the key as a Bearer token in the `Authorization` header:

```bash
curl -H "Authorization: Bearer <your-api-key>" \
  https://<platform-url>/api/projects/<workspace>/sessions
```

Replace `<your-api-key>`, `<platform-url>`, and `<workspace>` with your actual values.

### Managing keys

Open **Settings > API Keys** to view all active keys for the workspace. The key list shows:

- **Name** and **description** -- identify each key at a glance.
- **Created** -- how long ago the key was created.
- **Last used** -- when the key was last used to authenticate a request, or "Never" if unused.
- **Role** -- the permission level assigned to the key.

To revoke a key, click the delete icon next to it and confirm. Revocation is immediate -- any system using that key loses access as soon as you confirm.

### Security best practices

- **Treat API keys like credentials.** Do not commit them to version control or share them in plaintext.
- **Use the minimum required role.** If a pipeline only reads session results, assign the **View** role.
- **Set an expiration.** Prefer a finite lifetime and rotate keys before they expire.
- **Rotate keys regularly.** Create a replacement key, update your integrations, then revoke the old key.
- **Revoke unused keys.** Check the "Last used" column periodically and delete keys that are no longer in use.

## Best practices

- **One workspace per team or project** -- keeps secrets, integrations, and permissions cleanly separated.
- **Set a session timeout** -- prevents forgotten sessions from running indefinitely.
- **Use descriptive names** -- other users will see the workspace name when it is shared with them.
- **Rotate API keys regularly** -- treat them like any other credential.
