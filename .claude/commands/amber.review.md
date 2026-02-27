---
description: Perform a comprehensive code review using repository-specific standards from the Amber memory system.
---

## User Input

```text
$ARGUMENTS
```

You **MUST** consider the user input before proceeding (if not empty). The input may specify files, a PR number, a branch, or a focus area.

## Goal

Perform a stringent, standards-driven code review against this repository's documented patterns, security requirements, and architectural conventions.

## Execution Steps

### 1. Load Memory System

Read all of the following files to build your review context. Do not skip any.

1. `CLAUDE.md` (master project instructions)
2. `.claude/context/backend-development.md` (Go backend, Gin, K8s integration)
3. `.claude/context/frontend-development.md` (NextJS, Shadcn UI, React Query)
4. `.claude/context/security-standards.md` (auth, RBAC, token handling, container security)
5. `.claude/patterns/k8s-client-usage.md` (user token vs service account)
6. `.claude/patterns/error-handling.md` (consistent error patterns)
7. `.claude/patterns/react-query-usage.md` (data fetching patterns)

### 2. Identify Changes to Review

Determine the scope based on user input:

- **If a PR number is provided**: Use `gh pr diff <number>` to get the diff
- **If files/paths are provided**: Review those specific files
- **If a branch is provided**: Diff against `main`
- **If no input**: Review all uncommitted changes (`git diff` + `git diff --cached`)

### 3. Perform Review

Evaluate every changed file against the loaded standards. Apply ALL relevant checks — do not cherry-pick.

#### Review Axes

1. **Code Quality** — Does it follow CLAUDE.md patterns? Naming conventions? No unnecessary comments?
2. **Security** — User token auth (`GetK8sClientsForRequest`), RBAC checks before operations, token redaction in logs, input validation, SecurityContext on Job pods, no secrets in code
3. **Performance** — Unnecessary re-renders, missing query key parameters, N+1 queries, unbounded list operations
4. **Testing** — Adequate coverage for new functionality? Tests follow existing patterns?
5. **Architecture** — Follows project structure from memory context? Correct layer separation (api/ vs queries/ in frontend, handlers/ vs types/ in backend)?
6. **Error Handling** — Follows error handling patterns? No `panic()`, no silent failures, wrapped errors with context, generic user messages with detailed server logs

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

Assign each finding exactly one severity level:

- **Blocker** — Must fix before merge. Security vulnerabilities, data loss risk, service account misuse for user operations, token leaks
- **Critical** — Should fix before merge. RBAC bypasses, missing error handling on K8s operations, `any` types in new code, `panic()` in handlers
- **Major** — Important to address. Architecture violations, missing tests for new logic, performance concerns, pattern deviations
- **Minor** — Nice-to-have. Style improvements, documentation gaps, minor naming inconsistencies

### 5. Produce Review Report

Output the review in this exact format:

```markdown
# Claude Code Review

## Summary
[1-3 sentence overview of the changes and overall assessment]

## Issues by Severity

### Blocker Issues
[Must fix before merge — or "None" if clean]

### Critical Issues
[Should fix before merge — or "None"]

### Major Issues
[Important to address — or "None"]

### Minor Issues
[Nice-to-have improvements — or "None"]

## Positive Highlights
[Things done well — always include at least one]

## Recommendations
[Prioritized action items, most important first]
```

For each issue, include:
- File path and line number(s)
- What the problem is
- Which standard it violates (reference the memory file)
- Suggested fix (code snippet when helpful)

## Operating Principles

- **Be stringent**: This is a quality gate, not a rubber stamp. Flag real issues.
- **Be specific**: Reference exact file:line, exact standard violated, exact fix.
- **Be fair**: Always acknowledge what was done well in Positive Highlights.
- **No false positives**: Only flag issues backed by the loaded standards. Do not invent rules.
- **Existing code is not in scope**: Only review changed/added lines unless existing code is directly affected.

## Context

$ARGUMENTS
