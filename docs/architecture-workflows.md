# Architecture Workflows

Visual diagrams explaining how components interact in the Ambient Code Platform.

**Last Updated:** 2025-12-04

**For:** New contributors understanding the system

## Diagrams
### 1. High-Level System Architecture
This chart describes how the main components (Frontend, Backend, Operator, Runner) interact to execute an agentic session.

```mermaid
graph TD
    User[User] -->|Interacts| FE[Frontend NextJS]
    FE -->|REST API / WebSocket| BE[Backend API Go]
    
    subgraph K8sCluster [Kubernetes Cluster]
        BE -->|Creates/Updates CR| K8sAPI[Kubernetes API]
        
        Operator[Agentic Operator] -->|Watches CRs| K8sAPI
        Operator -->|Creates Job| Job[Runner Job]
        
        subgraph RunnerPod [Runner Pod]
            Runner[Claude Code Runner Python]
            RunnerShell[Runner Shell]
            ClaudeCLI[Claude Code CLI]
        end
        
        Job -->|Runs| Runner
        Runner -->|Connects WS| BE
    end
    
    Runner -->|Clones/Pushes| Git[External Git Provider]
    BE -->|Auth/Metadata| Git

    style FE fill:#e1f5fe,stroke:#01579b
    style BE fill:#e8f5e9,stroke:#1b5e20
    style Operator fill:#fff3e0,stroke:#e65100
    style Runner fill:#f3e5f5,stroke:#4a148c
```

#### Workflow Explanation
1.  **User** starts a session in the **Frontend**.
2.  **Frontend** calls **Backend** to create an `AgenticSession` Custom Resource (CR) in Kubernetes.
3.  **Operator** watches for the new CR and creates a Kubernetes **Job** running the **Runner** image.
4.  **Runner** starts up, connects back to the **Backend** via WebSocket, and executes the **Claude Code CLI**.
5.  Real-time logs and status are sent from Runner -> Backend -> Frontend.

---

### 2. Backend Internal Interactions
This chart details how the Backend service is organized and handles requests.

```mermaid
graph LR
    Request[HTTP Request] --> Router[Gin Router]
    
    subgraph BackendComponents [Backend Components]
        Router -->|Middleware Chain| Middleware[Auth & RBAC]
        Middleware -->|/api/projects...| SessionHandler[Session Handler]
        Router -->|/ws/...| WSHandler[WebSocket Handler]
        Router -->|/auth/...| AuthHandler[Auth Handler]
        
        SessionHandler -->|Read/List| UserClient[User-Scoped Client]
        SessionHandler -->|Write/Mint| SAClient[Backend SA Client]
        SessionHandler -->|Git Ops| GitService[Git Integration]
        
        WSHandler -->|Broadcasts| Hub[WebSocket Hub]
        Hub -->|Updates| SessionHandler
    end
    
    UserClient -->|RBAC Access| K8s[Kubernetes]
    SAClient -->|Privileged Ops| K8s
    GitService -->|API Calls| ExternalGit[GitHub/GitLab]

    style SessionHandler fill:#fff9c4
    style WSHandler fill:#fff9c4
    style UserClient fill:#bbdefb
    style SAClient fill:#ffccbc
    style Middleware fill:#e1bee7
```

#### Key Internal Flows
-   **Session Management:** `handlers/sessions.go` uses user-scoped clients for reads/validation and the backend service account for CR writes after RBAC authorization.
-   **Real-time Updates:** `websocket/` package manages connections from both the Frontend (users) and Runners (agents).
-   **Git Integration:** `git/` and `github/` packages handle token exchange and repository operations.
-   **Authentication & RBAC:** All user operations validated with user bearer tokens; backend service account used only for privileged operations after authorization checks.

---

### 3. Backend <-> Runner Interaction
This chart focuses on the communication between the running Agent (Runner) and the Backend.

```mermaid
sequenceDiagram
    participant Runner as Runner (Python)
    participant Backend as Backend (Go)
    participant K8s as Kubernetes
    
    Note over Runner, Backend: Initialization
    K8s->>Runner: Start Pod
    Runner->>Backend: WebSocket Connect /sessions/:id/ws
    Backend-->>Runner: Auth and Context
    
    Note over Runner, Backend: Execution Loop
    loop Execution
        Runner->>Runner: Execute Claude Step
        Runner->>Backend: Send Logs (WS Message)
        Runner->>Backend: Send Status Update (WS Message)
        
        opt Git Operations
            Runner->>Backend: Git Push/Pull Request
        end
    end
    
    Note over Runner, Backend: Completion
    Runner->>Backend: Final Result and Exit Code
    Backend->>K8s: Update Session Status (Phase: Completed)
```

#### Details
-   **Communication:** The Runner uses a wrapper (`wrapper.py`) around the `runner-shell` library to establish a persistent WebSocket connection.
-   **Protocol:** Messages include logs, status changes, and content synchronization.
-   **State:** The Backend updates the Kubernetes CR status based on these messages, which the Operator eventually observes.

---

### 4. Frontend <-> Backend Interaction
This chart shows how the User Interface communicates with the Backend.

```mermaid
sequenceDiagram
    participant FE as Frontend
    participant BE as Backend
    
    Note over FE, BE: Session Creation
    FE->>BE: POST /projects/:id/agentic-sessions
    BE->>BE: Create K8s Custom Resource
    BE-->>FE: 201 Created (Session JSON)
    
    Note over FE, BE: Monitoring
    FE->>BE: GET /projects/:id/agentic-sessions
    FE->>BE: WebSocket Connect /sessions/:id/ws
    
    loop Real-time Updates
        BE-->>FE: WS Message (Log Line)
        BE-->>FE: WS Message (Status: Running -> Completed)
    end
    
    Note over FE, BE: Session Control
    opt Stop Session
        FE->>BE: POST .../sessions/:id/stop
        BE->>BE: Update CR (DesiredPhase: Stopped)
    end
```

#### Key Interactions
-   **API Services:** Located in `frontend/src/services/api/`, these TypeScript modules handle strongly-typed REST calls.
-   **Live Data:** The frontend subscribes to the same WebSocket channels that the Runner publishes to, allowing users to see the agent's "thought process" and terminal output in real-time.

---

### 5. Operator <-> Runner Lifecycle
This chart illustrates how the Operator manages the Runner's lifecycle from creation to completion.

```mermaid
graph TD
    subgraph OperatorLogic [Operator Logic]
        Watch[Watch CR Event] -->|Phase: Pending| CreateJob[Create Kubernetes Job]
        CreateJob -->|Inject Env| Config[Env Vars: API Keys, Repo URL]
        CreateJob -->|Mount| PVC[Workspace PVC]
        
        CreateJob --> Monitor[Monitor Job Routine]
        Monitor -->|Poll Status| PodStatus[Check Pod Phase]
        
        PodStatus -->|Running| UpdateCR1[Update CR: Phase=Running]
        PodStatus -->|Completed| UpdateCR2[Update CR: Phase=Completed]
        PodStatus -->|Failed| UpdateCR3[Update CR: Phase=Failed]
        
        UpdateCR1 --> K8sAPI
        UpdateCR2 --> K8sAPI
        UpdateCR3 --> K8sAPI
    end

    subgraph K8s [Kubernetes]
        K8sAPI -->|Event| Watch
        CreateJob -->|Apply| JobResource[Job Resource]
        JobResource -->|Spawns| Pod[Runner Pod]
    end

    style OperatorLogic fill:#fff3e0,stroke:#e65100
    style K8s fill:#f3e5f5,stroke:#4a148c
```

#### Lifecycle Details
-   **Provisioning:** The Operator detects a new `AgenticSession` CR (Phase: Pending) and creates a batch `Job`.
-   **Configuration:** It injects critical environment variables (e.g., `ANTHROPIC_API_KEY` or `CLAUDE_CODE_USE_VERTEX`, `GIT_REPO_URL`) and mounts a Persistent Volume Claim (PVC) for the workspace.
-   **Monitoring:** A dedicated goroutine polls the Job/Pod status and updates the CR's `status.phase` accordingly.
-   **Cleanup:** If the session is stopped (DesiredPhase: Stopped), the Operator deletes the Job and Pods.

---

### 6. Runner Internal Execution Flow
This chart details the internal logic of the `wrapper.py` script running inside the container.

```mermaid
sequenceDiagram
    participant Wrapper as wrapper.py
    participant SDK as Claude SDK
    participant API as Anthropic API
    participant WS as WebSocket (Backend)
    participant Git as Git Remote

    Note over Wrapper: Startup
    Wrapper->>Wrapper: Parse Env Vars
    Wrapper->>Wrapper: Setup Workspace & Clone Repo
    Wrapper->>WS: Connect & Authenticate
    
    Note over Wrapper: AI Execution
    Wrapper->>SDK: Initialize (ClaudeAgentOptions)
    Wrapper->>SDK: client.send_message(prompt)
    
    loop Response Stream
        SDK->>API: Request Completion
        API-->>SDK: Stream Events
        
        alt Text Block
            SDK-->>Wrapper: TextChunk
            Wrapper->>WS: Send "agent_message"
        else Tool Use
            SDK-->>Wrapper: ToolUse (Read/Write/Exec)
            Wrapper->>WS: Send "tool_use"
        else Thinking
            SDK-->>Wrapper: ThinkingBlock
            Wrapper->>WS: Send "thinking"
        end
    end
    
    SDK-->>Wrapper: Final ResultMessage
    
    Note over Wrapper: Output Handling
    alt Auto Push Enabled
        Wrapper->>Git: git push output_branch
        opt Create PR
            Wrapper->>Git: Create Pull Request
        end
    end
    
    Wrapper->>Wrapper: Exit (0=Success, 1=Error)
```

#### Execution Details
-   **Engine:** The runner uses the **Claude Code SDK** (Anthropic) to drive the agentic loop. It does **not** currently use the OpenAI API, though the architecture is extensible.
-   **Streaming:** All "thoughts", terminal outputs, and tool executions are streamed in real-time to the user via the WebSocket connection.
-   **Output:**
    1.  **Real-time:** The conversation stream.
    2.  **Persistent:** Changes made to the filesystem (code edits) are persisted in the PVC and optionally pushed to the remote Git repository (`git push`) upon completion.
