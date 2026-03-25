---
title: "Amber"
---

[**Amber**](https://github.com/ambient-code/amber) is the Ambient Code Platform's AI-powered codebase intelligence agent. Amber reads your attached repositories, understands your project standards, and works alongside you -- from quick consultations to autonomous background maintenance. Amber maintains awareness of session-accessible files, runs configured lint checks, and is available whenever your deployment and integrations are online.

## What Amber can do

Amber covers a wide range of software engineering tasks:

| Capability | Examples |
|-----------|---------|
| **Code analysis** | Explain how a module works, trace a call chain, identify dead code. |
| **Bug fixing** | Reproduce, diagnose, and fix bugs -- with root cause analysis and tests. |
| **Issue triage** | Categorize new issues by severity and component, link related items, suggest assignees. |
| **Sprint planning** | Cluster issues into themes, estimate effort, recommend sprint priorities. |
| **PR reviews** | Check for standards violations, security concerns, and performance regressions. |
| **Refactoring** | Break large files into modules, extract patterns, unify duplicated logic. |
| **Test generation** | Write unit tests, contract tests, and edge-case coverage for untested code. |
| **Dependency monitoring** | Scan for outdated packages, upstream breaking changes, and security advisories. |
| **Health reports** | Produce periodic assessments of test coverage, tech debt, and codebase quality. |

## How to work with Amber

There are two main ways to interact with Amber: through sessions in the platform UI and through GitHub automation.

### Interactive sessions

The most common way to work with Amber is to create a session from the platform UI.

1. Open a workspace and click **New Session**.
2. Attach the repositories Amber should work with.
3. Optionally select a **workflow** (such as Bugfix or Triage) to give Amber a structured plan.
4. Write your prompt and start the session.

Once running, Amber appears in the chat interface just like any other session. You can ask questions, request changes, and review the results in real time. Amber displays every tool call it makes -- file reads, edits, shell commands, searches -- so you always know what happened.

**Example prompts for interactive sessions:**

- "What changed in the backend this week? Anything I should be concerned about?"
- "The `/sessions` endpoint returns 500 when the token is expired. Find the bug and fix it."
- "Review the open issues and group them by component. Which ones should we tackle this sprint?"
- "Refactor `handlers/sessions.go` -- it is too large. Break it into smaller, focused modules."

### GitHub automation

Amber can also work autonomously via GitHub. GitHub events such as issue creation or webhook integrations can trigger Amber to analyze issues, implement fixes, and open pull requests without you needing to open the platform UI.

The specific trigger methods depend on your deployment configuration. Common approaches include GitHub Actions workflows that start Amber on issue events, webhook integrations, and scheduled jobs.

Amber creates a feature branch, runs linters and tests, and opens a PR linked back to the original issue. By default, Amber does not merge its own PRs -- a human must review and merge.

## Workflows that pair well with Amber

Amber works with any prompt, but it is especially effective when paired with a workflow. Workflows provide a structured plan to follow, which improves consistency and output quality.

| Workflow | When to use it with Amber |
|----------|--------------------------|
| **Bugfix** | Systematic five-phase bug resolution: reproduce, diagnose, fix, test, verify. |
| **Triage** | Process an issue backlog and produce a prioritized, categorized breakdown. |
| **Spec-kit** | Generate a detailed specification from requirements, then implement to spec. |
| **Amber Interview** | Gather context and requirements through a structured question-and-answer format. |

See [Workflows](../concepts/workflows) for the full list and details on creating custom workflows.

## Amber's capabilities

Under the hood, Amber has access to the same toolset as any session on the platform, plus domain knowledge that makes it particularly effective at codebase work.

### Repository access

Amber reads and writes files across every repository attached to the session. It parses directory structures, import graphs, and language-specific patterns. When you attach multiple repositories, Amber can reason across all of them -- for instance, tracing how a backend API change affects the frontend.

### Tool use

During a session, Amber uses tools to do real work:

- **File operations** -- read, write, and edit files with surgical precision.
- **Shell commands** -- run builds, tests, linters, and Git operations.
- **Search** -- find code patterns, function definitions, and references across the codebase.
- **Web search and fetch** -- look up documentation, upstream changelogs, and API references.
- **MCP integrations** -- interact with GitHub (PRs, issues, commits), Jira, and other connected services through the workspace's configured integrations.

### Standards awareness

Amber reads your project's `CLAUDE.md` and any workflow instructions to apply your team's conventions. It follows your coding standards, commit message format, testing requirements, and tooling preferences. If your project says "use `gofmt`" or "zero `any` types in TypeScript," Amber enforces that in its own output.

### Confidence and transparency

Amber reports what it is doing and why. It shows its reasoning, flags uncertainty, and prompts for confirmation before making risky changes. When Amber opens a pull request, it includes:

- What changed and why.
- A confidence level (high, medium, or low).
- Rollback instructions in case something goes wrong.
- A risk assessment of the change's blast radius.

## Tips for getting the best results

**Be specific.** A prompt like "fix the bug in the login endpoint where expired tokens return 500 instead of 401" will get a targeted fix. A prompt like "make the code better" will get a generic response.

**Point to files.** If you know which files are involved, mention them. Amber focuses its analysis instead of scanning the entire codebase.

**Provide success criteria.** Tell Amber what "done" looks like: "all tests pass," "coverage above 60%," "no lint warnings." Amber uses these criteria to validate its own output.

**Start small.** If you are new to Amber, begin with a low-risk task -- a formatting fix, a missing test, or a code review. As you build confidence in the output, move to larger refactors and autonomous workflows.

**Use workflows for repeatable tasks.** If you run the same kind of task regularly (bug triage, sprint planning, test coverage), select a workflow. Workflows provide a proven process and produce more consistent output.

**Review before merging.** Amber does not merge its own pull requests by default. Always review the diff, check the test results, and verify the changes make sense in context.

**Give feedback.** If Amber's output missed the mark, provide corrections. Amber adjusts its approach based on your input within the session. Over time, your team's standards in `CLAUDE.md` and custom workflows encode your preferences so every session starts from a better baseline.
