# Example Skill: Error Handling Patterns

This is a **sample skill** showing what a formalized skill would look like for the Ambient Code Platform. This example takes the existing `.claude/patterns/error-handling.md` content and formats it as a complete skill.

**Location (when implemented):** `.ambient/skills/error-handling-patterns/SKILL.md`

---

# Error Handling Patterns Skill

**Skill Name:** error-handling-patterns
**Version:** 1.0.0
**Last Updated:** 2026-01-27
**Maintainer:** Platform Team

## Trigger Conditions

Use this skill when:
- Working on error handling in Go backend or operator
- Implementing new API endpoints
- Debugging error propagation issues
- Reviewing PRs with error handling changes
- Keywords: `error handling`, `gin.Context`, `http.Status`, `k8s errors`, `authorization`, `validation`

## Overview

Consistent error handling is critical for debugging, security, and user experience. This skill covers error handling patterns across the Ambient Code Platform's Go components (backend and operator).

**Core Principles:**
1. **Validate early, fail fast** - Catch errors at entry points
2. **Log with context** - Include project/session names, operations
3. **User-safe messages** - Don't expose internals or stack traces
4. **Proper HTTP status codes** - Match semantics (400, 401, 403, 404, 500)
5. **Structured errors** - Use `gin.H{"error": "message"}` consistently

---

## Pattern Reference

### Pattern 1: Resource Not Found (404)

**When to Use:**
- Kubernetes resource doesn't exist
- Session, project, or job not found

**Template:**
```go
func GetSession(c *gin.Context) {
    projectName := c.Param("projectName")
    sessionName := c.Param("sessionName")

    // 1. Validate authentication first
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        return
    }

    // 2. Fetch resource
    obj, err := reqDyn.Resource(gvr).Namespace(projectName).Get(ctx, sessionName, v1.GetOptions{})

    // 3. Check for not found BEFORE generic error
    if errors.IsNotFound(err) {
        c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
        return
    }

    // 4. Handle other errors
    if err != nil {
        log.Printf("Failed to get session %s/%s: %v", projectName, sessionName, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
        return
    }

    // 5. Success path
    c.JSON(http.StatusOK, obj)
}
```

**Key Points:**
- ✅ Check `errors.IsNotFound(err)` for K8s resources
- ✅ Log with context (project, session name, operation)
- ✅ Generic user message ("Failed to retrieve session")
- ✅ Proper status code (404 for not found, 500 for internal errors)

**Anti-Patterns:**
- ❌ Returning 500 for not found errors
- ❌ Exposing K8s error details to users
- ❌ Not logging errors
- ❌ Logging without context (which session failed?)

---

### Pattern 2: Validation Errors (400)

**When to Use:**
- Invalid request body
- Missing required fields
- Invalid Kubernetes resource names
- Business logic validation failures

**Template:**
```go
func CreateSession(c *gin.Context) {
    var req CreateSessionRequest

    // 1. Validate JSON binding
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }

    // 2. Validate resource name format (K8s DNS label)
    if !isValidK8sName(req.Name) {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "Invalid name: must be a valid Kubernetes DNS label",
        })
        return
    }

    // 3. Validate required fields
    if req.Prompt == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Prompt is required"})
        return
    }

    // 4. Business logic validation
    if req.MaxTokens < 0 || req.MaxTokens > 200000 {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": "MaxTokens must be between 0 and 200000",
        })
        return
    }

    // ... create session
}

// Helper function
func isValidK8sName(name string) bool {
    // K8s DNS label: lowercase alphanumeric + hyphens, max 63 chars
    matched, _ := regexp.MatchString(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`, name)
    return matched && len(name) <= 63
}
```

**Key Points:**
- ✅ Validate early, fail fast
- ✅ Specific error messages (what's wrong, not just "invalid")
- ✅ Consistent status code (400 Bad Request)
- ✅ K8s naming constraints enforced

**Anti-Patterns:**
- ❌ Vague errors ("Invalid input")
- ❌ Continuing after validation failure
- ❌ Not checking K8s name format (causes downstream errors)
- ❌ Returning 500 for validation errors

---

### Pattern 3: Authorization Errors (403)

**When to Use:**
- User lacks RBAC permissions
- Token is valid but insufficient privileges
- Resource in namespace user can't access

**Template:**
```go
func CreateSession(c *gin.Context) {
    projectName := c.Param("projectName")

    reqK8s, _ := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        return
    }

    // 1. Create SelfSubjectAccessReview
    ssar := &authv1.SelfSubjectAccessReview{
        Spec: authv1.SelfSubjectAccessReviewSpec{
            ResourceAttributes: &authv1.ResourceAttributes{
                Group:     "vteam.ambient-code",
                Resource:  "agenticsessions",
                Verb:      "create",
                Namespace: projectName,
            },
        },
    }

    // 2. Check authorization
    res, err := reqK8s.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, ssar, v1.CreateOptions{})
    if err != nil {
        log.Printf("Authorization check failed: %v", err)
        c.JSON(http.StatusForbidden, gin.H{"error": "Authorization check failed"})
        return
    }

    // 3. Deny if not allowed
    if !res.Status.Allowed {
        log.Printf("User denied create access to agenticsessions in namespace %s: %s", projectName, res.Status.Reason)
        c.JSON(http.StatusForbidden, gin.H{
            "error": "You do not have permission to create sessions in this project",
        })
        return
    }

    // ... proceed with creation
}
```

**Key Points:**
- ✅ Use `SelfSubjectAccessReview` for RBAC checks
- ✅ Log denial reason (for debugging)
- ✅ User-friendly error message (specific to operation)
- ✅ 403 Forbidden (not 401, token is valid)

**Anti-Patterns:**
- ❌ Returning 401 for authorization failures (401 = authentication, 403 = authorization)
- ❌ Not logging denial reason
- ❌ Proceeding without RBAC check
- ❌ Checking authorization after resource creation (TOCTOU issue)

---

### Pattern 4: Authentication Errors (401)

**When to Use:**
- Missing or invalid token
- Token expired
- Token malformed

**Template:**
```go
func GetK8sClientsForRequest(c *gin.Context) (*kubernetes.Clientset, dynamic.Interface) {
    // 1. Extract token from header
    authHeader := c.GetHeader("Authorization")
    if authHeader == "" {
        log.Printf("Missing Authorization header")
        return nil, nil
    }

    // 2. Validate Bearer format
    if !strings.HasPrefix(authHeader, "Bearer ") {
        log.Printf("Invalid Authorization header format")
        return nil, nil
    }

    token := strings.TrimPrefix(authHeader, "Bearer ")
    if token == "" {
        log.Printf("Empty bearer token")
        return nil, nil
    }

    // 3. Create K8s clients with user token
    config := &rest.Config{
        Host:        os.Getenv("K8S_API_SERVER"),
        BearerToken: token,
        TLSClientConfig: rest.TLSClientConfig{
            Insecure: false,
            CAFile:   "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt",
        },
    }

    clientset, err := kubernetes.NewForConfig(config)
    if err != nil {
        log.Printf("Failed to create K8s clientset: %v", err)
        return nil, nil
    }

    dynClient, err := dynamic.NewForConfig(config)
    if err != nil {
        log.Printf("Failed to create dynamic client: %v", err)
        return nil, nil
    }

    return clientset, dynClient
}

// Usage in handlers
func SomeHandler(c *gin.Context) {
    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        return
    }
    // ... use clients
}
```

**Key Points:**
- ✅ Validate token format and presence
- ✅ Log authentication failures (without token value)
- ✅ Return nil to signal auth failure
- ✅ Handlers check nil and return 401

**Anti-Patterns:**
- ❌ Logging token values
- ❌ Returning 403 for missing tokens (403 = authorized but forbidden)
- ❌ Falling back to service account (security issue)
- ❌ Not validating Bearer format

---

### Pattern 5: Conflict Errors (409)

**When to Use:**
- Resource already exists (duplicate name)
- Concurrent modification (optimistic locking)

**Template:**
```go
func CreateSession(c *gin.Context) {
    projectName := c.Param("projectName")
    sessionName := req.Name

    reqK8s, reqDyn := GetK8sClientsForRequest(c)
    if reqK8s == nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
        return
    }

    // ... validation, authorization ...

    // 1. Create the session
    obj, err := reqDyn.Resource(gvr).Namespace(projectName).Create(ctx, unstructuredObj, v1.CreateOptions{})

    // 2. Check for AlreadyExists error
    if errors.IsAlreadyExists(err) {
        c.JSON(http.StatusConflict, gin.H{
            "error": fmt.Sprintf("Session '%s' already exists in project '%s'", sessionName, projectName),
        })
        return
    }

    // 3. Handle other errors
    if err != nil {
        log.Printf("Failed to create session %s/%s: %v", projectName, sessionName, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
        return
    }

    c.JSON(http.StatusCreated, obj)
}
```

**Key Points:**
- ✅ Use `errors.IsAlreadyExists(err)` for K8s resources
- ✅ 409 Conflict status code
- ✅ Specific error message with resource name

**For Concurrent Modification:**
```go
// Update with optimistic locking
func UpdateSession(c *gin.Context) {
    // ... get session ...

    // 1. Modify the object
    unstructured.SetNestedField(obj.Object, newValue, "spec", "field")

    // 2. Update with current resourceVersion
    updated, err := reqDyn.Resource(gvr).Namespace(projectName).Update(ctx, obj, v1.UpdateOptions{})

    // 3. Check for conflict
    if errors.IsConflict(err) {
        c.JSON(http.StatusConflict, gin.H{
            "error": "Session was modified by another request. Please retry.",
        })
        return
    }

    if err != nil {
        log.Printf("Failed to update session: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
        return
    }

    c.JSON(http.StatusOK, updated)
}
```

---

### Pattern 6: Internal Server Errors (500)

**When to Use:**
- Unexpected errors (not validation, auth, or not-found)
- K8s API failures
- Database connection issues
- External service failures

**Template:**
```go
func SomeHandler(c *gin.Context) {
    // ... setup ...

    // 1. Operation that might fail unexpectedly
    result, err := externalService.DoSomething(params)
    if err != nil {
        // 2. Log with full context
        log.Printf("External service failed for project=%s, session=%s: %v",
                   projectName, sessionName, err)

        // 3. Generic user message (don't expose internals)
        c.JSON(http.StatusInternalServerError, gin.H{
            "error": "An unexpected error occurred. Please try again.",
        })
        return
    }

    // ... success path ...
}
```

**Key Points:**
- ✅ Log full error details (for debugging)
- ✅ Generic user message (security)
- ✅ 500 status code
- ✅ Include request context in logs

**Anti-Patterns:**
- ❌ Exposing error details to users (security issue)
- ❌ Returning 200 with error in body
- ❌ Not logging internal errors
- ❌ Using 500 for validation/auth errors

---

## Operator-Specific Patterns

### Pattern 7: Reconciliation Loop Errors

**When to Use:**
- Operator reconcile function
- Updating status subresource
- Creating/deleting child resources

**Template:**
```go
func (r *AgenticSessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    log := log.FromContext(ctx)

    // 1. Fetch the AgenticSession
    var session vteamv1alpha1.AgenticSession
    if err := r.Get(ctx, req.NamespacedName, &session); err != nil {
        if errors.IsNotFound(err) {
            // Resource deleted, stop reconciling
            log.Info("AgenticSession not found, likely deleted")
            return ctrl.Result{}, nil
        }
        // Unexpected error, retry
        log.Error(err, "Failed to fetch AgenticSession")
        return ctrl.Result{}, err
    }

    // 2. Handle reconciliation logic
    job, err := r.createJobForSession(ctx, &session)
    if err != nil {
        // 3. Update status to reflect error
        session.Status.Phase = "Failed"
        session.Status.Message = "Failed to create job"
        if updateErr := r.Status().Update(ctx, &session); updateErr != nil {
            log.Error(updateErr, "Failed to update status after job creation failure")
            // Return original error, not update error
            return ctrl.Result{}, err
        }

        // 4. Log and requeue
        log.Error(err, "Failed to create job", "session", session.Name)
        return ctrl.Result{RequeueAfter: 30 * time.Second}, err
    }

    // 5. Success: update status
    session.Status.Phase = "Running"
    session.Status.JobName = job.Name
    if err := r.Status().Update(ctx, &session); err != nil {
        log.Error(err, "Failed to update status")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

**Key Points:**
- ✅ Return `(ctrl.Result{}, nil)` for deleted resources
- ✅ Return `(ctrl.Result{}, err)` for retriable errors
- ✅ Update status subresource on failures
- ✅ Use `RequeueAfter` for rate-limited retries
- ✅ Log with structured fields

**Anti-Patterns:**
- ❌ Infinite requeue loops (missing backoff)
- ❌ Not updating status on errors
- ❌ Fighting over status subresource (concurrent updates)
- ❌ Panicking instead of returning errors

---

## Decision Tree: Which Error Pattern?

```
Is the error related to authentication?
├─ YES: Pattern 4 (401 Unauthorized)
└─ NO: Continue

Is the error related to authorization/permissions?
├─ YES: Pattern 3 (403 Forbidden)
└─ NO: Continue

Is it a validation error (bad input)?
├─ YES: Pattern 2 (400 Bad Request)
└─ NO: Continue

Is the resource not found?
├─ YES: Pattern 1 (404 Not Found)
└─ NO: Continue

Is it a conflict (duplicate, concurrent modification)?
├─ YES: Pattern 5 (409 Conflict)
└─ NO: Continue

Is it an unexpected error?
└─ YES: Pattern 6 (500 Internal Server Error)
```

---

## Quick Reference

### HTTP Status Codes

| Code | Name | Use When |
|------|------|----------|
| 200 | OK | Successful GET, PUT, PATCH |
| 201 | Created | Successful POST (resource created) |
| 204 | No Content | Successful DELETE |
| 400 | Bad Request | Validation error, malformed input |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Valid auth, insufficient permissions |
| 404 | Not Found | Resource doesn't exist |
| 409 | Conflict | Resource already exists, concurrent modification |
| 500 | Internal Server Error | Unexpected error, service failure |

### Common K8s Error Checks

```go
import "k8s.io/apimachinery/pkg/api/errors"

errors.IsNotFound(err)       // 404
errors.IsAlreadyExists(err)  // 409
errors.IsConflict(err)       // 409 (optimistic locking)
errors.IsForbidden(err)      // 403
errors.IsUnauthorized(err)   // 401
errors.IsInvalid(err)        // 400 (validation)
```

### Logging Template

```go
// Good: structured with context
log.Printf("Failed to %s: project=%s, session=%s, error=%v",
           operation, projectName, sessionName, err)

// Bad: no context
log.Printf("Error: %v", err)
```

---

## Testing Error Handling

### Unit Tests

```go
func TestGetSession_NotFound(t *testing.T) {
    // Setup
    router := gin.New()
    router.GET("/projects/:projectName/sessions/:sessionName", GetSession)

    // Mock K8s client to return NotFound error
    mockClient := &mockK8sClient{
        getError: errors.NewNotFound(schema.GroupResource{}, "test-session"),
    }

    // Execute
    req := httptest.NewRequest("GET", "/projects/test-project/sessions/test-session", nil)
    req.Header.Set("Authorization", "Bearer test-token")
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)

    // Assert
    assert.Equal(t, http.StatusNotFound, w.Code)
    assert.Contains(t, w.Body.String(), "Session not found")
}
```

### Integration Tests

```go
func TestSessionCreation_Duplicate(t *testing.T) {
    // Create session
    session := createTestSession(t, "test-project", "my-session")

    // Try to create duplicate
    resp, err := client.CreateSession("test-project", "my-session", sessionSpec)

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, http.StatusConflict, resp.StatusCode)
    assert.Contains(t, resp.Body, "already exists")
}
```

---

## Troubleshooting Guide

### Problem: Getting 500 errors for non-existent resources

**Cause:** Not checking `errors.IsNotFound(err)` before generic error handling

**Fix:**
```go
// BEFORE (wrong order)
if err != nil {
    return ctrl.Result{}, err  // Returns 500 for NotFound
}
if errors.IsNotFound(err) {
    // Never reached
}

// AFTER (correct order)
if errors.IsNotFound(err) {
    c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
    return
}
if err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
    return
}
```

---

### Problem: Users seeing K8s error messages

**Cause:** Returning raw errors to users

**Fix:**
```go
// BEFORE (exposes internals)
c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

// AFTER (user-safe)
log.Printf("Internal error: %v", err)
c.JSON(http.StatusInternalServerError, gin.H{"error": "An error occurred"})
```

---

### Problem: Operator infinite requeue loops

**Cause:** Not using backoff or RequeueAfter

**Fix:**
```go
// BEFORE (infinite loop)
return ctrl.Result{Requeue: true}, nil

// AFTER (rate-limited)
return ctrl.Result{RequeueAfter: 30 * time.Second}, err
```

---

## Related Skills

- **Go Backend API Development** - Overall backend patterns
- **Kubernetes Operator Development** - Operator-specific patterns
- **Security Standards** - Auth/authz details

---

## Changelog

- **1.0.0** (2026-01-27): Initial skill creation based on `.claude/patterns/error-handling.md`
