# Ambient Code Platform Architecture Diagrams

This document provides visual representations of the Ambient Code Platform architecture, component interactions, and key workflows.

## System Architecture Overview

```mermaid
graph TB
    subgraph "User Layer"
        UI[Next.js Frontend<br/>Shadcn UI]
        CLI[kubectl/oc CLI]
    end

    subgraph "API Layer"
        Backend[Go Backend API<br/>Gin + REST]
        OAuth[OpenShift OAuth Proxy]
    end

    subgraph "Control Plane"
        Operator[Agentic Operator<br/>Go Controller]
        K8sAPI[Kubernetes API Server]
    end

    subgraph "Execution Layer"
        Jobs[Kubernetes Jobs]
        Runners[Claude Code Runner Pods<br/>Python SDK]
    end

    subgraph "Storage"
        CRDs[(Custom Resources<br/>AgenticSession<br/>ProjectSettings<br/>RFEWorkflow)]
        PVCs[(PVCs<br/>Workspace Storage)]
        Secrets[(Secrets<br/>API Keys & Tokens)]
    end

    subgraph "External Services"
        Anthropic[Anthropic API<br/>Claude Models]
        GitHub[GitHub<br/>Repositories & PRs]
    end

    UI -->|HTTPS + Bearer Token| OAuth
    CLI -->|kubectl commands| K8sAPI
    OAuth -->|Validated Token| Backend
    Backend -->|Create/Update CRs| K8sAPI
    K8sAPI -->|Store| CRDs
    Operator -->|Watch| CRDs
    Operator -->|Create| Jobs
    Jobs -->|Execute| Runners
    Runners -->|Read/Write| PVCs
    Runners -->|API Calls| Anthropic
    Runners -->|Clone/Push| GitHub
    Backend -->|Read Secrets| K8sAPI
    Runners -->|Read Secrets| K8sAPI
    K8sAPI -->|Store| Secrets
    Backend -->|WebSocket Updates| UI

    style UI fill:#e1f5ff
    style Backend fill:#fff4e1
    style Operator fill:#ffe1f5
    style Runners fill:#e1ffe1
    style CRDs fill:#f5f5f5
    style Anthropic fill:#ffd4d4
    style GitHub fill:#d4e5ff
```

## Component Interaction Flow

```mermaid
sequenceDiagram
    participant User
    participant Frontend as Next.js Frontend
    participant OAuth as OAuth Proxy
    participant Backend as Go Backend API
    participant K8s as Kubernetes API
    participant Operator as Agentic Operator
    participant Job as K8s Job
    participant Runner as Claude Runner Pod
    participant Anthropic as Anthropic API

    User->>Frontend: Create Session via UI
    Frontend->>OAuth: POST /api/projects/my-project/agentic-sessions<br/>(Bearer Token)
    OAuth->>OAuth: Validate OAuth token
    OAuth->>Backend: Forward with X-Forwarded-User
    Backend->>Backend: Extract user identity
    Backend->>K8s: Check RBAC permissions<br/>(SelfSubjectAccessReview)
    K8s-->>Backend: Authorized
    Backend->>K8s: Create AgenticSession CR<br/>(using service account)
    K8s-->>Backend: CR created (UID returned)
    Backend-->>Frontend: 201 Created {uid, name}
    Frontend-->>User: Session created

    Note over Operator: Watches for new CRs
    K8s->>Operator: Watch event: CR ADDED
    Operator->>Operator: Reconcile AgenticSession
    Operator->>K8s: Create Secret (user token for runner)
    Operator->>K8s: Create PVC (workspace)
    Operator->>K8s: Create Job (with OwnerReference)
    K8s->>Job: Schedule Job
    Job->>Runner: Start pod
    Runner->>K8s: Read Secret (Anthropic API key)
    Runner->>Anthropic: Stream session execution
    Anthropic-->>Runner: AI responses
    Runner->>Runner: Execute code changes
    Runner->>K8s: Update CR status via patch
    K8s->>Backend: Status change event
    Backend->>Frontend: WebSocket notification
    Frontend-->>User: Show progress update
    Runner->>Runner: Complete session
    Runner->>K8s: Write results to CR status
    Job->>K8s: Job completed
    Operator->>K8s: Update CR phase: Completed
    K8s->>Backend: Final status update
    Backend->>Frontend: WebSocket: session complete
    Frontend-->>User: Show results
```

## Agentic Session Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Pending: User creates session

    Pending --> Creating: Operator creates Job
    Creating --> Running: Pod starts, runner executes

    Running --> Completed: Session succeeds
    Running --> Failed: Session errors
    Running --> Timeout: Exceeds time limit

    Completed --> [*]
    Failed --> [*]
    Timeout --> [*]

    note right of Pending
        CR created in K8s
        Spec validated
    end note

    note right of Creating
        Operator spawns:
        - Secret (tokens)
        - PVC (workspace)
        - Job (runner pod)
    end note

    note right of Running
        Runner pod:
        - Clones repos
        - Executes Claude CLI
        - Streams to Anthropic
        - Updates CR status
    end note

    note right of Completed
        Results written to:
        - CR status
        - Workspace PVC
        Optional: Push to GitHub
    end note
```

## Multi-Tenant Architecture

```mermaid
graph TB
    subgraph "Platform Shared Services"
        Backend[Backend API<br/>Service Account]
        Operator[Operator<br/>Service Account]
        Frontend[Frontend UI]
    end

    subgraph "Project A Namespace"
        direction TB
        CRs_A[AgenticSessions<br/>ProjectSettings]
        Jobs_A[Session Jobs]
        PVCs_A[Workspaces]
        Secrets_A[API Keys]
        RBAC_A[RoleBindings<br/>User A → Editor]
    end

    subgraph "Project B Namespace"
        direction TB
        CRs_B[AgenticSessions<br/>ProjectSettings]
        Jobs_B[Session Jobs]
        PVCs_B[Workspaces]
        Secrets_B[API Keys]
        RBAC_B[RoleBindings<br/>User B → Viewer]
    end

    User_A[User A<br/>OAuth Token A]
    User_B[User B<br/>OAuth Token B]

    User_A -->|Bearer Token A| Frontend
    User_B -->|Bearer Token B| Frontend
    Frontend -->|Token A| Backend
    Frontend -->|Token B| Backend

    Backend -->|User Token A<br/>RBAC: ✓ Edit| CRs_A
    Backend -->|User Token B<br/>RBAC: ✓ View| CRs_B
    Backend -.->|User Token A<br/>RBAC: ✗ Forbidden| CRs_B
    Backend -.->|User Token B<br/>RBAC: ✗ Forbidden| CRs_A

    Operator -->|Watch All Namespaces| CRs_A
    Operator -->|Watch All Namespaces| CRs_B
    Operator -->|Create/Update| Jobs_A
    Operator -->|Create/Update| Jobs_B

    Jobs_A --> PVCs_A
    Jobs_A --> Secrets_A
    Jobs_B --> PVCs_B
    Jobs_B --> Secrets_B

    style User_A fill:#cce5ff
    style User_B fill:#ffccf2
    style Backend fill:#fff4e1
    style Operator fill:#ffe1f5
```

## Custom Resource Definitions Structure

```mermaid
classDiagram
    class AgenticSession {
        +metadata: ObjectMeta
        +spec: AgenticSessionSpec
        +status: AgenticSessionStatus
    }

    class AgenticSessionSpec {
        +prompt: string
        +repos: []RepoConfig
        +mainRepoIndex: int
        +interactive: bool
        +timeout: int
        +model: string
    }

    class AgenticSessionStatus {
        +phase: string
        +startTime: string
        +completionTime: string
        +results: string
        +errorMessage: string
        +repoStatuses: []RepoStatus
    }

    class RepoConfig {
        +input: RepoInput
        +output: RepoOutput
    }

    class RepoInput {
        +repoURL: string
        +branch: string
    }

    class RepoOutput {
        +fork: bool
        +targetBranch: string
        +prTitle: string
    }

    class RepoStatus {
        +repoURL: string
        +pushed: bool
        +abandoned: bool
    }

    class ProjectSettings {
        +metadata: ObjectMeta
        +spec: ProjectSettingsSpec
    }

    class ProjectSettingsSpec {
        +anthropicAPIKey: string
        +defaultModel: string
        +defaultTimeout: int
        +allowedModels: []string
    }

    class RFEWorkflow {
        +metadata: ObjectMeta
        +spec: RFEWorkflowSpec
        +status: RFEWorkflowStatus
    }

    class RFEWorkflowSpec {
        +requirement: string
        +repos: []RepoConfig
    }

    class RFEWorkflowStatus {
        +phase: string
        +steps: []StepStatus
    }

    AgenticSession --> AgenticSessionSpec
    AgenticSession --> AgenticSessionStatus
    AgenticSessionSpec --> RepoConfig
    AgenticSessionStatus --> RepoStatus
    RepoConfig --> RepoInput
    RepoConfig --> RepoOutput
    ProjectSettings --> ProjectSettingsSpec
    RFEWorkflow --> RFEWorkflowSpec
    RFEWorkflow --> RFEWorkflowStatus
```

## Authentication and Authorization Flow

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant OAuth as OAuth Proxy
    participant Backend as Backend API
    participant K8s as Kubernetes API

    User->>Browser: Access UI
    Browser->>OAuth: GET /
    OAuth->>OAuth: Check session cookie
    alt No valid session
        OAuth->>User: Redirect to OpenShift OAuth
        User->>OAuth: Login with credentials
        OAuth->>OAuth: Create session cookie
    end
    OAuth->>Browser: Serve frontend app
    Browser->>User: Display UI

    Note over User,Browser: User creates session

    Browser->>OAuth: POST /api/projects/my-project/agentic-sessions<br/>Cookie: oauth-session
    OAuth->>OAuth: Validate session
    OAuth->>Backend: Forward request<br/>X-Forwarded-User: alice<br/>X-Forwarded-Email: alice@example.com

    Backend->>Backend: Extract user from header
    Backend->>Backend: Get user's OAuth token
    Backend->>K8s: SelfSubjectAccessReview<br/>(using user's token)

    alt User has permission
        K8s-->>Backend: Allowed: true
        Backend->>K8s: Create AgenticSession CR<br/>(using backend SA)
        K8s-->>Backend: CR created
        Backend-->>Browser: 201 Created
    else User lacks permission
        K8s-->>Backend: Allowed: false
        Backend-->>Browser: 403 Forbidden
    end
```

## Operator Reconciliation Pattern

```mermaid
flowchart TD
    A[Watch AgenticSession CRs] --> B{Event Type?}

    B -->|ADDED| C[New session created]
    B -->|MODIFIED| D[Session updated]
    B -->|DELETED| E[Session deleted]

    C --> F{Check phase}
    D --> F

    F -->|Pending| G[Begin reconciliation]
    F -->|Creating/Running| H[Check job status]
    F -->|Completed/Failed| I[Skip - terminal state]

    G --> J[Verify CR still exists]
    J -->|Not found| K[Skip - deleted]
    J -->|Exists| L[Extract spec fields]

    L --> M[Create Secret for tokens]
    M --> N[Create PVC for workspace]
    N --> O[Create Job with OwnerReference]

    O --> P[Update CR status: Creating]
    P --> Q[Start goroutine to monitor Job]
    Q --> R[Update CR status: Running]

    H --> S{Job status?}
    S -->|Succeeded| T[Extract results from pod logs]
    S -->|Failed| U[Extract error from pod status]
    S -->|Running| V[Wait for completion]

    T --> W[Update CR status: Completed]
    U --> X[Update CR status: Failed]

    E --> Y[Cleanup triggered by OwnerReferences]
    Y --> Z[Job deleted automatically]
    Z --> AA[PVC/Secret deleted automatically]

    W --> AB[End]
    X --> AB
    I --> AB
    K --> AB
    V --> H
```

## Data Flow: Session Creation to Completion

```mermaid
graph LR
    subgraph "1. User Request"
        A[User submits<br/>session request]
    end

    subgraph "2. API Processing"
        B[Backend validates<br/>request]
        C[Backend checks<br/>RBAC]
        D[Backend creates<br/>CR]
    end

    subgraph "3. Operator Reconciliation"
        E[Operator detects<br/>new CR]
        F[Create<br/>infrastructure]
        G[Spawn Job]
    end

    subgraph "4. Runner Execution"
        H[Runner pod<br/>starts]
        I[Clone repos]
        J[Execute Claude<br/>session]
        K[Apply changes]
    end

    subgraph "5. Results"
        L[Update CR<br/>status]
        M[Persist to<br/>workspace]
        N[Optional: Push<br/>to GitHub]
    end

    subgraph "6. User Visibility"
        O[WebSocket<br/>notifications]
        P[UI displays<br/>results]
    end

    A --> B
    B --> C
    C --> D
    D --> E
    E --> F
    F --> G
    G --> H
    H --> I
    I --> J
    J --> K
    K --> L
    L --> M
    M --> N
    L --> O
    O --> P

    style A fill:#e1f5ff
    style D fill:#fff4e1
    style G fill:#ffe1f5
    style J fill:#e1ffe1
    style L fill:#f5f5f5
    style P fill:#e1f5ff
```

## Component Deployment Topology

```mermaid
graph TB
    subgraph "OpenShift Cluster"
        subgraph "ambient-code Namespace<br/>(Platform Services)"
            FE_Deploy[Frontend Deployment<br/>replicas: 2]
            FE_Svc[Frontend Service]
            BE_Deploy[Backend Deployment<br/>replicas: 3]
            BE_Svc[Backend Service]
            OP_Deploy[Operator Deployment<br/>replicas: 1]

            FE_Deploy --> FE_Svc
            BE_Deploy --> BE_Svc
        end

        subgraph "project-alpha Namespace"
            CR1[AgenticSession: task-1]
            CR2[AgenticSession: task-2]
            Job1[Job: task-1]
            Job2[Job: task-2]
            PVC1[PVC: task-1-workspace]
            PVC2[PVC: task-2-workspace]

            CR1 -.OwnerReference.-> Job1
            CR2 -.OwnerReference.-> Job2
            Job1 --> PVC1
            Job2 --> PVC2
        end

        subgraph "project-beta Namespace"
            CR3[AgenticSession: analysis-1]
            Job3[Job: analysis-1]
            PVC3[PVC: analysis-1-workspace]

            CR3 -.OwnerReference.-> Job3
            Job3 --> PVC3
        end

        Route[OpenShift Route<br/>vteam.apps.cluster.example.com]
        Ingress[Ingress Controller]
    end

    Users[Users] -->|HTTPS| Route
    Route --> Ingress
    Ingress --> FE_Svc
    FE_Svc -->|API Calls| BE_Svc

    OP_Deploy -.Watches.-> CR1
    OP_Deploy -.Watches.-> CR2
    OP_Deploy -.Watches.-> CR3

    style FE_Deploy fill:#e1f5ff
    style BE_Deploy fill:#fff4e1
    style OP_Deploy fill:#ffe1f5
    style Job1 fill:#e1ffe1
    style Job2 fill:#e1ffe1
    style Job3 fill:#e1ffe1
```

## Technology Stack

```mermaid
graph LR
    subgraph "Frontend"
        NextJS[Next.js 14<br/>App Router]
        Shadcn[Shadcn UI<br/>Components]
        ReactQuery[React Query<br/>Data Fetching]
        TypeScript[TypeScript<br/>Type Safety]
    end

    subgraph "Backend"
        Go[Go 1.21+]
        Gin[Gin Web Framework]
        K8sClient[Kubernetes Client-Go]
        DynamicClient[Dynamic Client<br/>Unstructured]
    end

    subgraph "Operator"
        GoOp[Go 1.21+]
        Watch[Watch API]
        Reconcile[Reconciliation Loop]
    end

    subgraph "Runner"
        Python[Python 3.11+]
        ClaudeSDK[Claude Code SDK]
        Anthropic[Anthropic SDK]
        GitPython[GitPython]
    end

    subgraph "Infrastructure"
        K8s[Kubernetes 1.27+]
        OpenShift[OpenShift 4.14+]
        CRDs[Custom Resources]
    end

    NextJS --> Shadcn
    NextJS --> ReactQuery
    NextJS --> TypeScript

    Go --> Gin
    Go --> K8sClient
    Go --> DynamicClient

    GoOp --> Watch
    GoOp --> Reconcile

    Python --> ClaudeSDK
    Python --> Anthropic
    Python --> GitPython

    K8s --> OpenShift
    K8s --> CRDs
```

## Key Design Patterns

### 1. Kubernetes-Native Pattern
- **Custom Resources** represent desired state
- **Operator** reconciles actual state to desired state
- **OwnerReferences** enable automatic cleanup
- **Namespaces** provide multi-tenant isolation

### 2. Security Pattern
- **User Token Authentication**: Backend uses user's OAuth token for RBAC checks
- **Service Account for Writes**: Backend SA writes CRs after validation
- **Runner Token**: Operator mints temporary tokens for runner pods
- **Secret Management**: API keys stored in Kubernetes Secrets

### 3. Scalability Pattern
- **Stateless Components**: Frontend and Backend scale horizontally
- **Job-Based Execution**: Each session is an isolated Kubernetes Job
- **Resource Limits**: Jobs have CPU/memory limits to prevent resource exhaustion
- **Namespace Isolation**: Projects in separate namespaces prevent interference

### 4. Observability Pattern
- **CR Status**: Single source of truth for session state
- **Kubernetes Events**: Operator emits events for debugging
- **WebSocket Updates**: Real-time notifications to UI
- **Structured Logging**: JSON logs with correlation IDs

## Related Documentation

- [ADR-0001: Kubernetes-Native Architecture](../adr/0001-kubernetes-native-architecture.md)
- [ADR-0002: User Token Authentication](../adr/0002-user-token-authentication.md)
- [ADR-0003: Multi-Repo Support](../adr/0003-multi-repo-support.md)
- [Amber Workflow Diagrams](amber-workflow.md)
- [Getting Started Guide](../user-guide/getting-started.md)
