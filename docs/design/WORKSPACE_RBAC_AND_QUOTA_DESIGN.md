# Workspace RBAC and Quota Management Design

**Status:** MVP Design Phase  
**Last Updated:** February 10, 2026  
**Audience:** Implementation team ready to build

---

## Executive Summary

This document establishes the complete permissions and quota hierarchy for the Ambient Code Platform, including:

1. **Permissions Model**: Root User → Owner → Admin → User → Viewer (5-tier hierarchy)
2. **ProjectSettings Enhancement**: Owner/admin tracking with audit trail
3. **Namespace quota integration**: First-class quota and policy enforcement using Kubernetes ResourceQuota & LimitRange
4. **Langfuse Tracing**: Critical operations emitted for observability
5. **Delete Safety**: Confirmation pattern with workspace name verification

**MVP Scope**: Phases 1-2 (Permissions + Delete + Quota enforcement already in Phase 1)  
**Phase 2+**: Project transfer, advanced quota policies, cost attribution

---

## Part 1: Understanding the Current 3-Tier RBAC Model

### Current State (Today)

The platform currently has **3 Kubernetes ClusterRoles** bound at namespace level via RoleBindings:

```
ambient-project-view   ← Read-only: list/get sessions, settings, monitor jobs
         ↓
ambient-project-edit   ← Create/update sessions, create secrets (excludes RBAC management)
         ↓
ambient-project-admin  ← Full CRUD on everything: sessions, settings, secrets, RBAC, job deletion
```

**How It's Used Today:**

Each project (namespace) has RoleBindings that assign users/groups to one of these roles:

```yaml
# Example: User alice has admin on project-x
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ambient-permission-admin-alice-user
  namespace: project-x
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambient-project-admin  # ← One of the 3 roles
subjects:
  - kind: User
    name: alice@company.com
```

**Handler Integration:**

The backend checks permissions in two ways:

1. **Implicit via GetK8sClientsForRequest()**: User's Kubernetes RBAC is enforced automatically
   - User tries to create session → K8s API denies if no `create` verb on agenticsessions
   - Backend code doesn't need to check — K8s does it

2. **Explicit via AddProjectPermission/RemoveProjectPermission**:
   - Only admin role can create/delete RoleBindings
   - Handler validates: `if user doesn't have ambient-project-admin, reject`

**What's Missing:**

- ❌ No concept of **who created** the workspace
- ❌ No **owner** distinct from admin  
- ❌ No **multiple independent admins** (you can't have 2 admins managing each other)
- ❌ No **hierarchy**: All 3 admins are equal; one admin can remove another
- ❌ No **root user** to resolve disputes/transfers

---

## Part 2: New Permissions Model (5-Tier Hierarchy)

### Conceptual Hierarchy

```
┌─────────────────────────────────────────────────────────────┐
│ 🔒 ROOT USER (Platform Level)                              │
│ • Accepts workspace transfer requests                       │
│ • Resolves disputes/emergency access                        │
│ • Cannot delete workspaces (audit trail preserved)          │
───────────────────────────────────────────────────────────────│
│ 👑 OWNER (Workspace Level)                                 │
│ • Created workspace OR transferred to them                  │
│ • Can add/remove admins                                     │
│ • Can delete workspace (with confirmation)                  │
│ • Can view all audit logs                                   │
│ • Automatic implicit admin role (without RoleBinding)       │
───────────────────────────────────────────────────────────────│
│ 🔑 ADMIN (Workspace Level)                                 │
│ • Managed by owner(s)                                       │
│ • Can do everything except manage admins/delete workspace   │
│ • 1+ admins can exist per workspace                         │
│ • Maps to ambient-project-admin ClusterRole (unchanged)     │
───────────────────────────────────────────────────────────────│
│ ✏️ USER/EDITOR (Workspace Level)                           │
│ • Can create and edit sessions, workflows                   │
│ • Cannot manage RBAC, delete sessions, view secrets         │
│ • Maps to ambient-project-edit ClusterRole (unchanged)      │
───────────────────────────────────────────────────────────────│
│ 👁️ VIEWER (Workspace Level)                               │
│ • Read-only access                                          │
│ • Can monitor progress, view results                        │
│ • Maps to ambient-project-view ClusterRole (unchanged)      │
└─────────────────────────────────────────────────────────────┘
```

### Permission Matrix

| Operation | Root | Owner | Admin | User | Viewer |
|-----------|------|-------|-------|------|--------|
| **View workspace+sessions** | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Create session** | ✗ | ✓ | ✓ | ✓ | ✗ |
| **Delete session** | ✗ | ✓ | ✓ | ✗ | ✗ |
| **Manage secrets** | ✗ | ✓ | ✓ | ✗ | ✗ |
| **View audit log** | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Add admin** | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Remove admin** | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Delete workspace** | ✗ | ✓ | ✗ | ✗ | ✗ |
| **Transfer workspace** | ✓ | ✓* | ✗ | ✗ | ✗ |
| **Accept transfer** | ✓ | ✗ | ✗ | ✗ | ✗ |

*Owner can request transfer to another user; Root approves

### Typical Workflows

**Workspace Creation:**
```
User creates workspace → User becomes OWNER
Owner can immediately grant ADMIN to colleagues
Owner delegates session creation to ADMINs
Owner invites stakeholders as VIEWERs
```

**Admin Management:**
```
OWNER: "Add alice as admin"
  ↓
Backend: Add alice to ProjectSettings.spec.adminUsers
Backend: Create RoleBinding: alice → ambient-project-admin
Operator: Creates RoleBinding (idempotent)
✓ Alice can now create sessions, manage secrets, add more users
```

**Delete Workspace (Safety):**
```
OWNER clicks "Delete workspace"
  ↓
Dialog: "Type workspace name to confirm: ______"
OWNER types: "my-workspace"
  ↓
Backend DELETE /api/projects/my-workspace
  → Validate owner role
  → Emit Langfuse trace: "workspace_deleted"
  → Delete namespace (cascades all CRs, Jobs, PVCs)
  → Response: Audit entry created
```

**Workspace Transfer (Phase 2):**
```
OWNER: "Transfer to bob@company.com"
  ↓
ROOT USER receives notification
  ↓
ROOT approves/rejects transfer
  ↓
ProjectSettings.spec.owner = "bob@company.com"
  → Audit entry: "transferred_by: alice, to: bob"
  → alice loses owner permissions
  → bob gains owner permissions
```

---

## Part 3: ProjectSettings CR Enhancements

### Current Structure (Incomplete)

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: my-workspace
spec:
  groupAccess:
    - groupName: "engineering-team"
      role: "admin"
  defaultConfigRepo:
    gitUrl: "https://github.com/acme/defaults"
    branch: "main"
  # ❌ MISSING: Owner concept, admin tracking, audit trail
```

### Updated Structure (MVP)

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: my-workspace
  labels:
    ambient-code.io/managed: "true"
spec:
  # ============ OWNERSHIP & ADMIN MANAGEMENT ============
  owner: "alice@company.com"           # Immutable after creation
  
  adminUsers:                           # Mutable list of admins
    - "bob@company.com"
    - "charlie@company.com"
  
  # ============ GROUP-BASED ACCESS (EXISTING) ============
  groupAccess:
    - groupName: "engineering-team"
      role: "admin"
    - groupName: "product-team"
      role: "view"
  
  # ============ PROJECT METADATA ============
  displayName: "My Workspace"           # Human-friendly name
  description: "Frontend + Backend collab"
  
  # ============ QUOTA (NEW - Part of Phase 1) ============
  quota:
    maxConcurrentSessions: 5
    maxSessionDurationMinutes: 480      # 8 hours
    maxStorageGB: 100
    maxMonthlyTokens: 1000000
    cpuLimit: "4"                       # Kubernetes limit
    memoryLimit: "8Gi"
  
  # ============ DEFAULT CONFIG (EXISTING) ============
  defaultConfigRepo:
    gitUrl: "https://github.com/acme/defaults"
    branch: "main"
  
  # ============ NAMESPACE QUOTA REFERENCE (NEW - Phase 1) ============
  # quotaProfile maps to a predefined ResourceQuota + LimitRange profile
  quotaProfile: "development"   # Maps to a ResourceQuota/LimitRange example
  
  # ============ SETTINGS (FUTURE) ============
  # runnerSecretsName: "runner-config"   # Already used, not shown in this PR
  
status:
  # ============ RECONCILIATION STATUS ============
  observedGeneration: 5                 # Operator reconciliation gen
  phase: "Ready"                        # Ready | Error | Updating
  
  # ============ ADMIN ROLEBINDINGS ============
  adminRoleBindingsCreated:
    - "ambient-permission-admin-bob-user"
    - "ambient-permission-admin-charlie-user"
  
  # ============ AUDIT TRAIL ============
  createdAt: "2025-01-15T10:30:00Z"
  createdBy: "alice@company.com"
  lastModifiedAt: "2025-02-10T14:22:00Z"
  lastModifiedBy: "alice@company.com"  # Who made the last change
  
  # ============ OPERATIONAL STATUS ============
  lastReconcileTime: "2025-02-10T15:00:00Z"
  conditions:
    - type: "AdminsConfigured"
      status: "True"
      lastUpdateTime: "2025-02-10T15:00:00Z"
      reason: "AllAdminsActive"
      message: "All 2 admin RoleBindings created and active"
    - type: "NamespaceQuotaActive"
      status: "True"
      reason: "QuotaProfileExists"
      message: "Linked to quota profile 'development' (ResourceQuota/LimitRange)"
```

### CRD Schema Changes

```yaml
# Add these to ProjectSettings CRD
spec:
  type: object
  properties:
    owner:
      type: string
      description: "Email of workspace owner (immutable)"
      pattern: '^[^@]+@[^@]+$'
    
    adminUsers:
      type: array
      description: "List of admin email addresses"
      items:
        type: string
        pattern: '^[^@]+@[^@]+$'
    
    displayName:
      type: string
      maxLength: 255
    
    description:
      type: string
      maxLength: 1024
    
    quota:
      type: object
      properties:
        maxConcurrentSessions:
          type: integer
          minimum: 1
          maximum: 100
        maxSessionDurationMinutes:
          type: integer
          minimum: 5
          maximum: 2880  # 48 hours
        maxStorageGB:
          type: integer
          minimum: 1
          maximum: 10000
        maxMonthlyTokens:
          type: integer
          minimum: 100000
        cpuLimit:
          type: string
          pattern: '^[0-9]+m?$'  # e.g., "4", "2000m"
        memoryLimit:
          type: string
          pattern: '^[0-9]+(Mi|Gi)$'  # e.g., "8Gi"
    
    quotaProfile:
      type: string
      description: "References a predefined quota profile (maps to ResourceQuota + LimitRange)"

status:
  properties:
    adminRoleBindingsCreated:
      type: array
      items:
        type: string
    createdAt:
      type: string
      format: date-time
    createdBy:
      type: string
    lastModifiedAt:
      type: string
      format: date-time
    lastModifiedBy:
      type: string
```

---

## Part 4: Namespace quota integration (ResourceQuota + LimitRange)

### Why namespace quotas?

**Current State:**
- Kubernetes namespaces already provide strong primitives for resource limits (`ResourceQuota`, `LimitRange`) and for scoping resources by namespace.
- For MVP we prefer to use native Kubernetes primitives which are widely available and simpler to operate and maintain.

**This change means:**
- We will enforce per-workspace quotas using `ResourceQuota` and `LimitRange` on the namespace.
- The operator will reconcile `ProjectSettings.spec.quota` into namespace `ResourceQuota`/`LimitRange` objects.
- Multi-tenant fairness is handled by conservative default quotas per workspace (and reviewed by platform operators) rather than an external queueing system in Phase 1.

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ Namespace Quota Configuration                                 │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ResourceQuota (namespace-level total limits)                │
│    ├─ hard:
│    │   ├─ limits.cpu: "100"
│    │   ├─ limits.memory: "256Gi"
│    │   └─ persistentvolumeclaims: "100"
│                                                               │
│  LimitRange (per-pod min/max/defaults)                      │
│    ├─ default.requests.cpu: "200m"
│    ├─ default.requests.memory: "256Mi"
│    └─ default.limits.cpu: "4"
│                                                               │
│  ProjectSettings.spec.quota → reconciled into above objects  │
│                                                               │
└──────────────────────────────────────────────────────────────┘
                            ↓↓↓
       When user creates AgenticSession...
       ┌────────────────────────────────────────┐
       │ 1. Backend validates: user has create  │
       │    permission (RBAC)                   │
       │ 2. Backend creates AgenticSession CR   │
       │ 3. Operator creates Job/Pod in ns      │
       │ 4. K8s admission uses LimitRange/Quota │
       │    to enforce per-pod and namespace    │
       │    limits                              │
       │ 5. If limits exceeded, pod admission   │
       │    is rejected and backend returns 429│
       └────────────────────────────────────────┘
```

### User-facing: Quota Tiers (SaaS Mental Model)

Create preset quota profiles that teams can choose; the operator maps the chosen profile to `ResourceQuota` and `LimitRange` values:

```yaml
# Tier: Development (default for new workspaces)
name: development
spec:
  maxConcurrentSessions: 3
  maxSessionDurationMinutes: 120        # 2 hours
  maxStorageGB: 20
  cpuLimit: "2"
  memoryLimit: "4Gi"

# Tier: Production (for revenue-critical work)
name: production
spec:
  maxConcurrentSessions: 10
  maxSessionDurationMinutes: 480        # 8 hours
  maxStorageGB: 500
  cpuLimit: "8"
  memoryLimit: "32Gi"

# Tier: Unlimited (for platform team)
name: unlimited
spec:
  # No meaningful limits; based on physical cluster
  maxConcurrentSessions: 999
  maxSessionDurationMinutes: 43200      # 30 days
  maxStorageGB: 10000
  cpuLimit: "64"
  memoryLimit: "256Gi"
```

### Operator Responsibilities

**On ProjectSettings creation/update:**

```go
func reconcileProjectSettings(obj *unstructured.Unstructured) error {
  // 1. Compute desired ResourceQuota & LimitRange from spec.quota
  quota := getQuotaSpec(obj)

  // 2. Ensure ResourceQuota exists and matches desired limits
  ensureResourceQuota(namespace, quota)

  // 3. Ensure LimitRange exists with per-pod defaults/limits
  ensureLimitRange(namespace, quota)

  // 4. Ensure admin RoleBindings exist
  adminUsers := getAdminUsers(obj)
  for _, admin := range adminUsers {
    ensureAdminRoleBinding(namespace, admin)
  }

  // 5. Update status with reconciliation results
  updateStatus(namespace, map[string]interface{}{
    "phase": "Ready",
    "adminRoleBindingsCreated": []string{...},
    "namespaceQuotaProfile": quota.ProfileName,
  })

  return nil
}
```

**On AgenticSession creation:**

```go
func handleAgenticSessionCreated(session *unstructured.Unstructured) error {
  // 1. Get namespace ResourceQuota and LimitRange settings
  quota := getWorkspaceQuota(session.Namespace)

  // 2. Create Job/Pod with resource requests informed by quota
  podReqs := corev1.ResourceList{
    "cpu": resource.MustParse(quota.cpuLimit),
    "memory": resource.MustParse(quota.memoryLimit),
  }

  // 3. Create Job; if namespace ResourceQuota prevents admission,
  //    pod admission will fail and backend should report quota exceeded
  createJobWithRequests(session, podReqs)

  return nil
}
```

### Quota Enforcement Points

| Component | What It Enforces | Mechanism |
|-----------|-----------------|-----------|
| **Kubernetes ResourceQuota** | Namespace totals (cpu, memory, PVC count/size) | K8s admission control |
| **Kubernetes LimitRange** | Per-pod min/max/default CPU/Memory | Pod admission defaults/limits |
| **Operator** | Reconcile ProjectSettings → ResourceQuota/LimitRange | Create/update namespace objects |
| **Backend** | Role-based creation (who can create) | RBAC + permission checks |
| **Langfuse** | Token budget per workspace | Trace emission + analytics |

---

## Part 5: Langfuse Integration (Observability)

### Critical Operations to Trace

These should emit traces **immediately** (Phase 1):

```
PROJECT LIFECYCLE:
  ✓ project_created(owner, name, tier)
  ✓ project_deleted(owner, name, reason, audit_id)
  ✓ admin_added(workspace, by_who, added_who)
  ✓ admin_removed(workspace, by_who, removed_who)
  ✓ permissions_changed(workspace, by_who, change_type)

SESSION LIFECYCLE:
  ✓ session_created(workspace, creator, repo_count, timeout_minutes)
  ✓ session_started(workspace, session_id, model, token_estimate)
  ✓ session_completed(workspace, session_id, duration_seconds, tokens_used, status)
  ✓ session_failed(workspace, session_id, error_code, error_msg)
  ✓ session_timeout(workspace, session_id, duration_minutes)

QUOTA EVENTS:
  ✓ quota_limit_exceeded(workspace, resource_type, requested, limit)
  ✓ quota_tier_changed(workspace, from_tier, to_tier, by_who)

QUOTA EVENTS:
  ✓ workload_queued(workspace, session_id, position_in_queue, wait_estimate)
  ✓ workload_admitted(workspace, session_id, available_resources)
  ✓ workload_preempted(workspace, session_id, reason, higher_priority_id)
```

### Lower Priority (Phase 2+):

```
AGENT-SPECIFIC:
  - agent_step_executed(agent_type, input_tokens, output_tokens)
  - tool_called(tool_name, status, duration_ms)
  - rfe_phase_completed(workflow_id, phase, duration_minutes)

INFRASTRUCTURE:
  - job_scheduled(job_id, node, cpu, memory)
  - pvc_allocated(workspace, size_gb)
  - resource_cleanup(workspace, freed_resources)

COST & USAGE:
  - token_cost_calculated(workspace, session_id, cost_usd, model)
  - monthly_quota_reset(workspace, month)
```

### Implementation Pattern

**Backend Handler (for project operations):**

```go
func DeleteProject(c *gin.Context) {
  projectName := c.Param("projectName")
  user := c.GetString("user_id")  // From auth middleware
  
  // 1. Validate owner
  reqK8s, _ := GetK8sClientsForRequest(c)
  isOwner, err := validateOwner(reqK8s, projectName, user)
  if !isOwner {
    c.JSON(http.StatusForbidden, ...)
    return
  }
  
  // 2. Delete namespace (cascades to all CRs, Jobs, PVCs)
  err := reqK8s.CoreV1().Namespaces().Delete(ctx, projectName, v1.DeleteOptions{})
  if err != nil {
    c.JSON(http.StatusInternalServerError, ...)
    return
  }
  
  // 3. Emit Langfuse trace IMMEDIATELY
  if langfuseEnabled() {
    emit_langfuse_trace(LangfuseTraceOptions{
      Name: "project_deleted",
      Input: map[string]interface{}{
        "project_name": projectName,
        "owner": user,
        "timestamp": time.Now().RFC3339,
      },
      Output: map[string]interface{}{
        "status": "deleted",
        "cascaded_deletions": map[string]interface{}{
          "sessions": 5,
          "jobs": 5,
          "pvcs": 5,
          "services": 2,
        },
      },
      Session_id: getSessionTraceID(),
      User_id: user,
    })
  }
  
  c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}
```

**Operator (for session lifecycle):**

```go
func handleSessionCreated(obj *unstructured.Unstructured) {
  // ... setup ...
  
  // Emit trace
  if langfuseEnabled() {
    emit_langfuse_trace(LangfuseTraceOptions{
      Name: "session_created",
      Input: map[string]interface{}{
        "prompt": "[REDACTED]",  // Masking enabled by default
        "model": "claude-3.5-sonnet",
        "timeout_minutes": getSessionTimeout(obj),
        "repos": len(getRepos(obj)),
      },
      Session_id: obj.Name,
      User_id: getSessionCreator(obj),
      Metadata: map[string]interface{}{
        "workspace": obj.Namespace,
        "mode": "batch_or_interactive",
      },
    })
  }
}
```

### Mask by Default Pattern

```go
// In observability.py or similar
func _privacy_masking_function(trace_event: dict) -> dict:
    """Redact sensitive message content while preserving metrics"""
    if "input" in trace_event:
        trace_event["input_tokens"] = len(trace_event["input"])
        if not trace_event.get("content"):  # Already redacted
            trace_event["input"] = "[REDACTED]"
    
    if "output" in trace_event:
        trace_event["output_tokens"] = len(trace_event["output"])
        if not trace_event.get("content"):
            trace_event["output"] = "[REDACTED]"
    
    return trace_event
```

---

## Part 6: Delete Project Safety Pattern

### User Flow

```
1. User clicks Delete button
   ↓
2. Modal appears: "Deleting 'my-workspace' is PERMANENT"
   ├─ ⚠️ Warning: All sessions, data, history deleted forever
   ├─ Info: 5 active sessions will be terminated
   ├─ Info: 45 GB storage will be freed
   └─ Input: "Type workspace name to confirm: ________"

3. User types: "my-workspace"
   ↓
4. Backend: DELETE /api/projects/my-workspace
   ├─ Verify user is owner
   ├─ Verify workspace name matches
   ├─ Delete namespace (cascades all K8s resources)
   ├─ Emit Langfuse trace (project_deleted event)
   └─ Return confirmation with deleted resource counts

5. UI shows: "Workspace deleted successfully"
   └─ Redirect to projects list (should no longer exist)
```

### Delete Endpoint Implementation

```go
// DELETE /api/projects/:projectName
func DeleteProject(c *gin.Context) {
  projectName := c.Param("projectName")
  
  var req struct {
    ConfirmationName string `json:"confirmationName" binding:"required"`
  }
  if err := c.ShouldBindJSON(&req); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": "confirmationName required"})
    return
  }
  
  // 1. Verify owner role
  reqK8s, _ := GetK8sClientsForRequest(c)
  if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
    return
  }
  
  isOwner, err := isProjectOwner(reqK8s, projectName, c.GetString("user_id"))
  if !isOwner {
    c.JSON(http.StatusForbidden, gin.H{"error": "Only owner can delete"})
    return
  }
  
  // 2. Verify confirmation name matches
  if req.ConfirmationName != projectName {
    c.JSON(http.StatusBadRequest, gin.H{"error": "Workspace name mismatch"})
    return
  }
  
  // 3. Get resource counts before deletion (for audit)
  sessions, _ := countAgenticSessions(reqK8s, projectName)
  jobs, _ := countJobs(reqK8s, projectName)
  
  // 4. Delete namespace (cascades to all child resources)
  err = reqK8s.CoreV1().Namespaces().Delete(ctx, projectName, 
    &v1.DeleteOptions{GracePeriodSeconds: boolPtr(30)})
  if err != nil {
    log.Printf("Failed to delete project %s: %v", projectName, err)
    c.JSON(http.StatusInternalServerError, 
      gin.H{"error": "Failed to delete project"})
    return
  }
  
  // 5. Emit Langfuse trace
  if langfuseEnabled() {
    emitLangfuseTrace(LangfuseTrace{
      Name: "project_deleted",
      Input: map[string]interface{}{
        "project_name": projectName,
      },
      Output: map[string]interface{}{
        "status": "deleted",
        "deleted_sessions": sessions,
        "deleted_jobs": jobs,
        "timestamp": time.Now().RFC3339,
      },
      UserId: c.GetString("user_id"),
    })
  }
  
  // 6. Return confirmation
  c.JSON(http.StatusOK, gin.H{
    "message": "Workspace deleted",
    "project": projectName,
    "deleted_sessions": sessions,
    "deleted_jobs": jobs,
  })
}
```

### Frontend (Confirmation Dialog)

```typescript
// React component
export const DeleteProjectDialog = ({ projectName, onConfirm }) => {
  const [confirmationName, setConfirmationName] = useState("");
  const isValid = confirmationName === projectName;
  
  return (
    <Dialog>
      <DialogHeader>Delete Workspace</DialogHeader>
      <DialogContent>
        <Alert variant="destructive">
          <AlertTriangle className="w-4 h-4" />
          <AlertTitle>This action cannot be undone</AlertTitle>
          <AlertDescription>
            All sessions, data, and history will be permanently deleted.
          </AlertDescription>
        </Alert>
        
        <div className="space-y-4">
          <p>
            To confirm deletion, type the workspace name:
            <strong className="block mt-2">{projectName}</strong>
          </p>
          <Input
            placeholder="Type workspace name..."
            value={confirmationName}
            onChange={(e) => setConfirmationName(e.target.value)}
            autoFocus
          />
        </div>
      </DialogContent>
      <DialogFooter>
        <Button variant="ghost" onClick={() => /* close */}>
          Cancel
        </Button>
        <Button
          variant="destructive"
          disabled={!isValid}
          onClick={() => onConfirm(confirmationName)}
        >
          Delete Workspace Permanently
        </Button>
      </DialogFooter>
    </Dialog>
  );
};
```

---

## Part 7: MVP Implementation Phases

### Phase 1: Core Permissions + Delete + Quota (8-10 weeks)

**Week 1-2: Foundation**
- [ ] Update ProjectSettings CRD (owner, adminUsers, quota, quotaProfile)
- [ ] Update operator reconciliation (create admin RoleBindings, create/maintain ResourceQuota & LimitRange)
- [ ] Update backend handlers (validate owner, add admin, remove admin)
- [ ] Add Langfuse trace emission (project lifecycle + session lifecycle)

**Week 2-3: Delete Safety**
- [ ] Add DELETE /api/projects/:projectName handler with confirmation
- [ ] Add delete confirmation dialog to frontend
- [ ] E2E test delete flow with confirmation

**Week 3-4: Namespace quota integration**
- [ ] Prepare ResourceQuota and LimitRange examples for each quota tier
- [ ] Operator creates/updates ResourceQuota & LimitRange per workspace based on `spec.quotaProfile`
- [ ] AgenticSession handler relies on Kubernetes admission for quota enforcement; backend emits quota traces

**Week 4-5: Quota Enforcement**
- [ ] Operator monitors Workload admission
- [ ] Emit Langfuse trace: "quota_limit_exceeded"
- [ ] UI shows queue position when workload is queued
- [ ] Tests for quota limits

**Week 5-6: Migration**
- [ ] Script to migrate existing projects (set owner to creator, empty adminUsers)
- [ ] Operator reconciliation catches up to old projects
- [ ] Backward compat: Old projects without owner get default (first admin or platform owner)

**Week 6-7: Audit Trail**
- [ ] Update ProjectSettings status (createdAt, createdBy, lastModifiedAt, etc.)
- [ ] Operator maintains audit trail
- [ ] Backend returns audit trail in GetProjectSettings response

**Week 7-8: Testing & Polish**
- [ ] Unit tests (handlers, operators, permissions)
- [ ] Integration tests (RBAC + NamespaceQuota interaction)
- [ ] E2E tests (create → add admin → delete flow)
- [ ] Performance testing (parallel quota checks)

**Week 8-10: Documentation & Deployment**
- [ ] Update ADRs and context files
- [ ] Change `components/manifests/base/rbac/README.md`
- [ ] Write deployment guide for Namespace ResourceQuota / LimitRange (examples, runbook)
- [ ] Write admin/owner runbook

### Phase 2: Project Transfer + Root User (4-6 weeks)

**Goals:**
- [ ] OWNER can request transfer to another user
- [ ] ROOT USER can approve/reject transfers
- [ ] Audit trail tracks all transfers
- [ ] Longfuse trace: "project_transferred"

**New Endpoints:**
- POST /admin/transfer-requests (owner requests)
- GET /admin/transfer-requests (root lists pending)
- POST /admin/transfer-requests/:id/approve
- POST /admin/transfer-requests/:id/reject

**Root User Discovery:**
- Read from environment: `PLATFORM_ROOT_USER=platform-admin@company.com`
- Or lookup system group: `system:cluster-admins`

**New CRD: TransferRequest (optional)**
```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: TransferRequest
metadata:
  name: transfer-my-workspace-to-bob
spec:
  workspace: "my-workspace"
  requestedBy: "alice@company.com"
  targetUser: "bob@company.com"
  reason: "Leaving team, transferring to new owner"
  createdAt: "2025-02-10T15:00:00Z"
status:
  status: "pending"  # pending | approved | rejected
  approvedBy: ""
  approvalTime: ""
  rejectionReason: ""
```

### Phase 3+: Advanced Quota & Cost Attribution

**Future goals:**
- [ ] Tiered pricing (dev tier = free, prod tier = $X/month)
- [ ] Cost attribution per workspace
- [ ] Reserved quota (prepaid capacity)
- [ ] Burst quota (overflow with backpressure)
- [ ] Cost alerts and usage dashboard
- [ ] Chargeback reports

---

## Part 8: Root User Responsibilities

### Who is Root?

```
Option 1: Environment Variable (Simplest)
  PLATFORM_ROOT_USER=platform-admin@company.com

Option 2: Group-Based (Scales Better)
  system:cluster-admins (from OAuth/OpenShift)
  
Option 3: ClusterRole-Based (Most Explicit)
  ambient-platform-root (new ClusterRole)
```

**Recommendation for MVP**: Use environment variable + group fallback

### Root User Endpoint

```go
// GET /api/admin/system-info
// Returns info about root users (no auth required for discovery)
func GetSystemInfo(c *gin.Context) {
  c.JSON(http.StatusOK, gin.H{
    "rootUsers": []string{
      os.Getenv("PLATFORM_ROOT_USER"),
    },
    "namespaceQuotaEnabled": isNamespaceQuotaEnabled(),
    "langfuseEnabled": isLangfuseEnabled(),
  })
}

// GET /api/admin/pending-transfers
// Lists pending transfer requests (root user only)
func ListPendingTransfers(c *gin.Context) {
  if !isRootUser(c) {
    c.JSON(http.StatusForbidden, gin.H{"error": "Root user only"})
    return
  }
  
  // Return list of TransferRequest CRs (Phase 2)
  transfers, _ := listTransferRequests(c.Request.Context())
  c.JSON(http.StatusOK, gin.H{"transfers": transfers})
}
```

### Root User Capabilities

| Operation | Who Can Do | Notes |
|-----------|-----------|-------|
| View system metrics | Root + Platform ops | CPU usage, quota utilization |
| Adjust ClusterQueue limits | Root only | Redistribute quota between tiers |
| Approve project transfer | Root only | Only way to finalize transfer (Phase 2) |
| Override quota limits | Root only | Emergency access (logged + traced) |
| View all audit logs | Root only | Cross-workspace audit trail |
| Delete project (emergency) | Root only | If owner is unreachable |
| Create admin user | Root only | Bootstrap admin for new clusters |

---

## Part 9: Configuration Examples

### Tier Definition (Cluster-Level)

**File: `components/manifests/base/quotas/quota-tiers.yaml`**

```yaml
# Development Tier (Default)
apiVersion: vteam.ambient-code/v1alpha1
kind: QuotaTier
metadata:
  name: development
spec:
  displayName: "Development"
  description: "For prototyping and experimentation"
  maxConcurrentSessions: 3
  maxSessionDurationMinutes: 120
  maxStorageGB: 20
  maxMonthlyTokens: 100000
  cpuLimit: "2"
  memoryLimit: "4Gi"
  quotaProfileCluster: "development"

---
# Production Tier
apiVersion: vteam.ambient-code/v1alpha1
kind: QuotaTier
metadata:
  name: production
spec:
  displayName: "Production"
  description: "For revenue-critical and continuous workflows"
  maxConcurrentSessions: 10
  maxSessionDurationMinutes: 480
  maxStorageGB: 500
  maxMonthlyTokens: 5000000
  cpuLimit: "8"
  memoryLimit: "32Gi"
  quotaProfileCluster: "production"

---
# Unlimited Tier (Platform team only)
apiVersion: vteam.ambient-code/v1alpha1
kind: QuotaTier
metadata:
  name: unlimited
spec:
  displayName: "Unlimited"
  description: "For platform operations and testing"
  maxConcurrentSessions: 999
  maxSessionDurationMinutes: 43200  # 30 days
  maxStorageGB: 10000
  maxMonthlyTokens: 999999999
  cpuLimit: "64"
  memoryLimit: "256Gi"
  quotaProfileCluster: "unlimited"
```

### CreateProject with Tier Selection

**API Request:**

```json
POST /api/projects
{
  "name": "my-workspace",
  "displayName": "My Team Workspace",
  "description": "Frontend + Backend collaboration",
  "quotaTier": "development"  ← User selects tier
}
```

**Backend Handler:**

```go
func CreateProject(c *gin.Context) {
  var req struct {
    Name string `json:"name" binding:"required"`
    DisplayName string `json:"displayName"`
    QuotaTier string `json:"quotaTier"`  // "development" | "production" | etc.
  }
  c.ShouldBindJSON(&req)
  
  // Default tier if not specified
  if req.QuotaTier == "" {
    req.QuotaTier = "development"
  }
  
  // 1. Create namespace
  ns := &corev1.Namespace{...}
  K8sClient.CoreV1().Namespaces().Create(...)
  
  // 2. Create ProjectSettings with owner + tier
  quotaTier := getQuotaTier(req.QuotaTier)  // Load QuotaTier CR
  ps := &ProjectSettings{
    Spec: ProjectSettingsSpec{
      Owner: c.GetString("user_id"),
      AdminUsers: []string{c.GetString("user_id")},  // Owner is auto-admin
      DisplayName: req.DisplayName,
      Quota: quotaTier.Spec,
      QuotaProfile: req.QuotaTier,
    },
  }
  DynamicClient.Resource(projectSettingsGVR).Namespace(req.Name).Create(...)
  
  // 3. Emit Langfuse trace
  emitLangfuseTrace(LangfuseTrace{
    Name: "project_created",
    Input: map[string]interface{}{
      "name": req.Name,
      "tier": req.QuotaTier,
    },
    UserId: c.GetString("user_id"),
  })
  
  c.JSON(http.StatusCreated, gin.H{"project": req.Name})
}
```

---

## Part 10: Backward Compatibility & Migration

### Handling Existing Projects (No Owner)

**Script: `scripts/migrate-projectsettings.sh`**

```bash
#!/bin/bash
# Migrates existing ProjectSettings CRs to include owner/admins

# List all ProjectSettings without owner
kubectl get projectsettings --all-namespaces -o json | \
  jq '.items[] | select(.spec.owner == null)'

# For each ProjectSettings:
# 1. Find who has admin RoleBinding
# 2. Promote first admin as owner
# 3. Keep others as admins (in spec.adminUsers)
# 4. Set createdAt to now (or K8s creation timestamp if available)

for ps in $(kubectl get projectsettings -A | tail -n +2); do
  ns=$(echo $ps | awk '{print $1}')
  
  # Find admins from RoleBindings
  admins=$(kubectl get rolebindings -n $ns \
    -l "app=ambient-permission" \
    -o jsonpath='{.items[?(@.roleRef.name=="ambient-project-admin")].subjects[*].name}')
  
  if [ -z "$admins" ]; then
    echo "Warning: No admins found for $ns, skipping"
    continue
  fi
  
  # Set first admin as owner
  owner=$(echo $admins | awk '{print $1}')
  
  # Patch ProjectSettings
  kubectl patch projectsettings -n $ns projectsettings \
    --type merge \
    -p "{\"spec\": {\"owner\": \"$owner\"}}"
    
  echo "✓ Migrated $ns, owner=$owner"
done
```

### Operator Reconciliation (Idempotent)

**When handling existing ProjectSettings:**

```go
// If owner is empty (old CR), don't fail
// Just log warning and continue
if owner == "" {
  log.Printf("Warning: ProjectSettings in %s has no owner (legacy?)", ns)
  // Don't create OwnerReference or do anything special
  // Just ensure admin RoleBindings exist
}

// Always reconcile admin RoleBindings (idempotent)
for _, admin := range spec.AdminUsers {
  ensureAdminRoleBinding(ns, admin)
}

// If adminUsers is empty, try to infer from existing RoleBindings
if len(spec.AdminUsers) == 0 {
  inferred := inferAdminsFromRoleBindings(ns)
  log.Printf("Inferred admins from RoleBindings: %v", inferred)
  // Still create the RoleBindings (they already exist)
}
```

---

## Summary: The Rights Model at a Glance

```
👑 OWNER
  ├─ Can add/remove admins
  ├─ Can delete workspace
  ├─ Can view audit log
  └─ Receives transfer requests (Phase 2)

🔑 ADMIN (one or more)
  ├─ Can create/delete sessions
  ├─ Can manage secrets
  ├─ Cannot manage admins
  └─ Cannot delete workspace

✏️ USER/EDITOR
  ├─ Can create sessions
  ├─ Cannot delete sessions
  └─ Cannot manage anyone

👁️ VIEWER
  ├─ Can read everything
  └─ Cannot create anything

🔒 ROOT USER (Platform)
  ├─ Approves transfers (Phase 2)
  ├─ Adjusts cluster quotas
  └─ Emergency access only
```

---

## Files to Create/Modify (MVP)

```
NEW CRDS:
  ✓ components/manifests/base/quotas/quota-tiers.yaml

NEW MANIFESTS:
  ✓ components/manifests/quota/namespace-resourcequota.yaml
  ✓ components/manifests/quota/namespace-limitrange.yaml (per-project)
  ✓ components/manifests/quota/README.md (examples)

MODIFIED FILES:
  ✓ components/manifests/base/crds/projectsettings-crd.yaml (enhance schema)
  ✓ components/backend/types/common.go (ProjectSettings types)
  ✓ components/backend/handlers/projects.go (DeleteProject endpoint)
  ✓ components/backend/handlers/project_settings.go (new endpoints for admins)
  ✓ components/backend/handlers/permissions.go (verify owner for delete)
  ✓ components/operator/internal/handlers/projectsettings.go (reconcile admins + namespace quota)
  ✓ components/backend/observability.py (emit traces)
  ✓ components/frontend/src/pages/projects/[name]/settings.tsx (admin/delete UI)

SCRIPTS:
  ✓ scripts/migrate-projectsettings.sh (one-time migration)
```

**Total Scope: MVP implementation 8-10 weeks, fully scoped and ready to build.**
