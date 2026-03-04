---
title: "Core Concepts"
---

This page introduces the main building blocks of the Ambient Code Platform. Each concept links to a deeper reference page where available.

## Workspaces

A workspace is a project container that groups sessions, integrations, and team members together. Workspaces map to Kubernetes namespaces under the hood, providing resource isolation between teams.

Each workspace has:

- **Members and roles** -- Control who can create sessions, manage integrations, or administer the workspace.
- **Settings** -- Default model configuration and API keys.

Learn more in [Workspaces](../concepts/workspaces/).

## Sessions

A session is a single AI agent execution. When you create a session, ACP provisions a containerized environment, clones any requested repositories, and runs the agent with your prompt.

Sessions have a defined lifecycle:

**Pending** &rarr; **Creating** &rarr; **Running** &rarr; **Stopping** &rarr; **Stopped** / **Completed** / **Failed**

While running, you can interact with the agent through a chat interface, observe its progress in real time, and browse output artifacts. Sessions are configurable:

- **Model** -- Which LLM powers the agent (Claude Sonnet 4.5, Claude Opus 4.6, Claude Opus 4.5, or Claude Haiku 4.5).
- **Temperature** -- Controls response randomness (default: 0.7).
- **Max tokens** -- Maximum output tokens per response (default: 4000).
- **Timeout** -- Maximum execution time in seconds before the session is stopped (default: 300).
- **Repositories** -- One or more git repos cloned into the session workspace.
- **Workflow** -- An optional structured template guiding the agent's approach.

Learn more in [Sessions](../concepts/sessions/).

## Integrations

Integrations connect ACP to external services so agents can read from and write to the tools your team already uses. Integrations are configured globally and available across all workspaces.

| Integration | Auth Method | What Agents Can Do |
|---|---|---|
| **GitHub** | GitHub App or PAT | Clone repos, open PRs, read/comment on issues |
| **GitLab** | OAuth | Clone repos, open merge requests, interact with issues |
| **Jira** | OAuth | Read and update tickets, add comments, transition status |
| **Google Workspace** | OAuth | Access Drive documents, Calendar events, and Gmail for context |

Once connected, every session can use them.

Learn more in [Integrations](../concepts/integrations/).

## Workflows

Workflows are structured task templates that guide how an agent approaches a problem. They provide consistent, repeatable processes for common tasks.

**Built-in workflows:**

- **Bugfix** -- Diagnose and fix a reported bug, including tests.
- **Triage** -- Classify, prioritize, and route an issue.
- **Spec-kit** -- Generate a technical specification from requirements.
- **PRD/RFE** -- Produce a product requirements document or request for enhancement.

**Custom workflows** can be loaded from any git repository, letting teams codify their own processes and share them across the organization.

Learn more in [Workflows](../concepts/workflows/).

## Context and Artifacts

**Context** is the input an agent works with: cloned repositories, linked documents, integration data, and your prompt.

**Artifacts** are the outputs a session produces: modified files, generated documents, pull requests, and any other files the agent creates during execution. You can browse and download artifacts from the session detail page.

Learn more in [Context and Artifacts](../concepts/context-and-artifacts/).

## MCP Tools

The Model Context Protocol (MCP) lets agents call external tools during a session. MCP tools extend what an agent can do beyond code and text -- for example, querying a database, calling an internal API, or running a specialized analysis tool.

MCP tools are configured at the workspace level and made available to all sessions within that workspace.

## Next Steps

- **[Quickstart (UI)](quickstart-ui/)** -- Create your first session.
- **[GitHub Action](../extensions/github-action/)** -- Automate sessions from CI/CD.
