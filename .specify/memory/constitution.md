# Ambient Code Platform Constitution

The Ambient Code Platform is a Kubernetes-native AI automation platform combining multiple AI providers (Claude, Gemini, and future providers) with multi-agent collaboration capabilities. It enables intelligent agentic sessions through pluggable client interfaces including REST API, SDK, MCP, and web UI. The platform is MIT licensed and built on open source, standards-based technologies.

This constitution establishes non-negotiable principles for all components: Backend, Frontend, Operator, Runner, Public API, Ambient API Server, Ambient SDK (Go, Python, TypeScript), Ambient CLI (acpctl), Open WebUI, MCP Server (coming), and Mobile App (coming). It supersedes all other development guidelines.

---

## Core Principles

### I. Kubernetes-Native Architecture

All features MUST use Kubernetes primitives and patterns:

- **CRDs** for domain objects: `AgenticSession`, `ProjectSettings`, `RFEWorkflow`
- **Operators** for reconciliation loops and lifecycle management
- **Jobs and JobSets** for execution workloads with proper resource limits, timeouts, and failure policies
- **ConfigMaps and Secrets** for configuration (project-scoped isolation)
- **Services and Routes** for network exposure (ClusterIP internal, Ingress external)
- **RBAC** for authorization (namespace-scoped roles, minimal permissions)
- **Pluggable Interfaces**: REST API, SDK, MCP, Web UI (extensible, provider-agnostic)

---

### II. Security & Multi-Tenancy First

Security and isolation MUST be embedded from initial design:

- **Authentication**: All endpoints use user tokens via `GetK8sClientsForRequest()`. No unauthenticated endpoints except health/metrics.
- **Authorization**: RBAC checks before resource access. Deny by default.
- **Token Security**: NEVER log tokens/API keys. Store in Kubernetes Secrets.
- **Multi-Tenancy**: Project-scoped namespaces, network policies, resource quotas, separate service accounts.
- **Least Privilege**: Minimal service account permissions. No cluster-admin except installation.
- **Container Security**: `AllowPrivilegeEscalation: false`, drop all capabilities, run as non-root (UID > 1000).
- **Backend Service Account**: ONLY for CR writes and token minting. NEVER for user resource access.

---

### III. Type Safety & Error Handling

Production code MUST follow strict type safety:

- **No Panic**: FORBIDDEN in handlers/reconcilers. Use explicit error returns.
- **Error Context**: `fmt.Errorf("context: %w", err)`. Log before returning.
- **Type-Safe Access**: Use `unstructured.Nested*` helpers. ALWAYS check `found` boolean.
- **Frontend**: Zero `any` types. Define interfaces for all API responses. Strict null checks.
- **Graceful Degradation**: `IsNotFound` during cleanup is not an error. Handle missing fields gracefully.

---

### IV. Test-Driven Development

TDD is MANDATORY (Red-Green-Refactor):

- **Contract Tests**: Every API endpoint and library interface
- **Integration Tests**: Multi-component interactions, K8s operations, external APIs
- **Unit Tests**: Business logic, pure functions, edge cases
- **Permission Tests**: RBAC validation, multi-tenant isolation
- **E2E Tests**: Critical user journeys (project lifecycle, session execution)
- **Quality**: Deterministic (no flaky tests), fast (unit <1s, integration <10s), independent, cleanup resources

---

### V. Component Modularity

Code MUST be organized into clear, single-responsibility modules:

**Go (Backend/Operator)**:
- `handlers/` - HTTP/watch logic only. NO business logic.
- `services/` - Reusable business logic. No HTTP handling.
- `types/` - Pure data structures.
- `clients/` - K8s and external API wrappers.
- No cyclic dependencies (DAG imports).

**Frontend (Next.js)**:
- File colocation: Single-use with pages, reusable in `/components`
- Routes: `page.tsx`, `loading.tsx`, `error.tsx` required
- Component limit: 200 lines max, extract hooks/subcomponents

---

### VI. Observability & Monitoring

All components MUST support operational visibility:

- **Structured Logging**: Key-value pairs, context (namespace, resource, user), appropriate levels. Never log sensitive data.
- **Health Endpoints**: `/health` for liveness/readiness probes
- **Metrics Endpoints**: `/metrics` REQUIRED (Prometheus format). Latency percentiles, error rates, throughput.
- **Status Updates**: Use `UpdateStatus` subresource. Include phase, conditions, observedGeneration.
- **Events**: Emit Kubernetes events for state transitions.

---

### VII. Resource Lifecycle Management

Kubernetes resources MUST have proper lifecycle management:

- **OwnerReferences**: ALWAYS set on child resources. `Controller: true` for primary owner.
- **BlockOwnerDeletion**: NEVER use `true` (causes multi-tenant permission issues).
- **Idempotency**: Check existence first. Handle `AlreadyExists` gracefully.
- **Cleanup**: Rely on OwnerReferences for cascading deletes. Finalizers only for external cleanup.
- **Goroutine Safety**: Exit monitoring goroutines on resource deletion. Use context cancellation.

---

### VIII. Context Engineering & Prompt Optimization

AI output quality depends on input quality:

- **Context Budgets**: Respect provider-specific token limits. Design for portable context.
- **Prioritization**: System context → Conversation history → Examples → Background
- **Prompt Templates**: Standardized, versioned templates for common operations
- **Compression**: Summarize long sessions. Keep critical details.
- **Agent Personas**: Consistent roles and terminology
- **Pre-Deployment**: ALL prompts optimized for clarity and token efficiency before deployment

---

### IX. Data Access & Knowledge Augmentation

Enable agents to access external knowledge:

- **RAG**: Embed/index repositories. Semantic chunking (512-1024 tokens). Reranking for relevance.
- **MCP**: Support MCP servers with namespace isolation, rate limiting, graceful failure handling.
- **RLHF**: Capture user feedback. Refine prompts from patterns. Support A/B testing. Anonymize data.

---

### X. Commit Discipline & Code Review

Each commit MUST be atomic and reviewable:

**Line Thresholds** (excluding generated/vendor):
- Bug Fix: ≤150 lines
- Feature (Small): ≤300 lines
- Feature (Medium): ≤500 lines (requires design justification)
- Refactoring: ≤400 lines (behavior-preserving ONLY)
- Test Addition: ≤250 lines

**Format**: `type(scope): description` (feat, fix, refactor, test, docs, chore, perf, ci)

**PR Size**: ≤600 lines. Larger PRs MUST be broken up.

---

### XI. System Dynamics & Control Theory

Design systems around system dynamics principles:

- **Reinforcing Loops (Flywheels)**: Identify, accelerate, and create loops that compound capability
- **Balancing Loops (Governors)**: Identify and remove loops that limit performance
- **Loop States**: Absent (decide to create), Present (measure/tune), Throttled (find/remove constraint)
- **Design Questions**: Where are reinforcing loops? Where are balancing loops? What's the cheapest intervention?
- **Delay Factor**: Accept strategic delays for long-term compounding. Document expected payoff timelines.

---

### XII. Dependency Management

Dependencies MUST be managed securely and automatically:

- **Automated Security**: Dependabot/Renovate enabled. Automatic security updates. SBOM generation.
- **Version Pinning**: Exact versions in lock files. Review changelogs before major bumps.
- **Supply Chain**: Verify checksums. Use trusted registries. Sign release artifacts.
- **GitHub Actions**: NEVER use `pull_request_target`. Pin actions to SHA. Minimize secrets exposure.
- **Zero-Friction**: Security tooling MUST NOT slow development. Automated fixes preferred.

---

## Development Standards

### Go (Backend & Operator)

- **Formatting**: `gofmt`, `golangci-lint`, `go vet`, `goimports`
- **K8s Clients**: `GetK8sClientsForRequest()` for user ops. Service account ONLY for CR writes.
- **Status Updates**: Use `UpdateStatus` subresource. Never update spec and status together.

### Frontend (Next.js / TypeScript)

- **UI**: Shadcn components from `@/components/ui/*`. Extend via composition.
- **Data**: React Query hooks from `@/services/queries/*`. All mutations invalidate queries.
- **Loading/Empty States**: REQUIRED for all async operations and lists.

### Python (Runner)

- **Environment**: ALWAYS use virtual environments. Prefer `uv` over `pip`.
- **Formatting**: `ruff format` and `ruff check` before committing.
- **Quality**: Type hints for signatures. `mypy` for checking. No bare `except:`.

### Naming (vTeam → ACP)

- **Safe to update**: Documentation, comments, logs, UI text, new code
- **DO NOT update**: K8s API groups, CRDs, container names, resource names, environment variables

---

## Deployment & Operations

### Pre-Deployment

All code MUST pass validation:
- **Go**: `gofmt`, `go vet`, `golangci-lint`, `make test`
- **Frontend**: `npm run lint`, `npm run type-check`, `npm run build` (0 errors/warnings)
- **Python**: `ruff format --check`, `ruff check`, `mypy`, `pytest`

### Container Security

```yaml
securityContext:
  allowPrivilegeEscalation: false
  runAsNonRoot: true
  runAsUser: 1000
  capabilities:
    drop: [ALL]
```

All pods MUST have resource requests and limits.

### Scaling & Session Lifecycle

- Design for ~1000 active (Running) pods, ~5000 idle pods
- Per-user baseline: ~5 running, ~5 idle, ~50 stopped sessions
- **Proactive idle stopping**: Auto-transition to `Stopped` after configurable timeout
- **State hydration**: `init-hydrate` container restores from S3 on resume
- **State sync**: `sync.sh` sidecar continuously syncs to S3
- Phases: `Pending` → `Creating` → `Running` → `Stopping` → `Stopped` (or `Completed`/`Failed`)

---

## Governance

### Amendment Process

1. **Proposal**: Document rationale, impact analysis, alternatives
2. **Review**: Stakeholder feedback, revisions
3. **Approval**: Maintainer approval required
4. **Migration**: Update templates, documentation, provide migration guide
5. **Versioning**: MAJOR (breaking), MINOR (new principles), PATCH (clarifications)

### Compliance

- All PRs MUST verify constitution compliance
- Reviewers MUST check compliance before merge
- CI MUST enforce formatting, linting, tests, commit format
- Constitution supersedes all other practices

### Development Guidance

- **Primary**: `/AGENTS.md` (primary), `/CLAUDE.md` (symlink), `/.specify/memory/constitution.md`
- **Components**: `/components/*/README.md`
- **Docs**: `/docs/src/content/docs/` (Astro Starlight)
- **Templates**: `/.specify/templates/*.md`

---

## Amendment History

### Version 3.0.0 (2026-03-04)

- Multi-provider AI support (Claude, Gemini, future providers)
- New Principle XI: System Dynamics & Control Theory
- New Principle XII: Dependency Management
- Pluggable interfaces (REST API, SDK, MCP, Web UI)
- JobSets support, session lifecycle management, state hydration
- Documentation migrated to Astro Starlight
- AGENTS.md as primary (CLAUDE.md as symlink)
- New components: Ambient API Server, SDK, CLI, Open WebUI

### Version 2.0.0 (2025-01-22)

- Spec-kit alignment, enhanced structure, rationale sections

### Version 1.0.0 (2025-11-13)

- Official ratification, 10 core principles in force

---

**Version**: 3.0.0 | **Ratified**: 2026-03-04 | **Last Amended**: 2026-03-04
