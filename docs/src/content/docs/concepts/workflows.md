---
title: "Workflows"
---

A **workflow** is a structured task template that gives the AI agent a proven plan for a specific type of work. Instead of writing a long prompt every time, you select a workflow and the agent follows a defined process -- complete with system instructions, phases, and quality rubrics.

## How workflows work

When you attach a workflow to a session, the platform loads a set of instructions from a Git repository and injects them into the agent's context. The agent then follows the workflow's process rather than relying solely on your ad-hoc prompt.

Workflows typically include:

- A **system prompt** that sets the agent's role and approach.
- **Slash commands and skills** -- these are Claude Code features defined in the workflow's `.claude/` directory that trigger specific phases or actions.
- A **rubric** the agent uses to self-evaluate its output quality.

## Out-of-the-box workflows

The platform ships with several ready-to-use workflows:

| Workflow | What it does |
|----------|-------------|
| [**Bugfix**](/workflows/bugfix/) | Systematic multi-phase bug resolution: assess, reproduce, diagnose, fix, test, review, document, and submit a PR. |
| [**Triage**](/workflows/triage/) | Analyzes an issue backlog, categorizes items by severity and effort, and produces actionable reports with bulk operations. |
| [**Spec-kit**](/workflows/spec-kit/) | Specification-driven development: generates a detailed spec from requirements, plans implementation, breaks into tasks, and implements. |
| [**PRD / RFE**](/workflows/prd-rfe/) | Creates Product Requirements Documents and breaks them into actionable Request for Enhancement items with prioritization. |
| **Amber Interview** | Guided interview format for collecting user feedback through structured Q&A. (No detail page.) |
| **Template** | A minimal starting point for building your own custom workflow. (No detail page.) |

See the [Workflows section](/workflows/) for detailed documentation on the linked workflows above, including commands, phases, generated artifacts, and tips.

### Using a workflow in a session

1. Create a session and open it.
2. In the session sidebar, open the **Workflow** dropdown.
3. Select one of the out-of-the-box workflows.
4. Provide your prompt as usual -- the workflow adds structure around it.
5. Use the workflow's slash commands from the chat to trigger specific phases.

You can switch workflows on a running session at any time from the session sidebar.

## Custom workflows

If the built-in workflows do not fit your process, you can create your own from any Git repository. See [Custom Workflows](/workflows/custom/) for the full guide, including directory structure, `ambient.json` configuration, and development workflow.

### Quick overview

A custom workflow lives in a Git repository that the platform can access. The only required file is `.ambient/ambient.json`:

```json
{
  "name": "My Custom Workflow",
  "description": "A workflow that does X, Y, and Z",
  "systemPrompt": "You are a helpful assistant for...",
  "startupPrompt": "Welcome! Use /start to begin."
}
```

To load a custom workflow, select **Custom Workflow...** from the workflow dropdown in the session sidebar and enter the Git URL, branch, and path.

For detailed workflow internals, advanced configuration, and the full `ambient.json` schema, see [Custom Workflows](/workflows/custom/) and the [workflows repository](https://github.com/ambient-code/workflows).
