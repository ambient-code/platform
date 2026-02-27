---
title: "What is Ambient?"
---

The **Ambient Code Platform (ACP)** is a Kubernetes-native platform that lets development teams run AI-powered agents for real engineering work -- bug fixes, code analysis, sprint planning, issue triage, and more.

You define a task in natural language. ACP spins up a containerized agent session, connects it to your repositories and tools, and delivers results you can review, iterate on, and ship.

## What You Can Do

- **Fix bugs and write code** -- Point an agent at a repo, describe the problem, and get a working pull request.
- **Triage issues** -- Automatically classify, prioritize, and route incoming issues across your backlog.
- **Plan sprints** -- Generate sprint plans from your issue tracker with dependency analysis and effort estimates.
- **Analyze codebases** -- Get architectural reviews, security audits, or dependency assessments on demand.
- **Run structured workflows** -- Use built-in templates (Bugfix, Triage, Spec-kit, PRD/RFE) or bring your own from any git repo.

## How It Works

```
You describe a task  -->  ACP creates an agent session  -->  Agent works in a container  -->  You review the results
```

Sessions run as Kubernetes Jobs with full isolation. Each session gets its own workspace with cloned repositories, configured integrations, and access to external tools via MCP (Model Context Protocol).

## Key Capabilities

**Workspaces** -- Multi-tenant project containers where teams organize sessions, configure integrations, and manage permissions. Learn more in [Workspaces](../concepts/workspaces/).

**Sessions** -- AI agent execution environments with a chat interface, configurable model settings, timeout controls, and artifact output. Learn more in [Sessions](../concepts/sessions/).

**Integrations** -- Connect GitHub (App or PAT), GitLab (OAuth), Jira (OAuth), and Google Workspace (OAuth) so agents can read and write to your existing tools. Learn more in [Integrations](../concepts/integrations/).

**Workflows** -- Structured task templates that guide agent behavior. Use the built-in templates or point to a custom workflow in any git repo. Learn more in [Workflows](../concepts/workflows/).

**Multi-repo support** -- Clone multiple repositories into a single session for cross-repo analysis and changes.

## Getting Started

- **[Quickstart (UI)](quickstart-ui/)** -- Create your first session through the web interface.
- **[Core Concepts](concepts/)** -- Understand the building blocks before diving in.
