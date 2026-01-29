# Architecture Documentation

Technical architecture documentation for the Ambient Code Platform.

## 📐 Overview

The Ambient Code Platform follows a Kubernetes-native microservices architecture with Custom Resources, Operators, and Job-based execution.

```
User → Frontend → Backend API → K8s Operator → Runner Jobs → Claude Code CLI
```

## 🗂️ Architecture Documentation

### System Design
- **System Context** - High-level system boundaries and external integrations
- **Component Architecture** - Individual component designs
- **Data Flow** - How data moves through the system
- **Security Architecture** - Authentication, authorization, and security patterns

### Diagrams
**[Architecture Diagrams](diagrams/)** - Visual system representations
- [Platform Architecture](../platform-architecture.mmd) - Complete system diagram
- [Component Structure](../component-structure.mmd) - Component relationships
- [Deployment Stack](../deployment-stack.mmd) - Deployment topology
- [Agentic Session Flow](../agentic-session-flow.mmd) - Session lifecycle
- [UX Feature Workflow](diagrams/ux-feature-workflow.md) - Multi-agent workflow

### Key Components

#### Frontend (Next.js + Shadcn UI)
**Purpose:** Web interface for session management and monitoring

**Key Features:**
- Project and session CRUD operations
- Real-time WebSocket updates
- Repository browsing
- Multi-agent chat interface

**Documentation:** [components/frontend/README.md](../../components/frontend/README.md)

---

#### Backend API (Go + Gin)
**Purpose:** REST API managing Kubernetes Custom Resources

**Key Features:**
- Project-scoped endpoints with multi-tenant isolation
- User token-based authentication
- Git operations (clone, fork, PR creation)
- WebSocket support for real-time updates

**Documentation:** [components/backend/README.md](../../components/backend/README.md)

---

#### Agentic Operator (Go)
**Purpose:** Kubernetes controller watching Custom Resources

**Key Features:**
- Watches AgenticSession CRs and creates Jobs
- Monitors Job execution and updates CR status
- Handles timeouts and cleanup
- Manages runner pod lifecycle

**Documentation:** [components/operator/README.md](../../components/operator/README.md)

---

#### Claude Code Runner (Python)
**Purpose:** Job pod executing Claude Code CLI

**Key Features:**
- Claude Code SDK integration
- Multi-agent collaboration
- Workspace synchronization via PVC
- Anthropic API streaming

**Documentation:** [components/runners/claude-code-runner/README.md](../../components/runners/claude-code-runner/README.md)

---

## 🎯 Core Concepts

### Custom Resource Definitions (CRDs)

**AgenticSession** - Represents an AI execution session
- Spec: prompt, repos, interactive mode, timeout, model
- Status: phase, startTime, completionTime, results

**ProjectSettings** - Project-scoped configuration
- API keys, default models, timeout settings
- Namespace-isolated for multi-tenancy

**RFEWorkflow** - Request For Enhancement workflows
- 7-step agent council process
- Multi-agent collaboration

### Multi-Tenancy

- Each **project** maps to a Kubernetes **namespace**
- RBAC enforces namespace-scoped access
- User tokens determine permissions
- No cross-project data access

### Authentication & Authorization

- **Authentication:** OpenShift OAuth (production) or test tokens (dev)
- **Authorization:** User tokens with namespace-scoped RBAC
- **Backend Pattern:** Always use user-scoped K8s clients for operations
- **Security:** Token redaction, no service account fallback

See [ADR-0002: User Token Authentication](../adr/0002-user-token-authentication.md)

## 📋 Architectural Decision Records

**[ADR Directory](../adr/)** - Why we made key technical decisions

| ADR | Title | Status |
|-----|-------|--------|
| [0001](../adr/0001-kubernetes-native-architecture.md) | Kubernetes-Native Architecture | Accepted |
| [0002](../adr/0002-user-token-authentication.md) | User Token Authentication | Accepted |
| [0003](../adr/0003-multi-repo-support.md) | Multi-Repo Support | Accepted |
| [0004](../adr/0004-go-backend-python-runner.md) | Go Backend + Python Runner | Accepted |
| [0005](../adr/0005-nextjs-shadcn-react-query.md) | Next.js + Shadcn + React Query | Accepted |

**Format:** We follow the [ADR template](../adr/template.md) for all architectural decisions.

## 🔄 Request Flow

### Creating an Agentic Session

1. **User** submits session via web UI
2. **Frontend** sends POST to `/api/projects/:project/agentic-sessions`
3. **Backend** validates user token and creates `AgenticSession` CR
4. **Operator** watches CR, creates Kubernetes Job
5. **Job** runs Claude Code runner pod
6. **Runner** executes Claude Code CLI, streams results
7. **Operator** monitors Job, updates CR status
8. **Frontend** displays real-time updates via WebSocket

### Data Flow

```
User Input → Frontend (Next.js)
    ↓
Backend API (Go) → User Token Validation → RBAC Check
    ↓
Kubernetes API → AgenticSession CR created
    ↓
Operator (Go) → Watches CR → Creates Job
    ↓
Runner Pod (Python) → Executes Claude Code → Streams events
    ↓
Operator → Updates CR Status
    ↓
Backend → WebSocket → Frontend → User sees results
```

## 🔐 Security Architecture

### Authentication Layers
1. **OpenShift OAuth** (production) - Cluster-based identity
2. **User Tokens** - Bearer tokens for API authentication
3. **Service Accounts** - Limited to CR writes and token minting

### Authorization Model
- **Namespace-scoped RBAC** - Users only see their authorized projects
- **User-scoped K8s clients** - All API operations use user credentials
- **No privilege escalation** - Backend never falls back to service account

See [Security Standards](../../CLAUDE.md#security-patterns)

## 🧪 Testing Architecture

- **Unit Tests** - Component logic testing (Go, TypeScript)
- **Contract Tests** - API contract validation (Go)
- **Integration Tests** - End-to-end with real K8s (Go)
- **E2E Tests** - User journey testing with Cypress (Kind cluster)

See [Testing Documentation](testing/)

## 📚 Additional Resources

- **[Decisions Log](decisions.md)** - Chronological decision history
- **[Design Documents](design/)** - Feature design proposals
- **[Implementation Plans](implementation-plans/)** - Detailed implementation guides
- **[Labs](labs/)** - Hands-on learning exercises

## 🤝 Contributing to Architecture

When proposing architectural changes:

1. **Check existing ADRs** - Understand current decisions
2. **Draft ADR** - Use [template](adr/template.md)
3. **Discuss** - GitHub Discussions or issue
4. **Review** - Get feedback from maintainers
5. **Implement** - Code + tests + documentation
6. **Update** - Mark ADR as accepted, update relevant docs

---

**Questions?** Open a [GitHub Discussion](https://github.com/ambient-code/vTeam/discussions)
