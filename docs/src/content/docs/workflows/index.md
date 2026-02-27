---
title: "Workflows"
---

Workflows are structured task templates that give an AI agent a proven, repeatable process for a specific type of work. Instead of writing a detailed prompt each time you create a session, you select a workflow and the agent follows a defined methodology -- complete with phases, slash commands, specialized sub-agents, and quality rubrics.

## How workflows work

When you attach a workflow to a session, the platform clones a workflow definition from a Git repository and injects its instructions into the agent's context. The workflow defines:

- A **system prompt** that sets the agent's role, methodology, and behavioral guardrails.
- **Slash commands** that trigger specific phases (e.g., `/diagnose`, `/fix`, `/test`).
- **Skills** that provide reusable knowledge and can spawn sub-agents for specialized work.
- **Artifact paths** that tell the platform where to find generated outputs.
- An optional **rubric** the agent uses to self-evaluate output quality.

Workflows are discovered automatically. Any directory in the [workflows repository](https://github.com/ambient-code/workflows) that contains a valid `.ambient/ambient.json` file appears in the session creation UI.

## Out-of-the-box workflows

The platform ships with these ready-to-use workflows:

| Workflow | Description | Commands |
|----------|-------------|----------|
| [**Bugfix**](/workflows/bugfix/) | Systematic bug resolution with reproduction, diagnosis, fix, testing, and PR submission | `/assess`, `/reproduce`, `/diagnose`, `/fix`, `/test`, `/review`, `/document`, `/pr` |
| [**Triage**](/workflows/triage/) | Issue backlog analysis with actionable reports and bulk operations | Conversational |
| [**Spec-kit**](/workflows/spec-kit/) | Specification-driven development for feature planning and implementation | Key commands include `/speckit.specify`, `/speckit.plan`, `/speckit.tasks`, `/speckit.implement` |
| [**PRD / RFE**](/workflows/prd-rfe/) | Product requirements documentation and RFE task breakdown | Key commands include `/prd.discover`, `/prd.create`, `/rfe.breakdown`, `/rfe.prioritize` |

The platform also includes the **Amber Interview** workflow for collecting user feedback and a **Claude.md Generator** workflow. See the [workflows repository](https://github.com/ambient-code/workflows) for the full list.

## Using a workflow

1. Click **New Session** in your workspace.
2. In the creation dialog, open the **Workflow** dropdown.
3. Select one of the available workflows.
4. Provide your prompt as usual -- the workflow adds structure around it.
5. Once the session is running, use the workflow's slash commands from the chat to trigger specific phases.

You can also switch workflows on a running session from the session sidebar.

## Custom workflows

If the built-in workflows do not cover your process, you can bring your own workflow from any Git repository. See the [Custom Workflows](/workflows/custom/) guide for details on creating and loading custom workflows.
