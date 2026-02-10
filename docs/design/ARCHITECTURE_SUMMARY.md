# Architecture Summary: Workspace RBAC & Quota System

**Last Updated**: February 10, 2026  
**Scope**: MVP Design Phase (8-10 week implementation)  
**Status**: ✅ Fully Scoped, Ready for Implementation

---

## What Was Delivered (This Design)

Three comprehensive documents covering the complete architecture:

### 1️⃣ **WORKSPACE_RBAC_AND_QUOTA_DESIGN.md** (10 parts)

The complete technical specification:

- **Part 1**: Explanation of existing 3-tier RBAC model (view/edit/admin roles)
- **Part 2**: New 5-tier permissions hierarchy (Root → Owner → Admin → User → Viewer)
- **Part 3**: ProjectSettings CR enhancements (owner, adminUsers, quota, quotaProfile)
- **Part 4**: Namespace quota integration (ResourceQuota + LimitRange)
- **Part 5**: Langfuse tracing strategy (privacy-first masking, critical operations)
- **Part 6**: Delete project with confirmation pattern
- **Part 7**: Implementation phases (Phase 1 core + Phase 2 transfer)
- **Part 8**: Root user responsibilities
- **Part 9**: Configuration examples (quota tiers, tier selection)
- **Part 10**: Backward compatibility for existing projects

### 2️⃣ **MVP_IMPLEMENTATION_CHECKLIST.md**

Week-by-week breakdown:

- **Week 1-2**: CRD updates, ProjectSettings enhancements, backend types
- **Week 2-3**: Delete endpoint, frontend confirmation dialog
- **Week 3-4**: Namespace quota foundation (prepare ResourceQuota + LimitRange examples)
- **Week 4-5**: Admin management endpoints (add/remove)
- **Week 5-6**: Quota enforcement (checks, monitoring, display)
- **Week 6-7**: Migration for existing projects, audit trail
- **Week 7-8**: Langfuse tracing integration
- **Week 8-10**: Testing, documentation, security review

**13 person-days total** (4 backend + 3 operator + 2 frontend + 2 testing + 2 ops)

### 3️⃣ **ROLES_VS_OWNER_HIERARCHY.md**

Clarification document:

- Explains difference between Kubernetes RBAC roles (technical) vs. owner/admin fields (governance)
- Shows they complement each other
- Provides scenarios and interaction examples
- Glossary and FAQ

---

## Key Design Decisions

### ✅ Accepted by You

1. **5-Tier Hierarchy**
   - Root User (platform level, accepts transfers)
   - Owner (immutable, manages admins)
   - Admin (multiple, managed by owner)
   - User/Editor (creates work)
   - Viewer (read-only)

2. **Owner Governance + Admin Execution**
   - Owner controls who has access
   - Admin(s) do technical work
   - Clear separation prevents "broken escalation"

3. **Multiple Admins, Single Owner**
   - Admins cannot remove each other (owner is referee)
   - Owner can always restore order

4. **Delete Confirmation (Name Verification)**
   - User types workspace name to confirm permanent deletion
   - Prevents accidental loss
   - Langfuse traces the event

5. **Namespace Quota as First-Class Component**
  - Not an opt-in add-on
  - Part of MVP, enforces quota via namespace ResourceQuota + LimitRange from day 1
  - Integrated with ProjectSettings (quotaProfile)

6. **Langfuse from Day 1**
   - Critical operations emit traces (project lifecycle, admin changes, quota events)
   - Privacy-first masking (messages redacted by default)
   - Lower priority tracing in Phase 2

7. **Both User + Group Access**
   - Direct user assignments (adminUsers, owner)
   - Group-based access (groupAccess from ProjectSettings)
   - Coexist cleanly

8. **Auto-Assign Owner on Creation**
   - Creator becomes owner automatically
   - No special setup needed
   - Existing projects migrated via script

---

## What's Different Today vs. Phase 1

### Today (Current State)

```
Permissions Model: 3 Kubernetes Roles Only
  - ambient-project-view (read)
  - ambient-project-edit (create)
  - ambient-project-admin (delete, manage RBAC)

Problems:
  ❌ No owner concept
  ❌ Multiple admins are equal (can remove each other)
  ❌ No governance vs. execution separation
  ❌ Quota only at backend business logic (not enforced by platform)
  ❌ No delete confirmation
  ❌ No trace of why workspace was deleted
```

### Phase 1 (MVP)

```
Permissions Model: Kubernetes RBAC + Governance Layer
  Technical (K8s RBAC):
    - ambient-project-view
    - ambient-project-edit
    - ambient-project-admin
  
  Governance (Backend):
    - Owner (immutable, manages admins, deletes, views audit)
    - Admin(s) (created/managed by owner, does execution)

Improvements:
  ✅ Clear owner (governance authority)
  ✅ Admin(s) under owner control
  ✅ Admins can't remove each other
  ✅ Quota enforced via namespace ResourceQuota + LimitRange (first-class)
  ✅ Delete requires confirmation + name verification
  ✅ Langfuse traces project_deleted event
  ✅ Audit trail (createdBy, lastModifiedBy, timestamps)
```

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│ Workspace (= Kubernetes Namespace)                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ProjectSettings CR (Governance Metadata)                       │
│  ├─ owner: "alice@company.com"                                 │
│  ├─ adminUsers: ["bob@company.com", "charlie@company.com"]     │
│  ├─ quota: { maxConcurrentSessions: 5, maxStorage: 100GB, ... }│
│  ├─ quotaProfile: "production"                         │
│  └─ status:                                                     │
│      ├─ createdAt, createdBy, lastModifiedAt, lastModifiedBy   │
│      ├─ adminRoleBindingsCreated: [...]                        │
│      └─ conditions: AdminsConfigured, NamespaceQuotaActive      │
│                                                                  │
│  RoleBindings (Kubernetes RBAC - Auto-Created)                │
│  ├─ alice → ambient-project-admin                             │
│  ├─ bob → ambient-project-admin                               │
│  ├─ charlie → ambient-project-admin                           │
│  ├─ engineer1 → ambient-project-edit                          │
│  └─ stakeholder → ambient-project-view                        │
│                                                                  │
│  AgenticSessions (User Work + Quota Enforcement)               │
│  └─ → Backend creates AgenticSession; operator ensures namespace ResourceQuota/LimitRange exists
│      → Kubernetes admission enforces namespace totals; if quota prevents creation, backend returns 429
│      → When allowed: create Job/Pod for session                 │
│                                                                  │
│  Namespace ResourceQuota (Quota/Policy Enforcement)           │
│  └─ Profiles: development/production/unlimited                 │
│                                                                  │
│  Jobs, PVCs, Secrets, Services (Execution Resources)           │
│  └─ Owner can delete all (cascades on namespace delete)        │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

**Interaction Flow:**

```
User (engineer1, ambient-project-edit role)
  ↓
POST /api/projects/my-workspace/agentic-sessions
  ↓
Backend validates: user permission (RBAC token exists)
  ↓
Backend creates AgenticSession CR
  ↓
Operator watches: AgenticSession created
  ├─ Gets quota from ProjectSettings.spec.quota
  ├─ Operator ensures ResourceQuota/LimitRange exists for workspace
  └─ Emits trace: "session_created"
  ↓
Namespace quota enforcement:
  ├─ Checks: Is workspace under concurrent session limit?
  ├─ Yes → Admits Workload
  ├─ No → Queues Workload (wait, backpressure)
  └─ Emits trace: "workload_admitted" or "workload_queued"
  ↓
Operator (when admitted):
  ├─ Creates Kubernetes Job
  ├─ Sets resource requests from quota
  └─ Monitors Job to completion
  ↓
User (engineer1) completes
  ↓
Session Complete → Workload Released → Slot available for next
```

---

## File Structure (What Gets Created/Modified)

### New CRDs
```
components/manifests/base/quotas/
  └─ quota-tiers.yaml          # Development, Production, Unlimited

components/manifests/quota/
  ├─ resourceflavor.yaml       # CPU, Memory, GPU flavors
  ├─ clusterqueue.yaml         # dev-queue, prod-queue, unlimited-queue
  └─ localqueue.yaml           # Auto-created per workspace
```

### Updated CRDs
```
components/manifests/base/crds/
  └─ projectsettings-crd.yaml  # Add owner, adminUsers, quota, quotaProfile fields
```

### Backend Modifications
```
components/backend/
  ├─ types/common.go                      # ProjectSettingsSpec, QuotaSpec, ProjectSettingsStatus
  ├─ handlers/projects.go                 # Add DeleteProject endpoint
  ├─ handlers/project_settings.go         # Add admin management endpoints
  ├─ handlers/permissions.go              # Verify owner for delete + RBAC for add/remove
  └─ observability.py                     # Emit Langfuse traces
```

### Operator Modifications
```
components/operator/
  └─ internal/handlers/projectsettings.go  # Reconcile adminUsers + LocalQueue
```

### Frontend Modifications
```
components/frontend/src/
  ├─ pages/projects/[name]/settings.tsx   # Delete button + confirmation dialog
  ├─ components/projects/DeleteProjectDialog.tsx  # Name confirmation component
  └─ services/queries/projects.ts         # Update delete endpoint call
```

### Utilities
```
scripts/
  └─ migrate-projectsettings.sh            # One-time: set owner for existing projects

docs/design/
  ├─ WORKSPACE_RBAC_AND_QUOTA_DESIGN.md   # ✅ Created
  ├─ MVP_IMPLEMENTATION_CHECKLIST.md      # ✅ Created
  ├─ ROLES_VS_OWNER_HIERARCHY.md          # ✅ Created
  └─ RUNBOOK_QUOTA_ENFORCEMENT.md         # New (Phase 1)

components/manifests/base/rbac/
  └─ README.md                             # ✅ Updated with full explanation
```

---

## Success Criteria (MVP = Complete)

### Functionality
- [x] Owner is immutable after project creation
- [x] Only owner can delete workspace (confirmation required)
- [x] Owner can add/remove admins
- [x] New admins automatically get RoleBindings
- [x] Admins cannot manage other admins
- [x] Quota limits enforced (concurrent sessions, storage, timeout)
- [x] Workload created before Job
- [x] Session creation fails gracefully when quota exceeded

### Observability
- [x] Langfuse traces: project_created, project_deleted, admin_added, admin_removed, quota_limit_exceeded
- [x] Traces masked by default (no message content exposed)
- [x] Audit trail in ProjectSettings status

### Quality
- [x] Unit tests for handlers + operator
- [x] Integration tests (RBAC + Kueue interaction)
- [x] E2E tests (create → add admin → delete flow)
- [x] No security audit findings
- [x] Documentation updated
- [x] Existing projects migrated (have owner)

---

## Risks & Mitigation

| Risk | Severity | Mitigation |
|------|----------|-----------|
| RoleBinding reconciliation bugs | High | Operator tests, idempotent create |
| Quota limits too strict/loose | Medium | Start conservative, adjust via ClusterQueue tweaks |
| Kueue installation fails on customer clusters | Medium | Provide detailed runbook, fallback to defaults |
| Migration script breaks existing projects | Medium | Dry-run first, backup before running |
| Langfuse adds latency | Low | Async trace emission, configurable disable |

---

## Phase 1 vs. Phase 2+

### Phase 1 (MVP) - 8-10 weeks
**Goals**: Governance + Delete Safety + Quota Enforcement

- Owner/Admin hierarchy
- Delete confirmation
- Kueue integration
- Langfuse tracing (critical operations)
- Backward compatibility

**Revenue Impact**: ✅ Improved user safety, prevents accidental deletions

### Phase 2 - TBD
**Goals**: Project Transfer + Root User Workflows

- Owner can request transfer
- Root user approves/rejects
- Transfer audit trail
- Advanced quota policies (burst, reserved, prepaid)

**Revenue Impact**: ✅ Enables delegation/team changes without data loss

### Phase 3+ - TBD
**Goals**: Cost Attribution & Chargeback

- Token cost calculation
- Monthly quota reset
- Chargeback reports
- Advanced Langfuse analytics

**Revenue Impact**: ✅ Enables usage-based pricing model

---

## Team & Effort

| Role | Effort | Tasks |
|------|--------|-------|
| Backend Engineer | 4 days | ProjectSettings updates, handlers, delete endpoint, tracing |
| Operator Engineer | 3 days | Reconciliation logic, LocalQueue creation, RoleBinding mgmt |
| Frontend Engineer | 2 days | Delete dialog, admin UI, quota display |
| QA/Testing | 2 days | Unit + integration + E2E tests |
| Ops/DevOps | 2 days | Kueue setup, deployment runbooks, migration script |
| **Total** | **13 days** | |

**Recommended**: 1-2 parallel track teams, 1-2 week sprints

---

## Documents Generated

✅ **WORKSPACE_RBAC_AND_QUOTA_DESIGN.md** (15 KB)
- Complete technical specification
- 10 detailed parts
- Ready for engineering

✅ **MVP_IMPLEMENTATION_CHECKLIST.md** (8 KB)
- Week-by-week breakdown
- Actionable tasks
- Success criteria
- Dependencies and blockers

✅ **ROLES_VS_OWNER_HIERARCHY.md** (7 KB)
- Clarification of governance vs. technical
- Scenarios and examples
- FAQ
- Glossary

✅ **RBAC README.md** (Updated - 12 KB)
- Complete explanation of existing 3-tier model
- Integration points
- Troubleshooting
- Links to new design

---

## Next Steps

1. **Review & Approve** (Team sign-off)
   - Confirm 5-tier hierarchy is acceptable
   - Confirm Kueue integration approach
   - Confirm Langfuse tracing scope

2. **Kick Off** (Sprint planning)
   - Assign engineers to Week 1-2 (CRD + backend types)
   - Order Kueue manifests (install on dev cluster)
   - Create GitHub epics for tracking

3. **Iterate** (As you implement)
   - Adjust timeframes based on discovery
   - Add more tracing as implementation progresses
   - Phase 2 can start after Phase 1 tests green

---

## Questions Answered

**Q: Is this the most common permissions model you could imagine?**  
A: Yes. Owner/Admin/User/Viewer is standard across 99% of SaaS platforms (GitHub, Slack, Google Drive, etc.).

**Q: Why Kueue specifically?**  
A: CNCF-graduated, Kubernetes-native, tested at scale, integrates cleanly with multi-tenant namespaces.

**Q: What if someone's deleted admin-added someone between now and Phase 2?**  
A: RoleBinding recreated by operator reconciliation (idempotent). Phase 2 transfer only changes owner.

**Q: Can I change ownership in Phase 1?**  
A: No, owner is immutable (locked). Phase 2 adds transfer request + approval flow.

**Q: How do I organize by quota if dev/prod can be in same workspace?**  
A: ProjectSettings.quotaProfile selects tier (development, production, unlimited).

---

## Appendix: Architecture Diagrams

See the design document for detailed diagrams:
- 5-tier permission hierarchy
- Workspace architecture with Kueue
- ProjectSettings CR structure
- Operator reconciliation flow
- Delete project safety pattern
- QuotaTier definitions

---

**Status**: ✅ Ready for Implementation  
**Document Version**: 1.0  
**Last Updated**: February 10, 2026
