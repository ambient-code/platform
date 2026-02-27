---
title: "Bugfix Workflow"
---

The Bugfix workflow provides a systematic, multi-phase process for resolving software bugs. It guides the agent from initial assessment through reproduction, root cause analysis, fix implementation, testing, documentation, and pull request submission.

The workflow is orchestrated by **Amber**, who serves as a single point of contact and automatically coordinates specialized sub-agents based on the complexity of the task.

## When to use

- You have a bug report, issue URL, or symptom description and want a structured resolution process.
- You want the agent to follow engineering best practices: reproduce first, diagnose the root cause, implement a minimal fix, and verify with tests.
- You want comprehensive artifacts (reports, release notes, PR descriptions) generated alongside the fix.

## Phases and commands

The workflow defines eight sequential phases. You can start at any phase depending on how much context you already have.

### Phase 1 -- Assess (`/assess`)

Read the bug report, explain understanding of the problem, and propose a plan of attack.

**When to use**: Start here with any new bug report.

### Phase 2 -- Reproduce (`/reproduce`)

Systematically reproduce the bug and document the observable behavior.

- Parse bug reports and extract key information.
- Set up the environment matching the bug conditions.
- Attempt reproduction with variations to understand the boundaries.
- Document minimal reproduction steps.
- Create a reproduction report with severity assessment.

**Output**: `artifacts/bugfix/reports/reproduction.md`

### Phase 3 -- Diagnose (`/diagnose`)

Perform root cause analysis and assess impact across the codebase.

- Review the reproduction report and understand failure conditions.
- Analyze code paths and trace execution flow.
- Examine git history and recent changes.
- Form and test hypotheses about the root cause.
- Recommend a fix approach with confidence level.

**Output**: `artifacts/bugfix/analysis/root-cause.md`

### Phase 4 -- Fix (`/fix`)

Implement the bug fix following project conventions.

- Review the fix strategy from the diagnosis phase.
- Create a feature branch (`bugfix/issue-{number}-{description}`).
- Implement minimal code changes to fix the root cause.
- Run linters and formatters.
- Document implementation choices.

**Output**: Modified source files and `artifacts/bugfix/fixes/implementation-notes.md`

### Phase 5 -- Test (`/test`)

Verify the fix and create regression tests.

- Create a regression test that fails without the fix and passes with it.
- Write comprehensive unit tests for modified code.
- Run integration tests in realistic scenarios.
- Execute the full test suite to catch side effects.
- Perform manual verification of the original reproduction steps.

**Output**: New test files and `artifacts/bugfix/tests/verification.md`

### Phase 6 -- Review (`/review`)

Critically evaluate the fix and tests before proceeding. This phase is optional but recommended for complex or high-risk changes.

- Re-read all evidence: reproduction report, root cause analysis, code changes, and test results.
- Critique the fix: does it address the root cause or just suppress a symptom?
- Critique the tests: do they prove the bug is fixed, or do mocks hide real problems?
- Classify into a verdict and recommend next steps.

**Verdicts**: _Fix is inadequate_ (go back to `/fix`), _Tests are incomplete_ (add more tests), or _Fix and tests are solid_ (proceed to `/document`).

### Phase 7 -- Document (`/document`)

Create complete documentation for the fix.

- Update issue/ticket with root cause and fix summary.
- Create release notes entry and CHANGELOG addition.
- Update code comments with issue references.
- Draft PR description.

**Output**: `artifacts/bugfix/docs/` containing issue updates, release notes, changelog entries, and PR description.

### Phase 8 -- PR (`/pr`)

Create a pull request to submit the bug fix.

- Run pre-flight checks (authentication, remotes, git config).
- Stage changes and commit with conventional commit format.
- Push to a fork and create a draft PR targeting upstream.
- Handle common failures with clear fallbacks.

**Output**: A draft pull request URL or manual creation instructions.

## Agent orchestration

Rather than manually selecting agents, Amber assesses each phase and engages the right specialists automatically. The Bugfix workflow uses skill-based sub-agents (defined in `.claude/skills/`, not `.claude/agents/`):

- **Stella (Staff Engineer)** -- Complex debugging, root cause analysis, architectural issues.
- **Neil (Test Engineer)** -- Comprehensive test strategies, integration testing, automation.
- **Taylor (Team Member)** -- Straightforward implementations and documentation.

## Generated artifacts

All artifacts are organized under `artifacts/bugfix/`:

```
artifacts/bugfix/
  reports/
    reproduction.md
  analysis/
    root-cause.md
  fixes/
    implementation-notes.md
  tests/
    verification.md
  docs/
    issue-update.md
    release-notes.md
    changelog-entry.md
    pr-description.md
```

## Example scenarios

**You have a bug report:**

> "Fix bug #425 -- session status updates are failing."

The agent starts with `/assess` and `/reproduce`, works through diagnosis and a fix, verifies with tests, and submits a PR. The full lifecycle produces all artifacts.

**You already know the symptoms:**

> "Sessions are failing to update status in the operator."

Skip reproduction, jump to `/diagnose` for root cause analysis, then follow the remaining phases.

**You already know the root cause:**

> "Missing retry logic in UpdateStatus at operator/handlers/sessions.go:334."

Jump directly to `/fix`, then verify with `/test`, document, and submit.

## Tips

- **Reproduce first.** Flaky reproduction leads to incomplete diagnosis. Even failed reproduction attempts are valuable information.
- **Keep fixes minimal.** Only change what is necessary. Do not combine refactoring with bug fixes.
- **Test the test.** Verify that your regression test actually fails without the fix.
- **Use confidence levels.** The workflow tags actions as High (90-100%), Medium (70-89%), or Low (<70%) confidence. Low-confidence actions prompt escalation to a human.
- **Respect the target project.** The workflow adapts to the project's coding standards, test framework, and conventions rather than imposing its own.
