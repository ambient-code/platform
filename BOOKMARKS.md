# Bookmarks

Progressive disclosure for task-specific documentation and references.

## Table of Contents

- [Governance](#governance)
- [Architecture Decisions](#architecture-decisions)
- [Component Development Guides](#component-development-guides)
- [Component Guides](#component-guides)
- [Development Environment](#development-environment)
- [Testing](#testing)
- [Observability](#observability)
- [Design Documents](#design-documents)
- [Dependency Automation](#dependency-automation)
- [Amber Automation](#amber-automation)

---

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

## Cross-Cutting Conventions

| Document | Purpose |
|----------|---------|
| [CodeRabbit-Derived Conventions](docs/coderabbit-derived-conventions.md) | Image consistency, reconciliation patterns, error propagation, namespace-scoped keys, SecurityContext — derived from 971 CodeRabbit review comments |

## Component Development Guides

Convention documentation for each component. Loaded by review agents on demand.

| Guide | Scope |
|-------|-------|
| [Backend Development](components/backend/DEVELOPMENT.md) | Go patterns, K8s integration, handler conventions, user-scoped clients |
| [Backend Error Patterns](components/backend/ERROR_PATTERNS.md) | Consistent error patterns across backend and operator |
| [Backend K8s Client Patterns](components/backend/K8S_CLIENT_PATTERNS.md) | User token vs. service account — critical for RBAC compliance |
| [Frontend Development](components/frontend/DEVELOPMENT.md) | NextJS patterns, Shadcn, React Query, component guidelines |
| [Frontend React Query Patterns](components/frontend/REACT_QUERY_PATTERNS.md) | Data fetching hooks, mutations, cache invalidation |
| [Operator Development](components/operator/DEVELOPMENT.md) | OwnerReferences, reconciliation patterns, SecurityContext, resource limits |
| [Security Standards](docs/security-standards.md) | Auth flows, RBAC, token handling, container security |

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
