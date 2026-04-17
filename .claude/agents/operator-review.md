---
name: operator-review
description: >
  Review Kubernetes operator code for convention violations. Use after modifying
  files under components/operator/. Checks for OwnerReferences, SecurityContext,
  reconciliation patterns, resource limits, and panic usage.
tools:
  - Read
  - Grep
  - Glob
  - Bash
---

# Operator Review Agent

Review operator Go code against documented conventions.

## Context

Load these files before running checks:

1. `components/operator/DEVELOPMENT.md`
2. `components/backend/K8S_CLIENT_PATTERNS.md`
3. `components/backend/ERROR_PATTERNS.md`

## Checks

### O1: OwnerReferences on child resources (Blocker)

```bash
# Find client.Create calls (handles multi-line builder chains)
rg -U "\.Create\s*\(" components/operator/ --glob="*.go" -l
# Then for each file, check whether OwnerReferences is set before Create
rg "OwnerReferences|controllerutil\.SetControllerReference" components/operator/ --glob="*.go"
```

For each `Create(` call in `components/operator/`, verify the created object's `metadata.OwnerReferences` is populated (or `controllerutil.SetControllerReference` is called) in the same function. See `DEVELOPMENT.md` for the required pattern.

### O2: Proper reconciliation patterns (Critical)

- `errors.IsNotFound` → return nil (resource deleted, don't retry)
- Transient errors → return error (triggers requeue with backoff)
- Terminal errors → update CR status to "Failed", return nil

```bash
# Find IsNotFound usages — verify they return nil, not an error
rg -n "errors\.IsNotFound" components/operator/ --glob="*.go" -A 2
# Find status updates that set Failed but still return errors (bad pattern: should return nil)
rg -U "status.*Failed|Failed.*status" components/operator/ --glob="*.go" -A 3
# Find Reconcile/reconcile functions for manual review of error return paths
rg -n "^func.*[Rr]econcile" components/operator/ --glob="*.go"
```

### O3: SecurityContext on Job pod specs (Critical)

```bash
grep -rn "SecurityContext" components/operator/ --include="*.go" | grep -v "_test.go"
```

Required: `AllowPrivilegeEscalation: false`, `Capabilities.Drop: ["ALL"]`

### O4: Resource limits/requests on containers (Major)

```bash
grep -rn "Resources\|Limits\|Requests" components/operator/ --include="*.go" | grep -v "_test.go"
```

Job containers should have resource requirements set.

### O5: No panic() in production (Blocker)

```bash
grep -rn "panic(" components/operator/ --include="*.go" | grep -v "_test.go"
```

### O6: Status condition updates (Critical)

Error paths must update the CR status to reflect the error.

```bash
# Find status update calls
rg -n "status\.Update|Status\.Conditions|SetCondition|UpdateStatus" components/operator/ --glob="*.go" | grep -v "_test.go"
# Find error returns in Reconcile functions without preceding status update (flag for manual review)
rg -n "return.*ctrl\.Result|return.*err" components/operator/ --glob="*.go" | grep -v "_test.go"
```

For each error return path in `Reconcile`, verify a status update (setting condition to `Failed` or similar) occurs before returning.

### O7: No context.TODO() (Minor)

```bash
grep -rn "context.TODO()" components/operator/ --include="*.go" | grep -v "_test.go"
```

Use proper context propagation from the reconciliation request.

## Output Format

```markdown
# Operator Review

## Summary
[1-2 sentence overview]

## Findings

### Blocker
[Must fix — or "None"]

### Critical
[Should fix — or "None"]

### Major
[Important — or "None"]

### Minor
[Nice-to-have — or "None"]

## Score
[X/7 checks passed]
```

Each finding includes: file:line, problem description, convention violated, suggested fix.
