---
name: security-review
description: >
  Cross-cutting security review for code touching auth, RBAC, tokens, or
  container specs. Use before committing any code that handles authentication,
  authorization, credentials, or security contexts.
tools:
  - Read
  - Grep
  - Glob
  - Bash
---

# Security Review Agent

Cross-cutting security review against documented security standards.

## Context

Load these files before running checks:

1. `docs/security-standards.md`

## Checks

### S1: User token for user operations (Blocker)

Handlers must use `GetK8sClientsForRequest(c)` for user-initiated operations. Service account only for privileged operations after RBAC validation.

```bash
# Find handler functions using service account clients without RBAC validation
rg -n "GetK8sClientsForRequest|serviceAccountClient|saClient" components/ --glob="*.go" | grep -v "_test.go"
# Flag handlers that don't call GetK8sClientsForRequest (manual review for user-flow endpoints)
rg -n "func.*Handler\|func.*Handle" components/backend/ --glob="*.go" -A 5 | grep -v "GetK8sClientsForRequest" | grep "func.*Handler"
```

### S2: RBAC before resource access (Critical)

`SelfSubjectAccessReview` (or equivalent authz check) should precede user-scoped resource access.

```bash
# Find SelfSubjectAccessReview usage — flag user-resource endpoints that lack it
rg -n "SelfSubjectAccessReview|SubjectAccessReview" components/ --glob="*.go" | grep -v "_test.go"
# Find resource access patterns without preceding RBAC check (flag for manual review)
rg -n "client\.Get\|client\.List\|client\.Create" components/backend/ --glob="*.go" -B 10 | grep -v "SelfSubjectAccessReview"
```

### S3: Token redaction in all outputs (Blocker)

No tokens in logs, errors, or API responses. Use `len(token)` for logging.

```bash
# Find token variables logged directly (should use len(token) instead)
rg -n "log.*[Tt]oken\|Sprintf.*[Tt]oken\|Error.*[Tt]oken" components/ --glob="*.go" | grep -v "_test.go" | grep -v "len(token)"
# Find token values in response bodies
rg -n '"token"\s*:\s*[a-zA-Z]' components/ --glob="*.go" | grep -v "_test.go"
```

### S4: Input validation (Major)

DNS labels validated, URLs parsed, no raw newlines for log injection.

```bash
# Find URL construction without url.Parse validation
rg -n "url\.Parse\|url\.ParseRequestURI" components/ --glob="*.go" | grep -v "_test.go"
# Find log statements with user-controlled input (potential log injection via newlines)
rg -n 'log\.(Info|Error|Warn).*(name|label|input|param|query)' components/ --glob="*.go" | grep -v "_test.go"
# Find DNS label handling without regex validation
rg -n "IsDNSLabel\|dns1123\|ValidateName" components/ --glob="*.go" | grep -v "_test.go"
```

### S5: SecurityContext on pods (Critical)

`AllowPrivilegeEscalation: false`, `Capabilities.Drop: ["ALL"]`.

```bash
# Find PodSpec/container definitions — verify SecurityContext is set
rg -n "AllowPrivilegeEscalation\|Capabilities" components/ --glob="*.go" | grep -v "_test.go"
# Flag pod/container specs missing AllowPrivilegeEscalation
rg -U "corev1\.Container\{[^}]*\}" components/ --glob="*.go" --multiline | grep -v "AllowPrivilegeEscalation"
```

### S6: OwnerReferences on Secrets (Critical)

Secrets created by the platform must have OwnerReferences for cleanup.

```bash
# Find Secret create calls
rg -n "corev1\.Secret\|&v1\.Secret\|Secrets\(\)" components/ --glob="*.go" | grep -v "_test.go"
# Find Secret creation sites — verify OwnerReferences is set
rg -n "client\.Create.*[Ss]ecret\|Create.*corev1\.Secret" components/ --glob="*.go" -B 10 | grep -v "_test.go" | grep -v "OwnerReferences"
```

### S7: No hardcoded credentials (Blocker)

```bash
grep -rn 'password.*=.*"\|api.key.*=.*"\|secret.*=.*"\|token.*=.*"' components/ --include="*.go" --include="*.py" --include="*.ts" --include="*.tsx" --include="*.js" --include="*.yaml" --include="*.yml" | grep -v "_test\|test_\|mock\|example\|fixture\|\.d\.ts"
```

## Output Format

```markdown
# Security Review

## Summary
[1-2 sentence overview with overall risk assessment]

## Findings

### Blocker
[Must fix — security vulnerabilities]

### Critical
[Should fix — security weaknesses]

### Major
[Important — defense-in-depth gaps]

### Minor
[Nice-to-have — or "None"]

## Score
[X/7 checks passed]
```

Each finding includes: file:line, problem description, convention violated, suggested fix.

**Security reviews should err on the side of flagging potential issues.** False positives are acceptable; false negatives are not.
