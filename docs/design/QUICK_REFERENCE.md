# 📋 Design Summary Sheet

**Workspace RBAC & Quota System** | MVP Scope | 8-10 weeks | 13 person-days

---

## The Model at a Glance

```
                    🔒 ROOT USER
                   (Platform Level)
                         ↓
            Accept Transfer Requests (Phase 2)
                         
────────────────────────────────────────────────
                    👑 OWNER
                   (Workspace)
         Immutable | Can Delete | Manage Admins
                         ↓
            ┌─────────────┼─────────────┐
            ↓             ↓             ↓
          🔑 ADMIN    🔑 ADMIN    🔑 ADMIN (multiple)
         (technical)  (technical) (technical)
        Create Work   Create Work  Create Work
        No governance
                         ↓
            ┌──────────────┴──────────────┐
            ↓                             ↓
          ✏️ USER/EDITOR           👁️ VIEWER
        Create Sessions          Read-Only
        (ambient-project-edit)   (ambient-project-view)
```

---

## What Gets Built (Phase 1)

### Backend
- [ ] Delete endpoint with name confirmation
- [ ] Admin management (add/remove)
- [ ] Owner validation (before governance ops)
- [ ] Langfuse trace emission (5 events)

### Operator
- [ ] Reconcile adminUsers → RoleBindings
- [ ] Create namespace ResourceQuota / LimitRange from `ProjectSettings.spec.quota`
- [ ] Update audit trail (status fields)

### Frontend
- [ ] Delete confirmation dialog
- [ ] Admin management UI
- [ ] Quota display

### Infrastructure
- [ ] ProjectSettings CRD enhancement
- [ ] Namespace ResourceQuota / LimitRange examples
- [ ] QuotaTier definitions
- [ ] Migration script

---

## Key Files to Know

| File | Purpose | Status |
|------|---------|--------|
| `docs/design/WORKSPACE_RBAC_AND_QUOTA_DESIGN.md` | Complete spec (10 parts) | ✅ Created |
| `docs/design/MVP_IMPLEMENTATION_CHECKLIST.md` | Week-by-week tasks | ✅ Created |
| `docs/design/ROLES_VS_OWNER_HIERARCHY.md` | Governance vs. technical | ✅ Created |
| `docs/design/ARCHITECTURE_SUMMARY.md` | Executive overview | ✅ Created |
| `docs/design/README.md` | Navigation guide | ✅ Created |
| `components/manifests/base/rbac/README.md` | Enhanced RBAC explanation | ✅ Updated |

---

## Langfuse Events (MVP)

```
✅ project_created     ← Emitted when workspace created
✅ project_deleted     ← Emitted when owner deletes (with confirmation)
✅ admin_added         ← Emitted when owner adds admin
✅ admin_removed       ← Emitted when owner removes admin
✅ quota_limit_exceeded ← Emitted when session creation hits limit
```

**Masking**: All messages redacted by default  
**Future**: Can fill in more granular tracing in Phase 2+

---

## Three Tiers of Permission Enforcement

```
Layer 1: GOVERNANCE (Backend checks)
  "Is this person allowed to GOVERN?"
   ├─ Is alice = owner? Can delete/transfer
   ├─ Is bob = admin? Can manage users
   └─ Is charlie = user? Can create work

Layer 2: TECHNICAL (Kubernetes RBAC)
  "Is this person allowed to RUN this?"
   ├─ Create verb on agenticsessions?
   ├─ Delete verb on rolebindings?
   └─ List verb on secrets?

Layer 3: QUOTA (Kubernetes namespace ResourceQuota + LimitRange)
   "Is this work allowed to RUN?"
    ├─ Within namespace CPU/Memory totals?
    ├─ Within storage/PVC limits?
    └─ Within token budget enforced by backend/observability?
```

**They work together**: Governance → RBAC → NamespaceQuota → Execution

---

## Success Looks Like

```
✅ Alice creates workspace
   → alice = owner (immutable)
   
✅ Alice adds Bob as admin
   → Bob gets ambient-project-admin role
   → Bob cannot add others (alice only)
   
✅ Charlie (viewer) tries to create session
   → 403: viewers cannot create sessions
   
✅ Bob creates 6th session (limit is 5)
   → 429: quota exceeded, position in queue: 3
   
✅ Alice deletes workspace
   → Dialog: "Type workspace name"
   → Alice types: "my-workspace"
   → Deleted ✓
   → Langfuse trace emitted ✓
```

---

## Quick Start for Teams

### Week 1-2: I'm Starting
→ Read [`MVP_IMPLEMENTATION_CHECKLIST.md`](docs/design/MVP_IMPLEMENTATION_CHECKLIST.md) Week 1-2 section  
→ Copy ProjectSettings CRD schema from Part 3 of design doc  
→ Start with type definitions in `backend/types/common.go`

### Week 3: I'm Stuck
→ Reference [`WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`](docs/design/WORKSPACE_RBAC_AND_QUOTA_DESIGN.md) Part 4 (Namespace quota integration)  
→ Check [`ROLES_VS_OWNER_HIERARCHY.md`](docs/design/ROLES_VS_OWNER_HIERARCHY.md) for permission logic

### Week 5+: I Need Tests
→ See [`MVP_IMPLEMENTATION_CHECKLIST.md`](docs/design/MVP_IMPLEMENTATION_CHECKLIST.md) Week 8-10 (Testing)  
→ Use scenario walk-throughs as test cases

### Deployment Time
→ Follow [`ARCHITECTURE_SUMMARY.md`](docs/design/ARCHITECTURE_SUMMARY.md) "Success Criteria"  
→ Run migration script on existing projects  
→ Verify namespace `ResourceQuota` and `LimitRange` are applied

---

## Effort Breakdown

```
Backend                 4 days  ████░░░░░░
Operator                3 days  ███░░░░░░░
Frontend                2 days  ██░░░░░░░░
Testing                 2 days  ██░░░░░░░░
Ops/DevOps              2 days  ██░░░░░░░░
────────────────────────────────
TOTAL                  13 days  13x
```

**Total**: 8-10 weeks sequential (2-3 sprint cycles)  
**Parallelizable**: Backend + Frontend can run in parallel after CRD designs

---

## Decisions You Made (Locked In)

1. ✅ **5-tier hierarchy** (Root, Owner, Admin, User, Viewer)
2. ✅ **Owner = immutable** (until Phase 2 transfer)
3. ✅ **Multiple admins** (owner manages them)
4. ✅ **Namespace ResourceQuota = first-class** (not optional)
5. ✅ **Delete with name confirmation** (safety feature)
6. ✅ **Langfuse from day 1** (critical ops traced)
7. ✅ **Both user + group access** (coexist cleanly)
8. ✅ **8-10 week MVP timeline** (scoped for excellence)

---

## Phase 2 (Deferred)

These are NOT in Phase 1:

- ❌ Project transfer (awaiting Phase 2 design)
- ❌ Root user approval workflows
- ❌ Advanced quota policies (burst, reserved)
- ❌ Cost attribution & chargeback

---

## Living Documents

These are your source of truth:

📄 **WORKSPACE_RBAC_AND_QUOTA_DESIGN.md** (the spec)
- Update this as you discover implementation details
- Sections evolve week-by-week
- Stay in sync with code

📋 **MVP_IMPLEMENTATION_CHECKLIST.md** (the tasks)
- Copy tasks to Jira
- Uncheck as you complete
- Add blockers as you find them

📝 **ROLES_VS_OWNER_HIERARCHY.md** (the explanation)
- Keep for onboarding new team members
- Reference when questions arise
- Stable (shouldn't change much)

---

## Navigation Guide

**Architect or Lead?**  
→ `ARCHITECTURE_SUMMARY.md` (5 min)

**Ready to Code?**  
→ `MVP_IMPLEMENTATION_CHECKLIST.md` (30 min)

**Need to Understand Permissions?**  
→ `ROLES_VS_OWNER_HIERARCHY.md` (25 min)

**Building the Whole Thing?**  
→ `WORKSPACE_RBAC_AND_QUOTA_DESIGN.md` (60 min)

**Running This Project?**  
→ `design/README.md` (navigation guide)

---

## Summary

**We just delivered**:

✅ 47 KB of comprehensive design documentation  
✅ Complete technical specification (ready to implement)  
✅ Week-by-week implementation checklist  
✅ Architectural clarification (governance vs. technical)  
✅ Enhanced RBAC reference documentation

**You're ready to**:

→ Assign work to teams  
→ Schedule 8-10 week sprint cycle  
→ Start Week 1-2 (CRD + backend types)  
→ Deploy Phase 1 MVP  
→ Plan Phase 2 (transfer workflows)

**Next step**: Review with team, mark as "approved", kick off sprint planning

---

**Status**: ✅ Scope Complete  
**Date**: February 10, 2026  
**Version**: 1.0
