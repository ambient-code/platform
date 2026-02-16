# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The **Ambient Code Platform** is a Kubernetes-native AI automation platform that orchestrates intelligent agentic sessions through containerized microservices. The platform enables AI-powered automation for analysis, research, development, and content creation tasks via a modern web interface.

> **Note:** This project was formerly known as "vTeam". Technical artifacts (image names, namespaces, API groups, routes) still use "vteam" for backward compatibility. The docs use ACP naming.

### Core Architecture

The system follows a Kubernetes-native pattern with Custom Resources, Operators, and Job execution:

1. **Frontend** (NextJS + Shadcn): Web UI for session management and monitoring
2. **Backend API** (Go + Gin): REST API managing Kubernetes Custom Resources with multi-tenant project isolation
3. **Agentic Operator** (Go): Kubernetes controller watching CRs and creating Jobs
4. **Claude Code Runner** (Python): Job pods executing Claude Code CLI with multi-agent collaboration

### Agentic Session Flow

```
User Creates Session → Backend Creates CR → Operator Spawns Job →
Pod Runs Claude CLI → Results Stored in CR → UI Displays Progress
```

### Amber Background Agent

Automates common development tasks via GitHub Issues. See [Amber Quickstart](docs/amber-quickstart.md), [Full Documentation](docs/amber-automation.md), [Amber Config](.claude/amber-config.yml).

Labels: `amber:auto-fix`, `amber:refactor`, `amber:test-coverage`

## Memory System - Loadable Context

This repository uses a structured **memory system** to provide targeted, loadable context instead of relying solely on this CLAUDE.md file.

### Quick Reference

**Load these files when working in specific areas:**

| Task Type | Context File | Architecture View | Pattern File |
|-----------|--------------|-------------------|--------------|
| **Backend API work** | `.claude/context/backend-development.md` | `repomix-analysis/03-architecture-only.xml` | `.claude/patterns/k8s-client-usage.md` |
| **Frontend UI work** | `.claude/context/frontend-development.md` | `repomix-analysis/03-architecture-only.xml` | `.claude/patterns/react-query-usage.md` |
| **Security review** | `.claude/context/security-standards.md` | `repomix-analysis/03-architecture-only.xml` | `.claude/patterns/error-handling.md` |
| **Architecture questions** | - | `repomix-analysis/03-architecture-only.xml` | See ADRs below |

**Note:** We use a single repomix architecture view (grade 8.8/10, 187K tokens) for all tasks. See `.claude/repomix-guide.md` for details.

### Available Memory Files

**1. Context Files** (`.claude/context/`)

- `backend-development.md` - Go backend, K8s integration, handler patterns, operator patterns, API design, package organization, common mistakes
- `frontend-development.md` - NextJS, Shadcn UI, React Query patterns
- `security-standards.md` - Auth, RBAC, token handling, container security patterns

**2. Architectural Decision Records** (`docs/adr/`)

- Documents WHY decisions were made, not just WHAT
- `0001-kubernetes-native-architecture.md`
- `0002-user-token-authentication.md`
- `0003-multi-repo-support.md`
- `0004-go-backend-python-runner.md`
- `0005-nextjs-shadcn-react-query.md`

**3. Code Pattern Catalog** (`.claude/patterns/`)

- `error-handling.md` - Consistent error patterns (backend, operator, runner)
- `k8s-client-usage.md` - When to use user token vs. service account
- `react-query-usage.md` - Data fetching patterns (queries, mutations, caching)

**4. Other References**

- `.claude/repomix-guide.md` - Guide for using the architecture view effectively
- `docs/decisions.md` - Lightweight chronological record of major decisions
- `docs/DOCUMENTATION_MAP.md` - Quick-reference map for all documentation

## Development Commands

### Essential Commands

```bash
make kind-up           # Start local Kind cluster (recommended)
make build-all         # Build all container images
make deploy            # Deploy to cluster
make test              # Run tests
make lint              # Lint code
make clean             # Clean up deployment
```

### Local Development

**Kind (recommended):** `make kind-up` → access at `http://localhost:8080`. Full guide: `docs/developer/local-development/kind.md`

**CRC (OpenShift-specific):** `make dev-start`. Full guide: `docs/developer/local-development/crc.md`

**Hot-reloading:** `DEV_MODE=true make dev-start` (terminal 1), `make dev-sync` (terminal 2)

### Component Development

See component-specific documentation for detailed commands:

- **Backend** (`components/backend/README.md`)
- **Frontend** (`components/frontend/README.md`), also `DESIGN_GUIDELINES.md`
- **Operator** (`components/operator/README.md`)
- **Claude Code Runner** (`components/runners/claude-code-runner/README.md`)

### Build & Deploy Details

```bash
make build-all CONTAINER_ENGINE=podman    # Build with podman
make build-all PLATFORM=linux/arm64       # Build for ARM64
make push-all REGISTRY=quay.io/username   # Push to registry
make deploy NAMESPACE=my-namespace        # Deploy to custom namespace
```

### Documentation

```bash
pip install -r requirements-docs.txt && mkdocs serve   # Serve docs locally
mkdocs build                                            # Build static site
```

## Key Architecture Patterns

### Custom Resource Definitions (CRDs)

1. **AgenticSession** (`agenticsessions.vteam.ambient-code`): AI execution session with prompt, repos (multi-repo), interactive mode, timeout, model selection
2. **ProjectSettings** (`projectsettings.vteam.ambient-code`): Project-scoped configuration (API keys, defaults)
3. **RFEWorkflow** (`rfeworkflows.vteam.ambient-code`): 7-step agent council process for engineering refinement

### Multi-Repo Support

Each repo has required `input` (URL, branch) and optional `output` (fork/target). `mainRepoIndex` specifies the Claude working directory (default: 0). Per-repo status: `pushed` or `abandoned`.

### Interactive vs Batch Mode

- **Batch Mode** (default): Single prompt execution with timeout
- **Interactive Mode** (`interactive: true`): Long-running chat sessions using inbox/outbox files

## Configuration Standards

### Python

- **Virtual environments**: `python -m venv venv` or `uv venv`; prefer `uv` over `pip`
- **Formatting**: black (double quotes), isort with black profile
- **Linting**: flake8 (ignore E203, W503)

### Go

- **Formatting**: `go fmt ./...` (enforced)
- **Linting**: golangci-lint (install via `make install-tools`)
- **Testing**: Table-driven tests with subtests
- **Error handling**: Explicit error returns, no panic in production code

### Container Images

- **Default registry**: `quay.io/ambient_code`
- **Image tags**: vteam_frontend, vteam_backend, vteam_operator, vteam_claude_runner
- **Platform**: Default `linux/amd64`, ARM64 via `PLATFORM=linux/arm64`
- **Build tool**: Docker or Podman (`CONTAINER_ENGINE=podman`)

### Git Workflow

- **Default branch**: `main`
- **Feature branches**: Required for development
- **Commit style**: Conventional commits (squashed on merge)
- **Branch verification**: Always check current branch before file modifications

### Kubernetes/OpenShift

- **Default namespace**: `ambient-code` (production), `vteam-dev` (local dev)
- **CRD group**: `vteam.ambient-code`
- **API version**: `v1alpha1` (current)
- **RBAC**: Namespace-scoped service accounts with minimal permissions

## Backend and Operator Development Standards

**IMPORTANT**: When working on backend or operator code, you MUST load the detailed context files:

- **→ `.claude/context/backend-development.md`** — Critical rules, package organization, K8s client patterns, API design, operator patterns (watch loop, reconciliation, status updates, goroutine monitoring), common mistakes, pre-commit checklist, reference files
- **→ `.claude/patterns/k8s-client-usage.md`** — User-scoped vs service account client decision tree
- **→ `.claude/patterns/error-handling.md`** — Handler and operator error patterns with code examples
- **→ `.claude/context/security-standards.md`** — Token handling, RBAC enforcement, container security, input validation

**Pre-commit commands:**

```bash
cd components/backend && gofmt -l . && go vet ./... && golangci-lint run
cd components/operator && gofmt -l . && go vet ./... && golangci-lint run
gofmt -w components/backend components/operator   # Auto-format
```

## Frontend Development Standards

**→ Load `.claude/context/frontend-development.md`** for complete frontend standards, critical rules, and pre-commit checklist.

**→ See `components/frontend/DESIGN_GUIDELINES.md`** for detailed patterns and examples.

**→ See `.claude/patterns/react-query-usage.md`** for data fetching patterns.

## Langfuse Observability (LLM Tracing)

Optional Langfuse integration for LLM observability with privacy-first design (messages redacted by default).

**→ See `docs/observability/observability-langfuse.md`** for trace structure, configuration, and privacy details.

**→ See `docs/deployment/langfuse.md`** for deployment instructions.

## GitHub Actions CI/CD

- **components-build-deploy.yml**: Change-detection builds, multi-platform (amd64/arm64), pushes to `quay.io/ambient_code` on main
- **go-lint.yml** / **frontend-lint.yml**: Code quality (gofmt, go vet, golangci-lint, ESLint, TypeScript)
- **e2e.yml**: End-to-end Cypress tests in kind cluster
- **amber-issue-handler.yml**: Amber background agent automation
- **claude.yml** / **claude-code-review.yml**: Claude Code integration and automated code reviews
- **prod-release-deploy.yaml**: Production releases with semver and changelog

## Testing Strategy

**→ See `docs/testing/testing-summary.md`** for the complete test inventory matrix and CI/CD orchestration.

**→ See `docs/testing/e2e-guide.md`** and `e2e/README.md` for E2E testing with Cypress.

**Quick start:**

```bash
make test-e2e-local                          # E2E against local kind cluster
cd components/backend && go test ./...       # Backend unit tests
```

## Documentation

**→ See `docs/DOCUMENTATION_MAP.md`** for a complete map of all documentation.

**Standards**: Default to improving existing documentation rather than creating new files. Colocate docs with relevant code (e.g., `components/backend/README.md`). Only create top-level docs for cross-cutting concerns.
