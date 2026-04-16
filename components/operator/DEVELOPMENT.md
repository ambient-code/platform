# Operator Development Context

> Part of [CLAUDE.md Critical Conventions](../../CLAUDE.md#critical-conventions)

**When to load:** Working on the Kubernetes operator, reconciliation logic, or Job management

## Quick Reference

- **Language:** Go 1.21+
- **Pattern:** Controller-runtime based operator
- **Primary Files:** `internal/handlers/sessions.go`, `internal/config/config.go`

## Critical Rules

### OwnerReferences on All Child Resources

Every Job, Secret, and PVC the operator creates **must** set OwnerReferences pointing to the parent AgenticSession CR. This ensures automatic cleanup when the session is deleted.

```go
OwnerReferences: []metav1.OwnerReference{
    {
        APIVersion:         obj.GetAPIVersion(),
        Kind:               obj.GetKind(),
        Name:               obj.GetName(),
        UID:                obj.GetUID(),
        Controller:         boolPtr(true),
        // BlockOwnerDeletion omitted — causes permission issues in constrained RBAC environments
    },
},
```

### SecurityContext on Job Pod Specs

All Job pod specs must include a restrictive SecurityContext:

```go
SecurityContext: &corev1.SecurityContext{
    AllowPrivilegeEscalation: boolPtr(false),
    ReadOnlyRootFilesystem:   boolPtr(false),
    Capabilities: &corev1.Capabilities{
        Drop: []corev1.Capability{"ALL"},
    },
},
```

### Resource Limits and Requests

Job containers must specify resource requirements to prevent unbounded resource consumption.

### Reconciliation Error Handling

```go
// Resource deleted during reconciliation — NOT an error
if errors.IsNotFound(err) {
    log.Printf("Resource %s/%s deleted, skipping", namespace, name)
    return ctrl.Result{}, nil  // Don't requeue
}

// Transient error — return error to requeue
if err != nil {
    return ctrl.Result{}, fmt.Errorf("failed to get object: %w", err)
}
```

**Key patterns:**
- `IsNotFound` → return `ctrl.Result{}, nil` (resource gone, no retry)
- Transient errors → return `ctrl.Result{}, err` (triggers requeue with backoff)
- Terminal errors → update CR status to "Failed", return `ctrl.Result{}, nil` (don't retry)

### Status Updates on Error

When an operation fails, always update the CR status before returning:

```go
updateAgenticSessionStatus(namespace, name, map[string]interface{}{
    "phase":   "Failed",
    "message": fmt.Sprintf("Failed to create job: %v", err),
})
```

### Context Propagation

Use the context from the reconciliation request, not `context.TODO()`:

```go
// Bad
ctx := context.TODO()

// Good — use the ctx parameter from the Reconcile(ctx, req) signature
// The ctx is already provided as the first argument to Reconcile and phase handlers
```

### No panic() in Production

Same as backend: return `fmt.Errorf` with context instead. A panic crashes the entire operator, affecting all sessions.

## Pre-Commit Checklist

- [ ] OwnerReferences set on all child resources
- [ ] SecurityContext on all Job pod specs
- [ ] Resource limits/requests on containers
- [ ] Status updated on error paths
- [ ] No `panic()` in non-test code
- [ ] Proper context propagation (no `context.TODO()`)
- [ ] `gofmt -w .` applied
- [ ] `go vet ./...` passes
