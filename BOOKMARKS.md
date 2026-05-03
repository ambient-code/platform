# Bookmarks

Progressive disclosure for task-specific documentation and references.

## Table of Contents

- [Specs](#specs)
- [Workflows](#workflows)
- [Standards](#standards)
- [Governance](#governance)
- [Architecture Decisions](#architecture-decisions)
- [Component Guides](#component-guides)
- [Development Environment](#development-environment)
- [Testing](#testing)
- [Observability](#observability)
- [Design Documents](#design-documents)
- [Dependency Automation](#dependency-automation)
- [Amber Automation](#amber-automation)

---

## Specs

Desired state of the system, organized by capability domain.

| Spec | Domain | Purpose |
|------|--------|---------|
| [Ambient Data Model](specs/sessions/ambient-model.spec.md) | sessions | Platform-wide data model: projects, agents, sessions, credentials, RBAC |
| [Control Plane](specs/control-plane/control-plane.spec.md) | control-plane | CP architecture, runner structure, K8s provisioning |
| [Runner](specs/agents/runner.spec.md) | agents | Runner subprocess lifecycle, bridges, gRPC/HTTP endpoints |
| [MCP Server](specs/integrations/mcp-server.spec.md) | integrations | MCP tool definitions, sidecar and public endpoint modes |

Feature specs remain in numbered directories under `specs/` (e.g., `specs/001-*/spec.md`).

## Workflows

Agent-consumable procedures — step-by-step implementation workflows.

| Workflow | Domain | Purpose |
|----------|--------|---------|
| [Ambient Model](workflows/sessions/ambient-model.workflow.md) | sessions | Spec-driven implementation pipeline for data model changes |
| [Control Plane](workflows/control-plane/control-plane.workflow.md) | control-plane | CP + Runner implementation workflow |
| [MCP Server](workflows/integrations/mcp-server.workflow.md) | integrations | MCP server implementation workflow |

## Standards

Component-specific conventions loaded by review agents on demand.

| Standard | Domain | Scope |
|----------|--------|-------|
| [Backend Conventions](specs/standards/backend/conventions.spec.md) | backend | Go patterns, K8s integration, handler conventions, user-scoped clients |
| [Backend Error Handling](specs/standards/backend/error-handling.spec.md) | backend | Consistent error patterns across backend and operator |
| [Backend K8s Client](specs/standards/backend/k8s-client.spec.md) | backend | User token vs. service account — critical for RBAC compliance |
| [Frontend Conventions](specs/standards/frontend/conventions.spec.md) | frontend | NextJS patterns, Shadcn, React Query, component guidelines |
| [Frontend React Query](specs/standards/frontend/react-query.spec.md) | frontend | Data fetching hooks, mutations, cache invalidation |
| [Operator Conventions](specs/standards/control-plane/conventions.spec.md) | control-plane | OwnerReferences, reconciliation patterns, SecurityContext, resource limits |
| [Security Standards](specs/standards/security/security.spec.md) | security | Auth flows, RBAC, token handling, container security |
| [Cross-Cutting Conventions](specs/standards/platform/cross-cutting.spec.md) | platform | Image consistency, reconciliation, error propagation, namespace-scoped keys |

## Governance

| Document | Purpose |
|----------|---------|
| [ACP Constitution](.specify/memory/constitution.md) | 10 core principles: K8s-native, security, type safety, TDD, modularity, observability, lifecycle, context engineering, data access, commit discipline |
| [Runner Constitution](.specify/constitutions/runner.md) | Version pinning, automated freshness, image discipline, schema sync, bridge modularity |
| [SDD Preflight](.github/workflows/sdd-preflight.yml) | CI workflow enforcing constitution compliance on PRs |

## Architecture Decisions

| ADR | Decision |
|-----|----------|
| [ADR-0001](docs/internal/adr/0001-kubernetes-native-architecture.md) | CRDs, operators, and Job-based execution over traditional API |
| [ADR-0002](docs/internal/adr/0002-user-token-authentication.md) | User tokens for API ops instead of service accounts |
| [ADR-0003](docs/internal/adr/0003-multi-repo-support.md) | Multi-repository support in a single session |
| [ADR-0004](docs/internal/adr/0004-go-backend-python-runner.md) | Go for backend/operator, Python for runner |
| [ADR-0005](docs/internal/adr/0005-nextjs-shadcn-react-query.md) | NextJS + Shadcn + React Query frontend stack |
| [ADR-0006](docs/internal/adr/0006-ambient-runner-sdk-architecture.md) | Runner SDK design and architecture |
| [ADR-0007](docs/internal/adr/0007-unleash-feature-flags.md) | Unleash with workspace-scoped overrides |
| [ADR-0008](docs/internal/adr/0008-automate-code-reviews.md) | Automated inner-loop code review |
| [ADR-0009](docs/internal/adr/0009-rest-api-postgresql-trex-foundation.md) | REST API + PostgreSQL via rh-trex-ai |

## Component Guides

| Guide | Purpose |
|-------|---------|
| [Backend README](components/backend/README.md) | Go API development, testing, handler structure |
| [Backend Test Guide](components/backend/TEST_GUIDE.md) | Testing strategies, test utilities, integration test setup |
| [Frontend README](components/frontend/README.md) | NextJS development, local setup, environment config |
| [Frontend Design Guidelines](components/frontend/DESIGN_GUIDELINES.md) | Component patterns, Shadcn usage, type conventions |
| [Frontend Component Patterns](components/frontend/COMPONENT_PATTERNS.md) | Architecture patterns for React components |
| [Operator README](components/operator/README.md) | Operator development, watch patterns, reconciliation loop |
| [Runner README](components/runners/ambient-runner/README.md) | Python runner, Claude Code SDK integration |
| [Public API README](components/public-api/README.md) | Stateless gateway, token forwarding, input validation |
| [API Server Guide](components/ambient-api-server/CLAUDE.md) | rh-trex-ai REST API, plugin system, code generation |
| [SDK Guide](components/ambient-sdk/CLAUDE.md) | Go + Python client libraries for the public API |
| [CLI README](components/ambient-cli/README.md) | acpctl CLI for managing agentic sessions |
| [CodeRabbit Integration](docs/src/content/docs/features/coderabbit.md) | Setup, review gate, session credentials, `.coderabbit.yaml` config |

## Development Environment

| Guide | Purpose |
|-------|---------|
| [Kind](docs/internal/developer/local-development/kind.md) | Recommended local dev setup (Kubernetes in Docker) |
| [OpenShift](docs/internal/developer/local-development/openshift.md) | OpenShift Local (CRC) setup for OCP-specific features |
| [Hybrid](docs/internal/developer/local-development/hybrid.md) | Run components locally with breakpoint debugging |
| [Manifests](components/manifests/README.md) | Kustomize overlay structure, deploy.sh usage |

## Testing

| Guide | Purpose |
|-------|---------|
| [E2E Testing Guide](docs/internal/testing/e2e-guide.md) | Writing and running Cypress E2E tests |
| [E2E README](e2e/README.md) | Running E2E tests, environment setup, CI integration |

## Observability

| Guide | Purpose |
|-------|---------|
| [Overview](docs/internal/observability/README.md) | Monitoring, metrics, and tracing architecture |
| [Langfuse](docs/internal/observability/observability-langfuse.md) | LLM tracing with privacy-preserving defaults |
| [Operator Metrics](docs/internal/observability/operator-metrics-visualization.md) | Grafana dashboards for operator metrics |

## Design Documents

| Document | Purpose |
|----------|---------|
| [Declarative Session Reconciliation](docs/internal/design/declarative-session-reconciliation.md) | Session lifecycle via declarative status transitions |
| [Runner-Operator Contract](docs/internal/design/runner-operator-contract.md) | Interface contract between operator and runner pods |
| [Session Status Redesign](docs/internal/design/session-status-redesign.md) | Status field evolution and phase transitions |
| [Session Initialization Flow](docs/internal/design/session-initialization-flow.md) | How sessions are initialized and configured |
| [Spec-Runtime Synchronization](docs/internal/design/spec-runtime-synchronization.md) | Keeping spec and runtime state in sync |
| [Agent Runtime Registry](docs/internal/design/agent-runtime-registry-plan.md) | Agent runtime registry architecture |
| [CLI Runners](docs/internal/design/cli-runners-plan.md) | CLI runner design and implementation |
| [Status Update Comparison](docs/internal/design/status-update-comparison.md) | Comparison of status update approaches |

## Dependency Automation

| Resource | Purpose |
|----------|---------|
| [SDK Version Bump Workflow](.github/workflows/sdk-version-bump.yml) | Daily check for claude-agent-sdk + anthropic updates, auto-PR |
| [SDK Version Bump Script](scripts/sdk-version-bump.py) | PyPI version check, pyproject.toml update, changelog fetch |
| [SDK Feature Report Generator](scripts/sdk_report.py) | Parse release notes into structured feature data |

## Amber Automation

| Resource | Purpose |
|----------|---------|
| [Amber Config](.claude/amber-config.yml) | Automation policies and label mappings |
