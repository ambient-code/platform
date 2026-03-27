---
title: "Quick start"
---

Create your first AI agent session using the Ambient Code Platform web interface.

## Sign in

ACP uses SSO for authentication -- there is no separate account to create. Navigate to your organization's ACP URL and sign in with your existing credentials.

After signing in, you land on the **Workspaces** page.

## Create a workspace

Workspaces are project containers that hold your sessions, integrations, and team permissions.

1. Click **Create Workspace**.
2. Enter a **name** (e.g., `my-team`) and optional **description**.
3. Click **Create**.

You are now inside your workspace. From here you can configure integrations and start sessions.

## Configure integrations (optional)

Integrations let agents interact with your external services. You can skip this step and add integrations later.

To connect a service:

1. Open your workspace and navigate to **Settings** > **Integrations**.
2. Select the service you want to connect:
   - **GitHub** -- Authenticate via GitHub App installation or Personal Access Token.
   - **GitLab** -- Authenticate with a Personal Access Token.
   - **Jira** -- Authenticate with your Jira instance URL, email, and API token.
   - **Google Drive** -- Authenticate via OAuth to access Drive files.
3. Follow the OAuth or token flow to authorize ACP.

Once connected, agents in this workspace can read from and write to the linked service.

## Create your first session

1. From your workspace, click **New Session**.
2. In the create session dialog, configure:
   - **Display name** (optional) -- A human-readable label for the session.
   - **Model** -- Select an available LLM from the dropdown. Model availability and feature-gated access depend on your organization's configuration.
   - **Integrations** -- Review the read-only status indicators showing which integrations are connected.
3. Click **Create Session**.

The session enters the **Pending** state and begins provisioning.

## Work with the session

Once the session reaches the **Running** state, the session chat page opens. From here you can:

- **Type your prompt** in the chat input -- describe what you want the agent to do. Be specific: _"Fix the null pointer exception in the login handler and open a PR"_ works better than _"Fix bugs."_
- **Add repositories** from the sidebar to give the agent access to your code.
- **Select a workflow** from the sidebar to apply a structured task template (e.g., Bugfix, Triage).

## Monitor the session

The session moves through these states:

| Status | What is happening |
|---|---|
| **Pending** | Session is queued and waiting for resources. |
| **Creating** | The platform is provisioning the agent container. |
| **Running** | The agent is actively working on your task. |
| **Stopping** | The session is gracefully shutting down. |
| **Stopped** | The session was manually stopped before completion. |
| **Completed** | The agent has finished. Review the results. |
| **Failed** | Something went wrong. Check the session logs. |

While the session is running you can:

- **Chat** with the agent to provide clarification or redirect its work.
- **View logs** to see what the agent is doing in real time.
- **Browse Shared Artifacts** in the sidebar accordion to inspect files the agent has created or modified.

## Review results

When the session completes:

1. Open the session to see the full conversation and any final output.
2. Expand **Shared Artifacts** in the sidebar to browse files the agent produced.
3. If the agent opened a pull request, follow the link to review it in your source control provider.

## Keyboard shortcuts

Press **Cmd+K** (Mac) or **Ctrl+K** (Windows/Linux) to open the command palette.
From the command palette you can quickly navigate to any workspace, jump to settings or integrations, and start a new session.
Type in the search field to filter the available commands and workspaces.
