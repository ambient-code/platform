---
name: convention-eval
description: >
  Runs all convention checks across the full codebase and produces a scored
  alignment report. Dispatched by the /align skill.
tools:
  - Read
  - Grep
  - Glob
  - Bash
---

# Convention Evaluation Agent

Evaluate codebase adherence to documented conventions. Produce a scored report.

## Context Files

Load these before running checks:

1. `components/backend/DEVELOPMENT.md`
2. `components/backend/ERROR_PATTERNS.md`
3. `components/backend/K8S_CLIENT_PATTERNS.md`
4. `components/frontend/DEVELOPMENT.md`
5. `components/frontend/REACT_QUERY_PATTERNS.md`
6. `components/operator/DEVELOPMENT.md`
7. `docs/security-standards.md`

## Checks by Category

### Backend (8 checks, weight: 25%)

| # | Check | Severity |
|---|-------|----------|
| B1 | No `panic()` in production | Blocker |
| B2 | User-scoped clients for user ops | Blocker |
| B3 | No direct type assertions | Critical |
| B4 | No silent error handling | Critical |
| B5 | No internal error exposure | Major |
| B6 | No tokens in logs | Blocker |
| B7 | Error wrapping with %w | Major |
| B8 | Files under 400 lines | Minor |

### Frontend (8 checks, weight: 25%)

| # | Check | Severity |
|---|-------|----------|
| F1 | No raw HTML elements | Critical |
| F2 | No manual fetch() | Critical |
| F3 | No `interface` declarations | Major |
| F4 | No `any` types | Critical |
| F5 | Components under 200 lines | Minor |
| F6 | Loading/error states | Major |
| F7 | Colocated single-use components | Minor |
| F8 | Feature flag on new pages | Major |

### Operator (7 checks, weight: 20%)

| # | Check | Severity |
|---|-------|----------|
| O1 | OwnerReferences on child resources | Blocker |
| O2 | Proper reconciliation patterns | Critical |
| O3 | SecurityContext on Job pods | Critical |
| O4 | Resource limits/requests | Major |
| O5 | No `panic()` in production | Blocker |
| O6 | Status condition updates | Critical |
| O7 | No `context.TODO()` | Minor |

### Runner (4 checks, weight: 10%)

| # | Check | Severity |
|---|-------|----------|
| R1 | Proper async patterns | Major |
| R2 | Credential handling | Blocker |
| R3 | Error propagation | Critical |
| R4 | No hardcoded secrets | Blocker |

### Security (7 checks, weight: 20%)

| # | Check | Severity |
|---|-------|----------|
| S1 | User token for user ops | Blocker |
| S2 | RBAC before resource access | Critical |
| S3 | Token redaction | Blocker |
| S4 | Input validation | Major |
| S5 | SecurityContext on pods | Critical |
| S6 | OwnerReferences on Secrets | Critical |
| S7 | No hardcoded credentials | Blocker |

## Scoring

- Each check: Pass (1) or Fail (0)
- Category score: passes / total
- Overall score:
  - Full scope: weighted average across all categories
  - Scoped runs: renormalize weights to selected categories (e.g., backend-only uses 100% backend weight)

## Output Format

```markdown
# Convention Alignment Report

**Scope:** [full | backend | frontend | ...]
**Date:** [ISO date]
**Overall Score:** [X%]

## Category Scores

| Category | Score | Pass | Fail | Blockers |
|----------|-------|------|------|----------|
| Backend  | X/8   | X    | X    | X        |
| Frontend | X/8   | X    | X    | X        |
| Operator | X/7   | X    | X    | X        |
| Runner   | X/4   | X    | X    | X        |
| Security | X/7   | X    | X    | X        |

## Failures

### Blockers
[List with file:line references]

### Critical
[List with file:line references]

### Major / Minor
[List]

## Recommendations
[Top 3 priorities to improve alignment]
```
