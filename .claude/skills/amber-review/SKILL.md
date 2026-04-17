---
name: amber-review
description: >
  Perform a comprehensive code review using repository-specific standards. Use
  when reviewing code changes, checking PR quality, auditing convention compliance,
  or validating changes before merge. Triggers on: "review code", "check changes",
  "code review", "amber review", "review PR", "audit conventions", "quality check".
---

# Amber Review

Stringent, standards-driven code review against this repository's documented patterns, security requirements, and architectural conventions.

## User Input

```text
$ARGUMENTS
```

Consider the user input before proceeding (if not empty). The input may specify files, a PR number, a branch, or a focus area.

## Execution Steps

### 1. Load Review Context

Read all of the following files to build your review context. Do not skip any.

1. `CLAUDE.md` (master project instructions)
2. `components/backend/DEVELOPMENT.md` (Go backend, Gin, K8s integration)
3. `components/frontend/DEVELOPMENT.md` (NextJS, Shadcn UI, React Query)
4. `docs/security-standards.md` (auth, RBAC, token handling, container security)
5. `components/backend/K8S_CLIENT_PATTERNS.md` (user token vs service account)
6. `components/backend/ERROR_PATTERNS.md` (consistent error patterns)
7. `components/frontend/REACT_QUERY_PATTERNS.md` (data fetching patterns)
8. `components/operator/DEVELOPMENT.md` (K8s operator, reconciliation, OwnerReferences)

### 2. Identify Changes to Review

Determine the scope based on user input:

- **If a PR number is provided**: Use `gh pr diff <number>` to get the diff
- **If files/paths are provided**: Review those specific files
- **If a branch is provided**: Diff against `main`
- **If no input**: Review all uncommitted changes (`git diff` + `git diff --cached`)

### 3. Perform Review

Evaluate every changed file against the loaded standards. Apply ALL relevant checks.

#### Review Axes

1. **Code Quality** — Does it follow CLAUDE.md patterns? Naming conventions?
2. **Security** — User token auth (`GetK8sClientsForRequest`), RBAC checks, token redaction, input validation, SecurityContext on Job pods, no secrets in code
3. **Performance** — Unnecessary re-renders, missing query key parameters, N+1 queries, unbounded list operations
4. **Testing** — Adequate coverage for new functionality? Tests follow existing patterns?
5. **Architecture** — Follows project structure? Correct layer separation?
6. **Error Handling** — No `panic()`, no silent failures, wrapped errors with context, generic user messages with detailed server logs

#### Backend-Specific Checks (Go)

- [ ] All user operations use `GetK8sClientsForRequest`, never service account fallback
- [ ] No tokens in logs (use `len(token)`)
- [ ] Type-safe unstructured access (`unstructured.NestedMap`, not direct assertions)
- [ ] No `panic()` in production code
- [ ] Errors wrapped with `fmt.Errorf("context: %w", err)`
- [ ] `errors.IsNotFound` handled for 404 scenarios
- [ ] OwnerReferences set on child resources (Jobs, Secrets, PVCs)

#### Frontend-Specific Checks (TypeScript/React)

- [ ] Zero `any` types (use proper types or `unknown`)
- [ ] Shadcn UI components only (no custom buttons, inputs, dialogs)
- [ ] React Query for all data operations (no manual `fetch()` in components)
- [ ] `type` preferred over `interface`
- [ ] Single-use components colocated with their page
- [ ] Loading and error states handled
- [ ] Query keys include all relevant parameters

#### Security Checks (All Components)

- [ ] RBAC check performed before resource access
- [ ] No tokens or secrets in logs or error messages
- [ ] Input validated (K8s DNS labels, URL parsing)
- [ ] Log injection prevented (no raw newlines in logged user input)
- [ ] Generic error messages to users, detailed logs server-side
- [ ] Container SecurityContext: `AllowPrivilegeEscalation: false`, `Drop: ALL`

### 4. Classify Findings by Severity

- **Blocker** — Must fix before merge. Security vulnerabilities, data loss risk, service account misuse, token leaks
- **Critical** — Should fix before merge. RBAC bypasses, missing error handling, `any` types, `panic()` in handlers
- **Major** — Important to address. Architecture violations, missing tests, performance concerns
- **Minor** — Nice-to-have. Style improvements, documentation gaps

### 5. Produce Review Report

```markdown
# Claude Code Review

## Summary
[1-3 sentence overview]

## Findings

### Blocker
[Must fix — or "None"]

### Critical
[Should fix — or "None"]

### Major
[Important — or "None"]

### Minor
[Nice-to-have — or "None"]

## Positive Highlights
[Things done well — always include at least one]

## Recommendations
[Prioritized action items]
```

For each issue, include: file path and line number, what the problem is, which standard it violates, suggested fix.

## When to Use This vs Individual Agents

- **`/amber-review`**: Comprehensive single-session review across all components. Best for pre-merge quality gates.
- **Individual agents** (backend-review, frontend-review, operator-review, runner-review, security-review): Specialized checks for a single component. Best for focused work on one area or ongoing automated checks.
- **`/align`**: Codebase-wide convention scoring. Best for periodic health checks.

## Operating Principles

- **Be stringent**: This is a quality gate, not a rubber stamp.
- **Be specific**: Reference exact file:line, exact standard, exact fix.
- **Be fair**: Always acknowledge what was done well.
- **No false positives**: Only flag issues backed by loaded standards.
- **Existing code is not in scope**: Only review changed/added lines.
