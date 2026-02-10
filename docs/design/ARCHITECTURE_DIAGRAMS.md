# Workspace RBAC & Quota System - Architecture Diagrams

This document contains visual diagrams to help understand the workspace RBAC and quota management system design.

---

## 1. Permission Hierarchy Overview

```mermaid
graph TD
    A["🔒 ROOT USER<br/>(Platform Level)"]
    B["👑 OWNER<br/>(Workspace Level)"]
    C["🔑 ADMIN<br/>(Workspace Level)"]
    D["✏️ USER/EDITOR<br/>(Workspace Level)"]
    E["👁️ VIEWER<br/>(Workspace Level)"]
    
    A -->|"Transfers Workspace"| B
    A -->|"Approves/Rejects"| B
    B -->|"Manages"| C
    B -->|"Invites"| D
    B -->|"Invites"| E
    C -->|"Can be elevated to"| B
    D -->|"Can be elevated to"| C
    E -->|"Can be elevated to"| D
    
    style A fill:#ff6b6b,stroke:#c00,stroke-width:3px,color:#fff
    style B fill:#ffd93d,stroke:#c90,stroke-width:2px,color:#000
    style C fill:#6bcf7f,stroke:#090,stroke-width:2px,color:#fff
    style D fill:#4d96ff,stroke:#009,stroke-width:2px,color:#fff
    style E fill:#999,stroke:#666,stroke-width:2px,color:#fff
```

---

## 2. Permission Matrix - What Can Each Role Do?

```mermaid
graph LR
    subgraph "SESSION MANAGEMENT"
        V1["View Sessions"]
        C1["Create Session"]
        D1["Delete Session"]
    end
    
    subgraph "WORKSPACE MANAGEMENT"
        V2["View Audit Log"]
        M2["Manage Admins"]
        DW["Delete Workspace"]
    end
    
    subgraph "RESOURCE MANAGEMENT"
        M3["Manage Secrets"]
        V3["View Quota Status"]
    end
    
    Root["🔒 ROOT"]
    Owner["👑 OWNER"]
    Admin["🔑 ADMIN"]
    User["✏️ USER"]
    Viewer["👁️ VIEWER"]
    
    Root --> V1
    Owner --> V1
    Owner --> C1
    Owner --> D1
    Owner --> V2
    Owner --> M2
    Owner --> DW
    Owner --> M3
    
    Admin --> V1
    Admin --> C1
    Admin --> D1
    Admin --> M3
    
    User --> V1
    User --> C1
    
    Viewer --> V1
    
    style Root fill:#ff6b6b,color:#fff
    style Owner fill:#ffd93d,color:#000
    style Admin fill:#6bcf7f,color:#fff
    style User fill:#4d96ff,color:#fff
    style Viewer fill:#999,color:#fff
```

---

## 3. Workspace Creation & Setup Flow

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant Backend API
    participant K8s
    participant Operator
    
    User->>Frontend: Create Workspace
    Frontend->>Backend API: POST /api/projects
    
    Backend API->>Backend API: Validate user
    Backend API->>K8s: Create Namespace
    K8s-->>Backend API: Namespace created
    
    Backend API->>K8s: Create ProjectSettings CR
    Note over K8s: owner: user@company.com<br/>adminUsers: []<br/>quota: {...}
    K8s-->>Backend API: CR created
    
    Backend API->>K8s: Create RoleBinding (owner)
    Note over K8s: user → ambient-project-admin
    K8s-->>Backend API: RoleBinding created
    
    Backend API->>Backend API: Emit Langfuse trace
    Backend API-->>Frontend: 201 Created
    Frontend-->>User: Workspace ready!
    
    Operator->>K8s: Watch ProjectSettings
    Operator->>Operator: Reconcile quota & RBAC
```

---

## 4. Admin Management Lifecycle

```mermaid
graph TD
    Start["OWNER Adds Admin"] --> Backend["Backend: PUT /api/.../project-settings"]
    Backend --> Validate["Validate: User is owner"]
    Validate --> UpdateCR["Update ProjectSettings CR<br/>adminUsers += alice@example.com"]
    UpdateCR --> K8sDone["K8s CR updated"]
    K8sDone --> Operator["Operator: Watch ProjectSettings"]
    
    Operator --> OpValidate["Check spec.adminUsers"]
    OpValidate --> CreateRB["Create RoleBinding<br/>alice → ambient-project-admin"]
    CreateRB --> RBDone["RoleBinding exists"]
    RBDone --> Status["Update CR Status<br/>adminRoleBindingsCreated: [...]"]
    Status --> Ready["✅ Alice is now ADMIN"]
    Ready --> Permissions["✅ Alice can: Create sessions,<br/>Manage secrets, etc."]
    
    style Start fill:#ffd93d
    style Ready fill:#6bcf7f,color:#fff
    style Permissions fill:#4d96ff,color:#fff
```

---

## 5. Delete Workspace - Safety Confirmation

```mermaid
graph TD
    A["OWNER Clicks<br/>Delete Workspace"] --> B["Frontend Dialog:<br/>Confirm with workspace name"]
    B --> C["User Types:<br/>my-workspace"]
    C --> D{Name matches?}
    D -->|No| E["❌ Try again"]
    E --> C
    D -->|Yes| F["POST /api/projects/my-workspace/delete<br/>with confirmation token"]
    F --> G["Backend: Validate OWNER role"]
    G --> H["Emit Langfuse trace<br/>workspace_deleted"]
    H --> I["Delete Namespace<br/>cascades: Sessions, Jobs, PVCs"]
    I --> J["✅ Clean deletion<br/>Audit trail preserved"]
    
    style A fill:#ffd93d
    style F fill:#ff6b6b,color:#fff
    style J fill:#6bcf7f,color:#fff
    style E fill:#fff0f0
```

---

## 6. Kubernetes RBAC Integration

```mermaid
graph TB
    subgraph "Kubernetes Cluster"
        subgraph "my-workspace namespace"
            PS["ProjectSettings CR<br/>owner: alice<br/>adminUsers: [bob]"]
            RB1["RoleBinding<br/>alice →<br/>ambient-project-admin"]
            RB2["RoleBinding<br/>bob →<br/>ambient-project-admin"]
            RB3["RoleBinding<br/>charlie →<br/>ambient-project-view"]
        end
        
        subgraph "Cluster-level"
            CR1["ClusterRole:<br/>ambient-project-admin<br/>verbs: [create,delete,...]"]
            CR2["ClusterRole:<br/>ambient-project-view<br/>verbs: [get,list]"]
        end
    end
    
    PS --> RB1
    PS --> RB2
    PS --> RB3
    RB1 -.-> CR1
    RB2 -.-> CR1
    RB3 -.-> CR2
    
    style PS fill:#ffd93d,color:#000
    style RB1 fill:#6bcf7f,color:#fff
    style RB2 fill:#6bcf7f,color:#fff
    style RB3 fill:#4d96ff,color:#fff
    style CR1 fill:#f0ad4e,color:#fff
    style CR2 fill:#5bc0de,color:#fff
```

---

## 7. ProjectSettings CR Structure

```mermaid
graph TD
    PS["ProjectSettings CR"]
    
    Spec["spec:"]
    Owner["owner:<br/>alice@company.com"]
    Admins["adminUsers:<br/>- bob@company.com<br/>- charlie@company.com"]
    Meta["displayName: 'My Workspace'<br/>description: 'Frontend + Backend'"]
    Quota["quota:<br/>maxConcurrentSessions: 5<br/>maxSessionDurationMinutes: 480<br/>maxStorageGB: 100<br/>cpuLimit: '4'<br/>memoryLimit: '8Gi'"]
    Config["defaultConfigRepo:<br/>gitUrl: https://...<br/>branch: main"]
    Kueue["kueueWorkloadProfile:<br/>development"]
    
    Status["status:"]
    Created["createdAt: 2025-01-15T...<br/>createdBy: alice"]
    Modified["lastModifiedAt: 2025-02-10T...<br/>lastModifiedBy: alice"]
    RBs["adminRoleBindingsCreated: [...]"]
    Phase["phase: Ready"]
    Conditions["conditions: [...]"]
    
    PS --> Spec
    PS --> Status
    
    Spec --> Owner
    Spec --> Admins
    Spec --> Meta
    Spec --> Quota
    Spec --> Config
    Spec --> Kueue
    
    Status --> Created
    Status --> Modified
    Status --> RBs
    Status --> Phase
    Status --> Conditions
    
    style PS fill:#ffd93d,stroke:#c90,stroke-width:2px
    style Spec fill:#e8f4f8
    style Status fill:#f0f8e8
```

---

## 8. Kueue Integration Architecture

```mermaid
graph TB
    subgraph "Kueue Cluster-Level"
        RF["ResourceFlavor<br/>- gpu-a100: 10 GPUs<br/>- cpu-large: 64 CPUs"]
        CQ["ClusterQueue<br/>- dev-queue: 20% capacity<br/>- prod-queue: 70% capacity"]
    end
    
    subgraph "Per-Workspace"
        PS["ProjectSettings<br/>kueueWorkloadProfile:<br/>development"]
        LQ["LocalQueue<br/>my-workspace/dev<br/>maxRunningWorkloads: 5<br/>clusterQueue: dev-queue"]
    end
    
    subgraph "Session Execution"
        Job["Job spec.podTemplate<br/>requests:<br/>  cpu: 2<br/>  memory: 4Gi"]
        WL["Workload CR<br/>(created by operator)"]
    end
    
    RF --> CQ
    CQ --> LQ
    PS --> LQ
    LQ --> WL
    Job --> WL
    
    style RF fill:#ff9999,color:#fff
    style CQ fill:#ffcc99,color:#000
    style LQ fill:#99ccff,color:#fff
    style PS fill:#ffd93d,color:#000
    style WL fill:#99ff99,color:#000
    style Job fill:#cc99ff,color:#fff
```

---

## 9. Audit Trail & Langfuse Tracing

```mermaid
graph LR
    Event["User Action:<br/>Add Admin"]
    Backend["Backend<br/>Validation"]
    CRUpdate["ProjectSettings<br/>CR Updated"]
    AuditFields["status.lastModifiedBy<br/>status.lastModifiedAt"]
    Langfuse["Langfuse Trace<br/>admin_added"]
    Trace["Event:<br/>user=alice<br/>action=admin_added<br/>timestamp=..."]
    
    Event --> Backend
    Backend --> CRUpdate
    CRUpdate --> AuditFields
    CRUpdate --> Langfuse
    Langfuse --> Trace
    
    style Event fill:#4d96ff,color:#fff
    style Backend fill:#6bcf7f,color:#fff
    style CRUpdate fill:#ffd93d,color:#000
    style AuditFields fill:#99ccff,color:#000
    style Langfuse fill:#ff9999,color:#fff
    style Trace fill:#ffcc99,color:#000
```

---

## 10. Multi-Tenant Quota Enforcement

```mermaid
graph TB
    User1["User 1<br/>Workspace A"]
    User2["User 2<br/>Workspace B"]
    User3["User 3<br/>Workspace C"]
    
    PS1["ProjectSettings A<br/>maxConcurrentSessions: 5"]
    PS2["ProjectSettings B<br/>maxConcurrentSessions: 3"]
    PS3["ProjectSettings C<br/>maxConcurrentSessions: 10"]
    
    Kueue["Kueue<br/>Fair-share allocation"]
    
    Enforce["Operator enforces:<br/>- Session count per workspace<br/>- Duration per session<br/>- Token usage per month"]
    
    Result["End Result:<br/>No workspace starves others<br/>Platform resources shared fairly"]
    
    User1 --> PS1
    User2 --> PS2
    User3 --> PS3
    
    PS1 --> Kueue
    PS2 --> Kueue
    PS3 --> Kueue
    
    Kueue --> Enforce
    Enforce --> Result
    
    style Kueue fill:#ff9999,color:#fff,stroke:#c00,stroke-width:2px
    style Enforce fill:#99ccff,color:#fff
    style Result fill:#6bcf7f,color:#fff,stroke:#090,stroke-width:2px
```

---

## 11. Implementation Phases

```mermaid
gantt
    title Workspace RBAC & Quota Implementation Timeline
    dateFormat YYYY-MM-DD
    
    section Phase 1
    Owner field & audit trail :p1a, 2026-02-10, 30d
    Kueue quota integration :p1b, 2026-02-15, 40d
    Delete workspace safety :p1c, 2026-02-10, 35d
    Admin management UI :p1d, 2026-02-20, 45d
    
    section Phase 2
    Project transfer request :p2a, 2026-04-01, 25d
    Advanced quota policies :p2b, 2026-03-20, 40d
    Cost attribution :p2c, 2026-04-10, 30d
    
    section Testing & Deployment
    E2E testing :test, 2026-03-15, 30d
    Production deployment :deploy, 2026-04-15, 7d
```

---

## 12. Typical User Journeys

### Journey 1: Create Workspace & Invite Team

```mermaid
sequenceDiagram
    participant Alice as Alice (Creator)
    participant UI as Frontend UI
    participant API as Backend API
    participant K8s as Kubernetes
    
    Alice->>UI: Click "Create Workspace"
    UI->>API: POST /api/projects with name & description
    API->>K8s: Create namespace, ProjectSettings, RoleBinding
    K8s-->>API: Resources created
    API-->>UI: Workspace ready
    UI-->>Alice: Show settings page
    
    Note over Alice: Now Alice is OWNER
    
    Alice->>UI: Add admin: bob@company.com
    UI->>API: PUT /api/projects/.../project-settings
    API->>K8s: Update ProjectSettings.spec.adminUsers
    K8s-->>API: CR updated
    
    Note over K8s: Operator watches ProjectSettings
    
    API-->>UI: Admin added
    UI-->>Alice: ✅ Bob is now admin
    
    Note over Alice: Bob can now:<br/>Create sessions<br/>Manage team<br/>Invite others
```

### Journey 2: Create Session with Config Repo

```mermaid
sequenceDiagram
    participant User as User
    participant UI as Frontend
    participant API as Backend
    participant K8s as Kubernetes
    participant Pod as Runner Pod
    
    User->>UI: Click "New Session"
    Note over UI: Pre-fills configRepo<br/>from ProjectSettings.defaultConfigRepo
    User->>UI: Modify (optional) & Click "Create"
    
    UI->>API: POST /api/projects/.../sessions<br/>with configRepo: {...}
    API->>K8s: Create AgenticSession CR<br/>spec.configRepo: {...}
    K8s-->>API: Session created
    API-->>UI: Session ready
    
    Note over K8s: Operator watches AgenticSession
    K8s->>K8s: Create Job with PVC
    K8s->>Pod: Start runner pod
    
    Pod->>Pod: hydrate.sh:<br/>Clone config repo<br/>Overlay with session repo<br/>Start Claude Code runner
    
    Pod-->>UI: Ready for user interaction
    User->>Pod: Send first prompt
    Pod-->>User: Claude responds
```

---

## Key Takeaways

1. **5-Tier Hierarchy**: Root → Owner → Admin → User → Viewer provides clear governance
2. **Immutable Owner**: Created by user; can be transferred via Root approval
3. **Audit Trail**: Every change tracked in ProjectSettings.status
4. **Kueue Integration**: Platform-wide fair quota management
5. **Delete Safety**: Confirmation by name reduces accidental deletions
6. **Configuration Repo**: Workspace defaults for session configuration
7. **RBAC Separation**: Kubernetes ClusterRoles unchanged; governance added in CR

---

## Navigation

- [WORKSPACE_RBAC_AND_QUOTA_DESIGN.md](WORKSPACE_RBAC_AND_QUOTA_DESIGN.md) - Complete technical specification
- [MVP_IMPLEMENTATION_CHECKLIST.md](MVP_IMPLEMENTATION_CHECKLIST.md) - Week-by-week implementation plan
- [ROLES_VS_OWNER_HIERARCHY.md](ROLES_VS_OWNER_HIERARCHY.md) - Governance vs. technical permissions
- [QUICK_REFERENCE.md](QUICK_REFERENCE.md) - Quick lookup guide
