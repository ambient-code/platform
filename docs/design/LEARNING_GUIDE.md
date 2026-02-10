# Workspace RBAC & Quota System - Learning Guide

## 🎯 Purpose

This system adds **governance and quota management** to the Ambient Code Platform by introducing:

1. **Clear ownership** - Know who created each workspace
2. **Role-based access** - 5 tiers of permissions (Root → Owner → Admin → User → Viewer)
3. **Fair quota enforcement** - Platform-wide resource sharing via Kueue
4. **Safe deletions** - Prevent accidental workspace deletions
5. **Audit trail** - Track all permission changes

---

## 👥 Choose Your Learning Path

### For Project Managers / Non-Technical Users

**Understanding Roles (5 minutes)**

```
🔒 ROOT USER
   Purpose: Resolve disputes at platform level
   Example: "Approve Alice's request to transfer workspace to Bob"

👑 OWNER (Usually You)
   Purpose: You created the workspace, you control it
   Permissions: Invite team, promote admins, delete workspace
   Example: "Alice created the workspace, so Alice is OWNER"

🔑 ADMIN
   Purpose: Trusted teammates to manage the workspace
   Permissions: Create sessions, manage secrets, invite others
   Example: "Alice invited Bob as ADMIN to help run sessions"

✏️ USER / EDITOR
   Purpose: Team members who need to create sessions
   Permissions: Create sessions, work on them
   Example: "Charlie is a USER - can run sessions but can't invite others"

👁️ VIEWER
   Purpose: Stakeholders who need visibility
   Permissions: Read-only, see progress and results
   Example: "Manager watches session progress but can't change anything"
```

**Key Insight:** Owner > Admin > User > Viewer is like: CEO > Manager > Team Lead > Intern

---

### For Engineers / Technical Leads

**System Architecture (20 minutes)**

#### 1. What Changed?

**Before:** Only 3 roles, no ownership concept
```
ambient-project-view   ← Read-only
    ↓
ambient-project-edit   ← Create/update
    ↓
ambient-project-admin  ← Full control (no hierarchy)
```

**Now:** 5 roles with clear hierarchy and governance
```
🔒 ROOT (platform-level)
👑 OWNER (workspace-level, special)
🔑 ADMIN (workspace-level, multiple allowed)
✏️ USER (workspace-level)
👁️ VIEWER (workspace-level)
```

#### 2. Implementation - ProjectSettings CR Enhanced

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: my-workspace
spec:
  # GOVERNANCE (NEW)
  owner: "alice@company.com"           # Who created the workspace
  adminUsers:                          # Others who can manage
    - "bob@company.com"
    - "charlie@company.com"
  
  # QUOTA (NEW)
  quota:
    maxConcurrentSessions: 5           # Limit running sessions
    maxSessionDurationMinutes: 480     # 8-hour max per session
    maxStorageGB: 100                  # Total storage allowed
    cpuLimit: "4"                      # Resource limits
    memoryLimit: "8Gi"

status:
  # AUDIT TRAIL (NEW)
  createdAt: "2025-01-15T10:30:00Z"
  createdBy: "alice@company.com"
  lastModifiedAt: "2025-02-10T14:22:00Z"
  lastModifiedBy: "alice@company.com"
  
  # RBAC STATUS (NEW)
  adminRoleBindingsCreated:
    - "ambient-permission-admin-bob-user"
    - "ambient-permission-admin-charlie-user"
```

#### 3. Workflow: Add Admin

```
OWNER clicks "Add Admin: bob@company.com"
  ↓
Backend validates: Is alice the owner?
  ↓
Backend updates ProjectSettings.spec.adminUsers += "bob"
  ↓
Operator watches ProjectSettings change
  ↓
Operator creates RoleBinding: bob → ambient-project-admin
  ↓
Bob can now create sessions (K8s RBAC + frontend enforces)
  ↓
ProjectSettings.status.adminRoleBindingsCreated updated
```

#### 4. Kueue Integration

**What is Kueue?** Kubernetes queue management that prevents resource starvation

**How it works:**
```
ResourceFlavors (cluster-level resources)
    ↓
ClusterQueues (pool usage: 20% dev, 70% prod)
    ↓
LocalQueues (workspace-level: "my-workspace/dev")
    ↓
Sessions submit as Workloads
    ↓
Kueue schedules in FIFO order, respecting quotas
```

**Result:** No single workspace can starve others; fair-share allocation

#### 5. Delete Safety

```
OWNER clicks "Delete Workspace: my-workspace"
  ↓
Frontend dialog: "Type workspace name to confirm: ______"
  ↓
OWNER types: "my-workspace"
  ↓
Backend validates: Type matches name
  ↓
Backend validates: User is OWNER
  ↓
Emit Langfuse trace: workspace_deleted
  ↓
Delete namespace (cascades: Sessions, Jobs, PVCs)
  ↓
✅ Workspace gone but audit trail persists
```

**Why?** Prevent accidental `DELETE` command mishaps

---

### For Platform Operators

**Deployment & Configuration (15 minutes)**

#### Prerequisites

1. **Kueue must be installed**
   ```bash
   helm install kueue kueue/kueue
   ```

2. **Configure ResourceFlavors** (cluster resources available)
   ```yaml
   apiVersion: kueue.x-k8s.io/v1beta1
   kind: ResourceFlavor
   metadata:
     name: cpu-large
   spec:
     nodeLabels:
       kubernetes.io/instance-type: "large"
   ```

3. **Configure ClusterQueues** (quota buckets)
   ```yaml
   apiVersion: kueue.x-k8s.io/v1beta1
   kind: ClusterQueue
   metadata:
     name: dev-queue
   spec:
     maxRunningWorkloads: 50
     borrowingLimit: "50%"  # Can borrow from prod on weekend
     flavors:
       - name: cpu-large
         quota:
           - min: "4"
               max: "16"
   ```

#### Operator Responsibilities

When ProjectSettings.spec.adminUsers changes:

1. **Watch for changes** (operator reads ProjectSettings)
2. **Validate** (email format, not duplicate, etc.)
3. **Create/Delete RoleBindings** (use Operator service account)
4. **Update status** (adminRoleBindingsCreated list)
5. **Emit traces** (Langfuse for audit)

When ProjectSettings.spec.quota changes:

1. **Validate** (quotas are reasonable, Kueue supports them)
2. **Reconcile LocalQueue** (update maxRunningWorkloads, etc.)
3. **Emit Langfuse trace** (quota_changed)

#### Monitoring

```bash
# Check workspace quotas
kubectl get projectsettings -A

# Check admin RoleBindings created
kubectl describe ps projectsettings -n my-workspace

# Check Kueue workloads
kubectl get workloads -A

# Check Langfuse traces
# (Use Langfuse dashboard)
```

---

## 📊 Permission Matrix Deep Dive

| Operation | Root | Owner | Admin | User | Viewer |
|-----------|------|-------|-------|------|--------|
| **View Sessions** | ✓ | ✓ | ✓ | ✓ | ✓ |
| **Create Session** | ✗ | ✓ | ✓ | ✓ | ✗ |
| **Delete Session** | ✗ | ✓ | ✓ | ✗ | ✗ |
| **Edit Secrets** | ✗ | ✓ | ✓ | ✗ | ✗ |
| **View Audit Log** | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Add Admin** | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Remove Admin** | ✓ | ✓ | ✗ | ✗ | ✗ |
| **Delete Workspace** | ✗ | ✓ | ✗ | ✗ | ✗ |
| **Transfer Workspace** | ✓* | ✓† | ✗ | ✗ | ✗ |

*Root approves transfers | †Owner can request transfers

**Key:** 
- Upper roles have ALL permissions of lower roles
- Owner can do everything except transfer (must ask Root)
- Admin cannot manage RBAC or delete workspace

---

## 🔐 Kubernetes RBAC - How It Maps

```
┌─────────────────────────────────────────────────────────┐
│ ProjectSettings CR (Governance)                         │
│ owner: alice@company.com                                │
│ adminUsers: [bob@company.com]                           │
└─────────────────────────────────────────────────────────┘
                        ↓
        ┌───────────────┴───────────────┐
        ↓                               ↓
┌──────────────────────────┐    ┌──────────────────────────┐
│ RoleBinding: alice       │    │ RoleBinding: bob         │
│ → ambient-project-admin  │    │ → ambient-project-admin  │
└──────────────────────────┘    └──────────────────────────┘
        ↓                               ↓
        └───────────────┬───────────────┘
                        ↓
    ┌────────────────────────────────────┐
    │ ClusterRole: ambient-project-admin │
    │ verbs: [create, delete, update, ..] │
    └────────────────────────────────────┘
```

**What This Means:**
1. ProjectSettings is the source of truth (governance)
2. Operator creates RoleBindings based on ProjectSettings
3. K8s RBAC enforces the actual permissions
4. If ProjectSettings says alice is admin, she gets ambient-project-admin

---

## 🔄 Common Scenarios

### Scenario 1: Alice Creates Workspace

```
1. Alice: "Create Workspace: project-x"
2. Backend:
   - Creates namespace: project-x
   - Creates ProjectSettings with owner: alice
   - Creates RoleBinding: alice → ambient-project-admin
3. Operator:
   - Watches ProjectSettings
   - Confirms RoleBinding exists
4. Result:
   ✅ Alice is OWNER of project-x
   ✅ Alice can invite others
   ✅ Workspace ready to use
```

### Scenario 2: Alice Invites Bob as Admin

```
1. Alice: "Add Admin: bob@company.com"
2. Backend:
   - Validates: Is alice the owner? YES
   - Updates ProjectSettings.spec.adminUsers += bob
3. Operator:
   - Detects change
   - Creates RoleBinding: bob → ambient-project-admin
4. Result:
   ✅ Bob is now ADMIN
   ✅ Bob can create sessions, invite others
   ✅ BUT Bob cannot delete workspace or remove Alice as owner
```

### Scenario 3: Alice Deletes Workspace

```
1. Alice: "Delete Workspace"
2. Frontend: "Type workspace name: project-x"
3. Alice: "project-x" (types it correctly)
4. Backend:
   - Validates: Is alice the owner? YES
   - Validates: Type matches name? YES
   - Deletes namespace (cascades all resources)
   - Emit Langfuse: workspace_deleted
5. Result:
   ✅ Workspace deleted
   ✅ All sessions, jobs, PVCs cleaned up
   ✅ Audit trail shows who deleted when
```

### Scenario 4: Bob Tries to Delete Workspace (Should Fail)

```
1. Bob: "Delete Workspace"
2. Frontend: "Type workspace name: project-x"
3. Bob: "project-x" (types it correctly)
4. Backend:
   - Validates: Is bob the owner? NO (he's ADMIN)
   - Returns: 403 Forbidden
5. Result:
   ❌ Bob cannot delete (admin, not owner)
   ✅ Workspace protected
```

---

## 📈 Implementation Phases

### Phase 1 (MVP) - 8-10 Weeks
- ✅ Owner field in ProjectSettings (immutable)
- ✅ Admin management (add/remove admins)
- ✅ Audit trail (createdBy, lastModifiedBy, timestamps)
- ✅ Kueue integration (quota enforcement)
- ✅ Delete workspace safety confirmation
- ✅ Langfuse tracing for critical operations
- ✅ Full e2e tests and UI

### Phase 2 (Later)
- ❌ Workspace transfer (Owner → New Owner via Root approval)
- ❌ Advanced quota policies (time-based, cost-based limits)
- ❌ Cost attribution and chargeback
- ❌ Workspace templates and defaults

---

## 🧪 Testing Strategy

### Unit Tests (Backend)
```go
// Test owner is immutable
func TestOwnerImmutable(t *testing.T) {
    // Create workspace with alice as owner
    // Try to change to bob
    // Should fail
}

// Test admin management
func TestAddAdmin(t *testing.T) {
    // Alice (owner) adds bob (user) as admin
    // Check RoleBinding created
    // Bob can now create sessions
}

// Test quota enforcement
func TestQuotaExceeded(t *testing.T) {
    // Create 5 sessions (at limit)
    // Try to create 6th
    // Should fail: quota exceeded
}
```

### E2E Tests (Frontend + Backend)
```
Scenario: Create workspace, invite team, create session
1. Alice creates workspace "proj-x"
2. Alice adds bob as admin, charlie as user, dave as viewer
3. Bob creates session (should succeed)
4. Dave creates session (should fail - viewer role)
5. Alice deletes workspace with confirmation
6. Verify audit trail shows all changes
```

---

## 🔗 Related Documentation

- [WORKSPACE_RBAC_AND_QUOTA_DESIGN.md](WORKSPACE_RBAC_AND_QUOTA_DESIGN.md) - Complete technical spec (90+ min read)
- [MVP_IMPLEMENTATION_CHECKLIST.md](MVP_IMPLEMENTATION_CHECKLIST.md) - Week-by-week tasks (30 min read)
- [ROLES_VS_OWNER_HIERARCHY.md](ROLES_VS_OWNER_HIERARCHY.md) - Governance deep-dive (20 min read)
- [QUICK_REFERENCE.md](QUICK_REFERENCE.md) - API endpoints, CRD schema cheat sheet (10 min read)
- [ARCHITECTURE_DIAGRAMS.md](ARCHITECTURE_DIAGRAMS.md) - Visual diagrams (this file you just read)

---

## 💾 Quick Summary

| Aspect | Value |
|--------|-------|
| **Roles** | 5-tier: Root → Owner → Admin → User → Viewer |
| **Ownership** | Immutable after creation |
| **Admins** | Multiple allowed, managed by Owner |
| **Quota** | Per-workspace max concurrent sessions, duration, storage |
| **Kueue** | Fair-share queue management across all workspaces |
| **Audit** | CreatedAt, CreatedBy, LastModifiedAt, LastModifiedBy |
| **Safety** | Delete requires name confirmation |
| **Phases** | Phase 1 complete system, Phase 2+ transfers + cost tracking |

---

## ❓ FAQ

**Q: Can an admin remove the owner?**
A: No. Only the Root user can remove/transfer the owner. This prevents chaos.

**Q: Can a workspace have no owner?**
A: No. But you can transfer ownership via Root approval (Phase 2).

**Q: What happens if all admins are removed?**
A: Owner can still manage (even without admin role). Owner = implicit admin.

**Q: How does Kueue prevent starvation?**
A: FIFO queue + maxRunningWorkloads per workspace limits hogging resources.

**Q: Can quota be changed after creation?**
A: Yes. Owner can update ProjectSettings.spec.quota anytime.

**Q: What if someone deletes the ProjectSettings CR?**
A: Operator will recreate it (it's managed by operator). Deletion is blocked by ownerReference.

**Q: How long until Phase 2 (transfers)?**
A: TBD - depends on Phase 1 velocity and feedback. Estimated ~3 months after Phase 1 ships.

---

## 🚀 Next Steps

1. **Understand the Hierarchy** - Review the permission diagrams above
2. **Read the Full Spec** - WORKSPACE_RBAC_AND_QUOTA_DESIGN.md takes 90 minutes but is complete
3. **Check Implementation Plan** - MVP_IMPLEMENTATION_CHECKLIST.md shows week-by-week tasks
4. **Ask Questions** - This is complex; clarify any role/permission gaps now
5. **Plan Architecture** - Identify backend, operator, frontend changes needed
6. **Start Building** - Phase 1 is scoped at 13 person-days; estimated 8-10 weeks

**Estimated Total Learning Time:** 90 minutes to full understanding
