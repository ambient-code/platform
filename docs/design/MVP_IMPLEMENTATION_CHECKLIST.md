# MVP Implementation Checklist

**Scope**: 8-10 weeks to MVP (owner/admin permissions + delete safety + Kueue quota integration)

**Team**: Backend (4 days) + Operator (3 days) + Frontend (2 days) + Testing (2 days) + Ops (2 days) = 13 person-days

---

## Week 1-2: Foundation & CRD Updates

### ProjectSettings CRD Enhancement
- [ ] Backup existing ProjectSettings schema
- [ ] Add owner field (immutable string)
- [ ] Add adminUsers field (array of strings)
- [ ] Add quota fields (nested object)
- [ ] Add kueueWorkloadProfile field (string reference)
- [ ] Add displayName, description fields
- [ ] Add status fields: createdAt, createdBy, lastModifiedAt, lastModifiedBy
- [ ] Add status.adminRoleBindingsCreated array
- [ ] Add status.conditions array (AdminsConfigured, KueueQuotaActive)
- [ ] Add validation: owner != empty on stable API versions
- [ ] Test CRD validation with yq/kubectl dry-run

### Backend Type Updates
- [ ] Update `components/backend/types/common.go` with new types:
  - [ ] ProjectSettingsSpec (owner, adminUsers, quota, kueueWorkloadProfile)
  - [ ] QuotaSpec (maxConcurrentSessions, maxSessionDuration, etc.)
  - [ ] ProjectSettingsStatus (createdAt, createdBy, adminRoleBindingsCreated)
- [ ] Add helper functions:
  - [ ] IsProjectOwner(k8s, namespace, user) bool
  - [ ] GetProjectOwner(k8s, namespace) string
  - [ ] GetProjectAdmins(k8s, namespace) []string

### Operator Updates (handlers/projectsettings.go)
- [ ] Reconcile adminUsers: create RoleBindings for each admin
- [ ] Reconcile kueueWorkloadProfile: create/update LocalQueue
- [ ] Update status.adminRoleBindingsCreated (list of created RB names)
- [ ] Update status.phase (Ready | Error | Updating)
- [ ] Handle deleted admins (remove RoleBindings)
- [ ] Add idempotent RoleBinding creation (check if exists first)
- [ ] Update status conditions based on reconciliation results
- [ ] **Test**: Reconcile admin additions/removals, verify RoleBindings

---

## Week 2-3: Delete Endpoint & Frontend Safety

### Backend
- [ ] Add DELETE /api/projects/:projectName handler
  - [ ] Extract confirmationName from request body
  - [ ] Validate owner role (403 if not owner)
  - [ ] Validate confirmation name matches (400 if mismatch)
  - [ ] Get counts of sessions/jobs/pvcs before delete
  - [ ] Delete namespace via K8sClient (cascades all resources)
  - [ ] **Emit Langfuse trace: project_deleted**
  - [ ] Return success with deleted resource counts
- [ ] Add RBAC test: non-owner cannot delete
- [ ] Add RBAC test: wrong confirmation name rejected
- [ ] Add integration test: owner can delete + namespace gone

### Frontend
- [ ] Add Delete button to project settings page
  - [ ] Only visible to owner (check auth)
  - [ ] Opens confirmation dialog
- [ ] Create DeleteProjectDialog component
  - [ ] Shows warning: "This action cannot be undone"
  - [ ] Shows affected resources (5 active sessions, 45 GB storage, etc.)
  - [ ] Input field: "Type workspace name to confirm: ______"
  - [ ] Submit button disabled until input matches
  - [ ] Handles loading state (POST in progress)
  - [ ] Shows success: "Workspace deleted"
- [ ] **Test**: Can type name, confirm dialog, deletion happens

---

## Week 3-4: Kueue Integration Foundation

### Cluster Preparation
- [ ] Install Kueue operator on cluster
  - [ ] `kubectl apply -f kueue/install.yaml`
  - [ ] Wait for kueue-controller-manager pod ready
- [ ] Create ResourceFlavor manifests
  - [ ] default-flavor (CPU + Memory)
  - [ ] gpu-flavor (for future GPU workloads)
- [ ] Create ClusterQueue manifests
  - [ ] development-queue (20% cluster capacity, 50 max concurrent)
  - [ ] production-queue (70% cluster capacity, 200 max concurrent)
  - [ ] unlimited-queue (platform team only)
- [ ] Create admission check (PVC quota validation)

### Operator Kueue Integration
- [ ] Add Workload CR creation in session handler
  - [ ] Get workspace quota from ProjectSettings
  - [ ] Create Workload with pod template (CPU/Memory requests)
  - [ ] Set labels: workspace, session-id
  - [ ] Set OwnerReference to AgenticSession
- [ ] Add Workload monitoring
  - [ ] Watch Workload status.conditions
  - [ ] Admitted → Proceed to create Job
  - [ ] Evicted → Update session status, retry
  - [ ] Inadmissible → Return error, suggest queue position
- [ ] **Test**: Create session → Workload created → tracks admission

### Backend Awareness
- [ ] When session creation blocked by quota, return 429 with queue info
  - [ ] "max concurrent sessions exceeded, position in queue: 3"
- [ ] Add response header: X-Workload-Status (Pending | Admitted | Evicted)

---

## Week 4-5: Admin Management Endpoints

### Backend Handlers
- [ ] Add GET /api/projects/:projectName/admin-info
  - [ ] Return owner, adminUsers list, audit trail (createdAt, createdBy)
  - [ ] Only accessible to owner + admins
  - [ ] **Emit Langfuse: admin_info_read event (trace visibility)**
  
- [ ] Add POST /api/projects/:projectName/admins (add admin)
  - [ ] Request body: { "adminEmail": "bob@company.com" }
  - [ ] Validate owner role (403 if not owner)
  - [ ] Add to ProjectSettings.spec.adminUsers
  - [ ] Operator reconciles → creates RoleBinding
  - [ ] **Emit Langfuse: admin_added event**
  - [ ] Return updated admin list
  
- [ ] Add DELETE /api/projects/:projectName/admins/:adminEmail (remove admin)
  - [ ] Validate owner role (403 if not)
  - [ ] Remove from spec.adminUsers
  - [ ] Operator reconciles → deletes RoleBinding
  - [ ] **Emit Langfuse: admin_removed event**
  - [ ] Return updated admin list
  
- [ ] Update ADD/REMOVE permission handlers
  - [ ] Enforce: Only admins can add/remove users (not users)
  - [ ] Enforce: Only owner can manage admins

### RBAC Tests
- [ ] Owner can add admin (201 Created)
- [ ] Non-owner add admin → 403 Forbidden
- [ ] Owner can remove admin (200 OK)
- [ ] Admin cannot add anybody (403)
- [ ] User cannot add anybody (403)

---

## Week 5-6: Quota Enforcement

### ProjectSettings Enhancement
- [ ] Define quota fields in CRD (already done in week 1)
- [ ] Create QuotaTier CRDs (development, production, unlimited)

### Kueue Workload Enforcement
- [ ] Session handler sets CPU/Memory requests from quota
- [ ] Kueue enforces via ClusterQueue limits
- [ ] Monitor workload status for preemption events

### Backend Quota Checks (PreSession Validation)
- [ ] Before creating Workload, check:
  - [ ] Current concurrent sessions < quota.maxConcurrentSessions
  - [ ] Session duration <= quota.maxSessionDurationMinutes
  - [ ] Workspace storage + session size <= quota.maxStorageGB
- [ ] If exceeded: return 429 with "quota_exceeded" detail
- [ ] **Emit Langfuse: quota_limit_exceeded event**

### Operator Quota Monitoring
- [ ] Track total tokens used per workspace per month
- [ ] When approaching limit, add warning to status
- [ ] When exceeding, set status.phase = "QuotaExceeded"

### Frontend Display
- [ ] Show quota usage on project page
  - [ ] "1 of 3 concurrent sessions"
  - [ ] "215 GB of 500 GB storage"
  - [ ] Session queue position: "Position 3 in queue, ~5 min wait"

---

## Week 6-7: Migration & Audit Trail

### Migration Script
- [ ] Write `scripts/migrate-projectsettings.sh`
  - [ ] List all existing ProjectSettings (no owner)
  - [ ] For each: find first admin from RoleBindings
  - [ ] Patch ProjectSettings: set owner to first admin
  - [ ] Log progress (✓ Migrated ns, owner=user)
- [ ] Run dry-run on test cluster
- [ ] Run on production (backup first)
- [ ] Verify: every ProjectSettings now has owner

### Operator Backward Compatibility
- [ ] If spec.owner is empty (legacy): don't error
  - [ ] Log warning, skip owner-specific logic
  - [ ] Still reconcile adminUsers/RoleBindings normally
- [ ] After migration, operator updates createdAt/createdBy in status

### Status Subresource Updates
- [ ] Operator updates status fields:
  - [ ] status.createdAt (from K8s metadata.creationTimestamp or now)
  - [ ] status.createdBy (from owner or first admin found)
  - [ ] status.lastModifiedAt (now, on every reconcile)
  - [ ] status.lastModifiedBy (extract from admission webhook origin if available)
- [ ] Add UpdateStatus in operator reconciliation
- [ ] Test: status fields appear in kubectl describe

### Audit Log View
- [ ] Add GET /api/projects/:projectName/audit-log?limit=50&offset=0
  - [ ] Return chronological list of changes
  - [ ] Include: timestamp, user, action, before, after
  - [ ] Only accessible to owner + admins
  - [ ] **Source**: ProjectSettings status.conditions + admission webhook logs

---

## Week 7-8: Langfuse Tracing Integration

### Backend Trace Emission
- [ ] Identify critical entry points in handlers:
  - [ ] CreateProject (→ project_created)
  - [ ] DeleteProject (→ project_deleted)
  - [ ] AddAdmin (→ admin_added)
  - [ ] RemoveAdmin (→ admin_removed)
  - [ ] CreateSession (→ session_created) [already exists?]
  - [ ] DeleteSession (→ session_deleted)
  - [ ] Quota exceeded (→ quota_limit_exceeded)

- [ ] Call observability.emit_langfuse_trace() in each handler
  - [ ] Pass: name, input, output, userId, sessionId
  - [ ] Input: user request data
  - [ ] Output: server response data (e.g., deleted_sessions: 5)
  - [ ] Default masking: prompt/responses REDACTED

- [ ] Test: Enable Langfuse in local dev, verify traces appear

### Operator Trace Emission
- [ ] Identify reconciliation checkpoints:
  - [ ] AdminRoleBinding created (→ admin_rolebinding_created)
  - [ ] Workload created (→ workload_created)
  - [ ] Workload admitted (→ workload_admitted)
  - [ ] Admin RoleBinding deleted (→ admin_rolebinding_deleted)

- [ ] Call trace emission in operator handlers
- [ ] Include workspace + session metadata

### Configuration
- [ ] Read from environment:
  - [ ] LANGFUSE_ENABLED (default: false for dev, true for prod)
  - [ ] LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY
  - [ ] LANGFUSE_HOST
  - [ ] LANGFUSE_MASK_MESSAGES (default: true)

---

## Week 8-10: Testing & Documentation

### Unit Tests
- [ ] handlers/projects_test.go
  - [ ] DeleteProject with/without owner role
  - [ ] DeleteProject confirmation name validation
  - [ ] Admin add/remove permission checks
  
- [ ] handlers/permissions_test.go
  - [ ] Only admins can add/remove users
  - [ ] Owner can manage admins
  
- [ ] operators/projectsettings_test.go
  - [ ] AdminUsers reconciliation creates RoleBindings
  - [ ] Deleted admins → RoleBindings removed
  - [ ] LocalQueue creation from kueueWorkloadProfile
  - [ ] Status fields updated (createdAt, adminRoleBindingsCreated)

### Integration Tests
- [ ] Create project → owner=creator ✓
- [ ] Add admin → RoleBinding created ✓
- [ ] Remove admin → RoleBinding deleted ✓
- [ ] Delete project (owner only) ✓
- [ ] Concurrent session quota enforced ✓
- [ ] Workload created → job created after admission ✓

### E2E Tests (Cypress)
- [ ] Create workspace
- [ ] Add second admin
- [ ] Remove first admin
- [ ] View admin list
- [ ] Non-owner tries to delete → denied
- [ ] Owner deletes with confirmation
- [ ] Workspace disappears from list

### Documentation
- [ ] Update `components/manifests/base/rbac/README.md`
  - [ ] Explain new 5-tier model
  - [ ] Update permission matrix (admin vs owner)
  - [ ] Add example: delete project flow
  
- [ ] Create `docs/design/WORKSPACE_RBAC_AND_QUOTA_DESIGN.md` ✓ (done)
  
- [ ] Update `docs/deployment/README.md`
  - [ ] Add Kueue installation section
  - [ ] Explain quota tier setup
  - [ ] Migration steps for existing projects
  
- [ ] Create `RUNBOOK_QUOTA_ENFORCEMENT.md`
  - [ ] How to adjust ClusterQueue limits
  - [ ] How to manually override quota (emergency)
  - [ ] How to check workload status
  
- [ ] Update ADR if making architectural changes
  - [ ] Creates new ADR-XXXX: Owner/Admin Hierarchy
  - [ ] Or append to existing ADR
  
- [ ] Update CLAUDE.md with new patterns
  - [ ] ProjectSettings owner management
  - [ ] Langfuse trace emission pattern
  - [ ] Kueue integration pattern

### Performance Testing
- [ ] Load test: 1000 parallel project creations
  - [ ] Verify Kueue LocalQueue creation doesn't bottleneck
  - [ ] Verify RoleBinding reconciliation scales
  
- [ ] Quota check latency: DeleteProject with 50 related resources
  - [ ] Should be <500ms

### Security Review
- [ ] Confirm: Owner role properly enforced in delete handler
- [ ] Confirm: No tokens logged in Langfuse traces
- [ ] Confirm: Admin email validated before adding (no injection)
- [ ] Confirm: Migration script doesn't expose credentials
- [ ] Code review: All permission checks in place

---

## Blockers/Dependencies

| Item | Blocker? | Mitigation |
|------|----------|-----------|
| Kueue operator availability | No | Can deploy from kueue manifests |
| Langfuse availability | No | Can deploy locally or disable tracing |
| RBAC model decision | Yes | See Part 2 of design doc ✓ |
| Backward compat with existing projects | No | Migration script provided |
| Frontend component library | No | Already have Shadcn |
| E2E test environment | No | Already have Cypress + kind |

---

## Success Criteria (MVP Complete)

- [ ] Owner is immutable after project creation
- [ ] Only owner can delete workspace (with name confirmation)
- [ ] Owner can add/remove admins without affecting sessions
- [ ] New admins automatically get ambient-project-admin RoleBinding
- [ ] Quota limits enforced (quota_limit_exceeded → 429)
- [ ] Workload created before Job (Kueue integration working)
- [ ] Langfuse traces emitted for: project_created, project_deleted, admin_added, admin_removed, quota_limit_exceeded
- [ ] Existing projects migrated (have owner set)
- [ ] All E2E tests passing
- [ ] Documentation updated
- [ ] No security audit findings

**Estimated Timeline: 8-10 weeks with team of 4-5 engineers**

---

## Post-MVP (Phase 2+)

- [ ] Project transfer feature (owner → root approval)
- [ ] Advanced quota policies (burst, reserved, prepaid)
- [ ] Cost attribution per workspace
- [ ] Chargeback reports
- [ ] Admin escalation workflows
- [ ] Quota adjustment UI (admin-initiated)
