# CodeRabbit-Derived Conventions

Conventions added from analysis of 971 Critical+Major CodeRabbit review comments across 169 PRs (v0.0.22 through v0.2.0). Each convention addresses a recurring pattern that slipped past the AI agent during development.

## Image references must match across the stack

**Evidence**: 9 occurrences, 6 Critical. Top pattern by impact score (33).

Every image name and tag used in manifests, env vars (`RUNNER_IMAGE`, `STATE_SYNC_IMAGE`), kustomization overlays, kind load commands, and GHA build matrices must resolve to the same artifact.

After changing an image name or tag, grep all overlays, workflows, ConfigMaps, and Makefile targets. Mismatches cause silent deployment failures — pods pull a non-existent tag and enter `ImagePullBackOff`.

**Common violations**:
- Renaming an image in the build matrix but not in the kind overlay
- Updating `RUNNER_IMAGE` in one overlay but not the production env patch
- Partial releases deploying tags that were never built

## Reconcile, don't create-or-skip

**Evidence**: 6 occurrences, 1 Critical. Impact score 19.

Operator and backend code that creates K8s resources must use update-or-create (reconcile) patterns, not create-and-ignore-`AlreadyExists`. Treating `AlreadyExists` as success skips spec drift, label changes, and ownership updates.

**Pattern to avoid**:
```go
err := client.Create(ctx, obj)
if apierrors.IsAlreadyExists(err) {
    return nil // BAD: silently skips spec updates
}
```

**Correct pattern**:
```go
existing := &v1.RoleBinding{}
err := client.Get(ctx, key, existing)
if apierrors.IsNotFound(err) {
    return client.Create(ctx, obj)
}
// Update spec if it drifted
existing.Subjects = obj.Subjects
existing.RoleRef = obj.RoleRef
return client.Update(ctx, existing)
```

## Never silently swallow partial failures

**Evidence**: 6 occurrences. Impact score 18.

Every error path must propagate or explicitly log the failure. Do not discard errors from reconciliation loops, multi-step operations, or cleanup routines. If a step can fail independently, collect errors and return them together.

**Pattern to avoid**:
```go
for _, item := range items {
    if err := reconcile(item); err != nil {
        continue // BAD: silently drops the error
    }
}
```

**Correct pattern**:
```go
var errs []error
for _, item := range items {
    if err := reconcile(item); err != nil {
        errs = append(errs, fmt.Errorf("reconcile %s: %w", item.Name, err))
    }
}
return errors.Join(errs...)
```

## Namespace-scope shared state keys

**Evidence**: 6 occurrences, 1 Critical. Impact score 19.

Cache keys, status map entries, and derived identifiers that span multiple sessions or projects must include the namespace/project as a prefix. Bare `sessionID` or `name` keys collide across tenants.

**Pattern to avoid**:
```go
cache.Set(sessionID, status) // BAD: collides across projects
```

**Correct pattern**:
```go
cache.Set(fmt.Sprintf("%s/%s", namespace, sessionID), status)
```

## Restricted SecurityContext on all containers

**Evidence**: 3 occurrences. Impact score 9.

All init containers and sidecar containers in manifests must set:
- `runAsNonRoot: true`
- `capabilities.drop: ["ALL"]`
- `readOnlyRootFilesystem: true` (unless a specific write path is required)

This applies to all overlays including local-dev. OpenShift's restricted SCC will reject pods that don't meet these requirements.

```yaml
securityContext:
  runAsNonRoot: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  readOnlyRootFilesystem: true
  seccompProfile:
    type: RuntimeDefault
```

## Triage Pipeline

These conventions are maintained by the per-release CodeRabbit triage pipeline (`scripts/coderabbit-triage/`). Each release generates a metrics snapshot and trend report identifying new coverage gaps. See the [triage pipeline README](../scripts/coderabbit-triage/README.md) for usage.
