# Bookmarks

Progressive disclosure for task-specific documentation and references.

## Table of Contents

- [Architecture Decisions](#architecture-decisions)
- [Development Context](#development-context)
- [Code Patterns](#code-patterns)
- [Component Guides](#component-guides)
- [Deployment & Operations](#deployment--operations)
- [Testing](#testing)
- [Integrations](#integrations)
- [Observability](#observability)
- [Design Documents](#design-documents)
- [Amber Automation](#amber-automation)

---

## Architecture Decisions

### [ADR-0001: Kubernetes-Native Architecture](docs/adr/0001-kubernetes-native-architecture.md)

Why the platform uses CRDs, operators, and Job-based execution instead of a traditional API.

### [ADR-0002: User Token Authentication](docs/adr/0002-user-token-authentication.md)

Why user tokens are used for API operations instead of service accounts.

### [ADR-0003: Multi-Repo Support](docs/adr/0003-multi-repo-support.md)

Design for operating on multiple repositories in a single session.

### [ADR-0004: Go Backend, Python Runner](docs/adr/0004-go-backend-python-runner.md)

Language choices for each component and why.

### [ADR-0005: NextJS + Shadcn + React Query](docs/adr/0005-nextjs-shadcn-react-query.md)

Frontend technology stack decisions.

### [Decision Log](docs/decisions.md)

Chronological record of major decisions with links to ADRs.

---

## Development Context

### [Backend Development Context](.claude/context/backend-development.md)

Go backend patterns, K8s integration, handler conventions, user-scoped client usage.

### [Frontend Development Context](.claude/context/frontend-development.md)

NextJS patterns, Shadcn UI usage, React Query data fetching, component guidelines.

### [Security Standards](.claude/context/security-standards.md)

Auth flows, RBAC enforcement, token handling, container security patterns.

### [Architecture View](repomix-analysis/03-architecture-only.xml)

Full codebase architecture analysis (187K tokens). Load for cross-cutting questions.

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

## Deployment & Operations

### [Kind Local Development](docs/developer/local-development/kind.md)

Recommended local dev setup using Kind (Kubernetes in Docker).

### [CRC Local Development](docs/developer/local-development/crc.md)

OpenShift Local (CRC) setup for OpenShift-specific features.

### [OpenShift Deployment](docs/deployment/OPENSHIFT_DEPLOY.md)

Production OpenShift deployment guide.

### [OpenShift OAuth](docs/deployment/OPENSHIFT_OAUTH.md)

OAuth proxy configuration for cluster authentication.

### [Manifests README](components/manifests/README.md)

Kustomize overlay structure, deploy.sh usage.

### [Git Authentication](docs/deployment/git-authentication.md)

Git credential setup for session pods.

---

## Testing

### [E2E Testing Guide](docs/testing/e2e-guide.md)

Writing and running Cypress E2E tests.

### [E2E README](e2e/README.md)

Running E2E tests, environment setup, CI integration.

### [Testing Summary](docs/testing/testing-summary.md)

Overview of all test types (unit, contract, integration, E2E).

---

## Integrations

### [GitHub App Setup](docs/integrations/GITHUB_APP_SETUP.md)

GitHub App installation and configuration.

### [GitLab Integration](docs/integrations/gitlab-integration.md)

GitLab connectivity, self-hosted support, token setup.

### [Google Workspace](docs/integrations/google-workspace.md)

Google Drive and Workspace integration.

---

## Observability

### [Langfuse Deployment](docs/deployment/langfuse.md)

LLM tracing with privacy-preserving defaults.

### [Observability Overview](docs/observability/README.md)

Monitoring, metrics, and tracing architecture.

### [Operator Metrics](docs/observability/operator-metrics-visualization.md)

Grafana dashboards for operator metrics.

---

## Design Documents

### [Declarative Session Reconciliation](docs/design/declarative-session-reconciliation.md)

Session lifecycle management through declarative status transitions.

### [Runner-Operator Contract](docs/design/runner-operator-contract.md)

Interface contract between operator and runner pods.

### [Session Status Redesign](docs/design/session-status-redesign.md)

Status field evolution and phase transitions.

---

## Amber Automation

### [Amber Quickstart](docs/amber-quickstart.md)

Get started with Amber background agent in 5 minutes.

### [Amber Full Guide](docs/amber-automation.md)

Complete automation documentation for GitHub Issue-driven workflows.

### [Amber Config](.claude/amber-config.yml)

Automation policies and label mappings.
