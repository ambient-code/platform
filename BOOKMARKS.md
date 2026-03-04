# Bookmarks

Progressive disclosure for task-specific documentation and references.

## Table of Contents

- [Architecture Decisions](#architecture-decisions)
- [Development Context](#development-context)
- [Code Patterns](#code-patterns)
- [Component Guides](#component-guides)
- [Development Environment](#development-environment)
- [Testing](#testing)
- [Observability](#observability)
- [Design Documents](#design-documents)
- [Amber Automation](#amber-automation)

---

## Architecture Decisions

### [ADR-0001: Kubernetes-Native Architecture](docs/internal/adr/0001-kubernetes-native-architecture.md)

Why the platform uses CRDs, operators, and Job-based execution instead of a traditional API.

### [ADR-0002: User Token Authentication](docs/internal/adr/0002-user-token-authentication.md)

Why user tokens are used for API operations instead of service accounts.

### [ADR-0003: Multi-Repo Support](docs/internal/adr/0003-multi-repo-support.md)

Design for operating on multiple repositories in a single session.

### [ADR-0004: Go Backend, Python Runner](docs/internal/adr/0004-go-backend-python-runner.md)

Language choices for each component and why.

### [ADR-0005: NextJS + Shadcn + React Query](docs/internal/adr/0005-nextjs-shadcn-react-query.md)

Frontend technology stack decisions.

### [ADR-0006: Ambient Runner SDK Architecture](docs/internal/adr/0006-ambient-runner-sdk-architecture.md)

Runner SDK design and architecture.

---

## Development Context

### [Backend Development Context](.claude/context/backend-development.md)

Go backend patterns, K8s integration, handler conventions, user-scoped client usage.

### [Frontend Development Context](.claude/context/frontend-development.md)

NextJS patterns, Shadcn UI usage, React Query data fetching, component guidelines.

### [Security Standards](.claude/context/security-standards.md)

Auth flows, RBAC enforcement, token handling, container security patterns.

---

## Code Patterns

### [Error Handling Patterns](.claude/patterns/error-handling.md)

Consistent error patterns across backend, operator, and runner.

### [K8s Client Usage Patterns](.claude/patterns/k8s-client-usage.md)

When to use user token vs. service account clients. Critical for RBAC compliance.

### [React Query Usage Patterns](.claude/patterns/react-query-usage.md)

Data fetching hooks, mutations, cache invalidation, optimistic updates.

---

## Component Guides

### [Backend README](components/backend/README.md)

Go API development, testing, handler structure.

### [Backend Test Guide](components/backend/TEST_GUIDE.md)

Testing strategies, test utilities, integration test setup.

### [Frontend README](components/frontend/README.md)

NextJS development, local setup, environment config.

### [Frontend Design Guidelines](components/frontend/DESIGN_GUIDELINES.md)

Component patterns, Shadcn usage, type conventions, pre-commit checklist.

### [Frontend Component Patterns](components/frontend/COMPONENT_PATTERNS.md)

Architecture patterns for React components.

### [Operator README](components/operator/README.md)

Operator development, watch patterns, reconciliation loop.

### [Runner README](components/runners/claude-code-runner/README.md)

Python runner development, Claude Code SDK integration.

### [Public API README](components/public-api/README.md)

Stateless gateway design, token forwarding, input validation.

---

## Development Environment

### [Kind Local Development](docs/internal/developer/local-development/kind.md)

Recommended local dev setup using Kind (Kubernetes in Docker).

### [CRC Local Development](docs/internal/developer/local-development/crc.md)

OpenShift Local (CRC) setup for OpenShift-specific features.

### [Minikube Setup](docs/internal/developer/local-development/minikube.md)

Older approach, still supported.

### [Hybrid Development](docs/internal/developer/local-development/hybrid.md)

Run components locally with breakpoint debugging.

### [Manifests README](components/manifests/README.md)

Kustomize overlay structure, deploy.sh usage.

---

## Testing

### [E2E Testing Guide](docs/internal/testing/e2e-guide.md)

Writing and running Cypress E2E tests.

### [E2E README](e2e/README.md)

Running E2E tests, environment setup, CI integration.

---

## Observability

### [Observability Overview](docs/internal/observability/README.md)

Monitoring, metrics, and tracing architecture.

### [Langfuse Integration](docs/internal/observability/observability-langfuse.md)

LLM tracing with privacy-preserving defaults.

### [Operator Metrics](docs/internal/observability/operator-metrics-visualization.md)

Grafana dashboards for operator metrics.

---

## Design Documents

### [Declarative Session Reconciliation](docs/internal/design/declarative-session-reconciliation.md)

Session lifecycle management through declarative status transitions.

### [Runner-Operator Contract](docs/internal/design/runner-operator-contract.md)

Interface contract between operator and runner pods.

### [Session Status Redesign](docs/internal/design/session-status-redesign.md)

Status field evolution and phase transitions.

### [Session Initialization Flow](docs/internal/design/session-initialization-flow.md)

How sessions are initialized and configured.

### [Spec-Runtime Synchronization](docs/internal/design/spec-runtime-synchronization.md)

Keeping spec and runtime state in sync.

---

## Amber Automation

### [Amber Config](.claude/amber-config.yml)

Automation policies and label mappings.
