# Permissions Model: Roles vs. Owner/Admin Hierarchy

**Quick Answer: What's the difference between the 3 roles (view/edit/admin) and the owner/admin concept in Phase 1?**

---

## Today: 3 ClusterRoles (Kubernetes RBAC Only)

```
Every user gets ONE of these roles per workspace:

┌─ ambient-project-view    (read-only)
├─ ambient-project-edit    (create sessions)
└─ ambient-project-admin   (delete sessions, manage RBAC)
```

**Created via**: RoleBindings (one per user)  
**How**: Backend creates automatically when user adds someone via `/permissions` endpoint  
**Enforcement**: Kubernetes RBAC (automatic, at API level)

**Problem**: No hierarchy. Multiple admins are equal. One admin can remove another. No "owner" concept.

---

## Phase 1 (Coming): Owner + Admin Hierarchy

```
On top of the 3 roles, add:

┌─ Owner (metadata in ProjectSettings.spec)
│   ├─ Can add/remove admins
│   ├─ Can delete workspace
│   └─ Can view audit logs
│
├─ Admin (list in ProjectSettings.spec.adminUsers)
│   ├─ Gets ambient-project-admin role automatically
│   ├─ Managed by owner
│   └─ Cannot add/remove other admins
│
├─ User (ambient-project-edit role)
│   ├─ Creates sessions
│   └─ Cannot manage RBAC
│
└─ Viewer (ambient-project-view role)
    └─ Read-only
```

**Created via**: Metadata in ProjectSettings CR + backend handlers  
**How**: Owner field (immutable), adminUsers list (mutable by owner)  
**Enforcement**: Both Kubernetes RBAC + backend permission checks

---

## How They Work Together

### Scenario 1: Alice Creates a Workspace

```
1. POST /api/projects
   → Backend creates namespace
   → Creates ProjectSettings CR with owner=alice
   → Creates RoleBinding: alice → ambient-project-admin
   
2. ProjectSettings state:
   spec:
     owner: alice@company.com
     adminUsers: []  # Empty; alice is owner, not in admin list
   
3. Kubernetes RoleBinding state:
   - amber-permission-admin-alice-user → ambient-project-admin
   
4. Alice's effective permissions:
   ✓ As OWNER: Can add admins, can delete workspace, can view audit logs
   ✓ As ADMIN (implicit): Can create/delete sessions (from ClusterRole)
```

### Scenario 2: Alice Adds Bob as Admin

```
1. POST /api/projects/my-workspace/admins
   body: { adminEmail: "bob@company.com" }
   
   Backend checks: Is alice the owner? YES ✓
   
2. Backend adds bob to ProjectSettings.spec.adminUsers:
   spec:
     owner: alice@company.com
     adminUsers: ["bob@company.com"]
   
3. Operator reconciles:
   - Sees bob in adminUsers list
   - Creates RoleBinding: bob → ambient-project-admin
   
4. Bob's effective permissions:
   ✓ As ADMIN: Can create/delete sessions
   ✗ NOT admin of admins: Cannot add/remove users (owner only)
   ✗ NOT owner: Cannot delete workspace
```

### Scenario 3: Bob (Admin) Tries to Add Charlie

```
1. POST /api/projects/my-workspace/admins
   body: { adminEmail: "charlie@company.com" }
   
   Backend checks: Is bob the owner?
      → Look up ProjectSettings.spec.owner
      → owner = alice, not bob
      → Response: 403 Forbidden "Only owner can add admins"
   
Bob is ADMIN (can do technical work) but NOT OWNER (cannot do governance work).
```

### Scenario 4: Alice Deletes Workspace

```
1. DELETE /api/projects/my-workspace
   header: { confirmationName: "my-workspace" }
   
2. Backend checks:
   - Is alice the owner? YES ✓
   - Confirmation name matches? YES ✓
   
3. Backend deletes namespace (cascades all resources)
   
4. Kubernetes cascade:
   - Namespace deleted
   - All RoleBindings deleted
   - All Jobs/Pods/PVCs deleted
   - ProjectSettings CR deleted
   
5. Emit Langfuse trace: project_deleted
```

---

## The 3 Roles (Unchanged from Today)

These continue to exist and enforce **technical permissions** (who can do what operation):

| Role | User Permission | Edit Permission | Admin Permission |
|------|-----------------|-----------------|------------------|
| **ambient-project-view** | List sessions | No | No |
| **ambient-project-edit** | Create sessions, create secrets | Yes | No |
| **ambient-project-admin** | Delete sessions, modify RBAC, view secrets | Yes | Yes |

**How you get a role**: Owner adds you via the admin management API OR inherited from group membership

**Who enforces**: Kubernetes (every API call checked against ClusterRole)

---

## The Owner/Admin Fields (New in Phase 1)

These control **governance permissions** (who can manage the workspace):

| Field | Example | Who Sets | Who Can Change |
|-------|---------|----------|-----------------|
| **owner** | "alice@..." | Backend (on create) | Root user only (Phase 2 transfer) |
| **adminUsers** | ["bob@...", "charlie@..."] | Backend | OWNER only |

**How they work**: Stored in ProjectSettings.spec, used by backend handlers for permission checks

**Who enforces**: Backend (permission check before modifying RoleBindings, namespace ops)

---

## Three-Way Interaction Example

Alice (Owner) creates workspace → Adds Bob as Admin → Bob creates session → Alice deletes workspace

```
┌──────────────────────────────────────────────────────────────────────┐
│ ProjectSettings                                                        │
│                                                                       │
│  spec:                                                                │
│    owner: alice@company.com              ← Governance: who manages    │
│    adminUsers: ["bob@company.com"]       ← Governance: delegation     │
│    quota:                                ← Also governance             │
│      maxConcurrentSessions: 5                                         │
│                                                                       │
│  status:                                                              │
│    adminRoleBindingsCreated:                                          │
│      - "amber-permission-admin-bob-user" ← Link to technical RBAC     │
└──────────────────────────────────────────────────────────────────────┘
                            ↓↓↓ Operator watches this ↓↓↓
┌──────────────────────────────────────────────────────────────────────┐
│ RoleBindings (Kubernetes RBAC)                                        │
│                                                                       │
│  amber-permission-admin-bob-user:                                    │
│    roleRef: ambient-project-admin       ← Technical: what can do      │
│    subjects: [User: bob@company.com]                                  │
│                                                                       │
│  amber-permission-view-stakeholder-user:                             │
│    roleRef: ambient-project-view        ← Inherited from owner's add  │
│    subjects: [User: view-only@company.com]                           │
└──────────────────────────────────────────────────────────────────────┘
                            ↓↓↓ K8s checks this ↓↓↓
```

**Alice wants to delete workspace**:
- Backend checks: Is alice = owner? YES ✓ (governance, not RBAC)
- Backend deletes namespace
- K8s cascades: RoleBindings gone, no more technical permissions

**Bob tries to add new admin**:
- Backend checks: Is bob = owner? NO (governance check)
- Returns 403, operation rejected (never reaches K8s RBAC)

**Bob creates session**:
- Backend extracts bob's token
- K8s checks: Does bob's user have "create" verb on agenticsessions?
- K8s finds RoleBinding: bob → ambient-project-admin
- K8s checks ambient-project-admin: has "create"? YES ✓
- K8s approves (technical, automatic)

---

## Why Two Levels?

### Governance Level (ProjectSettings metadata)

**Why needed?**
- Immutable owner prevents accidental loss of workspace control
- Admins can't remove each other (owner is referee)
- Owner can make policy decisions (quota tier, who gets access)
- Audit trail: who created, who last modified

**Enforcement by**: Backend (custom code)  
**Example checks**: `if user != owner { return 403 }`

### Technical Level (Kubernetes RBAC)

**Why needed?**
- Automatic enforcement (no custom code to maintain)
- Integrates with K8s ecosystem (kubectl auth can-i, audit logs)
- Scales to 1000s of users without custom DB
- Fine-grained (verb-level: get, create, delete, etc.)

**Enforcement by**: Kubernetes (API server)  
**Example checks**: K8s checks ClusterRole for "create" verb

### They're Complementary

```
Governance Layer:
  "Is this person allowed to MANAGE this workspace?"
  → Checked by: Backend handler (owner validation)
  → Enforces: Who can add/remove users, delete workspace

Technical Layer:
  "Is this person allowed to RUN this operation?"
  → Checked by: Kubernetes API
  → Enforces: Who can create sessions, delete jobs, manage secrets
```

---

## Current vs. Phase 1 Behavior

### Today (Before Phase 1)

```
POST /api/projects/test-ws/admins
  body: { adminEmail: "new-admin@..." }
  
    ✓ Any admin can add users
    ✓ Users listed via RoleBindings only
    ✗ No owner concept
    ✗ No audit trail of who added whom
    ✗ Can't distinguish "operator" from "governance": all admins equal
```

### Phase 1 (After Implementation)

```
POST /api/projects/test-ws/admins
  body: { adminEmail: "new-admin@..." }
  
    ✓ Only OWNER can add users (checked at backend before K8s)
    ✓ Users listed in ProjectSettings.spec.adminUsers (permanent record)
    ✓ RoleBindings auto-created by operator (linked to spec)
    ✓ Audit trail: createdBy, lastModifiedBy, timestamp
    ✓ Clear roles: Owner does governance, Admin does execution
```

---

## Glossary

| Term | Definition | Location |
|------|-----------|----------|
| **ClusterRole** | Kubernetes resource defining verbs (create, delete, list) on resource types (sessions, secrets, jobs) | `components/manifests/base/rbac/*.yaml` |
| **RoleBinding** | Kubernetes resource linking user/group to a ClusterRole in a namespace | Created by backend dynamically |
| **Owner** | User who created workspace, can manage admins and delete workspace | `ProjectSettings.spec.owner` |
| **Admin** | User appointed by owner, has ambient-project-admin ClusterRole | `ProjectSettings.spec.adminUsers[]` |
| **User/Editor** | User with ambient-project-edit role, can create sessions | Implicit in RoleBinding |
| **Viewer** | User with ambient-project-view role, read-only | Implicit in RoleBinding |
| **Governance** | High-level decisions (owner, admins, quota tier, deletion) | Backend validation |
| **Technical** | Low-level permissions (create, delete, update verbs) | Kubernetes RBAC |

---

## FAQ

**Q: Do I need to change code when adding a new admin in Phase 1?**  
A: No. Backend automatically creates RoleBinding via operator reconciliation.

**Q: If I'm an admin, can I see who the owner is?**  
A: Yes, admins can call GET /projects/:name/admin-info (returns owner, admin list, audit trail).

**Q: Can there be multiple owners?**  
A: No, owner is singular (immutable). But multiple admins can exist (added by owner).

**Q: What happens if owner leaves?**  
A: Owner can add another admin before leaving. In Phase 2, can approve transfer to root user.

**Q: How do RoleBindings stay in sync with spec.adminUsers?**  
A: Operator watches ProjectSettings, reconciles RoleBindings idempotently.

**Q: What if backend and K8s disagree on permissions?**  
A: Backend check happens FIRST. If backend says "no" (governance), K8s never sees request.

**Q: Why not just use K8s RBAC for everything?**  
A: K8s RBAC is technical (create/delete/update). We need governance layer (owner/admin, policy, deletion approval).

---

## See Also

- **Complete design**: `docs/design/WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`
- **Implementation checklist**: `docs/design/MVP_IMPLEMENTATION_CHECKLIST.md`
- **RBAC manifest details**: `components/manifests/base/rbac/README.md`
- **Current roles**: `components/manifests/base/rbac/ambient-project-*.yaml`
