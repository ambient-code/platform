# Platform Architecture Diagrams

Comprehensive architecture documentation for the Ambient Code Platform (formerly vTeam), a Kubernetes-native AI automation platform.

## System Overview

```mermaid
graph TB
    subgraph "User Interface Layer"
        UI[Next.js Frontend<br/>Shadcn UI + React Query]
    end

    subgraph "API Layer"
        API[Go Backend API<br/>Gin + K8s Client]
        WS[WebSocket Server<br/>Real-time Updates]
    end

    subgraph "Kubernetes Control Plane"
        OP[Agentic Operator<br/>CR Watcher]
        CRD1[AgenticSession CRD]
        CRD2[ProjectSettings CRD]
        CRD3[RFEWorkflow CRD]
    end

    subgraph "Execution Layer"
        JOB[Kubernetes Jobs]
        RUNNER[Claude Code Runner<br/>Python + Claude SDK]
        PVC[Persistent Volume<br/>Workspace Storage]
    end

    subgraph "External Services"
        GIT[GitHub/GitLab<br/>Repository Access]
        CLAUDE[Anthropic API<br/>Claude Models]
        OAUTH[OpenShift OAuth<br/>Authentication]
    end

    UI -->|REST API| API
    UI -->|WebSocket| WS
    API -->|Create/Update CRs| CRD1
    API -->|Manage Settings| CRD2
    API -->|Orchestrate Workflows| CRD3

    OP -->|Watch| CRD1
    OP -->|Watch| CRD2
    OP -->|Watch| CRD3
    OP -->|Create| JOB

    JOB -->|Run| RUNNER
    RUNNER -->|Read/Write| PVC
    RUNNER -->|Clone/Push| GIT
    RUNNER -->|API Calls| CLAUDE

    API -->|Authenticate| OAUTH
    UI -->|Login| OAUTH
```

## Component Architecture

```mermaid
graph LR
    subgraph "Frontend (Next.js)"
        direction TB
        PAGES[App Router Pages]
        COMP[Shadcn Components]
        QUERY[React Query Hooks]
        PAGES --> COMP
        COMP --> QUERY
    end

    subgraph "Backend (Go)"
        direction TB
        ROUTES[HTTP Routes]
        HANDLERS[Request Handlers]
        K8S[K8s Client Layer]
        ROUTES --> HANDLERS
        HANDLERS --> K8S
    end

    subgraph "Operator (Go)"
        direction TB
        WATCH[Watch Loops]
        RECON[Reconciliation Logic]
        STATUS[Status Updater]
        WATCH --> RECON
        RECON --> STATUS
    end

    subgraph "Runner (Python)"
        direction TB
        SDK[Claude Code SDK]
        EXEC[Execution Engine]
        SYNC[Workspace Sync]
        SDK --> EXEC
        EXEC --> SYNC
    end

    QUERY -->|HTTP/WS| ROUTES
    K8S -->|Custom Resources| WATCH
    RECON -->|Create Jobs| EXEC
```

## Agentic Session Lifecycle

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant Backend
    participant Operator
    participant Job
    participant Runner
    participant Claude

    User->>Frontend: Create AgenticSession
    Frontend->>Backend: POST /api/projects/:project/agentic-sessions
    Backend->>Backend: Validate user token (GetK8sClientsForRequest)
    Backend->>Backend: Check RBAC permissions (SelfSubjectAccessReview)
    Backend->>Backend: Create AgenticSession CR (via K8s API)
    Backend-->>Frontend: 201 Created {uid, name}

    Operator->>Operator: Watch detects new AgenticSession
    Operator->>Operator: Read CR spec (prompt, repos, model)
    Operator->>Operator: Create Job with Runner pod
    Operator->>Operator: Update status: phase=Pending

    Job->>Runner: Start pod
    Runner->>Runner: Clone repositories
    Runner->>Runner: Setup workspace
    Runner->>Claude: Initialize Claude Code session
    Runner->>Operator: Update status: phase=Running

    loop Agent Execution
        Runner->>Claude: Stream prompt/responses
        Claude-->>Runner: Agent actions
        Runner->>Runner: Execute actions (read, write, bash)
        Runner->>Operator: Update progress
    end

    Runner->>Runner: Commit changes (if any)
    Runner->>Runner: Push to fork/branch
    Runner->>Operator: Update status: phase=Completed
    Runner->>Job: Exit with success

    Operator->>Operator: Detect Job completion
    Operator->>Operator: Update CR final status
    Operator->>Operator: Trigger cleanup (optional)

    Frontend->>Backend: GET /api/projects/:project/agentic-sessions/:name
    Backend-->>Frontend: Session status with results
    Frontend-->>User: Display results
```

## Authentication & Authorization Flow

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant OAuth
    participant Backend
    participant K8s

    User->>Frontend: Access UI
    Frontend->>OAuth: Redirect to login
    OAuth->>OAuth: Authenticate user
    OAuth-->>Frontend: Return bearer token
    Frontend->>Frontend: Store token

    User->>Frontend: Create resource
    Frontend->>Backend: API request + Authorization: Bearer {token}
    Backend->>Backend: Extract token from header
    Backend->>Backend: Create user-scoped K8s clients

    Backend->>K8s: SelfSubjectAccessReview (with user token)
    K8s->>K8s: Check RBAC policies
    K8s-->>Backend: Allowed: true/false

    alt User authorized
        Backend->>K8s: Create/Update resource (with user token)
        K8s-->>Backend: Success
        Backend-->>Frontend: 200/201 Success
        Frontend-->>User: Operation successful
    else User not authorized
        Backend-->>Frontend: 403 Forbidden
        Frontend-->>User: Access denied
    end
```

## Multi-Tenancy Architecture

```mermaid
graph TB
    subgraph "OpenShift Cluster"
        subgraph "Shared Services Namespace"
            BACKEND[Backend API]
            OPERATOR[Agentic Operator]
            FRONTEND[Frontend UI]
        end

        subgraph "Project A Namespace"
            CRA1[AgenticSession CRs]
            PSA[ProjectSettings CR]
            JOBA[Jobs]
            PVCA[PVCs]
            SECRETA[Secrets]
        end

        subgraph "Project B Namespace"
            CRB1[AgenticSession CRs]
            PSB[ProjectSettings CR]
            JOBB[Jobs]
            PVCB[PVCs]
            SECRETB[Secrets]
        end

        subgraph "Project C Namespace"
            CRC1[AgenticSession CRs]
            PSC[ProjectSettings CR]
            JOBC[Jobs]
            PVCC[PVCs]
            SECRETC[Secrets]
        end
    end

    BACKEND -->|User Token Auth| CRA1
    BACKEND -->|User Token Auth| CRB1
    BACKEND -->|User Token Auth| CRC1

    OPERATOR -->|Watch All Namespaces| CRA1
    OPERATOR -->|Watch All Namespaces| CRB1
    OPERATOR -->|Watch All Namespaces| CRC1

    OPERATOR -->|Create| JOBA
    OPERATOR -->|Create| JOBB
    OPERATOR -->|Create| JOBC

    style CRA1 fill:#e1f5ff
    style CRB1 fill:#fff4e1
    style CRC1 fill:#f0e1ff
```

## Data Flow Architecture

```mermaid
flowchart LR
    subgraph "Input"
        REQ[User Request<br/>Prompt + Repos]
    end

    subgraph "API Processing"
        VAL[Validation]
        AUTH[Authorization]
        TRANS[CR Translation]
    end

    subgraph "Kubernetes Storage"
        CR[Custom Resource<br/>AgenticSession]
        STATUS[Status Subresource]
    end

    subgraph "Operator Processing"
        WATCH[Watch Event]
        RECON[Reconciliation]
        JOBCREATE[Job Creation]
    end

    subgraph "Execution"
        POD[Runner Pod]
        WORKSPACE[PVC Workspace]
        RESULTS[Execution Results]
    end

    subgraph "Output"
        UPDATE[Status Updates]
        RESPONSE[API Response]
        DISPLAY[UI Display]
    end

    REQ --> VAL
    VAL --> AUTH
    AUTH --> TRANS
    TRANS --> CR

    CR --> WATCH
    WATCH --> RECON
    RECON --> JOBCREATE
    JOBCREATE --> POD

    POD --> WORKSPACE
    WORKSPACE --> RESULTS
    RESULTS --> UPDATE
    UPDATE --> STATUS

    STATUS --> RESPONSE
    RESPONSE --> DISPLAY
```

## Development Topology (OpenShift Local)

```mermaid
graph TB
    subgraph "Developer Workstation"
        IDE[IDE/Editor<br/>VSCode]
        CLI[Command Line<br/>make, oc, kubectl]
    end

    subgraph "OpenShift Local (CRC)"
        subgraph "vteam-dev namespace"
            FE[Frontend Pod<br/>localhost:3000]
            BE[Backend Pod<br/>:8080]
            OP[Operator Pod]
            SESSION[Session Pods<br/>Dynamic]
        end

        subgraph "Storage"
            PVC_DEV[PVCs]
            SECRET_DEV[Secrets]
        end
    end

    subgraph "External"
        GITHUB_DEV[GitHub]
        ANTHROPIC_DEV[Anthropic API]
    end

    IDE -->|make dev-start| CLI
    CLI -->|Deploy| FE
    CLI -->|Deploy| BE
    CLI -->|Deploy| OP

    FE -->|HTTP| BE
    BE -->|K8s API| SESSION
    OP -->|Create| SESSION

    SESSION -->|Mount| PVC_DEV
    SESSION -->|Clone/Push| GITHUB_DEV
    SESSION -->|API Calls| ANTHROPIC_DEV

    IDE -->|Hot Reload<br/>make dev-sync| FE
```

## Production Deployment Topology

```mermaid
graph TB
    subgraph "External Access"
        USERS[Users<br/>Web Browsers]
        GHOOK[GitHub Webhooks]
    end

    subgraph "OpenShift Cluster"
        subgraph "Ingress Layer"
            ROUTE[OpenShift Routes]
            OAUTH_PROXY[OAuth Proxy]
        end

        subgraph "ambient-code namespace"
            direction TB
            FE_PROD[Frontend Pods<br/>3 replicas<br/>HPA enabled]
            BE_PROD[Backend Pods<br/>3 replicas<br/>HPA enabled]
            OP_PROD[Operator Pod<br/>1 replica]
        end

        subgraph "Project Namespaces (Dynamic)"
            NS1[project-alpha]
            NS2[project-beta]
            NS3[project-gamma]
        end

        subgraph "Monitoring"
            PROM[Prometheus]
            GRAFANA[Grafana]
            LOGS[Logging Stack]
        end
    end

    subgraph "External Services"
        GITHUB_PROD[GitHub Enterprise]
        GITLAB_PROD[GitLab]
        ANTHROPIC_PROD[Anthropic API]
        LANGFUSE[LangFuse<br/>Observability]
    end

    USERS -->|HTTPS| ROUTE
    ROUTE --> OAUTH_PROXY
    OAUTH_PROXY --> FE_PROD

    FE_PROD -->|REST/WS| BE_PROD
    BE_PROD -->|K8s API| NS1
    BE_PROD -->|K8s API| NS2
    BE_PROD -->|K8s API| NS3

    OP_PROD -->|Watch| NS1
    OP_PROD -->|Watch| NS2
    OP_PROD -->|Watch| NS3

    GHOOK -->|Webhooks| BE_PROD

    BE_PROD -->|Metrics| PROM
    OP_PROD -->|Metrics| PROM
    BE_PROD -->|Logs| LOGS

    NS1 -->|Git Ops| GITHUB_PROD
    NS1 -->|Git Ops| GITLAB_PROD
    NS1 -->|API| ANTHROPIC_PROD
    NS1 -->|Traces| LANGFUSE
```

## Network & Service Communication

```mermaid
graph TB
    subgraph "Public Zone"
        INTERNET[Internet]
    end

    subgraph "Ingress Zone"
        ROUTER[OpenShift Router<br/>*.apps.cluster.com]
        LB[Load Balancer]
    end

    subgraph "Application Zone"
        FE_SVC[frontend-service<br/>ClusterIP:3000]
        BE_SVC[backend-service<br/>ClusterIP:8080]
        WS_SVC[websocket-service<br/>ClusterIP:8081]
    end

    subgraph "Control Plane Zone"
        API_SERVER[Kubernetes API Server<br/>:6443]
        ETCD[etcd<br/>CR Storage]
    end

    subgraph "Execution Zone"
        RUNNER_PODS[Runner Pods<br/>Dynamic IPs]
        PVC_STORAGE[Persistent Volumes]
    end

    subgraph "External Zone"
        GIT_EXTERNAL[GitHub/GitLab<br/>HTTPS:443]
        ANTHROPIC_EXTERNAL[Anthropic API<br/>HTTPS:443]
    end

    INTERNET -->|HTTPS:443| LB
    LB --> ROUTER
    ROUTER -->|Route: vteam.apps| FE_SVC
    ROUTER -->|Route: api.vteam.apps| BE_SVC
    ROUTER -->|Route: ws.vteam.apps| WS_SVC

    FE_SVC -->|HTTP:8080| BE_SVC
    FE_SVC -->|WS:8081| WS_SVC

    BE_SVC -->|HTTPS:6443<br/>User Token| API_SERVER
    WS_SVC -->|HTTPS:6443<br/>Service Account| API_SERVER

    API_SERVER --> ETCD

    RUNNER_PODS -->|Mount| PVC_STORAGE
    RUNNER_PODS -->|HTTPS:443| GIT_EXTERNAL
    RUNNER_PODS -->|HTTPS:443| ANTHROPIC_EXTERNAL
```

## Custom Resource Structure

```mermaid
classDiagram
    class AgenticSession {
        +apiVersion: vteam.ambient-code/v1alpha1
        +kind: AgenticSession
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
        +options: map[string]interface
    }

    class RepoConfig {
        +input: InputConfig
        +output: OutputConfig
    }

    class InputConfig {
        +url: string
        +branch: string
        +token: string
    }

    class OutputConfig {
        +pushTo: string (fork|same|none)
        +targetBranch: string
        +targetRepo: string
    }

    class AgenticSessionStatus {
        +phase: string (Pending|Running|Completed|Failed)
        +startTime: string
        +completionTime: string
        +results: string
        +error: string
        +repoStatuses: []RepoStatus
    }

    class RepoStatus {
        +repoIndex: int
        +status: string (pushed|abandoned)
        +branch: string
        +commitSHA: string
    }

    class ProjectSettings {
        +apiVersion: vteam.ambient-code/v1alpha1
        +kind: ProjectSettings
        +metadata: ObjectMeta
        +spec: ProjectSettingsSpec
    }

    class ProjectSettingsSpec {
        +anthropicApiKey: string
        +defaultModel: string
        +defaultTimeout: int
        +gitHubToken: string
        +gitLabToken: string
    }

    class RFEWorkflow {
        +apiVersion: vteam.ambient-code/v1alpha1
        +kind: RFEWorkflow
        +metadata: ObjectMeta
        +spec: RFEWorkflowSpec
        +status: RFEWorkflowStatus
    }

    class RFEWorkflowSpec {
        +description: string
        +agents: []AgentConfig
        +steps: []WorkflowStep
    }

    AgenticSession --> AgenticSessionSpec
    AgenticSession --> AgenticSessionStatus
    AgenticSessionSpec --> RepoConfig
    RepoConfig --> InputConfig
    RepoConfig --> OutputConfig
    AgenticSessionStatus --> RepoStatus
    ProjectSettings --> ProjectSettingsSpec
    RFEWorkflow --> RFEWorkflowSpec
```

## Resource Ownership & Cleanup

```mermaid
graph TD
    AS[AgenticSession CR<br/>UID: abc-123]

    AS -->|OwnerReference<br/>Controller: true| JOB[Job<br/>claude-session-job]
    AS -->|OwnerReference| SECRET[Secret<br/>anthropic-api-key]
    AS -->|OwnerReference| PVC[PVC<br/>workspace-pvc]

    JOB -->|OwnerReference<br/>Controller: true| POD[Pod<br/>claude-runner-pod]

    DELETE[Delete AgenticSession]
    DELETE -->|Cascade Delete| AS
    AS -.->|Automatic Cleanup| JOB
    AS -.->|Automatic Cleanup| SECRET
    AS -.->|Automatic Cleanup| PVC
    JOB -.->|Automatic Cleanup| POD

    style DELETE fill:#ff9999
    style AS fill:#99ccff
    style JOB fill:#99ff99
    style POD fill:#ffff99
```

## Error Handling & Status Transitions

```mermaid
stateDiagram-v2
    [*] --> Pending: CR Created

    Pending --> Running: Job Started
    Pending --> Failed: Job Creation Failed

    Running --> Completed: Success
    Running --> Failed: Execution Error
    Running --> Timeout: Timeout Exceeded

    Failed --> [*]: Cleanup
    Completed --> [*]: Cleanup
    Timeout --> Failed: Mark as Failed

    note right of Pending
        Operator creates Job
        Updates status.phase
    end note

    note right of Running
        Runner executes prompt
        Updates progress
        Streams results
    end note

    note right of Completed
        Results in status.results
        Commits pushed (if applicable)
        repoStatuses updated
    end note

    note right of Failed
        Error in status.error
        Logs available in pod
        Manual intervention needed
    end note
```

## Multi-Repo Workflow

```mermaid
sequenceDiagram
    participant User
    participant Backend
    participant Operator
    participant Runner
    participant Repo1 as github.com/org/repo1
    participant Repo2 as github.com/org/repo2
    participant Fork1 as github.com/user/repo1

    User->>Backend: Create AgenticSession<br/>repos[0]: repo1 (main)<br/>repos[1]: repo2 (ref)
    Backend->>Operator: AgenticSession CR
    Operator->>Runner: Start Job

    Runner->>Repo1: Clone (main repo)
    Runner->>Repo2: Clone (reference repo)
    Runner->>Runner: Set working dir to repos[0]

    Runner->>Runner: Execute agent actions<br/>Read from both repos<br/>Write to main repo

    Runner->>Fork1: Push changes to fork
    Runner->>Runner: Update repoStatuses[0]: pushed
    Runner->>Runner: Update repoStatuses[1]: abandoned

    Runner->>Operator: Status update
    Operator->>Backend: CR status updated
    Backend->>User: Results with repo statuses
```

## Monitoring & Observability

```mermaid
graph TB
    subgraph "Application Metrics"
        BE_METRICS[Backend Metrics<br/>HTTP request duration<br/>K8s API calls]
        OP_METRICS[Operator Metrics<br/>Reconciliation loops<br/>Job creation]
        RUNNER_METRICS[Runner Metrics<br/>Session duration<br/>API token usage]
    end

    subgraph "Infrastructure Metrics"
        POD_METRICS[Pod Metrics<br/>CPU, Memory<br/>Restart count]
        PVC_METRICS[Storage Metrics<br/>Usage, IOPS]
        NETWORK_METRICS[Network Metrics<br/>Throughput, Errors]
    end

    subgraph "Business Metrics"
        SESSION_COUNT[Active Sessions<br/>Success/Fail Rate]
        USER_ACTIVITY[User Activity<br/>API calls per user]
        COST_METRICS[Cost Metrics<br/>API token usage<br/>Compute hours]
    end

    subgraph "Collection Layer"
        PROM[Prometheus]
        LOKI[Loki Logs]
        LANGFUSE_OBS[LangFuse Traces]
    end

    subgraph "Visualization"
        GRAFANA_DASH[Grafana Dashboards]
        ALERTS[AlertManager]
    end

    BE_METRICS --> PROM
    OP_METRICS --> PROM
    RUNNER_METRICS --> PROM
    POD_METRICS --> PROM
    PVC_METRICS --> PROM
    NETWORK_METRICS --> PROM

    SESSION_COUNT --> PROM
    USER_ACTIVITY --> PROM
    COST_METRICS --> LANGFUSE_OBS

    PROM --> GRAFANA_DASH
    LOKI --> GRAFANA_DASH
    LANGFUSE_OBS --> GRAFANA_DASH

    PROM --> ALERTS
```

## Security Architecture

```mermaid
graph TB
    subgraph "Authentication"
        OAUTH[OpenShift OAuth<br/>User Authentication]
        TOKEN[Bearer Tokens<br/>User Identity]
    end

    subgraph "Authorization"
        RBAC[RBAC Policies<br/>Namespace-scoped]
        SSAR[SelfSubjectAccessReview<br/>Permission Checks]
    end

    subgraph "Secret Management"
        K8S_SECRET[Kubernetes Secrets<br/>API Keys, Tokens]
        PROJECTION[Secret Projection<br/>Environment Variables]
    end

    subgraph "Network Security"
        NETPOL[NetworkPolicies<br/>Pod Isolation]
        TLS[TLS Encryption<br/>In-transit]
    end

    subgraph "Pod Security"
        PSA[Pod Security Admission<br/>Restricted Profile]
        SECCTX[SecurityContext<br/>Non-root, Read-only FS]
        CAPSYS[Capabilities<br/>Drop ALL]
    end

    subgraph "Audit & Compliance"
        AUDIT_LOG[K8s Audit Logs]
        TOKEN_REDACTION[Token Redaction<br/>in Logs]
    end

    OAUTH --> TOKEN
    TOKEN --> RBAC
    RBAC --> SSAR

    K8S_SECRET --> PROJECTION
    PROJECTION --> POD[Runner Pods]

    NETPOL --> POD
    TLS --> POD

    PSA --> POD
    SECCTX --> POD
    CAPSYS --> POD

    POD --> AUDIT_LOG
    POD --> TOKEN_REDACTION
```

## Key Design Patterns

### User Token Authentication Pattern

All user-initiated API operations MUST use user-scoped Kubernetes clients:

```go
// ALWAYS use for user operations
reqK8s, reqDyn := GetK8sClientsForRequest(c)
if reqK8s == nil {
    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
    return
}
```

Backend service account is ONLY used for:
1. Writing CRs after validation
2. Minting tokens/secrets for runners
3. Cross-namespace operations backend is authorized for

### Status Subresource Pattern

Always update CR status via the `/status` subresource to prevent race conditions:

```go
_, err = DynamicClient.Resource(gvr).
    Namespace(ns).
    UpdateStatus(ctx, obj, v1.UpdateOptions{})
```

### OwnerReference Pattern

Set OwnerReferences on all child resources for automatic cleanup:

```go
ownerRef := v1.OwnerReference{
    APIVersion: "vteam.ambient-code/v1alpha1",
    Kind:       "AgenticSession",
    Name:       sessionName,
    UID:        sessionUID,
    Controller: boolPtr(true),
}
```

### Watch Loop with Reconnection

Operator watches must handle channel closures and reconnect:

```go
for {  // Infinite loop
    watcher, err := client.Watch(ctx, v1.ListOptions{})
    if err != nil {
        time.Sleep(5 * time.Second)
        continue
    }
    for event := range watcher.ResultChan() {
        handleEvent(event)
    }
    watcher.Stop()
    time.Sleep(2 * time.Second)
}
```

## References

- **ADR-0001**: [Kubernetes-Native Architecture](../adr/0001-kubernetes-native-architecture.md)
- **ADR-0002**: [User Token Authentication](../adr/0002-user-token-authentication.md)
- **ADR-0003**: [Multi-Repo Support](../adr/0003-multi-repo-support.md)
- **ADR-0004**: [Go Backend + Python Runner](../adr/0004-go-backend-python-runner.md)
- **CLAUDE.md**: Backend and Operator Development Standards
- **Amber Workflow**: [amber-workflow.md](amber-workflow.md)
