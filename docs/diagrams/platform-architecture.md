# Ambient Code Platform Architecture

This document provides comprehensive architecture diagrams for the Ambient Code Platform, showing system components, data flows, and integration points.

## System Overview

```mermaid
graph TB
    subgraph "User Interface"
        UI[Frontend - NextJS + Shadcn<br/>Port: 3000]
    end

    subgraph "API Layer"
        API[Backend API - Go + Gin<br/>Port: 8080<br/>REST API + WebSocket]
    end

    subgraph "Kubernetes Control Plane"
        OPERATOR[Agentic Operator - Go<br/>Watches Custom Resources<br/>Creates Jobs]
    end

    subgraph "Execution Layer"
        RUNNER[Claude Code Runner<br/>Python + Claude CLI<br/>Pod Execution]
    end

    subgraph "Kubernetes Resources"
        CR[Custom Resources<br/>AgenticSession<br/>ProjectSettings<br/>RFEWorkflow]
        JOB[Kubernetes Jobs<br/>Pod Management]
        PVC[Persistent Volumes<br/>Workspace Storage]
        SECRET[Secrets<br/>API Keys & Tokens]
    end

    subgraph "External Services"
        ANTHROPIC[Anthropic API<br/>Claude AI Models]
        GITHUB[GitHub<br/>Repository Access]
        GITLAB[GitLab<br/>Repository Access]
    end

    UI -->|HTTP/WebSocket| API
    API -->|Create/Update| CR
    OPERATOR -->|Watch| CR
    OPERATOR -->|Create| JOB
    JOB -->|Spawn| RUNNER
    RUNNER -->|Read/Write| PVC
    RUNNER -->|Access| SECRET
    RUNNER -->|AI Requests| ANTHROPIC
    RUNNER -->|Clone/Push| GITHUB
    RUNNER -->|Clone/Push| GITLAB
    API -->|Read Status| CR
    UI -->|Display| UI

    classDef frontend fill:#61dafb,stroke:#20232a,stroke-width:2px,color:#000
    classDef backend fill:#00add8,stroke:#007d9c,stroke-width:2px,color:#fff
    classDef operator fill:#326ce5,stroke:#1a4b99,stroke-width:2px,color:#fff
    classDef runner fill:#3776ab,stroke:#204060,stroke-width:2px,color:#fff
    classDef k8s fill:#f0f0f0,stroke:#666,stroke-width:1px,color:#000
    classDef external fill:#ff9900,stroke:#cc7a00,stroke-width:2px,color:#000

    class UI frontend
    class API backend
    class OPERATOR operator
    class RUNNER runner
    class CR,JOB,PVC,SECRET k8s
    class ANTHROPIC,GITHUB,GITLAB external
```

## Agentic Session Lifecycle

This diagram shows the complete lifecycle of an agentic session from creation to completion.

```mermaid
sequenceDiagram
    participant User
    participant Frontend
    participant Backend
    participant K8s as Kubernetes API
    participant Operator
    participant Job
    participant Runner
    participant Anthropic

    User->>Frontend: Create Session<br/>(Prompt, Repos, Settings)
    Frontend->>Backend: POST /api/projects/:project/agentic-sessions

    Note over Backend: Validate User Token<br/>Check RBAC Permissions

    Backend->>K8s: Create AgenticSession CR<br/>(Custom Resource)
    K8s-->>Backend: CR Created (UID)
    Backend-->>Frontend: 201 Created
    Frontend-->>User: Session Created

    Note over Operator: Watch Loop Detects<br/>New AgenticSession

    Operator->>K8s: Read AgenticSession
    K8s-->>Operator: CR Details

    Operator->>K8s: Create Job<br/>(with OwnerReference)
    K8s-->>Operator: Job Created

    Operator->>K8s: Update CR Status<br/>Phase: Creating

    K8s->>Job: Schedule Pod
    Job->>Runner: Start Container

    Runner->>K8s: Read Secrets<br/>(API Keys, Git Tokens)
    K8s-->>Runner: Secret Data

    Runner->>Runner: Clone Repositories
    Runner->>Anthropic: AI Request<br/>(Claude Code CLI)

    loop AI Processing
        Anthropic-->>Runner: Streaming Response
        Runner->>Runner: Execute Tools<br/>(Read, Write, Edit, Bash)
        Runner->>K8s: Update CR Status<br/>Progress Updates
    end

    Anthropic-->>Runner: Task Complete

    Runner->>Runner: Commit Changes
    Runner->>Runner: Push to Git

    Runner->>K8s: Update CR Status<br/>Phase: Completed<br/>Results

    Runner->>Runner: Exit (Success)

    Note over Operator: Monitor Detects<br/>Job Completion

    Operator->>K8s: Update CR Status<br/>CompletionTime

    Frontend->>Backend: GET Session Status<br/>(Polling/WebSocket)
    Backend->>K8s: Read AgenticSession
    K8s-->>Backend: CR with Status
    Backend-->>Frontend: Session Complete
    Frontend-->>User: Display Results
```

## Component Architecture

### Frontend (NextJS + Shadcn UI)

```mermaid
graph LR
    subgraph "Frontend Components"
        APP[Next.js App Router<br/>App Directory Structure]
        UI[Shadcn UI Components<br/>Accessible Design System]
        RQ[React Query<br/>Data Fetching & Caching]
        WS[WebSocket Client<br/>Real-time Updates]
    end

    subgraph "API Communication"
        HTTP[HTTP Client<br/>Fetch with Auth]
        SOCKET[Socket.io Client<br/>Status Streaming]
    end

    APP --> UI
    APP --> RQ
    RQ --> HTTP
    WS --> SOCKET
    HTTP -->|REST API| BACKEND[Backend API]
    SOCKET -->|WebSocket| BACKEND

    classDef component fill:#61dafb,stroke:#20232a,stroke-width:2px
    classDef client fill:#4a9eff,stroke:#2060c0,stroke-width:2px

    class APP,UI,RQ,WS component
    class HTTP,SOCKET client
```

### Backend API (Go + Gin)

```mermaid
graph TB
    subgraph "Request Handling"
        ROUTER[Gin Router<br/>Route Registration]
        MIDDLEWARE[Middleware Chain<br/>Auth, CORS, Logging]
    end

    subgraph "Handlers"
        PROJECTS[Project Handlers<br/>CRUD Operations]
        SESSIONS[Session Handlers<br/>Lifecycle Management]
        RFE[RFE Handlers<br/>Workflow Orchestration]
    end

    subgraph "Kubernetes Integration"
        USER_CLIENT[User-Scoped K8s Client<br/>Token-Based Auth]
        SA_CLIENT[Service Account Client<br/>CR Write Operations]
    end

    subgraph "External Integration"
        GIT[Git Operations<br/>Clone, Fork, PR]
        GITHUB_API[GitHub API<br/>Repository Management]
        GITLAB_API[GitLab API<br/>Repository Management]
    end

    ROUTER --> MIDDLEWARE
    MIDDLEWARE --> PROJECTS
    MIDDLEWARE --> SESSIONS
    MIDDLEWARE --> RFE

    PROJECTS --> USER_CLIENT
    SESSIONS --> USER_CLIENT
    SESSIONS --> SA_CLIENT
    RFE --> USER_CLIENT

    SESSIONS --> GIT
    GIT --> GITHUB_API
    GIT --> GITLAB_API

    classDef handler fill:#00add8,stroke:#007d9c,stroke-width:2px
    classDef k8s fill:#326ce5,stroke:#1a4b99,stroke-width:2px
    classDef external fill:#ff9900,stroke:#cc7a00,stroke-width:2px

    class ROUTER,MIDDLEWARE,PROJECTS,SESSIONS,RFE handler
    class USER_CLIENT,SA_CLIENT k8s
    class GIT,GITHUB_API,GITLAB_API external
```

### Agentic Operator (Go)

```mermaid
graph TB
    subgraph "Watch Coordination"
        MAIN[Main Watch Loop<br/>Resource Monitoring]
    end

    subgraph "Watch Handlers"
        SESSION_WATCH[AgenticSession Watcher<br/>Job Creation & Monitoring]
        NS_WATCH[Namespace Watcher<br/>Project Setup]
        SETTINGS_WATCH[ProjectSettings Watcher<br/>Configuration Sync]
    end

    subgraph "Reconciliation"
        RECONCILE[Reconcile Logic<br/>Desired vs Actual State]
        STATUS[Status Updates<br/>UpdateStatus Subresource]
    end

    subgraph "Job Management"
        CREATE[Job Creation<br/>Pod Spec Generation]
        MONITOR[Job Monitoring<br/>Completion Detection]
        CLEANUP[Resource Cleanup<br/>OwnerReference Cascade]
    end

    MAIN --> SESSION_WATCH
    MAIN --> NS_WATCH
    MAIN --> SETTINGS_WATCH

    SESSION_WATCH --> RECONCILE
    RECONCILE --> CREATE
    RECONCILE --> STATUS
    CREATE --> MONITOR
    MONITOR --> STATUS
    MONITOR --> CLEANUP

    classDef watch fill:#326ce5,stroke:#1a4b99,stroke-width:2px
    classDef logic fill:#00add8,stroke:#007d9c,stroke-width:2px
    classDef job fill:#4a9eff,stroke:#2060c0,stroke-width:2px

    class MAIN,SESSION_WATCH,NS_WATCH,SETTINGS_WATCH watch
    class RECONCILE,STATUS logic
    class CREATE,MONITOR,CLEANUP job
```

### Claude Code Runner (Python)

```mermaid
graph TB
    subgraph "Initialization"
        INIT[Container Start<br/>Load Configuration]
        SECRET_READ[Read Secrets<br/>API Keys & Tokens]
    end

    subgraph "Repository Setup"
        CLONE[Git Clone<br/>Multi-Repo Support]
        WORKSPACE[Workspace Setup<br/>PVC Mount]
    end

    subgraph "AI Execution"
        SDK[Claude Code SDK<br/>Multi-Agent Support]
        ANTHROPIC_CLIENT[Anthropic Client<br/>Streaming API]
    end

    subgraph "Tool Execution"
        READ[Read Tool<br/>File Operations]
        WRITE[Write Tool<br/>File Creation]
        EDIT[Edit Tool<br/>Precise Modifications]
        BASH[Bash Tool<br/>Command Execution]
        GREP[Grep Tool<br/>Code Search]
        GLOB[Glob Tool<br/>File Pattern Matching]
    end

    subgraph "Output & Status"
        RESULTS[Result Aggregation<br/>Session Output]
        STATUS_UPDATE[Status Updates<br/>CR Annotation]
        GIT_PUSH[Git Push<br/>Commit & PR]
    end

    INIT --> SECRET_READ
    SECRET_READ --> CLONE
    CLONE --> WORKSPACE
    WORKSPACE --> SDK
    SDK --> ANTHROPIC_CLIENT

    ANTHROPIC_CLIENT --> READ
    ANTHROPIC_CLIENT --> WRITE
    ANTHROPIC_CLIENT --> EDIT
    ANTHROPIC_CLIENT --> BASH
    ANTHROPIC_CLIENT --> GREP
    ANTHROPIC_CLIENT --> GLOB

    READ --> RESULTS
    WRITE --> RESULTS
    EDIT --> RESULTS
    BASH --> RESULTS
    GREP --> RESULTS
    GLOB --> RESULTS

    RESULTS --> STATUS_UPDATE
    RESULTS --> GIT_PUSH

    classDef init fill:#3776ab,stroke:#204060,stroke-width:2px
    classDef repo fill:#4a9eff,stroke:#2060c0,stroke-width:2px
    classDef ai fill:#ff6b6b,stroke:#cc5555,stroke-width:2px
    classDef tool fill:#51cf66,stroke:#40a647,stroke-width:2px
    classDef output fill:#ffd43b,stroke:#ccaa2e,stroke-width:2px

    class INIT,SECRET_READ init
    class CLONE,WORKSPACE repo
    class SDK,ANTHROPIC_CLIENT ai
    class READ,WRITE,EDIT,BASH,GREP,GLOB tool
    class RESULTS,STATUS_UPDATE,GIT_PUSH output
```

## Data Flow Architecture

### Session Creation Flow

```mermaid
graph LR
    USER[User Input] --> |1. Submit Form| FRONTEND[Frontend]
    FRONTEND --> |2. POST Request<br/>Bearer Token| BACKEND[Backend API]
    BACKEND --> |3. Validate Token<br/>RBAC Check| K8S_AUTH[K8s AuthZ]
    K8S_AUTH --> |4. Authorized| BACKEND
    BACKEND --> |5. Create CR| K8S_API[Kubernetes API]
    K8S_API --> |6. CR Stored| ETCD[etcd]
    ETCD --> |7. Watch Event| OPERATOR[Operator]
    OPERATOR --> |8. Create Job| K8S_API
    K8S_API --> |9. Schedule Pod| SCHEDULER[K8s Scheduler]
    SCHEDULER --> |10. Assign Node| NODE[Cluster Node]
    NODE --> |11. Pull Image<br/>Start Container| RUNNER[Runner Pod]

    classDef user fill:#61dafb,stroke:#20232a,stroke-width:2px
    classDef app fill:#00add8,stroke:#007d9c,stroke-width:2px
    classDef k8s fill:#326ce5,stroke:#1a4b99,stroke-width:2px
    classDef runtime fill:#3776ab,stroke:#204060,stroke-width:2px

    class USER,FRONTEND user
    class BACKEND app
    class K8S_AUTH,K8S_API,ETCD,SCHEDULER,NODE k8s
    class OPERATOR,RUNNER runtime
```

### Authentication & Authorization Flow

```mermaid
sequenceDiagram
    participant User
    participant Browser
    participant OAuth as OAuth Proxy
    participant Frontend
    participant Backend
    participant K8s as Kubernetes RBAC

    User->>Browser: Access Application
    Browser->>OAuth: Request Page

    alt Not Authenticated
        OAuth->>User: Redirect to Login
        User->>OAuth: OpenShift Credentials
        OAuth->>OAuth: Validate User
        OAuth->>Browser: Set Cookie + Token
    end

    Browser->>Frontend: Load Application<br/>(with token)
    Frontend->>Backend: API Request<br/>Authorization: Bearer {token}

    Backend->>Backend: Extract Token from Header
    Backend->>K8s: Create K8s Client<br/>(User Token)

    Backend->>K8s: SelfSubjectAccessReview<br/>(Check Permissions)
    K8s-->>Backend: Allowed/Denied

    alt Authorized
        Backend->>K8s: Perform Operation<br/>(List, Create, Update, Delete)
        K8s-->>Backend: Resource Data
        Backend-->>Frontend: 200 OK + Data
    else Unauthorized
        Backend-->>Frontend: 401/403 Error
        Frontend-->>User: Show Error
    end
```

## Multi-Tenancy Architecture

```mermaid
graph TB
    subgraph "Shared Infrastructure"
        FRONTEND[Frontend Pod<br/>Shared UI]
        BACKEND[Backend Pod<br/>Shared API]
        OPERATOR[Operator Pod<br/>Shared Controller]
    end

    subgraph "Project A Namespace"
        PA_CR[AgenticSessions<br/>Project A]
        PA_SETTINGS[ProjectSettings<br/>API Keys]
        PA_JOB[Jobs<br/>Session Pods]
        PA_PVC[PVCs<br/>Workspaces]
    end

    subgraph "Project B Namespace"
        PB_CR[AgenticSessions<br/>Project B]
        PB_SETTINGS[ProjectSettings<br/>API Keys]
        PB_JOB[Jobs<br/>Session Pods]
        PB_PVC[PVCs<br/>Workspaces]
    end

    subgraph "RBAC Isolation"
        RA[RoleBinding A<br/>User A → Project A]
        RB[RoleBinding B<br/>User B → Project B]
    end

    FRONTEND -.->|User A Token| BACKEND
    FRONTEND -.->|User B Token| BACKEND

    BACKEND -->|User A Operations| PA_CR
    BACKEND -->|User B Operations| PB_CR

    OPERATOR -->|Watch All| PA_CR
    OPERATOR -->|Watch All| PB_CR

    PA_CR --> PA_JOB
    PB_CR --> PB_JOB

    RA -.->|Enforce| PA_CR
    RB -.->|Enforce| PB_CR

    classDef shared fill:#00add8,stroke:#007d9c,stroke-width:2px
    classDef projecta fill:#51cf66,stroke:#40a647,stroke-width:2px
    classDef projectb fill:#ffd43b,stroke:#ccaa2e,stroke-width:2px
    classDef rbac fill:#ff6b6b,stroke:#cc5555,stroke-width:2px

    class FRONTEND,BACKEND,OPERATOR shared
    class PA_CR,PA_SETTINGS,PA_JOB,PA_PVC projecta
    class PB_CR,PB_SETTINGS,PB_JOB,PB_PVC projectb
    class RA,RB rbac
```

## Deployment Architecture

### Development Environment (OpenShift Local)

```mermaid
graph TB
    subgraph "OpenShift Local (CRC)"
        subgraph "vteam-dev Namespace"
            ROUTE[OpenShift Route<br/>*.apps-crc.testing]
            FE_SVC[Frontend Service<br/>ClusterIP]
            BE_SVC[Backend Service<br/>ClusterIP]

            FE_POD[Frontend Pod<br/>NextJS Dev Server]
            BE_POD[Backend Pod<br/>Go API]
            OP_POD[Operator Pod<br/>Watch Controller]

            RUNNER_JOB[Runner Jobs<br/>Session Execution]

            PVC_STORAGE[PVCs<br/>Session Workspaces]
        end

        ROUTE --> FE_SVC
        ROUTE --> BE_SVC
        FE_SVC --> FE_POD
        BE_SVC --> BE_POD

        OP_POD -.->|Create| RUNNER_JOB
        RUNNER_JOB -.->|Mount| PVC_STORAGE
    end

    DEV[Developer] -->|Browser| ROUTE
    FE_POD -->|Hot Reload| DEV_FILES[Local Files<br/>File Sync]

    classDef route fill:#ff9900,stroke:#cc7a00,stroke-width:2px
    classDef service fill:#326ce5,stroke:#1a4b99,stroke-width:2px
    classDef pod fill:#00add8,stroke:#007d9c,stroke-width:2px
    classDef storage fill:#51cf66,stroke:#40a647,stroke-width:2px

    class ROUTE route
    class FE_SVC,BE_SVC service
    class FE_POD,BE_POD,OP_POD,RUNNER_JOB pod
    class PVC_STORAGE storage
```

### Production Environment (OpenShift Cluster)

```mermaid
graph TB
    subgraph "Production Cluster"
        subgraph "Ingress Layer"
            LB[Load Balancer<br/>External IP]
            ROUTER[OpenShift Router<br/>HAProxy]
        end

        subgraph "ambient-code Namespace"
            OAUTH[OAuth Proxy<br/>Authentication]

            FE_DEPLOY[Frontend Deployment<br/>Replicas: 3<br/>HPA Enabled]
            BE_DEPLOY[Backend Deployment<br/>Replicas: 3<br/>HPA Enabled]
            OP_DEPLOY[Operator Deployment<br/>Replicas: 1]

            FE_SVC[Frontend Service]
            BE_SVC[Backend Service]

            CONFIG[ConfigMaps<br/>Environment Config]
            SECRETS[Secrets<br/>API Keys, Tokens]
        end

        subgraph "Project Namespaces"
            PROJ1[Project 1<br/>AgenticSessions + Jobs]
            PROJ2[Project 2<br/>AgenticSessions + Jobs]
            PROJN[Project N<br/>AgenticSessions + Jobs]
        end

        subgraph "Storage"
            SC[StorageClass<br/>Dynamic Provisioning]
            PV_POOL[Persistent Volumes<br/>Workspace Storage]
        end
    end

    USERS[End Users] -->|HTTPS| LB
    LB --> ROUTER
    ROUTER --> OAUTH
    OAUTH --> FE_SVC
    FE_SVC --> FE_DEPLOY
    FE_DEPLOY --> BE_SVC
    BE_SVC --> BE_DEPLOY

    BE_DEPLOY -.->|RBAC| PROJ1
    BE_DEPLOY -.->|RBAC| PROJ2
    BE_DEPLOY -.->|RBAC| PROJN

    OP_DEPLOY -.->|Watch| PROJ1
    OP_DEPLOY -.->|Watch| PROJ2
    OP_DEPLOY -.->|Watch| PROJN

    FE_DEPLOY -.->|Mount| CONFIG
    BE_DEPLOY -.->|Mount| CONFIG
    OP_DEPLOY -.->|Mount| CONFIG

    PROJ1 -.->|Provision| SC
    PROJ2 -.->|Provision| SC
    PROJN -.->|Provision| SC
    SC -.->|Create| PV_POOL

    classDef ingress fill:#ff9900,stroke:#cc7a00,stroke-width:2px
    classDef app fill:#00add8,stroke:#007d9c,stroke-width:2px
    classDef project fill:#51cf66,stroke:#40a647,stroke-width:2px
    classDef storage fill:#ffd43b,stroke:#ccaa2e,stroke-width:2px
    classDef auth fill:#ff6b6b,stroke:#cc5555,stroke-width:2px

    class LB,ROUTER ingress
    class FE_DEPLOY,BE_DEPLOY,OP_DEPLOY,FE_SVC,BE_SVC,CONFIG,SECRETS app
    class PROJ1,PROJ2,PROJN project
    class SC,PV_POOL storage
    class OAUTH auth
```

## Key Architectural Principles

### 1. Kubernetes-Native Design

- **Custom Resource Definitions (CRDs)**: AgenticSession, ProjectSettings, RFEWorkflow
- **Operator Pattern**: Reconciliation loop watches for CR changes
- **Job-based Execution**: Stateless runner pods for AI tasks
- **OwnerReferences**: Automatic resource cleanup via Kubernetes garbage collection

### 2. Security-First Architecture

- **User Token Authentication**: All API operations use user's Kubernetes token
- **RBAC Enforcement**: Namespace-scoped permissions via RoleBindings
- **Service Account Isolation**: Backend service account only for CR write operations
- **Secret Management**: Kubernetes Secrets for API keys and Git tokens

### 3. Multi-Tenancy

- **Project-based Isolation**: Each project maps to a Kubernetes namespace
- **Resource Quotas**: Per-namespace CPU/memory limits
- **Network Policies**: Component isolation and secure communication
- **Audit Logging**: Track all user operations

### 4. Scalability & Performance

- **Horizontal Pod Autoscaling**: Frontend and Backend scale with load
- **Concurrent Job Execution**: Multiple sessions run in parallel
- **Resource Limits**: Proper requests/limits for optimal scheduling
- **WebSocket Streaming**: Real-time status updates without polling

### 5. Extensibility

- **Multi-Agent Support**: Claude Code SDK enables specialized agents
- **Multi-Repo Sessions**: Operate on multiple repositories simultaneously
- **Custom Workflows**: RFE workflows orchestrate multi-step processes
- **Provider Agnostic**: GitHub and GitLab support with extensible design

## Component Communication

### Protocol Matrix

| Source | Target | Protocol | Port | Purpose |
|--------|--------|----------|------|---------|
| Frontend | Backend | HTTP/HTTPS | 8080 | REST API calls |
| Frontend | Backend | WebSocket | 8080 | Real-time status updates |
| Backend | Kubernetes API | HTTPS | 6443 | CR operations (user token) |
| Backend | Kubernetes API | HTTPS | 6443 | CR write (service account) |
| Operator | Kubernetes API | HTTPS | 6443 | Watch CRs, Create Jobs |
| Runner | Anthropic API | HTTPS | 443 | AI model inference |
| Runner | GitHub API | HTTPS | 443 | Repository operations |
| Runner | GitLab API | HTTPS | 443 | Repository operations |
| User | Frontend | HTTPS | 443 | Browser access |

### Network Topology

```mermaid
graph TB
    subgraph "External Network"
        USERS[Users<br/>Internet]
        ANTHROPIC[Anthropic API<br/>claude.ai]
        GITHUB[GitHub<br/>github.com]
        GITLAB[GitLab<br/>gitlab.com]
    end

    subgraph "Cluster Network"
        subgraph "Public Services"
            INGRESS[Ingress Controller<br/>Port 443]
        end

        subgraph "Internal Services"
            FE[Frontend Service<br/>ClusterIP:3000]
            BE[Backend Service<br/>ClusterIP:8080]
        end

        subgraph "Control Plane"
            K8S_API[Kubernetes API<br/>Port 6443]
        end

        subgraph "Pods"
            FE_POD[Frontend Pods]
            BE_POD[Backend Pods]
            OP_POD[Operator Pod]
            RUNNER_POD[Runner Pods]
        end
    end

    USERS -->|HTTPS| INGRESS
    INGRESS --> FE
    FE --> FE_POD
    FE_POD -->|HTTP| BE
    BE --> BE_POD
    BE_POD -->|HTTPS| K8S_API
    OP_POD -->|HTTPS| K8S_API
    RUNNER_POD -->|HTTPS| ANTHROPIC
    RUNNER_POD -->|HTTPS| GITHUB
    RUNNER_POD -->|HTTPS| GITLAB

    classDef external fill:#ff9900,stroke:#cc7a00,stroke-width:2px
    classDef public fill:#ff6b6b,stroke:#cc5555,stroke-width:2px
    classDef internal fill:#51cf66,stroke:#40a647,stroke-width:2px
    classDef control fill:#326ce5,stroke:#1a4b99,stroke-width:2px
    classDef pod fill:#00add8,stroke:#007d9c,stroke-width:2px

    class USERS,ANTHROPIC,GITHUB,GITLAB external
    class INGRESS public
    class FE,BE internal
    class K8S_API control
    class FE_POD,BE_POD,OP_POD,RUNNER_POD pod
```

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - Development standards and patterns
- [README.md](../../README.md) - Project overview and quick start
- [ADR-0001: Kubernetes-Native Architecture](../adr/0001-kubernetes-native-architecture.md)
- [ADR-0002: User Token Authentication](../adr/0002-user-token-authentication.md)
- [Backend Development Context](../../.claude/context/backend-development.md)
- [Frontend Development Context](../../.claude/context/frontend-development.md)
