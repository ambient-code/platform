# Backend Development Context

**When to load:** Working on Go backend API, handlers, or Kubernetes integration

## Quick Reference

- **Language:** Go 1.21+
- **Framework:** Gin (HTTP router)
- **K8s Client:** client-go + dynamic client
- **Primary Files:** `components/backend/handlers/*.go`, `components/backend/types/*.go`

## Critical Rules

### Authentication & Authorization

**ALWAYS use user-scoped clients for API operations:**

```go
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
    c.Abort()
    return
}
```

**FORBIDDEN:** Using backend service account (`DynamicClient`, `K8sClient`) for user-initiated operations

**Backend service account ONLY for:**

- Writing CRs after validation (handlers/sessions.go:417)
- Minting tokens/secrets for runners (handlers/sessions.go:449)
- Cross-namespace operations backend is authorized for

### Token Security

**NEVER log tokens:**

```go
// ❌ BAD
log.Printf("Token: %s", token)

// ✅ GOOD
log.Printf("Processing request with token (len=%d)", len(token))
```

**Token redaction in logs:** See `server/server.go:22-34` for custom formatter

### Error Handling

**Pattern for handler errors:**

```go
// Resource not found
if errors.IsNotFound(err) {
    c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
    return
}

// Generic error
if err != nil {
    log.Printf("Failed to create session %s in project %s: %v", name, project, err)
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
    return
}
```

### Type-Safe Unstructured Access

**FORBIDDEN:** Direct type assertions

```go
// ❌ BAD - will panic if type is wrong
spec := obj.Object["spec"].(map[string]interface{})
```

**REQUIRED:** Use unstructured helpers

```go
// ✅ GOOD
spec, found, err := unstructured.NestedMap(obj.Object, "spec")
if !found || err != nil {
    return fmt.Errorf("spec not found")
}
```

## Common Tasks

### Adding a New API Endpoint

1. **Define route:** `routes.go` with middleware chain
2. **Create handler:** `handlers/[resource].go`
3. **Validate project context:** Use `ValidateProjectContext()` middleware
4. **Get user clients:** `GetK8sClientsForRequest(c)`
5. **Perform operation:** Use `reqDyn` for K8s resources
6. **Return response:** Structured JSON with appropriate status code

### Adding a New Custom Resource Field

1. **Update CRD:** `components/manifests/base/[resource]-crd.yaml`
2. **Update types:** `components/backend/types/[resource].go`
3. **Update handlers:** Extract/validate new field in handlers
4. **Update operator:** Handle new field in reconciliation
5. **Test:** Create sample CR with new field

## Pre-Commit Checklist

- [ ] All user operations use `GetK8sClientsForRequest`
- [ ] No tokens in logs
- [ ] Errors logged with context
- [ ] Type-safe unstructured access
- [ ] `gofmt -w .` applied
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` passes

## Key Files

- `handlers/sessions.go` - AgenticSession lifecycle (3906 lines)
- `handlers/middleware.go` - Auth, RBAC validation
- `handlers/helpers.go` - Utility functions (StringPtr, BoolPtr)
- `types/session.go` - Type definitions
- `server/server.go` - Server setup, token redaction

## Exception: Public API Gateway Service

The `components/public-api/` service is a **stateless HTTP gateway** that does NOT follow the standard backend patterns above. This is intentional:

- **No K8s Clients**: Does NOT use `GetK8sClientsForRequest()` or access Kubernetes directly
- **No RBAC Permissions**: ServiceAccount has NO RoleBindings
- **Token Forwarding Only**: Proxies requests to backend with user's token in `Authorization` header
- **Backend Validates**: All K8s operations and RBAC enforcement happen in the backend service

The public-api is a thin shim layer that extracts/validates tokens, extracts project context, validates input parameters (prevents injection attacks), and forwards requests with proper authorization headers.

## Package Organization

**Backend Structure** (`components/backend/`):

```
backend/
├── handlers/          # HTTP handlers grouped by resource
│   ├── sessions.go    # AgenticSession CRUD + lifecycle
│   ├── projects.go    # Project management
│   ├── rfe.go         # RFE workflows
│   ├── helpers.go     # Shared utilities (StringPtr, etc.)
│   └── middleware.go  # Auth, validation, RBAC
├── types/             # Type definitions (no business logic)
│   ├── session.go
│   ├── project.go
│   └── common.go
├── server/            # Server setup, CORS, middleware
├── k8s/               # K8s resource templates
├── git/, github/      # External integrations
├── websocket/         # Real-time messaging
├── routes.go          # HTTP route registration
└── main.go            # Wiring, dependency injection
```

**Operator Structure** (`components/operator/`):

```
operator/
├── internal/
│   ├── config/        # K8s client init, config loading
│   ├── types/         # GVR definitions, resource helpers
│   ├── handlers/      # Watch handlers (sessions, namespaces, projectsettings)
│   └── services/      # Reusable services (PVC provisioning, etc.)
└── main.go            # Watch coordination
```

**Rules**:

- Handlers contain HTTP/watch logic ONLY
- Types are pure data structures
- Business logic in separate service packages
- No cyclic dependencies between packages

## Resource Management

**OwnerReferences Pattern**:

```go
ownerRef := v1.OwnerReference{
    APIVersion: obj.GetAPIVersion(),
    Kind:       obj.GetKind(),
    Name:       obj.GetName(),
    UID:        obj.GetUID(),
    Controller: boolPtr(true),
    // BlockOwnerDeletion: intentionally omitted (permission issues)
}

job := &batchv1.Job{
    ObjectMeta: v1.ObjectMeta{
        Name: jobName,
        Namespace: namespace,
        OwnerReferences: []v1.OwnerReference{ownerRef},
    },
}
```

**Cleanup Patterns**:

```go
policy := v1.DeletePropagationBackground
err := K8sClient.BatchV1().Jobs(ns).Delete(ctx, jobName, v1.DeleteOptions{
    PropagationPolicy: &policy,
})
if err != nil && !errors.IsNotFound(err) {
    log.Printf("Failed to delete job: %v", err)
    return err
}
```

## API Design Patterns

**Project-Scoped Endpoints**:

```go
r.GET("/api/projects/:projectName/agentic-sessions", ValidateProjectContext(), ListSessions)
r.POST("/api/projects/:projectName/agentic-sessions", ValidateProjectContext(), CreateSession)
r.GET("/api/projects/:projectName/agentic-sessions/:sessionName", ValidateProjectContext(), GetSession)
```

**Middleware Chain** (order matters):

```go
r.Use(gin.Recovery())
r.Use(gin.LoggerWithFormatter(customRedactingFormatter))
r.Use(cors.New(corsConfig))
r.Use(forwardedIdentityMiddleware())
r.Use(ValidateProjectContext())
```

**Response Patterns**:

```go
c.JSON(http.StatusOK, gin.H{"items": sessions})
c.JSON(http.StatusCreated, gin.H{"message": "Session created", "name": name, "uid": uid})
c.Status(http.StatusNoContent)
c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
```

## Operator Patterns

**Watch Loop with Reconnection**:

```go
func WatchAgenticSessions() {
    gvr := types.GetAgenticSessionResource()
    for {
        watcher, err := config.DynamicClient.Resource(gvr).Watch(ctx, v1.ListOptions{})
        if err != nil {
            log.Printf("Failed to create watcher: %v", err)
            time.Sleep(5 * time.Second)
            continue
        }
        for event := range watcher.ResultChan() {
            switch event.Type {
            case watch.Added, watch.Modified:
                obj := event.Object.(*unstructured.Unstructured)
                handleEvent(obj)
            case watch.Deleted:
                // Handle cleanup
            }
        }
        watcher.Stop()
        time.Sleep(2 * time.Second)
    }
}
```

**Reconciliation Pattern**:

```go
func handleEvent(obj *unstructured.Unstructured) error {
    name := obj.GetName()
    namespace := obj.GetNamespace()

    currentObj, err := getDynamicClient().Get(ctx, name, namespace)
    if errors.IsNotFound(err) {
        return nil
    }

    status, found, _ := unstructured.NestedMap(currentObj.Object, "status")
    phase := getPhaseOrDefault(status, "Pending")
    if phase != "Pending" {
        return nil
    }

    if _, err := getResource(name); err == nil {
        return nil
    }

    createResource(...)
    updateStatus(namespace, name, map[string]interface{}{"phase": "Creating"})
    return nil
}
```

**Status Updates** (use UpdateStatus subresource):

```go
func updateAgenticSessionStatus(namespace, name string, updates map[string]interface{}) error {
    gvr := types.GetAgenticSessionResource()
    obj, err := config.DynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, v1.GetOptions{})
    if errors.IsNotFound(err) {
        return nil
    }
    if obj.Object["status"] == nil {
        obj.Object["status"] = make(map[string]interface{})
    }
    status := obj.Object["status"].(map[string]interface{})
    for k, v := range updates {
        status[k] = v
    }
    _, err = config.DynamicClient.Resource(gvr).Namespace(namespace).UpdateStatus(ctx, obj, v1.UpdateOptions{})
    if errors.IsNotFound(err) {
        return nil
    }
    return err
}
```

**Goroutine Monitoring**:

```go
go monitorJob(jobName, sessionName, namespace)

func monitorJob(jobName, sessionName, namespace string) {
    for {
        time.Sleep(5 * time.Second)
        if _, err := getSession(namespace, sessionName); errors.IsNotFound(err) {
            return
        }
        job, err := K8sClient.BatchV1().Jobs(namespace).Get(ctx, jobName, v1.GetOptions{})
        if errors.IsNotFound(err) {
            return
        }
        if job.Status.Succeeded > 0 {
            updateStatus(namespace, sessionName, map[string]interface{}{
                "phase": "Completed",
                "completionTime": time.Now().Format(time.RFC3339),
            })
            cleanup(namespace, jobName)
            return
        }
    }
}
```

## Common Mistakes to Avoid

**Backend**:

- Using service account client for user operations (always use user token)
- Not checking if user-scoped client creation succeeded
- Logging full token values (use `len(token)` instead)
- Not validating project access in middleware
- Type assertions without checking: `val := obj["key"].(string)` (use `val, ok := ...`)
- Not setting OwnerReferences (causes resource leaks)
- Treating IsNotFound as fatal error during cleanup
- Exposing internal error details to API responses (use generic messages)

**Operator**:

- Not reconnecting watch on channel close
- Processing events without verifying resource still exists
- Updating status on main object instead of /status subresource
- Not checking current phase before reconciliation (causes duplicate resources)
- Creating resources without idempotency checks
- Goroutine leaks (not exiting monitor when resource deleted)
- Using `panic()` in watch/reconciliation loops
- Not setting SecurityContext on Job pods

## Reference Files

**Backend**:

- `components/backend/handlers/sessions.go` - Complete session lifecycle, user/SA client usage
- `components/backend/handlers/middleware.go` - Auth patterns, token extraction, RBAC
- `components/backend/handlers/helpers.go` - Utility functions (StringPtr, BoolPtr)
- `components/backend/types/common.go` - Type definitions
- `components/backend/server/server.go` - Server setup, middleware chain, token redaction
- `components/backend/routes.go` - HTTP route definitions and registration

**Operator**:

- `components/operator/internal/handlers/sessions.go` - Watch loop, reconciliation, status updates
- `components/operator/internal/config/config.go` - K8s client initialization
- `components/operator/internal/types/resources.go` - GVR definitions
- `components/operator/internal/services/infrastructure.go` - Reusable services

## Recent Issues & Learnings

- **2024-11-15:** Fixed token leak in logs - never log raw tokens
- **2024-11-10:** Multi-repo support added - `mainRepoIndex` specifies working directory
- **2024-10-20:** Added RBAC validation middleware - always check permissions
