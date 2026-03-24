# Proposal: Remove AgenticSession CRD and Move Reconciliation to Control Plane

## Executive Summary

This proposal outlines a plan to eliminate the `AgenticSession` CRD by moving its reconciliation logic directly into the `ambient-control-plane` component. This architectural change will simplify the system by reducing the number of moving parts while maintaining full functionality.

**⚠️ CRITICAL: This proposal has been reviewed from a Senior Distinguished Engineer perspective and contains significant architectural flaws that must be addressed before implementation.**

## Current Architecture vs. Proposed Architecture

### Current: Two-Stage Reconciliation
```
API Session → Control Plane → AgenticSession CRD → Operator → Kubernetes Resources
```

### Proposed: Direct Reconciliation  
```
API Session → Control Plane → Kubernetes Resources
```

## Resources Currently Created by Operator

Based on analysis of `components/operator/internal/controller/agenticsession_controller.go` and `internal/handlers/sessions.go`, the operator creates the following resources for each AgenticSession:

### 1. **Pod** (Main Runner)
- **Location**: `sessions.go` lines 864-1449 in `handleAgenticSessionEvent()`
- **Components**:
  - **Init Container** (`init-hydrate`): State hydration from S3, repo cloning
  - **Main Container** (`ambient-code-runner`): Claude Code CLI execution
  - **Sidecar Container** (`state-sync`): Continuous S3 synchronization
- **Volumes**: EmptyDir workspace, secret mounts for credentials

### 2. **RBAC Resources**
- **ServiceAccount**: `ambient-session-{sessionName}` (lines 2429-2444)
- **Role**: `ambient-session-{sessionName}-role` with AgenticSession and SelfSubjectAccessReview permissions (lines 2446-2477)  
- **RoleBinding**: Binds ServiceAccount to Role (lines 2479-2495)

### 3. **Secret** (Authentication Token)
- **Name**: `ambient-runner-token-{sessionName}` (lines 2509-2552)
- **Purpose**: Kubernetes service account token for runner API access

### 4. **Service** (Pod Access)
- **Name**: `session-{sessionName}` (lines 1403-1434)
- **Type**: ClusterIP
- **Purpose**: Exposes runner's FastAPI server for AG-UI content access

### 5. **Conditional Secrets** (Shared, Copied)
- **Vertex AI**: `ambient-vertex` - Google Cloud credentials (lines 2169-2280)
- **Langfuse**: `ambient-admin-langfuse-secret` - Observability credentials (lines 569-594)
- **Cleanup Logic**: Deleted when no active sessions remain using reference counting

## Migration Plan

### Phase 1: Extend Control Plane with Kubernetes Resource Management

#### 1.1 Add Kubernetes Client Dependencies
```go
// Add to control plane dependencies
"k8s.io/client-go/kubernetes"
"k8s.io/apimachinery/pkg/apis/meta/v1"
corev1 "k8s.io/api/core/v1"
rbacv1 "k8s.io/api/rbac/v1" 
```

#### 1.2 Extend Session Reconciler
Create new package `internal/k8s/` with resource managers:

**File: `internal/k8s/pod_manager.go`**
- Port `handleAgenticSessionEvent()` logic from operator
- Create init, main, and sidecar containers
- Handle volume mounts and resource requests
- Implement pod status monitoring

**File: `internal/k8s/rbac_manager.go`**  
- Port RBAC creation from `regenerateRunnerToken()`
- ServiceAccount, Role, RoleBinding management
- Token secret generation and refresh

**File: `internal/k8s/service_manager.go`**
- Port service creation logic
- ClusterIP service for runner FastAPI access

**File: `internal/k8s/secret_manager.go`**
- Conditional secret copying (Vertex AI, Langfuse)
- Reference counting for shared secrets
- Proper cleanup when sessions terminate

#### 1.3 Integrate with Existing Session Reconciler
Modify `internal/reconciler/sessions.go` to:
```go
func (r *SessionReconciler) reconcileSession(session *api.Session) error {
    switch session.Status {
    case "Pending":
        return r.createKubernetesResources(session)
    case "Running": 
        return r.monitorKubernetesResources(session)
    case "Stopping":
        return r.cleanupKubernetesResources(session)
    }
}

func (r *SessionReconciler) createKubernetesResources(session *api.Session) error {
    // Create RBAC resources
    if err := r.rbacManager.EnsureServiceAccount(session); err != nil {
        return err
    }
    
    // Create pod
    if err := r.podManager.CreateRunnerPod(session); err != nil {
        return err
    }
    
    // Create service
    if err := r.serviceManager.EnsureService(session); err != nil {
        return err
    }
    
    return nil
}
```

### Phase 2: Implement Resource Lifecycle Management

#### 2.1 Owner Reference Management
Since we no longer have AgenticSession CRDs as controller owners, implement alternative cleanup:

**Option A: Control Plane as Owner**
- Use control plane deployment as owner reference
- Requires careful cleanup logic when control plane restarts

**Option B: Label-Based Cleanup**  
- Use consistent labels: `ambient.session.id={sessionId}`
- Implement periodic cleanup of orphaned resources
- More resilient to control plane restarts

**Recommended: Option B with immediate cleanup**
```go
// Label all resources with session ID
labels := map[string]string{
    "ambient.session.id": session.ID,
    "ambient.component":  "session-runner", 
    "ambient.managed-by": "control-plane",
}

// Immediate cleanup on session deletion/completion
func (r *SessionReconciler) cleanupSession(sessionID string) error {
    selector := fmt.Sprintf("ambient.session.id=%s", sessionID)
    
    // Delete pod, service, RBAC resources by label selector
    return r.k8sClient.DeleteCollection(ctx, &corev1.Pod{}, 
        client.InNamespace(namespace),
        client.MatchingLabels(labels))
}
```

#### 2.2 Status Reconciliation  
Update session status based on pod conditions:
```go
func (r *SessionReconciler) updateSessionStatus(session *api.Session) error {
    pod, err := r.getPodForSession(session.ID)
    if err != nil {
        return err
    }
    
    switch pod.Status.Phase {
    case corev1.PodPending:
        session.Status = "Creating"
    case corev1.PodRunning:
        session.Status = "Running"  
    case corev1.PodSucceeded:
        session.Status = "Completed"
    case corev1.PodFailed:
        session.Status = "Failed"
    }
    
    return r.apiClient.UpdateSession(session)
}
```

### Phase 3: Remove AgenticSession CRD Dependencies

#### 3.1 Update API Session Schema
Extend API Session model to include fields currently in AgenticSession:
```go
type Session struct {
    // Existing fields...
    
    // New fields from AgenticSession
    LLMSettings     *LLMSettings     `json:"llm_settings,omitempty"`
    RunnerSettings  *RunnerSettings  `json:"runner_settings,omitempty"`  
    Timeout         int              `json:"timeout,omitempty"`
    KubeCRName      string           `json:"kube_cr_name,omitempty"` // Remove after migration
}
```

#### 3.2 Migrate Existing AgenticSession Data
Create migration script to:
1. List all existing AgenticSessions across namespaces
2. Update corresponding API Sessions with AgenticSession data
3. Verify data integrity before proceeding with deletion

#### 3.3 Remove Operator Dependencies
1. **Update Manifests**: Remove operator deployment and RBAC
2. **Remove CRD**: Delete `agenticsessions.vteam.ambient-code` CRD definition
3. **Update Scripts**: Modify `deploy.sh` to skip operator components
4. **Update Tests**: Remove operator-specific e2e tests

### Phase 4: Configuration and Operational Changes

#### 4.1 Control Plane RBAC Enhancement
Grant control plane permissions to manage session resources:
```yaml
# Add to control plane ClusterRole
rules:
- apiGroups: [""]
  resources: ["pods", "services", "serviceaccounts", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
- apiGroups: ["rbac.authorization.k8s.io"]  
  resources: ["roles", "rolebindings"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

#### 4.2 Namespace Strategy
**Option A: Single Namespace**
- All sessions run in `ambient-code` namespace
- Simpler RBAC, easier management
- May have naming conflicts with many sessions

**Option B: Dynamic Namespaces** (Current operator behavior)
- Create project-specific namespaces: `{project-name}-{suffix}`
- Requires namespace creation/deletion logic
- Better isolation, matches current behavior

**Recommended: Option B** to maintain compatibility

#### 4.3 Observability and Monitoring
- **Metrics**: Port operator metrics to control plane (session counts, durations, failures)
- **Logging**: Structured logging for resource creation/deletion events  
- **Health Checks**: Monitor control plane's Kubernetes API connectivity
- **Alerting**: Alert on failed resource creation or orphaned resources

## Benefits of This Approach

### 1. **Simplified Architecture**
- Eliminates operator component entirely (fewer moving parts)
- Reduces system complexity and failure modes
- Single point of session lifecycle management

### 2. **Reduced Latency**
- Direct API Session → Kubernetes resources (eliminates intermediate CRD step)
- Faster session startup times
- Real-time session state updates

### 3. **Better Resource Management**  
- Direct ownership and cleanup by control plane
- More predictable garbage collection
- Reduced orphaned resource risks

### 4. **Operational Simplicity**
- Fewer components to monitor and maintain
- Single place to debug session issues
- Unified logging and metrics

### 5. **Resource Efficiency**
- Eliminates operator deployment overhead
- No CRD storage overhead in etcd
- Reduced controller-runtime overhead

## Risks and Mitigation

### 1. **Control Plane Complexity**
- **Risk**: Control plane becomes too complex
- **Mitigation**: Modular design with clear separation of concerns, comprehensive testing

### 2. **Kubernetes API Load**  
- **Risk**: Direct resource management increases API calls
- **Mitigation**: Implement client-side caching, rate limiting, and batch operations

### 3. **Error Handling**
- **Risk**: Less sophisticated error handling than controller-runtime
- **Mitigation**: Port operator's retry logic and error classification

### 4. **Migration Complexity**
- **Risk**: Data loss during migration
- **Mitigation**: Comprehensive migration testing, rollback procedures

## Implementation Timeline

### Week 1-2: Foundation
- Implement Kubernetes client integration in control plane
- Create resource manager interfaces and basic implementations
- Add unit tests for resource creation logic

### Week 3-4: Core Features  
- Implement pod, RBAC, service, secret management
- Add session status monitoring and updates
- Integration testing with existing API

### Week 5-6: Migration & Testing
- Create migration tooling and procedures
- Comprehensive e2e testing of new architecture
- Performance and load testing

### Week 7-8: Deployment & Cleanup
- Deploy to staging/production environments
- Monitor system behavior and performance
- Remove operator components and CRD definitions

---

# 🚨 SENIOR DISTINGUISHED ENGINEER REVIEW

## Critical Architectural Flaws

### 1. **FUNDAMENTAL FLAW: Big Bang Migration Strategy**

**Problem**: The proposal suggests eliminating the operator and CRD entirely in a single migration. This is a **high-risk, all-or-nothing approach** that violates basic production system migration principles.

**Impact**: 
- Zero rollback capability once CRDs are deleted
- Single point of failure during migration
- No ability to A/B test or gradual rollout
- Risk of total session system outage

### 2. **MISSING: Side-by-Side Coexistence Strategy**

**Problem**: No consideration for running both systems simultaneously during transition.

**Required Approach**: 
```
Phase 1: Side-by-Side Coexistence
├── Backend → AgenticSession CRD → Operator → Pods (labels: created-by=operator)
└── Control Plane → Direct API Session → Pods (labels: created-by=control-plane)

Phase 2: Gradual Migration  
├── Route traffic to control plane incrementally
├── Monitor both systems in parallel
└── Validate feature parity before operator removal

Phase 3: Safe Decomission
├── Ensure zero operator-managed sessions
└── Remove operator and CRDs only after complete migration
```

### 3. **OWNERSHIP MODEL VIOLATION**

**Problem**: Proposal suggests "Control Plane as Owner" which violates Kubernetes ownership principles.

**Issues**:
- Control plane deployment restart would orphan ALL session resources
- No true controller ownership (control plane is not a CR)
- Garbage collection would fail catastrophically

**Correct Approach**: 
```yaml
# Option A: Self-Owned Pods (Recommended)
metadata:
  ownerReferences: []  # No owner - managed by control plane via labels
  labels:
    ambient.session.id: "xyz123"
    ambient.created-by: "control-plane"
    ambient.component: "session-runner"

# Option B: Session-Specific Dummy Owner  
# Create minimal "SessionRef" CRD per session as owner
apiVersion: ambient-code.io/v1alpha1
kind: SessionRef
metadata:
  name: session-xyz123
spec:
  sessionId: xyz123
  source: api-server
```

### 4. **RESOURCE CONFLICT RISKS**

**Problem**: No namespace/naming strategy to prevent resource conflicts between systems.

**Conflict Scenarios**:
- Session IDs collision between operator and control plane
- Service name conflicts in same namespace  
- RBAC resource name conflicts
- Secret naming conflicts

**Required Mitigation**:
```yaml
# Operator Resources (existing)
metadata:
  name: ambient-session-{sessionName}
  labels:
    ambient.created-by: "operator"
    ambient.source: "agenticsession-crd"

# Control Plane Resources (new)  
metadata:
  name: ambient-cp-session-{sessionId}  # Different prefix
  labels:
    ambient.created-by: "control-plane"
    ambient.source: "api-session"
```

### 5. **STATE SYNCHRONIZATION GAPS**

**Problem**: No plan for keeping API Session state in sync with Kubernetes resource state.

**Missing Components**:
- Pod status → API Session status reconciliation
- Pod failure handling and session state updates
- Network partition handling between control plane and API server
- Resource state drift detection and correction

**Required Design**:
```go
type SessionStateSync struct {
    apiClient     *APIClient
    k8sClient     *KubernetesClient
    stateStore    *StateStore  // Local state cache
}

func (s *SessionStateSync) ReconcileSessionState(sessionID string) error {
    // 1. Get API session state
    apiSession := s.apiClient.GetSession(sessionID)
    
    // 2. Get Kubernetes resource state  
    pod := s.k8sClient.GetPod(sessionID)
    
    // 3. Resolve conflicts and update both sides
    return s.synchronizeState(apiSession, pod)
}
```

### 6. **FAILURE MODE ANALYSIS MISSING**

**Problem**: No analysis of failure scenarios and recovery procedures.

**Critical Failure Modes**:
- Control plane pod restart during session creation
- Kubernetes API unavailability  
- Control plane crash with partial resource creation
- API server unavailability during status updates

**Required Resilience Design**:
- Idempotent resource creation with proper retry logic
- Checkpoint-based recovery after control plane restart
- Graceful degradation when API server is unreachable
- Dead letter queue for failed reconciliation attempts

### 7. **RBAC SECURITY MODEL FLAWS**

**Problem**: Proposed RBAC is too broad and violates least-privilege principle.

**Security Issues**:
```yaml
# OVERLY BROAD - Can manage ANY pod/secret in cluster
rules:
- apiGroups: [""]
  resources: ["pods", "services", "serviceaccounts", "secrets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

**Secure Design**:
```yaml
# Restricted to specific label selectors
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  resourceNames: []  # Restricted by label selector in code
  # Only pods with ambient.created-by=control-plane labels
```

### 8. **OPERATIONAL COMPLEXITY INCREASE**

**Problem**: Proposal claims "operational simplicity" but actually increases complexity.

**Hidden Complexities**:
- Control plane now must handle Kubernetes API errors, retries, rate limiting
- Manual resource cleanup procedures (no automatic garbage collection)
- Complex state synchronization between API server and K8s
- Custom metric collection and alerting for resource health
- Manual scaling and resource management

**Reality**: This moves complexity from well-tested controller-runtime framework to custom code.

## Recommended Alternative Architecture

### Phase 1: Side-by-Side Implementation (8 weeks)

```go
// Control plane supports both modes
type SessionReconciler struct {
    mode SessionReconciliationMode  // "crd" or "direct"
}

type SessionReconciliationMode string
const (
    CRDMode    SessionReconciliationMode = "crd"     // Use operator
    DirectMode SessionReconciliationMode = "direct"  // Direct k8s
)
```

**Benefits**:
- Zero risk to existing sessions
- A/B testing capability
- Feature parity validation
- Gradual traffic migration
- Full rollback capability

### Phase 2: Feature Flag Controlled Migration (4 weeks)

```yaml
# ConfigMap: ambient-control-plane-config
data:
  session-reconciliation-mode: "crd"  # or "direct"
  migration-percentage: "10"          # Gradually increase
  migration-session-filters: "new-sessions-only"
```

### Phase 3: Safe Decomissioning (2 weeks)

Only after **100% of sessions** are running via control plane:
1. Verify zero AgenticSession CRDs exist
2. Remove operator deployment
3. Remove CRD definitions
4. Clean up operator RBAC

## Revised Implementation Strategy

### 1. **Dual-Mode Session Reconciler**
```go
func (r *SessionReconciler) reconcileSession(session *api.Session) error {
    switch r.config.ReconciliationMode {
    case CRDMode:
        return r.createAgenticSessionCRD(session)  // Existing
    case DirectMode:  
        return r.createKubernetesResources(session) // New
    }
}
```

### 2. **Resource Ownership Strategy**
```yaml
# Self-owned resources with cleanup finalizers
metadata:
  finalizers: ["ambient-code.io/control-plane-cleanup"]
  labels:
    ambient.session.id: "{sessionId}"
    ambient.created-by: "control-plane"
    ambient.generation: "{timestamp}"  # For cleanup batching
```

### 3. **State Synchronization Layer**
```go
type SessionStateManager struct {
    apiClient APIClient
    k8sWatcher *K8sResourceWatcher
    syncQueue *WorkQueue
}

// Bidirectional state sync
func (s *SessionStateManager) Start() {
    go s.watchAPISessionChanges()    // API → K8s  
    go s.watchK8sResourceChanges()   // K8s → API
    go s.processStateSyncQueue()     // Conflict resolution
}
```

### 4. **Gradual Migration Controller**
```go
type MigrationController struct {
    percentage    int  // 0-100
    filters       []SessionFilter  
    dryRun        bool
}

func (m *MigrationController) ShouldMigrateSession(session *api.Session) bool {
    if m.dryRun { return false }
    return m.matchesFilters(session) && m.withinPercentage()
}
```

## Conclusion

The original proposal, while well-intentioned, represents a **dangerous big-bang migration** that could cause system-wide outages. The side-by-side coexistence approach is the **only safe way** to implement this architectural change in a production system.

**Key Principles**:
- **Never remove working systems without proven replacements**
- **Always maintain rollback capability**  
- **Implement gradual migration with feature flags**
- **Design for failure scenarios from day one**

## Next Steps

1. **STOP**: Do not implement original proposal
2. **Redesign**: Implement dual-mode reconciliation architecture  
3. **Validate**: Extensive testing of direct mode in isolation
4. **Graduate**: Gradual migration with comprehensive monitoring
5. **Cleanup**: Remove operator only after complete validation