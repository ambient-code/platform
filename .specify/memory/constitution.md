# Ambient Code Platform Constitution

**Version**: 2.0.0
**Status**: RATIFIED
**Ratified**: 2025-01-22
**Last Amended**: 2025-01-22
**Spec-Kit Compatible**: Yes

---

## Preamble

This constitution establishes the foundational principles, development standards, and governance framework for the **Ambient Code Platform** (formerly vTeam). It serves as the authoritative guide for all technical decisions, architectural patterns, and development practices.

The Ambient Code Platform is a Kubernetes-native AI automation platform that combines Claude Code CLI with multi-agent collaboration capabilities, enabling intelligent agentic sessions through a modern web interface.

### Purpose & Scope

This constitution:
- Defines non-negotiable technical principles that ensure platform quality, security, and scalability
- Establishes development standards across all components (Frontend, Backend, Operator, Runner)
- Provides governance processes for amendments and compliance
- Guides AI agents and human developers in consistent, high-quality implementation

### Constitutional Authority

This constitution supersedes all other development guidelines, coding standards, and best practices documentation. When conflicts arise, constitutional principles take precedence.

---

## Table of Contents

1. [Core Principles](#core-principles)
2. [Development Standards](#development-standards)
3. [Deployment & Operations](#deployment--operations)
4. [Governance](#governance)
5. [Amendment History](#amendment-history)

---

## Core Principles

### I. Kubernetes-Native Architecture

**Status**: MANDATORY
**Applies To**: All components (Backend, Operator, Runner orchestration)

All features MUST be built using Kubernetes primitives and patterns:

- **Custom Resource Definitions (CRDs)** for domain objects
  - `AgenticSession` - AI execution sessions
  - `ProjectSettings` - Multi-tenant project configuration
  - `RFEWorkflow` - Request for Enhancement workflows

- **Operators** for reconciliation loops and lifecycle management
  - Watch for CR changes
  - Reconcile to desired state
  - Handle edge cases and failure scenarios

- **Jobs** for execution workloads
  - Proper resource limits (CPU, memory)
  - Timeout configuration
  - Failure policies and retry logic

- **ConfigMaps and Secrets** for configuration management
  - Secrets for sensitive data (API keys, tokens)
  - ConfigMaps for non-sensitive configuration
  - Project-scoped isolation

- **Services and Routes** for network exposure
  - Internal ClusterIP for inter-component communication
  - NodePort/LoadBalancer/Ingress for external access
  - TLS termination at appropriate layers

- **RBAC** for authorization boundaries
  - Namespace-scoped roles for multi-tenancy
  - ClusterRoles only when truly cluster-wide access needed
  - Service accounts with minimal permissions

**Rationale**: Kubernetes-native design ensures portability, scalability, and enterprise-grade operational tooling. Violations create operational complexity, reduce platform value, and prevent leveraging Kubernetes ecosystem tools.

**Violation Consequences**: Features built outside Kubernetes patterns create:
- Operational silos requiring custom tooling
- Scaling limitations
- Security gaps in multi-tenant isolation
- Incompatibility with enterprise deployment requirements

---

### II. Security & Multi-Tenancy First

**Status**: NON-NEGOTIABLE
**Applies To**: All components with user-facing endpoints or resource access

Security and isolation MUST be embedded in every component from initial design:

#### Authentication
- All user-facing endpoints MUST use user tokens via `GetK8sClientsForRequest()`
- No unauthenticated endpoints except health checks and metrics
- Token validation on every request
- Session management following security best practices

#### Authorization
- RBAC checks MUST be performed before resource access
- Project-scoped authorization enforced at API layer
- Validate user permissions before Kubernetes client operations
- Deny by default - explicit grants required

#### Token Security
- NEVER log tokens, API keys, or sensitive headers
- Use redaction in logs (e.g., `token=****`)
- Secrets stored in Kubernetes Secrets with appropriate RBAC
- Rotate tokens regularly (document rotation procedures)

#### Multi-Tenancy
- Project-scoped namespaces with strict isolation
- Network policies preventing cross-project access
- Resource quotas per project/namespace
- Separate service accounts per project

#### Principle of Least Privilege
- Service accounts with minimal permissions
- No cluster-admin except for installation
- Namespace-admin only for project owners
- Read-only access by default

#### Container Security
- SecurityContext with `AllowPrivilegeEscalation: false`
- Drop all capabilities, add only required ones
- Run as non-root user (UID > 1000)
- Read-only root filesystem where possible

#### Backend Service Account Usage
- Backend service account ONLY for:
  - CR writes to Kubernetes
  - Token minting for temporary access
- NEVER use as fallback for failed user authentication
- Never use for user resource access

**Rationale**: Security breaches and privilege escalation destroy trust and platform viability. Multi-tenant isolation is non-negotiable for enterprise deployment. Security cannot be retrofitted - it must be foundational.

**Violation Consequences**: Security violations can lead to:
- Unauthorized access to sensitive data
- Cross-tenant data leakage
- Privilege escalation attacks
- Compliance failures and legal liability
- Complete loss of customer trust

---

### III. Type Safety & Error Handling

**Status**: NON-NEGOTIABLE
**Applies To**: All production code paths (handlers, reconcilers, business logic)

Production code MUST follow strict type safety and error handling rules:

#### No Panic in Production
- **FORBIDDEN** in handlers, reconcilers, or any production path
- Use explicit error returns instead
- Panic only in `main()` for fatal startup errors
- Recover from panics in goroutines if absolutely necessary

#### Explicit Error Handling (Go)
- Return errors with context: `fmt.Errorf("context: %w", err)`
- Wrap errors to preserve stack and context
- Log errors before returning: `log.Error("failed to do X", "error", err, "namespace", ns)`
- Never ignore errors with `_` without explicit justification

#### Type-Safe Unstructured Access (Go)
- Use `unstructured.Nested*` helpers
- ALWAYS check `found` boolean before using values
- Cast to correct type before use
- Handle missing fields gracefully

```go
// GOOD
if val, found, err := unstructured.NestedString(obj.Object, "spec", "field"); err == nil && found {
    // use val safely
} else {
    return fmt.Errorf("field not found or invalid: %w", err)
}

// BAD
val := obj.Object["spec"].(map[string]interface{})["field"].(string)  // NEVER DO THIS
```

#### Frontend Type Safety (TypeScript)
- Zero `any` types without explicit `eslint-disable` justification
- Define interfaces for all API responses
- Use generics for type-safe data fetching
- Strict null checks enabled in tsconfig

#### Structured Error Context
- Log errors before returning
- Include relevant context: namespace, resource name, operation
- Use structured logging with key-value pairs
- Never expose internal details to users (sanitize error messages)

#### Graceful Degradation
- `IsNotFound` during cleanup is not an error
- Handle missing optional fields gracefully
- Provide sensible defaults
- Document degraded behavior

**Rationale**:
- Runtime panics crash operator reconciliation loops and kill services
- Type assertions without checks cause nil pointer dereferences
- Explicit error handling ensures debuggability and operational stability
- Type safety catches bugs at compile time instead of production

**Violation Consequences**:
- Service crashes and unavailability
- Silent data corruption
- Impossible to debug production issues
- Cascade failures across components

---

### IV. Test-Driven Development

**Status**: MANDATORY
**Applies To**: All new functionality and bug fixes

TDD is MANDATORY for all new functionality following the Red-Green-Refactor cycle:

#### Test-First Development
1. **Red**: Write failing test demonstrating desired behavior
2. **Green**: Implement minimal code to pass test
3. **Refactor**: Improve code quality while keeping tests green

#### Required Test Categories

**Contract Tests**
- Every API endpoint MUST have contract tests
- Every library interface MUST have contract tests
- Test request/response schemas
- Validate error responses and status codes

**Integration Tests**
- Multi-component interactions MUST have integration tests
- Database operations
- Kubernetes client operations
- External API integrations

**Unit Tests**
- Business logic MUST have unit tests
- Pure functions and calculations
- State transformations
- Edge cases and error conditions

**Permission Tests**
- RBAC boundary validation
- Multi-tenant isolation verification
- Authorization checks for all endpoints
- Token validation and expiration

**E2E Tests**
- Critical user journeys MUST have end-to-end tests
- Project creation and deletion
- Session execution lifecycle
- Integration with Git providers

#### Coverage Standards

- **Maintain high test coverage** across all categories
- Critical paths MUST have comprehensive coverage (>90%)
- CI/CD pipeline MUST enforce test passing before merge
- Coverage reports generated automatically in CI
- New code MUST NOT decrease overall coverage

#### Test Quality Standards

- Tests MUST be deterministic (no flaky tests)
- Tests MUST run quickly (unit tests <1s, integration <10s)
- Tests MUST be independent (no shared state)
- Tests MUST clean up resources
- Tests MUST use meaningful assertions (not just checking for errors)

**Rationale**: Tests written after implementation miss edge cases and don't drive design. TDD ensures testability, catches regressions early, documents expected behavior, and enables confident refactoring.

**Violation Consequences**:
- Bugs discovered in production instead of development
- Fear of refactoring leads to code rot
- Regression bugs on every change
- Impossible to safely modify complex code

---

### V. Component Modularity

**Status**: MANDATORY
**Applies To**: All components and codebases

Code MUST be organized into clear, single-responsibility modules:

#### Backend & Operator (Go)

**Handlers**
- HTTP/watch logic ONLY
- Parse requests, validate inputs
- Call service layer for business logic
- Format responses
- NO business logic in handlers

**Types**
- Pure data structures
- No methods containing business logic
- Validation methods acceptable
- JSON/YAML tags for serialization

**Services**
- Reusable business logic
- No direct HTTP handling
- No direct Kubernetes client usage (accept as dependency)
- Return domain errors, not HTTP errors

**Clients**
- Kubernetes client wrappers
- External API clients
- Connection pooling and retries
- Error translation to domain errors

**No Cyclic Dependencies**
- Package imports MUST form a Directed Acyclic Graph (DAG)
- Use dependency injection to break cycles
- Introduce interfaces to decouple packages

#### Frontend (NextJS/React)

**File Colocation**
- Single-use components colocated with pages
- Reusable components in `/components`
- Types in same file or adjacent `.types.ts`
- Hooks in `/hooks` or colocated

**Route Structure**
- All routes MUST have `page.tsx`
- All routes MUST have `loading.tsx` for Suspense
- All routes MUST have `error.tsx` for error boundaries
- Use `layout.tsx` for shared layouts

**Component Size Limits**
- Components over 200 lines MUST be broken down
- Extract hooks for complex logic
- Extract subcomponents for UI sections
- One primary component per file

#### File Organization Patterns

```
components/
  backend/
    handlers/       # HTTP handlers
    services/       # Business logic
    types/          # Domain types
    clients/        # External clients
    utils/          # Utilities

  frontend/
    app/            # Next.js App Router
      (routes)/
        page.tsx
        loading.tsx
        error.tsx
    components/     # Reusable components
      ui/           # UI primitives (Shadcn)
    services/       # API clients
      api/          # API functions
      queries/      # React Query hooks
    types/          # TypeScript types
    lib/            # Utilities
```

**Rationale**: Modular architecture enables parallel development, simplifies testing, reduces cognitive load, and prevents tight coupling. Cyclic dependencies create maintenance nightmares and make testing impossible.

**Violation Consequences**:
- Impossible to test in isolation
- Changes ripple unpredictably
- Cannot parallelize development
- Circular dependencies cause import errors
- Large files are overwhelming and error-prone

---

### VI. Observability & Monitoring

**Status**: MANDATORY
**Applies To**: All services and operators

All components MUST support operational visibility from day one:

#### Structured Logging

- Use structured logs with key-value pairs
- Include context: namespace, resource name, operation, user
- Use appropriate log levels (DEBUG, INFO, WARN, ERROR)
- Never log sensitive data (tokens, passwords, API keys)

```go
// Go structured logging example
log.Info("created session",
    "namespace", namespace,
    "session", sessionName,
    "user", userID)
```

```typescript
// TypeScript structured logging example
logger.info('Session created', {
    namespace,
    sessionName,
    userId
});
```

#### Health Endpoints

- `/health` endpoints for ALL services
- Liveness probe: service is running
- Readiness probe: service can handle traffic
- Include dependency checks (DB, K8s API)

#### Metrics Endpoints (REQUIRED)

- `/metrics` endpoints REQUIRED for all services
- Prometheus format on dedicated management port
- Standard labels: service, namespace, version
- Expose on separate port from application (e.g., :9090)

**Key Metrics to Expose**:
- **Latency**: p50, p95, p99 percentiles
- **Error Rates**: by endpoint, operation, error type
- **Throughput**: requests per second, sessions per minute
- **Component-Specific Metrics**:
  - Session execution time (critical for vTeam)
  - Queue depth and wait time
  - Resource utilization (CPU, memory)
  - Job success/failure rates

**Metrics Standards**:
```
# Counter: Total requests
http_requests_total{service="backend",method="POST",endpoint="/api/sessions",status="200"} 1234

# Histogram: Request duration
http_request_duration_seconds_bucket{service="backend",endpoint="/api/sessions",le="0.1"} 100
http_request_duration_seconds_bucket{service="backend",endpoint="/api/sessions",le="0.5"} 450

# Gauge: Current active sessions
active_sessions{namespace="project-foo"} 5
```

#### Status Updates (Kubernetes)

- Use `UpdateStatus` subresource for CR status changes
- Never update spec and status in same operation
- Include phase, conditions, observedGeneration
- Emit events for state transitions

#### Event Emission

- Emit Kubernetes events for operator actions
- Use appropriate event types (Normal, Warning)
- Include helpful messages for users
- Reference related resources

#### Error Context

- Errors MUST include actionable context for debugging
- Include what operation failed
- Include relevant resource identifiers
- Suggest potential remediation steps
- Never expose internal implementation details to users

**Rationale**: Production systems fail. Without observability, debugging is impossible and Mean Time To Recovery (MTTR) explodes. Metrics enable proactive monitoring, capacity planning, and SLA tracking.

**Violation Consequences**:
- Cannot debug production issues
- No visibility into performance degradation
- Cannot detect partial outages
- No data for capacity planning
- SLA violations without warning

---

### VII. Resource Lifecycle Management

**Status**: MANDATORY
**Applies To**: All Kubernetes resource creation and deletion

Kubernetes resources MUST have proper lifecycle management to prevent resource leaks:

#### OwnerReferences (ALWAYS Required)

- ALWAYS set on child resources (Jobs, Secrets, PVCs, Services, ConfigMaps)
- Use `Controller: true` for primary owner
- Enables automatic cascading deletion
- Prevents orphaned resources

```go
// Example: Set owner reference
ownerRef := metav1.OwnerReference{
    APIVersion:         session.APIVersion,
    Kind:               session.Kind,
    Name:               session.Name,
    UID:                session.UID,
    Controller:         ptr.To(true),
    BlockOwnerDeletion: ptr.To(false), // Important for multi-tenant
}
job.OwnerReferences = []metav1.OwnerReference{ownerRef}
```

#### BlockOwnerDeletion (Do NOT Use)

- **Never set `BlockOwnerDeletion: true`**
- Causes permission issues in multi-tenant environments
- Prevents cascading deletion when expected
- Users without delete permissions on children cannot delete parent

#### Idempotency

- Resource creation MUST check existence first
- Handle `AlreadyExists` errors gracefully
- Compare existing resource with desired state
- Update if necessary, no-op if already correct

```go
// Example: Idempotent create
secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
if apierrors.IsNotFound(err) {
    // Create new secret
    secret, err = clientset.CoreV1().Secrets(namespace).Create(ctx, newSecret, metav1.CreateOptions{})
} else if err != nil {
    return fmt.Errorf("failed to get secret: %w", err)
}
// Secret exists, optionally update if needed
```

#### Cleanup on Deletion

- Rely on OwnerReferences for automatic cascading deletes
- Use finalizers ONLY when external cleanup required
- Remove finalizers after cleanup completes
- Handle finalizer failures gracefully

#### Goroutine Safety

- Exit monitoring goroutines when parent resource deleted
- Use context cancellation for graceful shutdown
- Prevent goroutine leaks on resource deletion

```go
// Example: Context-aware goroutine
go func(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            log.Info("stopping monitor", "reason", ctx.Err())
            return
        case <-ticker.C:
            // Do monitoring work
        }
    }
}(ctx)
```

**Rationale**: Resource leaks waste cluster capacity, cause quota exhaustion, and eventually lead to outages. Proper lifecycle management ensures automatic cleanup and prevents orphaned resources that accumulate over time.

**Violation Consequences**:
- Orphaned resources accumulate indefinitely
- Quota exhaustion prevents new resources
- Manual cleanup required (toil)
- Production outages from resource exhaustion
- Storage leaks from abandoned PVCs

---

### VIII. Context Engineering & Prompt Optimization

**Status**: MANDATORY
**Applies To**: AI agent prompts, context management, session design

The Ambient Code Platform is a context engineering hub - AI output quality depends on input quality:

#### Context Budgets

- Respect token limits (200K for Claude Sonnet 4.5)
- Track context usage during session
- Warn when approaching limits
- Implement context pruning strategies

#### Context Prioritization

Order of importance:
1. **System context**: Core instructions, principles, API contracts
2. **Conversation history**: Recent turns most important
3. **Examples**: Relevant examples and patterns
4. **Background**: Nice-to-have context

#### Prompt Templates

- Use standardized templates for common operations
- Templates for: RFE analysis, code review, refactoring, testing
- Version templates for reproducibility
- Document template variables and expected outputs

#### Context Compression

- Summarize long-running sessions to preserve history within budget
- Keep critical details, compress verbose explanations
- Maintain conversation flow and context
- Store full history externally if needed

#### Agent Personas

- Maintain consistency through well-defined agent roles
- Document each agent's responsibilities and capabilities
- Use consistent terminology across agents
- Avoid persona drift during long sessions

#### Pre-Deployment Optimization

- ALL prompts MUST be optimized for clarity and token efficiency before deployment
- Remove redundant instructions
- Consolidate repeated patterns
- Use clear, concise language
- Test prompts before production deployment

#### Incremental Loading

- Build context incrementally
- Avoid reloading static content repeatedly
- Cache stable context (API docs, principles)
- Load only relevant sections on-demand

**Rationale**: Poor context management causes hallucinations, inconsistent outputs, wasted API costs, and unreliable results. Context engineering is a first-class engineering discipline for AI platforms - treat it with the same rigor as code.

**Violation Consequences**:
- AI hallucinations and incorrect outputs
- Token limit exceeded (truncated context)
- Inconsistent behavior across sessions
- Excessive API costs
- Poor user experience

---

### IX. Data Access & Knowledge Augmentation

**Status**: MANDATORY
**Applies To**: AI agents, data retrieval systems, learning mechanisms

Enable agents to access external knowledge and learn from interactions:

#### Retrieval-Augmented Generation (RAG)

**Embedding & Indexing**:
- Embed and index repository contents for semantic search
- Chunk semantically (512-1024 tokens per chunk)
- Use consistent embedding models across platform
- Update indexes on repository changes

**Retrieval**:
- Apply reranking to improve relevance
- Return top-k results with confidence scores
- Provide context around matched chunks
- Handle no-results gracefully

**Chunking Strategy**:
- Respect code structure (functions, classes, modules)
- Include surrounding context (imports, comments)
- Overlap chunks for continuity (100-200 tokens)
- Preserve semantic meaning

#### Model Context Protocol (MCP)

**MCP Server Support**:
- Support MCP servers for structured data access
- Enable tools, resources, and prompts via MCP
- Document available MCP servers

**Isolation & Security**:
- Enforce namespace isolation for MCP access
- Validate MCP server responses
- Rate limit MCP calls
- Audit MCP operations

**Failure Handling**:
- Handle MCP server failures gracefully
- Provide fallback behavior when MCP unavailable
- Log MCP errors for debugging
- Don't block core functionality on MCP

#### Reinforcement Learning from Human Feedback (RLHF)

**User Feedback Capture**:
- Capture user ratings (thumbs up/down, 1-5 stars)
- Store with session metadata
- Include context: task type, model, parameters

**Pattern Analysis**:
- Refine prompts from user feedback patterns
- Identify successful vs unsuccessful patterns
- Analyze feedback by task category

**A/B Testing**:
- Support A/B testing of prompt variations
- Track performance metrics by variant
- Statistical significance testing
- Gradual rollout of winning variants

**Privacy**:
- Anonymize feedback data
- Allow opt-out from feedback collection
- Don't include sensitive user data

**Rationale**: Static prompts have limited effectiveness. Platforms must continuously improve through knowledge retrieval and learning from user feedback. RAG provides grounding in facts, MCP enables structured integrations, RLHF enables continuous improvement.

**Violation Consequences**:
- AI responses lack grounding in project reality
- Cannot access external knowledge sources
- No improvement over time
- Users repeat themselves (no memory)
- Platform capabilities stagnate

---

### X. Commit Discipline & Code Review

**Status**: MANDATORY
**Applies To**: All code commits and pull requests

Each commit MUST be atomic, reviewable, and independently testable:

#### Line Count Thresholds

**What counts toward limits**:
- ✅ Source code (`*.go`, `*.ts`, `*.tsx`, `*.py`)
- ✅ Configuration specific to feature (YAML, JSON)
- ✅ Test code
- ❌ Generated code (CRDs, OpenAPI, mocks)
- ❌ Lock files (`go.sum`, `package-lock.json`)
- ❌ Vendored dependencies
- ❌ Binary files

**Thresholds** (excluding generated code, test fixtures, vendor/deps):

**Bug Fix**: ≤150 lines
- Single issue resolution
- Includes test demonstrating the bug
- Includes fix verification
- Minimal scope to reduce risk

**Feature (Small)**: ≤300 lines
- Single user-facing capability
- Includes unit + contract tests
- Updates relevant documentation
- Focused on one feature

**Feature (Medium)**: ≤500 lines
- Multi-component feature
- Requires design justification in commit message
- MUST be reviewable in 30 minutes
- Consider splitting if possible

**Refactoring**: ≤400 lines
- Behavior-preserving changes ONLY
- MUST NOT mix with feature/bug changes
- Existing tests MUST pass unchanged
- Explain refactoring motivation

**Documentation**: ≤200 lines
- Pure documentation changes
- Can be larger for initial docs
- Update multiple docs together
- Keep focused on one topic

**Test Addition**: ≤250 lines
- Adding missing test coverage
- MUST NOT include implementation changes
- Explain what's being tested and why
- Separate PR from feature implementation

#### Mandatory Exceptions

Require justification in PR description:

**Code Generation**: Generated CRD YAML, OpenAPI schemas, protobuf
- Include generation command in PR
- Review generated output for correctness
- Commit generated files separately from manual changes

**Data Migration**: Database migrations, fixture updates
- Test migrations forward and backward
- Include rollback procedure
- Document migration impact

**Dependency Updates**: `go.mod`, `package.json`, `requirements.txt`
- Review changelog for breaking changes
- Update code for breaking changes
- Test thoroughly after upgrades

**Configuration**: Kubernetes manifests for new components (≤800 lines)
- Review resource limits and quotas
- Validate RBAC configurations
- Test deployment in dev environment

#### Commit Requirements

**Atomic Commits**:
- Single logical change that can be independently reverted
- Each commit passes all tests and linters
- No "fix previous commit" or "WIP" commits
- Squash before PR submission

**Conventional Format**: `type(scope): description`

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring (no behavior change)
- `test`: Adding or fixing tests
- `docs`: Documentation changes
- `chore`: Maintenance (dependencies, config)
- `perf`: Performance improvement
- `ci`: CI/CD configuration

**Scopes**: Component names
- `backend`: Backend API
- `frontend`: Next.js frontend
- `operator`: Kubernetes operator
- `runner`: Claude Code runner

**Examples**:
```
feat(backend): add GitLab repository support
fix(frontend): correct session status polling interval
refactor(operator): extract reconciliation logic to services
test(backend): add integration tests for permissions API
docs(readme): update GitLab setup instructions
```

**Message Content**:
- Explain WHY, not WHAT (code shows what)
- First line: concise summary (<72 chars)
- Body: detailed explanation (wrap at 72 chars)
- Reference issues: "Fixes #123" or "Relates to #456"

#### Review Standards

**PR Size Limits**:
- PR over 600 lines MUST be broken into multiple PRs
- Each PR should have clear, independent value
- Incremental delivery preferred over "big bang" merges

**Review Process**:
- Each commit reviewed independently
- Enable per-commit review in GitHub
- Large PRs require design doc or RFC first
- Two approvals for complex/risky changes

**Review Checklist**:
- [ ] Follows constitutional principles
- [ ] Tests included and passing
- [ ] Documentation updated
- [ ] No security vulnerabilities
- [ ] Performance impact acceptable
- [ ] Backward compatible (or breaking change justified)

**Rationale**: Large commits hide bugs, slow reviews, complicate bisecting, and create merge conflicts. Specific thresholds provide objective guidance while exceptions handle legitimate cases. Small, focused commits enable faster feedback, easier debugging (git bisect), and safer reverts.

**Violation Consequences**:
- Slow or blocked code reviews
- Bugs hidden in large diffs
- Cannot bisect to find regression
- Difficult or impossible to revert safely
- Merge conflicts with parallel work

---

## Development Standards

### Go Code (Backend & Operator)

#### Formatting

- Run `gofmt -w .` before committing
- Use `golangci-lint run` for comprehensive linting
- Run `go vet ./...` to detect suspicious constructs
- Use `goimports` for import organization

#### Error Handling

See [Principle III: Type Safety & Error Handling](#iii-type-safety--error-handling)

Additional Go-specific guidance:
- Wrap errors with context using `fmt.Errorf("operation: %w", err)`
- Check errors immediately after calls
- Don't use `panic()` in libraries or handlers
- Use `errors.Is()` and `errors.As()` for error inspection

#### Kubernetes Client Patterns

**User Operations**:
- Use `GetK8sClientsForRequest(c)` for user-scoped operations
- Always validate user has permission before action
- Use impersonation for multi-tenant isolation

**Service Account**:
- ONLY for CR writes and token minting
- Never as fallback for user authentication
- Minimal RBAC permissions

**Status Updates**:
- Use `UpdateStatus` subresource
- Never update spec and status together
- Include `observedGeneration` to track reconciliation

**Watch Loops**:
- Reconnect on channel close with exponential backoff
- Use informers for efficient watching
- Handle watch errors gracefully

#### Project Structure

```
components/backend/
  cmd/                 # Main entry points
  handlers/            # HTTP handlers
  services/            # Business logic
  clients/             # K8s and external clients
  types/               # Domain types
  middleware/          # HTTP middleware
  utils/               # Utilities
  config/              # Configuration
```

---

### Frontend Code (Next.js / TypeScript)

#### UI Components

**Shadcn UI Usage**:
- Use components from `@/components/ui/*`
- Don't modify Shadcn components directly
- Extend via composition, not modification

**Type Definitions**:
- Use `type` instead of `interface` for object shapes
- Use `interface` only for extensible contracts
- Export types alongside components

**Loading States**:
- All buttons MUST show loading state during async operations
- Use `disabled={isLoading}` with spinner
- Prevent double-submission

**Empty States**:
- All lists MUST have empty states
- Provide helpful guidance on next steps
- Use illustrations or icons

#### Data Operations

**React Query Hooks**:
- Use hooks from `@/services/queries/*`
- Colocate query hooks with API functions
- Use query keys consistently

**Mutations**:
- All mutations MUST invalidate relevant queries
- Show loading state during mutation
- Handle errors with toast notifications
- Optimistic updates for better UX

**No Direct fetch() in Components**:
- Use API functions from `@/services/api/*`
- Centralize error handling
- Type-safe API responses

#### File Organization

**Colocation**:
- Single-use components colocated with pages
- Reusable components in `/components`
- Types in `.types.ts` files
- Hooks in `/hooks` or colocated

**Route Structure**:
All routes MUST have:
- `page.tsx` - Main page component
- `loading.tsx` - Suspense loading state
- `error.tsx` - Error boundary
- `layout.tsx` - Shared layout (optional)

**Component Size**:
- Components over 200 lines MUST be broken down
- Extract hooks for complex logic
- Extract subcomponents for UI sections
- One primary component per file

#### Project Structure

```
components/frontend/
  src/
    app/              # Next.js App Router
      (routes)/       # Route groups
        page.tsx
        loading.tsx
        error.tsx
      api/            # API routes
    components/       # Reusable components
      ui/             # Shadcn UI components
      workspace-sections/  # Feature components
    services/         # API & queries
      api/            # API client functions
      queries/        # React Query hooks
    types/            # TypeScript types
      api/            # API response types
    lib/              # Utilities
      utils.ts        # Helper functions
      env.ts          # Environment variables
```

---

### Python Code (Runner)

#### Environment

**Virtual Environments**:
- ALWAYS use virtual environments
- `python -m venv venv` or `uv venv`
- Never install packages globally

**Package Management**:
- Prefer `uv` over `pip` for faster installs
- Pin versions in `requirements.txt`
- Use `requirements-dev.txt` for dev dependencies

#### Formatting

**Black**:
- Use `black` with 88 character line length
- Run before committing: `black .`

**isort**:
- Use `isort` with black profile
- Run before committing: `isort .`

**flake8/ruff**:
- Run linters before committing
- Fix all linting errors

#### Code Quality

**Type Hints**:
- Use type hints for function signatures
- Use `mypy` for type checking
- Gradually add types to existing code

**Error Handling**:
- Use specific exception types
- Avoid bare `except:`
- Log errors before re-raising

**Logging**:
- Use standard `logging` module
- Structured logging with context
- Never use `print()` in production code

---

### Naming & Legacy Migration

#### vTeam → ACP Transition

Replace usage of "vTeam" with "ACP" (Ambient Code Platform) where safe and unobtrusive:

**Safe to Update** (non-breaking):
- User-facing documentation and README files
- Code comments and inline documentation
- Log messages and error messages
- UI text and labels
- Variable names in new code
- New function and class names

**DO NOT Update** (breaking changes - maintain for backward compatibility):
- Kubernetes API group: `vteam.ambient-code`
- Custom Resource Definitions (CRD kinds)
- Container image names: `vteam_frontend`, `vteam_backend`, etc.
- Kubernetes resource names: deployments, services, routes
- Environment variables referenced in deployment configs
- File paths in scripts referencing namespaces/resources
- Git repository name and URLs

#### Incremental Approach

1. **Documentation First**: Update README, CLAUDE.md, `/docs`
2. **UI Text**: Update new features with ACP terminology
3. **New Code**: Use ACP naming in new modules
4. **No Mass Renames**: Update organically during feature work
5. **Document Legacy**: Maintain "Legacy vTeam References" section

**Rationale**: Project rebranded from vTeam to Ambient Code Platform, but technical artifacts retain "vteam" for backward compatibility. Gradual, safe migration improves clarity while avoiding breaking changes for existing deployments.

---

## Deployment & Operations

### Pre-Deployment Validation

All code MUST pass validation checks before deployment:

#### Go Components (Backend & Operator)

```bash
# Formatting check
gofmt -l .

# Vet for suspicious constructs
go vet ./...

# Comprehensive linting
golangci-lint run

# Run all tests
make test
```

**CI Requirements**:
- All checks must pass
- Zero linting errors
- All tests pass
- Code coverage maintained or improved

#### Frontend

```bash
# ESLint checks
npm run lint

# TypeScript type checking
npm run type-check

# Build for production (must pass)
npm run build  # Must complete with 0 errors, 0 warnings
```

**CI Requirements**:
- No ESLint errors or warnings
- No TypeScript errors
- Production build succeeds
- No console logs in production code

#### Python (Runner)

```bash
# Format checking
black --check .
isort --check .

# Linting
ruff check .

# Type checking
mypy .

# Tests
pytest
```

---

### Container Security

All container deployments MUST follow security best practices:

#### SecurityContext (Mandatory)

```yaml
securityContext:
  allowPrivilegeEscalation: false
  runAsNonRoot: true
  runAsUser: 1000
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: true  # where possible
```

#### Image Security

- **Base Images**: Use minimal base images (distroless, alpine)
- **Scanning**: Scan images for vulnerabilities before deployment
- **Signing**: Sign production images
- **Registries**: Use private registries for sensitive images

#### Resource Limits

All pods MUST have resource limits:

```yaml
resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

---

### Production Requirements

#### Security

Apply [Principle II: Security & Multi-Tenancy First](#ii-security--multi-tenancy-first) requirements.

**Additional Production Requirements**:
- **Image Scanning**: Scan container images for vulnerabilities before deployment
- **Secret Rotation**: Implement regular rotation of API keys and tokens
- **Audit Logging**: Enable audit logging for all resource modifications
- **Network Policies**: Implement network policies restricting pod-to-pod communication
- **TLS**: Use TLS for all external and internal communication
- **Vulnerability Management**: Process for responding to CVEs

#### Monitoring

Implement [Principle VI: Observability & Monitoring](#vi-observability--monitoring) requirements.

**Additional Production Requirements**:
- **Centralized Logging**: Set up log aggregation (ELK, Loki, CloudWatch)
- **Alerting Infrastructure**: Configure alerts for critical conditions
- **Dashboards**: Create operational dashboards for key metrics
- **On-Call**: Define on-call procedures and runbooks
- **SLOs**: Define Service Level Objectives for critical paths

#### Scaling

Design for scale and multi-tenancy:

**Horizontal Pod Autoscaling**:
- Configure HPA based on CPU/memory usage
- Use custom metrics where appropriate
- Set min/max replicas appropriately

**Resource Planning**:
- Set appropriate resource requests and limits
- Plan for job concurrency and queue management
- Design for multi-tenancy with shared infrastructure

**Database Strategy**:
- **Do NOT use etcd as database** for unbounded objects like CRs
- Use external database (Postgres) for persistent data
- Implement connection pooling
- Regular backups and disaster recovery

**Capacity Planning**:
- Monitor resource usage trends
- Plan for peak loads
- Implement rate limiting and quotas
- Load test before scaling up

---

## Governance

### Amendment Process

Changing this constitution requires following a formal process:

#### 1. Proposal

- **Document**: Write proposal with clear rationale
- **Impact Analysis**: Evaluate impact on:
  - Existing code and patterns
  - Templates (spec, plan, tasks, checklist)
  - Developer workflows
  - Production deployments
- **Alternatives**: Consider alternatives and tradeoffs

#### 2. Review

- **Stakeholder Review**: Share with maintainers and team
- **Feedback Period**: Allow time for feedback and discussion
- **Revisions**: Incorporate feedback into proposal

#### 3. Approval

- **Approval Required**: Project maintainer approval required
- **Consensus**: Strive for consensus on significant changes
- **Documentation**: Document decision rationale

#### 4. Migration

- **Update Templates**: Update all dependent templates
  - `spec-template.md`
  - `plan-template.md`
  - `tasks-template.md`
  - `checklist-template.md`
- **Update Documentation**: Update CLAUDE.md and other docs
- **Announce**: Communicate changes to team
- **Migration Path**: Provide migration guide for existing code

#### 5. Versioning

Increment version according to semantic versioning:

---

### Version Policy

Constitution versioning follows semantic versioning (MAJOR.MINOR.PATCH):

**MAJOR** (X.0.0):
- Backward incompatible governance/principle removals
- Redefinition of core principles
- Changes requiring code rewrites
- Breaking changes to development workflow

**MINOR** (x.Y.0):
- New principle/section added
- Material expansion of existing guidance
- New mandatory requirements
- New development standards

**PATCH** (x.y.Z):
- Clarifications without semantic changes
- Wording improvements
- Typo fixes
- Non-semantic refinements
- Examples added

**Current Version**: 2.0.0 (RATIFIED)

---

### Compliance

**Pull Request Requirements**:
- All PRs MUST verify constitution compliance
- Reference relevant constitutional principles
- Justify any deviations (must be exceptional)
- Pre-commit checklists MUST be followed

**Code Review**:
- Reviewers MUST check constitutional compliance
- Violations MUST be addressed before merge
- Patterns violating principles MUST be refactored

**Automated Checks**:
- CI MUST enforce formatting and linting
- CI MUST enforce test requirements
- CI MUST check commit message format
- Commit size validation tooling recommended

**Escalation**:
- Constitution violations can block PRs
- Complexity violations MUST be justified in implementation plans
- Repeated violations escalated to maintainers

**Priority**:
- Constitution supersedes all other practices and guidelines
- When conflicts arise, constitution takes precedence
- Update other documentation to align with constitution

---

### Development Guidance

Runtime development guidance is maintained in:

**Primary Documents**:
- `/CLAUDE.md` - Claude Code development patterns and examples
- `/.specify/memory/constitution.md` - This document (constitutional principles)
- `/README.md` - Project overview and quick start

**Component Documentation**:
- `/components/backend/README.md` - Backend API documentation
- `/components/frontend/README.md` - Frontend development guide
- `/components/operator/README.md` - Operator development guide
- `/components/runners/claude-code-runner/README.md` - Runner documentation

**Extended Documentation**:
- `/docs/*.md` - MkDocs documentation for deployment, operations, integrations
- `/CONTRIBUTING.md` - Contribution guidelines
- `/e2e/README.md` - End-to-end testing guide

**Template Files**:
- `/.specify/templates/spec-template.md` - Feature specification template
- `/.specify/templates/plan-template.md` - Implementation plan template
- `/.specify/templates/tasks-template.md` - Task breakdown template
- `/.specify/templates/checklist-template.md` - Quality checklist template

---

## Amendment History

### Version 2.0.0 (2025-01-22) - RATIFIED

**Status**: RATIFIED - Spec-Kit Alignment Release

**Major Changes**:
- Reformatted constitution to follow spec-kit conventions
- Enhanced structure with clear sections and navigation
- Added comprehensive rationale for each principle
- Added violation consequences for each principle
- Expanded development standards with code examples
- Improved formatting and readability
- Added table of contents
- Enhanced governance section with detailed processes
- Better alignment with spec-kit methodology

**Rationale**: Align constitution with spec-kit conventions to improve usability for both AI agents and human developers. Clearer structure and rationale improve decision-making and reduce ambiguity.

---

### Version 1.0.0 (2025-11-13) - RATIFIED

**Status**: RATIFIED - Official Ratification

**Changes**:
- Constitution officially ratified and adopted
- All 10 core principles now in force
- Development standards and governance policies active

---

### Version 0.2.0 (2025-11-XX) - DRAFT

**Status**: DRAFT

**Changes**:
- Added Development Standards: Naming & Legacy Migration subsection
- Safe vs. breaking change guidance for vTeam → ACP transition
- Incremental migration approach (documentation first, then UI, then code)
- DO NOT update list: API groups, CRDs, container names, K8s resources
- Safe to update: docs, comments, logs, UI text, new variable names

**Rationale**: Gradual migration improves clarity while preserving backward compatibility

---

### Version 0.1.0 (2025-11-XX) - DRAFT

**Status**: DRAFT

**Changes**:
- Added Principle X: Commit Discipline & Code Review
  - Line count thresholds by change type
  - Mandatory exceptions for generated code, migrations, dependencies
  - Conventional commit format requirements
  - PR size limits with justification requirements
  - Measurement guidelines

**Rationale**: Small, focused commits enable faster feedback, easier debugging, and safer reverts

---

### Version 0.0.1 (2025-11-XX) - DRAFT

**Status**: DRAFT

**Changes**:
- Added Principle VIII: Context Engineering & Prompt Optimization
- Added Principle IX: Data Access & Knowledge Augmentation
- Enhanced Principle IV: E2E testing, coverage standards, CI/CD automation
- Enhanced Principle VI: /metrics endpoint REQUIRED, simplified key metrics
- Simplified Principle IX: Consolidated RAG/MCP/RLHF into concise bullets
- Removed redundant sections
- Consolidated Development Standards
- Reduced total length by ~30 lines while maintaining clarity

**Template Updates**:
- ✅ plan-template.md - References constitution check dynamically
- ✅ tasks-template.md - Added Phase 3.9 for commit planning/validation (T036-T040)
- ✅ spec-template.md - No updates needed

**Follow-up TODOs**:
- Implement /metrics endpoints in all components
- Create prompt template library
- Design RAG pipeline architecture
- Add commit size validation tooling (pre-commit hook or CI check)
- Update PR template to include commit discipline checklist
- Continue vTeam → ACP migration incrementally (docs → UI → code)

---

**Version**: 2.0.0
**Ratified**: 2025-01-22
**Last Amended**: 2025-01-22
**Next Review**: 2025-04-22 (quarterly review recommended)
