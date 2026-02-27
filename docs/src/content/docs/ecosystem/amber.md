---
title: "Amber"
---

[**Amber**](https://github.com/ambient-code/amber) is the Ambient Code Platform's AI-powered colleague. She is a codebase intelligence agent that reads your repositories, understands your project standards, and works alongside you -- from quick consultations to autonomous background maintenance. Think of Amber as a senior engineer who never forgets a file, never skips a lint check, and is always available.

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

Once running, Amber appears in the chat interface just like any other session. You can ask questions, request changes, and review her work in real time. She shows you every tool call she makes -- file reads, edits, shell commands, searches -- so you always know what happened.

**Example prompts for interactive sessions:**

- "What changed in the backend this week? Anything I should be concerned about?"
- "The `/sessions` endpoint returns 500 when the token is expired. Find the bug and fix it."
- "Review the open issues and group them by component. Which ones should we tackle this sprint?"
- "Refactor `handlers/sessions.go` -- it is too large. Break it into smaller, focused modules."

### GitHub automation

Amber can also work autonomously via GitHub. She can be triggered by GitHub events such as issue creation or webhook integrations, allowing her to analyze issues, implement fixes, and open pull requests without you needing to open the platform UI.

The specific trigger methods depend on your deployment configuration. Common approaches include GitHub Actions workflows that invoke Amber on issue events, webhook integrations, and scheduled jobs.

Amber creates a feature branch, runs linters and tests, and opens a PR linked back to the original issue. By default she does not merge her own PRs, though this can be configured via the `AUTO_MERGE_ENABLED` setting.

## Workflows that pair well with Amber

Amber works with any prompt, but she is especially effective when paired with a workflow. Workflows give her a structured plan to follow, which improves consistency and output quality.

| Workflow | When to use it with Amber |
|----------|--------------------------|
| **Bugfix** | Systematic five-phase bug resolution: reproduce, diagnose, fix, test, verify. |
| **Triage** | Process an issue backlog and produce a prioritized, categorized breakdown. |
| **Spec-kit** | Generate a detailed specification from requirements, then implement to spec. |
| **Amber Interview** | Gather context and requirements through a structured question-and-answer format. |

See [Workflows](/concepts/workflows) for the full list and details on creating custom workflows.

## Amber's capabilities

Under the hood, Amber has access to the same toolset as any session on the platform, plus domain knowledge that makes her particularly effective at codebase work.

### Repository access

Amber reads and writes files across every repository attached to the session. She understands directory structures, import graphs, and language-specific patterns. When you attach multiple repositories, she can reason across all of them -- for instance, tracing how a backend API change affects the frontend.

### Tool use

During a session, Amber uses tools to do real work:

- **File operations** -- read, write, and edit files with surgical precision.
- **Shell commands** -- run builds, tests, linters, and Git operations.
- **Search** -- find code patterns, function definitions, and references across the codebase.
- **Web search and fetch** -- look up documentation, upstream changelogs, and API references.
- **MCP integrations** -- interact with GitHub (PRs, issues, commits), Jira, and other connected services through the workspace's configured integrations.

### Standards awareness

Amber reads your project's `CLAUDE.md` and any workflow instructions to understand your team's conventions. She follows your coding standards, commit message format, testing requirements, and tooling preferences. If your project says "use `gofmt`" or "zero `any` types in TypeScript," Amber enforces that in her own work.

### Confidence and transparency

Amber tells you what she is doing and why. She shows her reasoning, flags uncertainty, and asks before making risky changes. When she opens a pull request, it includes:

- What she changed and why.
- A confidence level (high, medium, or low).
- Rollback instructions in case something goes wrong.
- A risk assessment of the change's blast radius.

## Tips for getting the best results

**Be specific.** A prompt like "fix the bug in the login endpoint where expired tokens return 500 instead of 401" will get a targeted fix. A prompt like "make the code better" will get a generic response.

**Point to files.** If you know which files are involved, mention them. Amber will focus her analysis instead of scanning the entire codebase.

**Provide success criteria.** Tell Amber what "done" looks like: "all tests pass," "coverage above 60%," "no lint warnings." She uses these criteria to evaluate her own work.

**Start small.** If you are new to Amber, begin with a low-risk task -- a formatting fix, a missing test, or a code review. As you build confidence in her output, move to larger refactors and autonomous workflows.

**Use workflows for repeatable tasks.** If you run the same kind of task regularly (bug triage, sprint planning, test coverage), select a workflow. It gives Amber a proven process to follow and produces more consistent output.

**Review before merging.** Amber never merges her own pull requests by default. Always review the diff, check the test results, and verify the changes make sense in context.

**Give feedback.** If Amber's output missed the mark, tell her. She adjusts her approach based on your input within the session. Over time, your team's standards in `CLAUDE.md` and custom workflows encode your preferences so every session starts from a better baseline.
