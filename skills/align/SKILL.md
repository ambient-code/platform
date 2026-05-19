---
name: align
description: >
  Run a convention alignment check across the codebase to measure adherence to
  documented standards. Use when you want to check health of the codebase,
  verify convention compliance, get a scored report, find violations, check
  alignment before a release, or run periodic quality scans. Triggers on:
  "check conventions", "alignment scan", "codebase health", "are we following
  our standards", "convention violations", "quality check", "how aligned are we".
---

# Convention Alignment Check

Measure codebase adherence to documented conventions. Produces a scored report across 5 categories with ~34 checks.

## Usage

```text
/align                # Full codebase scan
/align backend        # Backend checks only
/align frontend       # Frontend checks only
/align operator       # Operator checks only
/align runner         # Runner checks only
/align security       # Security checks only
```

## User Input

```text
$ARGUMENTS
```

## How It Works

1. **Parse scope** from `$ARGUMENTS` (default: full scan)
2. **Dispatch** the `convention-eval` agent with the scope
   - The agent runs in its own context window
   - It loads component-level convention docs
   - It runs all checks via grep/glob/bash
   - It produces a scored report
3. **Display** the report to the user

## Dispatching the Agent

Use the Agent tool to dispatch the convention-eval agent:

```javascript
Agent({
  description: "Convention alignment check",
  prompt: "You are the convention-eval agent. Read your definition at .claude/agents/convention-eval.md, then run a ${scope} convention alignment check. Scope: ${scope} (one of: full, backend, frontend, operator, runner, security). Load all context files listed in your definition, run the checks, and report findings in the standard output format with scores."
})
```

If no scope is specified, run all categories.

## Report Format

The agent produces a markdown report with:
- Overall weighted score (0-100%)
- Per-category scores (Backend, Frontend, Operator, Runner, Security)
- Pass/fail per check with file:line references
- Failures grouped by severity (Blocker > Critical > Major > Minor)
- Top 3 recommendations for improvement

## Categories and Weights

| Category | Checks | Weight | Key concerns |
|----------|--------|--------|-------------|
| Backend | 8 | 25% | panic, service account misuse, error handling |
| Frontend | 8 | 25% | any types, raw HTML, manual fetch |
| Operator | 7 | 20% | OwnerReferences, SecurityContext, reconciliation |
| Runner | 4 | 10% | async patterns, credential handling |
| Security | 7 | 20% | RBAC, token redaction, input validation |

## Interpreting Results

- **90-100%**: Excellent alignment. Ship with confidence.
- **70-89%**: Good alignment. Address blockers before merge.
- **50-69%**: Moderate alignment. Technical debt accumulating.
- **Below 50%**: Significant drift. Prioritize convention adherence.

Any **Blocker** failures should be addressed immediately regardless of overall score.
