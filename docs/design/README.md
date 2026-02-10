# Design Documentation Index

**Workspace RBAC & Quota System - Design Phase Complete**

---

## 📋 Choose Your Path

### 🏗️ If You're an **Architect** or **Team Lead**

**Start here**: [`ARCHITECTURE_SUMMARY.md`](ARCHITECTURE_SUMMARY.md)
- Executive overview (5 min read)
- Key design decisions
- What's different today vs. Phase 1
- Team effort & timeline
- Success criteria

**Then read**: [`ROLES_VS_OWNER_HIERARCHY.md`](ROLES_VS_OWNER_HIERARCHY.md)
- Understand relationship between RBAC roles and governance
- See 3-way interaction examples
- Clarify governance vs. technical permissions

### 👨‍💻 If You're an **Engineer** Ready to Build

**Start here**: [`MVP_IMPLEMENTATION_CHECKLIST.md`](MVP_IMPLEMENTATION_CHECKLIST.md)
- Week-by-week breakdown
- Checkbox tasks (copy to Jira)
- What gets created/modified
- 13 person-days of work

**Then read**: [`WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`](WORKSPACE_RBAC_AND_QUOTA_DESIGN.md)
- Complete technical specification
- CRD schemas (copy-paste ready)
- Handler signatures
- Operator reconciliation examples
- Langfuse trace event names

### 📊 If You're **Product** or **Managing Stakeholders**

**Start here**: [`ARCHITECTURE_SUMMARY.md`](ARCHITECTURE_SUMMARY.md)
- What "Owner" and "Admin" mean
- How delete confirmation protects users
- Why Kueue matters (quota enforcement)
- Phase 1 vs. Phase 2 vs. Phase 3

**Then read**: [`ROLES_VS_OWNER_HIERARCHY.md`](ROLES_VS_OWNER_HIERARCHY.md) → FAQ section
- Answers to common questions
- Use case scenarios
- Permission matrix

### 🔧 If You're **DevOps** or **Infra**

**Start here**: [`WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`](WORKSPACE_RBAC_AND_QUOTA_DESIGN.md) → Part 4 (Kueue Integration)
- ResourceFlavors setup
- ClusterQueue configuration
- LocalQueue per workspace
- Cluster-level quota buckets

**Then read**: (After MVP deployment) `RUNBOOK_QUOTA_ENFORCEMENT.md` (Phase 1 creation)
- How to adjust limits
- Emergency override procedures
- Monitoring Kueue health

---

## 📚 Complete Design Documents

### 1. WORKSPACE_RBAC_AND_QUOTA_DESIGN.md
**Length**: ~15 KB | **Read Time**: 60 min | **For**: Engineers + Architects

**Contains**:
- Part 1: Explanation of existing 3-tier RBAC
- Part 2: New 5-tier permissions hierarchy (detailed)
- Part 3: ProjectSettings CR enhancements (with schema)
- Part 4: Kueue integration (architecture + examples)
- Part 5: Langfuse tracing (critical operations + masking)
- Part 6: Delete project safety pattern
- Part 7: Implementation phases (Phase 1, 2, 3)
- Part 8: Root user responsibilities
- Part 9: Configuration examples
- Part 10: Backward compatibility

**Start at**: [docs/design/WORKSPACE_RBAC_AND_QUOTA_DESIGN.md](WORKSPACE_RBAC_AND_QUOTA_DESIGN.md)

---

### 2. MVP_IMPLEMENTATION_CHECKLIST.md
**Length**: ~8 KB | **Read Time**: 30 min | **For**: Engineers + Project Managers

**Contains**:
- Week 1-2: Foundation & CRD updates
- Week 2-3: Delete endpoint & frontend
- Week 3-4: Kueue foundation
- Week 4-5: Admin management
- Week 5-6: Quota enforcement
- Week 6-7: Migration & audit trail
- Week 7-8: Langfuse tracing
- Week 8-10: Testing & documentation

**Each week has**:
- Specific tasks (checkboxes)
- Files to create/modify
- Tests to write
- Dependencies

**Start at**: [docs/design/MVP_IMPLEMENTATION_CHECKLIST.md](MVP_IMPLEMENTATION_CHECKLIST.md)

---

### 3. ROLES_VS_OWNER_HIERARCHY.md
**Length**: ~7 KB | **Read Time**: 25 min | **For**: Everyone (clarification)

**Contains**:
- Difference between 3 roles (technical) and governance
- How they work together
- 4 detailed scenario walk-throughs
- Permission matrix
- Glossary
- FAQ (common questions)

**Best for**: Understanding the complete permissions model

**Start at**: [docs/design/ROLES_VS_OWNER_HIERARCHY.md](ROLES_VS_OWNER_HIERARCHY.md)

---

### 4. ARCHITECTURE_SUMMARY.md
**Length**: ~5 KB | **Read Time**: 20 min | **For**: Decision makers

**Contains**:
- Accepted design decisions (with reasons)
- What's different today vs. Phase 1
- Architecture overview diagram (ASCII)
- File structure
- Success criteria
- Risk mitigation
- Team effort breakdown
- Next steps

**Start at**: [docs/design/ARCHITECTURE_SUMMARY.md](ARCHITECTURE_SUMMARY.md)

---

### 5. Updated: components/manifests/base/rbac/README.md
**Length**: ~12 KB | **Read Time**: 40 min | **For**: Understanding current state

**Contains**:
- Complete breakdown of each ClusterRole
- How RBAC works today (before Phase 1)
- View + Edit + Admin roles explained
- Permission matrix
- Integration points
- Troubleshooting

**Start at**: [components/manifests/base/rbac/README.md](../base/rbac/README.md)

---

## 🎯 Quick Reference: What Gets Built

### Phase 1 (MVP) - 8-10 weeks

**CRDs**:
- ✅ ProjectSettings (enhanced with owner, adminUsers, quota, kueueWorkloadProfile)
- ✅ QuotaTier (define tiers: development, production, unlimited)
- ✅ Kueue ResourceFlavor, ClusterQueue, LocalQueue (quota enforcement)

**Backend Handlers** (~200 lines new code):
- ✅ DELETE /api/projects/:projectName (delete with name confirmation)
- ✅ POST /api/projects/:projectName/admins (add admin, owner only)
- ✅ DELETE /api/projects/:projectName/admins/:adminEmail (remove admin, owner only)
- ✅ GET /api/projects/:projectName/admin-info (return owner, admins, audit trail)

**Operator Reconciliation** (~100 lines):
- ✅ Watch ProjectSettings.spec.adminUsers changes
- ✅ Create/delete RoleBindings for each admin
- ✅ Create LocalQueue for each workspace (linked to quota tier)
- ✅ Update status fields (createdAt, createdBy, adminRoleBindingsCreated)

**Frontend** (~200 lines):
- ✅ Delete button on project settings
- ✅ DeleteProjectDialog with name confirmation
- ✅ Admin management UI (add/remove)
- ✅ Display quota usage

**Langfuse Traces** (5 events):
- ✅ project_created
- ✅ project_deleted
- ✅ admin_added
- ✅ admin_removed
- ✅ quota_limit_exceeded

**Migration** (script):
- ✅ One-time script to set owner for existing projects

---

## 🚦 How to Use These Documents

### Scenario 1: "I need to implement this"
1. Read `MVP_IMPLEMENTATION_CHECKLIST.md`
2. Keep `WORKSPACE_RBAC_AND_QUOTA_DESIGN.md` open alongside
3. Copy CRD schemas, handler signatures from Part 3, Part 5

### Scenario 2: "I need to explain this to stakeholders"
1. Show `ARCHITECTURE_SUMMARY.md` (5 min overview)
2. Walk through permission matrix in `ROLES_VS_OWNER_HIERARCHY.md`
3. Show Phase 1 vs. today comparison in `ARCHITECTURE_SUMMARY.md`

### Scenario 3: "I need to understand why this design?"
1. Read Part 2 (5-tier hierarchy) in `WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`
2. Read `ROLES_VS_OWNER_HIERARCHY.md` (governance vs. technical)
3. See "Why Two Levels?" section for reasoning

### Scenario 4: "I need to set up Kueue"
1. Jump to Part 4 (Kueue Integration) in `WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`
2. Copy ClusterQueue + ResourceFlavor manifests
3. Reference `MVP_IMPLEMENTATION_CHECKLIST.md` Week 3-4 for deployment steps

### Scenario 5: "I need to write tests"
1. Read `MVP_IMPLEMENTATION_CHECKLIST.md` Week 8-10 (Testing section)
2. Check Part 5 in design doc for Langfuse trace format
3. Use scenario walk-throughs in `ROLES_VS_OWNER_HIERARCHY.md` as test cases

---

## 📊 Document Statistics

| Document | Size | Read Time | Audience |
|----------|------|-----------|----------|
| WORKSPACE_RBAC_AND_QUOTA_DESIGN.md | 15 KB | 60 min | Engineers + Architects |
| MVP_IMPLEMENTATION_CHECKLIST.md | 8 KB | 30 min | Engineers + PMs |
| ROLES_VS_OWNER_HIERARCHY.md | 7 KB | 25 min | Everyone |
| ARCHITECTURE_SUMMARY.md | 5 KB | 20 min | Decision makers |
| RBAC README.md (enhanced) | 12 KB | 40 min | Current state context |
| **Total** | **47 KB** | **175 min** | |

---

## ✅ Checklist for Review

Before implementation, confirm:

- [ ] **5-tier hierarchy accepted** (Root, Owner, Admin, User, Viewer)
- [ ] **Owner = immutable after creation** (only root can transfer in Phase 2)
- [ ] **Multiple admins OK** (managed by owner, can't remove each other)
- [ ] **Kueue integrated** (first-class component, not optional)
- [ ] **Langfuse from day 1** (critical operations traced)
- [ ] **Delete confirmation required** (name verification)
- [ ] **Phase 2 out of scope** (project transfer deferred)
- [ ] **Quota tiers** (development, production, unlimited)
- [ ] **Backward compat** (migration script provided)
- [ ] **8-10 week timeline** (13 person-days effort)

---

## 🔗 Related Documents (Existing)

These documents provide context for the new design:

- **ADR-0001**: Kubernetes-Native Architecture (why K8s at all)
- **ADR-0002**: User Token Authentication (why we use user tokens)
- **ADR-0003**: Multi-Repository Support (context for sessions)
- **docs/decisions.md**: Decision log (recent decisions timeline)
- **docs/DOCUMENTATION_MAP.md**: Complete docs overview
- **CLAUDE.md**: Platform overview and quick reference

---

## 🛠️ Tools & Resources

### For CRD Implementation
- `components/manifests/base/crds/projectsettings-crd.yaml`
- Copy ProjectSettings CRD schema from Part 3 of design doc
- Validate with: `kubectl apply -f file.yaml --dry-run=client`

### For Handler Implementation
- Reference: `components/backend/handlers/permissions.go` (similar pattern)
- Copy handler signatures from Part 3 of design doc
- Use `GetK8sClientsForRequest()` for user token validation

### For Operator Implementation
- Reference: `components/operator/internal/handlers/sessions.go` (similar pattern)
- Copy reconciliation loop from Part 4 of design doc
- Test with: `kubectl describe projectsettings -n test-ws`

### For Frontend Implementation
- Reference: `components/frontend/src/components/ui/` (Shadcn components)
- Copy dialog pattern from Part 6 of design doc
- Use existing form patterns from project settings page

### For Kueue Setup
- Download: [Kueue manifests](https://github.com/kubernetes-sigs/kueue/releases)
- Copy cluster setup from Part 4 of design doc
- Test with: `kubectl get clusterqueue` (should list dev, prod, unlimited)

---

## 📞 Questions?

Specific questions about:

- **5-tier model**: See `ROLES_VS_OWNER_HIERARCHY.md` FAQ
- **Implementation**: See `MVP_IMPLEMENTATION_CHECKLIST.md` for your week
- **CRD schema**: See Part 3 of `WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`
- **Kueue**: See Part 4 of `WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`
- **Langfuse**: See Part 5 of `WORKSPACE_RBAC_AND_QUOTA_DESIGN.md`
- **Current RBAC**: See `components/manifests/base/rbac/README.md`

---

## 🎉 Summary

You now have:

✅ **Complete technical specification** (15 KB design doc)  
✅ **Week-by-week implementation plan** (8 KB checklist)  
✅ **Architectural clarification** (7 KB role explanation)  
✅ **Executive summary** (5 KB overview)  
✅ **Enhanced RBAC documentation** (12 KB reference)  

**Total**: ~47 KB of comprehensive, actionable design documentation  
**Ready**: For immediate implementation (8-10 weeks)  
**Scope**: Fully scoped, zero ambiguity  

---

**Status**: ✅ Design Phase Complete - Ready for Implementation  
**Version**: 1.0  
**Date**: February 10, 2026
