---
title: "Bugfix Workflow"
---

# Bug Fix Workflow Tutorial

This tutorial walks you through using the Bug Fix workflow on the Ambient
Code Platform to diagnose and fix software bugs. By the end you will know how
to start a session, move through each phase of the workflow, and submit a pull
request with the fix.

## Getting Started

### Log in and create a session

Open [red.ht/ambient](https://red.ht/ambient) and start a new session. Select
the **Bug Fix** workflow and the model you want to use.

### Choosing a model

At the time of writing, **Claude Opus** is the model we find works best for
fixing bugs. If your bug is straightforward (a clear error message pointing to
an obvious cause), a less powerful model may be fine and can be quicker and
cheaper.

Keep in mind, though, that if the bug turns out to be more complex than you
expect, a less powerful model can end up being *slower* and *more expensive*. A
bug that a stronger model solves in one pass may take several rounds of
back-and-forth with a weaker model, consuming more time and tokens overall. When
in doubt, start with a capable model.

### Adding repository context

You can add the GitHub repository you want to work on by clicking the
**Add Context** tab in the Explorer panel before you begin. However, this is
not strictly necessary. If you give the workflow a link to the repository, or
to an issue in that repository, it will clone the repo into your session
automatically. Adding it in the Explorer panel can still be helpful for
simplifying the model's task by ensuring it has the repo you want available
before starting.

## Starting the Workflow

There are several ways to kick things off once your session is running.

### Option 1: Paste a bug issue link

Simply paste a link to a GitHub issue and the workflow will clone the repository
and begin walking you through the bug fix process interactively. For example:

```text
https://github.com/llamastack/llama-stack/issues/5119
```

The workflow will assess the issue, summarize its understanding, and ask for
your input before moving on to the next step.  Alternatively, you could start with:

```text
/assess https://github.com/llamastack/llama-stack/issues/5119
```

That makes it more explicit that you want to start at the beginning with the assessment.

### Option 2: Speedrun a bug issue

If you want the workflow to run through every phase without stopping for
input between steps, use the `/speedrun` command followed by the issue link:

```text
/speedrun https://github.com/llamastack/llama-stack/issues/5119
```

See [Speedrun mode](#speedrun-mode) below for details on what this does and
when to use it.

### Option 3: Describe the bug yourself

You do not need a GitHub issue at all. You can describe the bug in your own
words and provide a link to the repository:

```text
The Milvus vector store integration rejects collection names that contain
hyphens. Repo: https://github.com/llamastack/llama-stack
```

The workflow will clone the repo and proceed from there. This works with or
without `/speedrun`. Prefix your description with `/speedrun` to run through
all phases automatically, or leave it off to step through interactively.

## Workflow Phases

The Bug Fix workflow is organized into a series of phases. The in-product
greeting summarizes six core phases (Assess through Document), but in practice
the full workflow includes eight: the six core phases plus Review (an optional
quality check) and PR (to submit the fix). All eight are described below.

In the default interactive mode the workflow completes one phase at a time,
presents you with results and recommendations, and then waits for you to decide
what to do next. You can follow the suggested order, skip ahead, or go back to
revisit an earlier phase at any point.

Most phases produce a **report**, a Markdown file you can view in the
**Files** section of the Explorer panel. These reports are internal to
your session (they are not published to GitHub or anywhere else) and serve
two purposes:

1. **Visibility for you.** The reports give you a deeper view of what the
   workflow found and decided at each step, so you can provide more informed
   input and steer the process effectively.

2. **Memory for the workflow.** The platform can only keep a finite amount of
   context in memory at a time. By the time the workflow reaches the Test or
   Review phase, for example, the details of the original assessment may no
   longer be in its working context. The written reports let later phases refer
   back to earlier analysis without losing important details.

### Assess

**Command:** `/assess`

The first phase reads the bug report (or your description) and presents a
summary of its understanding: what the bug is, where it likely occurs, what
information is available, what is missing, and a proposed plan for
reproducing it. No code is changed or executed in this phase; it is purely
analysis and planning.

This is your chance to correct any misunderstandings or fill in missing
details before work begins.

**Report:** `artifacts/bugfix/reports/assessment.md`

### Reproduce

**Command:** `/reproduce`

This phase attempts to reproduce the bug in a controlled environment. It
follows the plan from the Assess phase, documents the steps taken, and records
what happened. Even a failed reproduction attempt is documented, since
understanding *why* a bug is hard to reproduce is useful information for
diagnosis.

**Report:** `artifacts/bugfix/reports/reproduction.md`

### Diagnose

**Command:** `/diagnose`

Root cause analysis. The workflow traces through the code, examines git
history, forms hypotheses about what is going wrong, and tests them. The goal
is to understand *why* the bug happens, not just *what* happens. The diagnosis
also assesses how much of the codebase is affected and recommends an approach
for the fix.

**Report:** `artifacts/bugfix/analysis/root-cause.md`

### Fix

**Command:** `/fix`

Implements the bug fix. The workflow creates a feature branch, makes the
minimal code changes needed to resolve the root cause, runs linters and
formatters, and documents the implementation choices it made.

**Report:** `artifacts/bugfix/fixes/implementation-notes.md`

### Test

**Command:** `/test`

Verifies the fix. The workflow creates regression tests that fail without the
fix and pass with it, runs the project's full test suite to check for
unintended side effects, and performs manual verification of the original
reproduction steps.

**Report:** `artifacts/bugfix/tests/verification.md`

### Review

**Command:** `/review`

An optional but recommended self-review phase. The workflow steps back and
critically evaluates the fix and tests: Does the fix address the root cause or
just suppress a symptom? Do the tests actually prove the bug is resolved, or
could they be hiding problems? The review produces a verdict:

- **Fix and tests are solid.** Ready to proceed to documentation and PR.
- **Fix is adequate but tests are incomplete.** Additional testing is
  recommended before proceeding.
- **Fix is inadequate.** The fix needs more work. The workflow will suggest
  going back to the Fix phase with specific guidance.

**Report:** `artifacts/bugfix/review/verdict.md`

### Document

**Command:** `/document`

Creates the supporting documentation for the fix: release notes, a changelog
entry, an issue update, and a draft pull request description.

**Reports:** `artifacts/bugfix/docs/` (multiple files including
`release-notes.md`, `changelog-entry.md`, and `pr-description.md`)

### PR

**Command:** `/pr`

The final phase. The workflow pushes the feature branch to a fork and creates a
draft pull request against the upstream repository. If automated PR creation
fails for any reason (e.g., missing permissions), it provides instructions for
creating the PR manually.

## Speedrun Mode

**Command:** `/speedrun`

Speedrun mode runs through all remaining phases in sequence without pausing for
your input between steps. It is useful when you have a well-defined bug and
trust the workflow to handle the full lifecycle autonomously.

Even in speedrun mode, the workflow will stop and ask you for guidance if it
hits a situation that needs human judgment, such as when the root cause is
unclear, if there are multiple valid fix approaches with different trade-offs,
or if a security concern arises. It will also stop if it hits an obstacle it
cannot get past on its own, for example if you gave it a link to a repository
that does not exist.

You can also interrupt a speedrun at any time by clicking the **stop** button
in the chat UI. Once stopped, you can interact with the session freely: review
what has been done so far, request changes, provide additional context, or tell
it to continue the speedrun from where it left off.

### Why speedrun is not the default

The default interactive mode pauses after each phase so you can review the
results and steer the process. For many bugs, especially complex ones or bugs
where the report is incomplete or ambiguous, this deliberate pacing is
valuable. Reviewing the assessment before reproduction begins, or checking the
diagnosis before a fix is attempted, lets you catch mistakes early and provide
context that the workflow might not have.

Speedrun is best suited for well-understood bugs where you are confident the
report contains enough information for the workflow to proceed on its own.

### Speedrun examples

Run from the beginning with a bug issue link:

```text
/speedrun https://github.com/llamastack/llama-stack/issues/5119
```

Continue from wherever you are in the flow (the workflow detects which phases are already complete):

```text
/speedrun
```

Jump ahead to a specific phase:

```text
/speedrun Jump ahead to /fix
```

## Viewing Reports and Artifacts

As the workflow progresses, reports and other artifacts are written to the
`artifacts/bugfix/` directory. You can view these files at any time in the
**Files** section of the Explorer panel. The reports are Markdown files that
provide a detailed record of each phase's analysis, decisions, and outcomes.

## Automation (for advanced users)

Everything above describes using the Bug Fix workflow through the interactive
UI. The Ambient Code Platform also supports triggering sessions
programmatically, so you can incorporate bug fixing into CI/CD pipelines,
scheduled jobs, or custom tooling.

### How it works

Under the hood, creating a session is a single HTTP POST to the platform's
backend API. The request body includes the prompt, the repositories to clone,
and an `activeWorkflow` object that tells the platform which workflow to load.
Anything that can make an authenticated HTTP request can create a bug fix
session.

Here is the equivalent `curl` call:

```bash
curl -X POST \
  "${ACP_API_URL}/projects/${ACP_PROJECT}/agentic-sessions" \
  -H "Authorization: Bearer ${ACP_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "initialPrompt": "/speedrun https://github.com/llamastack/llama-stack/issues/5119",
    "activeWorkflow": {
      "gitUrl": "https://github.com/ambient-code/workflows",
      "branch": "main",
      "path": "workflows/bugfix"
    },
    "repos": [
      {"url": "https://github.com/llamastack/llama-stack", "branch": "main", "autoPush": true}
    ],
    "llmSettings": {"model": "claude-sonnet-4-5"},
    "timeout": 1800
  }'
```

The key fields are:

- **`initialPrompt`** is what the agent sees as its first instruction. This is
  where you pass the bug description, issue link, or `/speedrun` command.
- **`activeWorkflow`** tells the platform which workflow to load. For the Bug
  Fix workflow, point it at this repository with the path `workflows/bugfix`.
  If you omit this field, the session starts with no workflow.
- **`repos`** is an array of repositories to clone into the session. Set
  `autoPush` to `true` if you want the agent to push its fix automatically.
- **`llmSettings.model`** selects the model.
- **`timeout`** is the session timeout in seconds.

Authentication uses a bearer token in the `Authorization` header.

### GitHub Action

The [`ambient-action`](https://github.com/ambient-code/ambient-action) GitHub
Action wraps this API call for use in GitHub workflows. You can use it to
automatically kick off a bug fix session when a new issue is opened, when a
label is applied, or on any other GitHub event.

The action supports two modes:

- **Fire-and-forget.** Create the session and let the GitHub workflow continue.
  The agent runs in the background.
- **Wait-for-completion.** Block the GitHub workflow until the session finishes
  (or times out), then use the session result in subsequent steps.

Here is an example that triggers a bug fix speedrun whenever an issue is
labeled `auto-fix`:

```yaml
name: Auto Bug Fix
on:
  issues:
    types: [labeled]

jobs:
  fix:
    if: github.event.label.name == 'auto-fix'
    runs-on: ubuntu-latest
    steps:
      - uses: ambient-code/ambient-action@v0.0.2
        with:
          api-url: ${{ secrets.ACP_URL }}
          api-token: ${{ secrets.ACP_TOKEN }}
          project: my-team
          prompt: |
            /speedrun https://github.com/${{ github.repository }}/issues/${{ github.event.issue.number }}
          repos: |
            [{"url": "https://github.com/${{ github.repository }}", "branch": "main", "autoPush": true}]
          workflow: |
            {"gitUrl": "https://github.com/ambient-code/workflows", "branch": "main", "path": "workflows/bugfix"}
          wait: true
          timeout: 30
```

When `wait` is `true`, the action outputs `session-name`, `session-uid`,
`session-phase`, and `session-result`, which you can use in subsequent workflow
steps. See the
[action repository](https://github.com/ambient-code/ambient-action) for the
full list of inputs and outputs.

### Tools that do NOT yet support workflows

Several other tools can create and manage ACP sessions. At the time of writing,
none of them support setting `activeWorkflow`, which means they cannot load the
structured Bug Fix workflow described in this tutorial. They will create a
session, but it will not have the phases, commands, or reports that the workflow
provides.

- **[`acpctl`](https://github.com/ambient-code/platform/tree/main/components/ambient-cli)**
  is a command-line tool for the platform. It can create sessions and check
  their status, but does not have a flag for specifying a workflow.
- **[`mcp-acp`](https://github.com/ambient-code/mcp)** is an MCP server that
  lets you manage ACP sessions from Claude Desktop, Claude Code, or any
  MCP-compatible client. It has a "bugfix" template, but that template only
  sets a generic prompt and model. It does not load the Bug Fix workflow.
- **[Ambient Platform SDKs](https://github.com/ambient-code/platform/tree/main/components/ambient-sdk)**
  provide client libraries in Go, Python, and TypeScript for programmatic
  session management.

If you want to automate bug fix sessions from a script, CI pipeline, or any
other environment, you do not need any of these tools. The platform's backend
API accepts a straightforward JSON POST request (as shown in the `curl` example
above). The `activeWorkflow` field in that request is what tells the platform to
load the Bug Fix workflow into the session. Any HTTP client in any language can
make that call. The GitHub Action is just a convenience wrapper that does the
same thing and adds polling for completion.
