# Workspace RBAC & Quota System - Quick Slides

> 📊 Visual summary of the workspace governance and quota system proposal

---

## Slide 1: What Problem Does This Solve?

### Current State (❌ Problems)
```
❌ No clear ownership - Who created the workspace?
❌ All admins are equal - Can't distinguish leadership
❌ No fair quota - One workspace can hog all resources
❌ Risky deletes - Easy to accidentally delete workspace
❌ No audit trail - Can't track who changed what
```

### New State (✅ Solutions)
```
✅ Clear owner - Workspace creator = owner
✅ Hierarchy - Owner > Admin > User > Viewer
✅ Fair quota - Namespace ResourceQuota + LimitRange ensure fair sharing
✅ Safe delete - Requires name confirmation
✅ Full audit - Track createdBy, lastModifiedBy, timestamps
```

---

## Slide 2: The 5-Tier Permission Model

```
                    🔒 ROOT USER
                    (Platform Admin)
                         ↓
                    👑 OWNER  ← Typically you
                    (Workspace Creator)
                         ↓
                    🔑 ADMIN
                    (Trusted Teammates)
                         ↓
                    ✏️ USER/EDITOR
                    (Team Members)
                         ↓
                    👁️ VIEWER
                    (Stakeholders)
```

**Key:** Each role includes all permissions of roles below it

---

## Slide 3: What Can Each Role Do?

| Action | Root | Owner | Admin | User | Viewer |
|--------|------|-------|-------|------|--------|
| View sessions | ✅ | ✅ | ✅ | ✅ | ✅ |
| Create sessions | ❌ | ✅ | ✅ | ✅ | ❌ |
| Delete sessions | ❌ | ✅ | ✅ | ❌ | ❌ |
| **Manage admins** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Delete workspace** | ❌ | ✅ | ❌ | ❌ | ❌ |
| View audit log | ✅ | ✅ | ❌ | ❌ | ❌ |

**Key Actions are in bold** - Only Owner, Admin, or Root can do these

---

## Slide 4: Typical Team Setup

```
ALICE (Creator)
  ↓
  └─ Role: OWNER
     └─ Invites Bob and Charlie as ADMINS
        └─ Bob and Charlie:
           • Can create sessions
           • Can approve PRs
           • Can invite users
        └─ BUT cannot:
           • Delete workspace
           • Remove each other

DAVE (Team Member)
  ↓
  └─ Role: USER/EDITOR
     └─ Can create sessions
     └─ Can run workflows
     └─ Cannot invite or manage

EVE (Manager)
  ↓
  └─ Role: VIEWER
     └─ Can see progress
     └─ Can view results
     └─ Cannot make changes
```

---

## Slide 5: ProjectSettings - The Single Source of Truth

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: projectsettings
  namespace: my-workspace
spec:
  # WHO IS WHO?
  owner: "alice@company.com"
  adminUsers:
    - "bob@company.com"
    - "charlie@company.com"
  
  # LIMITS
  quota:
    maxConcurrentSessions: 5
    maxSessionDurationMinutes: 480
    maxStorageGB: 100
    cpuLimit: "4"
    memoryLimit: "8Gi"

status:
  # AUDIT TRAIL
  createdAt: "2025-01-15T10:30:00Z"
  createdBy: "alice@company.com"
  lastModifiedAt: "2025-02-10T14:22:00Z"
  lastModifiedBy: "alice@company.com"
  
  # RBAC STATUS
  adminRoleBindingsCreated:
    - "ambient-permission-admin-bob-user"
    - "ambient-permission-admin-charlie-user"
```

**This CR controls:** Who can do what + Resource limits + Audit trail

---

## Slide 6: Add Admin - Step by Step

```
Step 1: OWNER clicks "Add Admin: bob@company.com" in UI
                         ↓
Step 2: Backend validates "Am I the owner?" → YES ✅
                         ↓
Step 3: Backend updates ProjectSettings CR
        adminUsers: ["bob@company.com"]
                         ↓
Step 4: Operator watches ProjectSettings change
                         ↓
Step 5: Operator creates RoleBinding
        bob → ambient-project-admin
                         ↓
Step 6: Update ProjectSettings.status
        adminRoleBindingsCreated: ["bob-user"]
                         ↓
✅ Bob is now ADMIN - can create sessions, manage team
```

**Time:** ~5 seconds

---

## Slide 7: Delete Workspace - Safety First

```
OWNER clicks "Delete Workspace"
        ↓
Frontend Dialog pops up:
"⚠️  This cannot be undone. Type workspace name to confirm:"
        ↓
OWNER types: "my-workspace" (must match exactly)
        ↓
Backend validates:
  1. Is user the OWNER? YES ✅
  2. Does typed name match? YES ✅
  3. Should we really do this? YES ✅
        ↓
Backend deletes namespace (cascades all resources)
        ↓
Emit audit trace: workspace_deleted
        ↓
✅ Gone forever (but audit trail stays)
```

**Why?** Prevents accidental `rm -rf /` type mistakes

---

## Slide 8: Quota Management - Namespace ResourceQuota

```
WITHOUT Namespace Quotas (Old Way)
  Problem:
  - Alice's workspace hogs all resources
  - Bob's sessions get stuck waiting
  - No fair sharing

WITH Namespace Quotas (New Way)
  Workspace A quota: 5 concurrent sessions
       ↓
  Workspace B quota: 3 concurrent sessions
       ↓
  Workspace C quota: 10 concurrent sessions
       ↓
  CLUSTER TOTAL: 50 concurrent (if enough hardware)
       ↓
  Namespace quotas + backend enforcement: fair sharing and admission control
       ↓
  Result: No workspace starves others ✅
```

**How it works:**
1. Each workspace gets a ResourceQuota + LimitRange based on `quotaProfile`
2. Kubernetes enforces namespace-level resource totals (CPU, memory, storage, count)
3. If quota prevents creation, backend emits quota events and UI shows limits/position
4. Operator can adjust namespace quotas via profiles for different tiers

---

## Slide 9: Audit Trail - What Gets Tracked?

```
Every workspace tracks:

createdAt: "2025-01-15T10:30:00Z"
  ↳ When was this workspace created?

createdBy: "alice@company.com"
  ↳ Who created it?

lastModifiedAt: "2025-02-10T14:22:00Z"
  ↳ When was it last changed?

lastModifiedBy: "alice@company.com"
  ↳ Who made the last change?

Changes tracked via Langfuse:
  ✓ admin_added: "bob@company.com"
  ✓ admin_removed: "charlie@company.com"
  ✓ quota_updated: maxConcurrentSessions 3→5
  ✓ workspace_deleted: "my-workspace"

Result: Complete history of who did what when ✅
```

---

## Slide 10: Kubernetes RBAC - How It Maps

```
┌────────────────────────────────────────┐
│ ProjectSettings (Governance)           │
│ owner: alice                           │
│ adminUsers: [bob, charlie]             │
└────────────────────────────────────────┘
              ↓
      ┌───────┴────────┐
      ↓                ↓
┌──────────┐      ┌──────────┐
│bob user  │      │charlie   │
│  RB      │      │  RB      │
└────┬─────┘      └────┬─────┘
     │                 │
     └─────────┬───────┘
              ↓
    ┌──────────────────────┐
    │ambient-project-admin │
    │  ClusterRole         │
    │  verbs: create, etc. │
    └──────────────────────┘

RESULT:
  ✅ alice: has admin (owner)
  ✅ bob: has admin (RoleBinding)
  ✅ charlie: has admin (RoleBinding)
  ✅ K8s RBAC enforces: only they can create resources
```

---

## Slide 11: Implementation Timeline

```
PHASE 1 (MVP) - Weeks 1-10
├─ Week 1-2: Owner field + Audit trail
├─ Week 2-3: Admin management backend
├─ Week 3-4: Namespace quota integration
├─ Week 4-5: Delete safety UI
├─ Week 5-7: Full CRUD + testing
├─ Week 7-9: E2E testing + bug fixes
└─ Week 9-10: Production deployment

PHASE 2 (Later) - Weeks 11+
├─ Workspace transfer (Owner → New Owner)
├─ Advanced quota policies (time-based, cost-based)
├─ Cost attribution and chargeback
└─ Workspace templates

TOTAL: ~13 person-days (4 backend + 3 operator + 2 frontend + 2 testing + 2 ops)
ESTIMATED: 8-10 weeks elapsed time
```

---

## Slide 12: Key Takeaways

✅ **5-tier hierarchy** provides clear governance  
✅ **Immutable owner** prevents transfers without authority  
✅ **Multiple admins** share workspace management  
✅ **Namespace quota integration** ensures fair resource sharing  
✅ **Quota per workspace** prevents starvation  
✅ **Delete safety** requires name confirmation  
✅ **Full audit trail** tracks all changes  
✅ **Backward compatible** - existing K8s RBAC unchanged  

---

## Slide 13: Common Questions Answered

**Q: Can an admin remove the owner?**
→ No. Only Root can remove owner. This prevents chaos.

**Q: What if all admins leave?**
→ Owner is implicit admin and can always manage.

**Q: Can I change the quota?**
→ Yes. Owner can update quota anytime in ProjectSettings.

**Q: What happens if workspace deletes?**
→ All sessions, jobs, PVCs cascade-deleted. Audit trail stays.

**Q: Can namespace quotas reject my session?**
→ Yes, if workspace hits maxConcurrentSessions limit. Must wait queue.

**Q: Does Root need one in each workspace?**
→ No. Root only needed for transfers. Normal workspaces don't see Root.

---

## Slide 14: Next Steps

1. **Review** permisson diagrams (Slide 2-3)
2. **Understand** typical team setup (Slide 4)  
3. **Learn** ProjectSettings structure (Slide 5)
4. **Read** full design document (WORKSPACE_RBAC_AND_QUOTA_DESIGN.md)
5. **Plan** implementation (MVP_IMPLEMENTATION_CHECKLIST.md)
6. **Start** building Phase 1

**Est. learning time:** 90 minutes → Full understanding

---

## 📚 Document Guide

| Document | Time | Content |
|----------|------|---------|
| **LEARNING_GUIDE.md** | 30 min | Beginner-friendly explanations |
| **ARCHITECTURE_DIAGRAMS.md** | 20 min | Visual diagrams + sequence flows |
| **QUICK_SLIDES.md** | 15 min | This file - executive summary |
| **WORKSPACE_RBAC_AND_QUOTA_DESIGN.md** | 90 min | Complete technical specification |
| **MVP_IMPLEMENTATION_CHECKLIST.md** | 30 min | Week-by-week task breakdown |
| **ROLES_VS_OWNER_HIERARCHY.md** | 20 min | Deep governance explanation |
| **QUICK_REFERENCE.md** | 10 min | API endpoints + schema cheat sheet |

**Total:** ~3.5 hours for complete mastery

---

## 🎓 Learning Paths by Role

### Project Manager / Product Owner (45 min)
1. Slides 1-4 (this file) - 15 min
2. LEARNING_GUIDE.md Scenarios section - 20 min
3. FAQ questions - 10 min

### Software Engineer (120 min)
1. All slides (this file) - 20 min
2. ARCHITECTURE_DIAGRAMS.md - 30 min
3. WORKSPACE_RBAC_AND_QUOTA_DESIGN.md - 70 min

### Platform Operator (90 min)
1. LEARNING_GUIDE.md "For Platform Operators" - 20 min
2. WORKSPACE_RBAC_AND_QUOTA_DESIGN.md Part 4 (Namespace quota integration) - 30 min
3. MVP_IMPLEMENTATION_CHECKLIST.md - 30 min
4. Deployment questions - 10 min

### Executive / Stakeholder (15 min)
1. Slides 1-2, 11-12 (this file) - 10 min
2. Key Takeaways (Slide 12) - 5 min

---

## 🚀 Ready to Dive Deeper?

- Start with **LEARNING_GUIDE.md** for detailed explanations
- Reference **ARCHITECTURE_DIAGRAMS.md** for visuals
- Read **WORKSPACE_RBAC_AND_QUOTA_DESIGN.md** for the full spec
- Build using **MVP_IMPLEMENTATION_CHECKLIST.md** as guide

Questions? Issues? Clarifications needed? Ask now before implementation starts!
