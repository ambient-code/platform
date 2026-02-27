---
title: "Quickstart"
---

This guide walks you through creating your first AI agent session using the Ambient Code Platform web interface.

## Sign In

ACP uses SSO for authentication -- there is no separate account to create. Navigate to your organization's ACP URL and sign in with your existing credentials.

After signing in, you land on the **Workspaces** page.

## Create a Workspace

Workspaces are project containers that hold your sessions, integrations, and team permissions.

1. Click **Create Workspace**.
2. Enter a **name** (e.g., `my-team`) and optional **description**.
3. Click **Create**.

You are now inside your workspace. From here you can configure integrations and start sessions.

## Configure Integrations (Optional)

Integrations let agents interact with your external services. You can skip this step and add integrations later.

To connect a service:

1. Open your workspace and navigate to **Settings** > **Integrations**.
2. Select the service you want to connect:
   - **GitHub** -- Authenticate via GitHub App installation or Personal Access Token.
   - **GitLab** -- Authenticate via OAuth.
   - **Jira** -- Authenticate via OAuth.
   - **Google Workspace** -- Authenticate via OAuth (covers Drive, Calendar, and Gmail).
3. Follow the OAuth or token flow to authorize ACP.

Once connected, agents in this workspace can read from and write to the linked service.

## Create Your First Session

1. From your workspace, click **New Session**.
2. In the create session dialog, configure:
   - **Display name** (optional) -- A human-readable label for the session.
   - **Model** -- Select the LLM to use from the dropdown (Claude Sonnet 4.5, Claude Opus 4.6, Claude Opus 4.5, or Claude Haiku 4.5).
   - **Integrations** -- Review the read-only status indicators showing which integrations are connected.
3. Click **Create Session**.

The session enters the **Pending** state and begins provisioning.

## Work with the Session

Once the session reaches the **Running** state, the session chat page opens. From here you can:

- **Type your prompt** in the chat input -- describe what you want the agent to do. Be specific: _"Fix the null pointer exception in the login handler and open a PR"_ works better than _"Fix bugs."_
- **Add repositories** from the sidebar to give the agent access to your code.
- **Select a workflow** from the sidebar to apply a structured task template (e.g., Bugfix, Triage).

## Monitor the Session

The session moves through these states:

| Status | What is happening |
|---|---|
| **Pending** | Session is queued and waiting for resources. |
| **Creating** | Kubernetes is provisioning the agent container. |
| **Running** | The agent is actively working on your task. |
| **Stopping** | The session is gracefully shutting down. |
| **Stopped** | The session was manually stopped before completion. |
| **Completed** | The agent has finished. Review the results. |
| **Failed** | Something went wrong. Check the session logs. |

While the session is running you can:

- **Chat** with the agent to provide clarification or redirect its work.
- **View logs** to see what the agent is doing in real time.
- **Browse Shared Artifacts** in the sidebar accordion to inspect files the agent has created or modified.

## Review Results

When the session completes:

1. Open the session to see the full conversation and any final output.
2. Expand **Shared Artifacts** in the sidebar to browse files the agent produced.
3. If the agent opened a pull request, follow the link to review it in your source control provider.

## Next Steps

- **[Core Concepts](/getting-started/concepts/)** -- Understand workspaces, sessions, and workflows in depth.
- **[GitHub Action](/extensions/github-action/)** -- Trigger sessions from CI/CD pipelines.
