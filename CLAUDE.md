# Ambient Code Platform

Kubernetes-native AI automation platform that orchestrates agentic sessions through containerized microservices. Built with Go (backend, operator), NextJS + Shadcn (frontend), Python (runner), and Kubernetes CRDs.

> Technical artifacts still use "vteam" for backward compatibility.

## Structure

- `components/backend/` - Go REST API (Gin), manages K8s Custom Resources with multi-tenant project isolation
- `components/frontend/` - NextJS web UI for session management and monitoring
- `components/operator/` - Go Kubernetes controller, watches CRDs and creates Jobs
- `components/runners/ambient-runner/` - Python runner executing Claude Code CLI in Job pods
- `components/ambient-cli/` - Go CLI (`acpctl`), manages agentic sessions from the command line
- `components/public-api/` - Stateless HTTP gateway, proxies to backend (no direct K8s access)
- `components/manifests/` - Kustomize-based deployment manifests and overlays
- `e2e/` - Cypress end-to-end tests
- `docs/` - Astro Starlight documentation site

## Key Files

- CRD definitions: `components/manifests/base/crds/agenticsessions-crd.yaml`, `projectsettings-crd.yaml`
- Session lifecycle: `components/backend/handlers/sessions.go`, `components/operator/internal/handlers/sessions.go`
- Auth & RBAC middleware: `components/backend/handlers/middleware.go`
- K8s client init: `components/operator/internal/config/config.go`
- Runner entry point: `components/runners/ambient-runner/main.py`
- Route registration: `components/backend/routes.go`
- Frontend API layer: `components/frontend/src/services/api/`, `src/services/queries/`

## Session Flow

```
User Creates Session → Backend Creates CR → Operator Spawns Job →
Pod Runs Claude CLI → Results Stored in CR → UI Displays Progress
```

## Commands

```shell
make build-all                 # Build all container images
make deploy                    # Deploy to cluster
make test                      # Run tests
make lint                      # Lint code
make kind-up                   # Start local Kind cluster
make test-e2e-local            # Run E2E tests against Kind
make benchmark                 # Run component benchmark harness
```

### Per-Component

```shell
# Backend / Operator (Go)
cd components/backend && gofmt -l . && go vet ./... && golangci-lint run
cd components/operator && gofmt -l . && go vet ./... && golangci-lint run

# Frontend
cd components/frontend && npm run build  # Must pass with 0 errors, 0 warnings

# Runner (Python)
cd components/runners/ambient-runner && uv venv && uv pip install -e .

# Docs
cd docs && npm run dev  # http://localhost:4321
```

### Benchmarking

```shell
# Human-friendly summary
make benchmark

# Agent / automation friendly output
make benchmark FORMAT=tsv

# Single component
make benchmark COMPONENT=frontend MODE=cold
```

Benchmark notes:

- `frontend` requires **Node.js 20+**
- `FORMAT=tsv` is preferred for agents to minimize token usage
- `warm` measures rebuild proxies, not browser-observed hot reload latency
- See `scripts/benchmarks/README.md` for semantics and caveats

## Critical Context

- **User token auth required**: All user-facing API ops use `GetK8sClientsForRequest(c)`, never the backend service account
- **OwnerReferences on all child resources**: Jobs, Secrets, PVCs must have controller owner refs
- **No `panic()` in production**: Return explicit `fmt.Errorf` with context
- **No `any` types in frontend**: Use proper types, `unknown`, or generic constraints
- **Conventional commits**: Squashed on merge to `main`

## Governance & Agent Guidelines

This repository uses a layered documentation system for AI agent governance:

| Document | Purpose |
|----------|---------|
| `CLAUDE.md` (this file) | Architecture overview, commands, critical context |
| `CONTRIBUTING.md` | Human contributor guide: workflow, standards, PR process |
| `BOOKMARKS.md` | Progressive disclosure index to deeper docs |
| `.claude/context/` | Task-specific context files (backend, frontend, security) |
| `.claude/patterns/` | Reusable code patterns (error handling, K8s client, React Query) |
| `.claude/skills/` | Discoverable skills for common tasks (dev-cluster, pr-fixer, unleash-flag) |
| `.claude/commands/` | Custom slash commands (review, analyze, implement, etc.) |
| `.claude/amber-config.yml` | Amber agent automation policies and risk classifications |

### Agent Rules

When working in this codebase, AI agents MUST:

1. **Use user-scoped K8s clients** for user operations — never fall back to the backend service account
2. **Never log tokens or secrets** — sanitize all log output
3. **Run linters before committing** — `gofmt`, `ruff format`, ESLint as appropriate
4. **Squash commits on merge** — use conventional commit prefixes (`feat:`, `fix:`, `docs:`, etc.)
5. **Respect PR size limits** — bug fixes ≤150 lines, features ≤300 lines (≤500 with justification)
6. **Never force-push** or modify security-critical code without human review
7. **Never skip CI checks** — all tests must pass before merge
8. **Escalate when unsure** — request human help if root cause is unclear, an architectural decision is needed, or confidence is below 80%

### Context Files

Load these for task-specific guidance:

- `.claude/context/backend-development.md` — Go backend patterns, K8s integration, handler conventions
- `.claude/context/frontend-development.md` — NextJS patterns, Shadcn UI, React Query data fetching
- `.claude/context/security-standards.md` — Auth flows, RBAC enforcement, token handling, container security

### Code Patterns

Reference these for consistent implementation:

- `.claude/patterns/error-handling.md` — Error wrapping, sentinel errors, user-facing error responses
- `.claude/patterns/k8s-client-usage.md` — User-scoped vs service-account clients, RBAC checks
- `.claude/patterns/react-query-usage.md` — Query keys, mutations, optimistic updates, cache invalidation

## Architecture Decisions

Key ADRs in `docs/internal/adr/`:

- **ADR-0001**: Kubernetes-native architecture (CRDs + operators + Job-based execution)
- **ADR-0002**: User token authentication (user tokens, not service accounts)
- **ADR-0003**: Multi-repo support (operating on multiple repos per session)
- **ADR-0004**: Go backend, Python runner (language choices per component)
- **ADR-0005**: NextJS + Shadcn + React Query (frontend stack)
- **ADR-0006**: Ambient Runner SDK architecture
- **ADR-0007**: Unleash feature flags

## Pre-commit Hooks

The project uses the [pre-commit](https://pre-commit.com/) framework to run linters locally before every commit. Configuration lives in `.pre-commit-config.yaml`.

### Install

```bash
make setup-hooks
```

### What Runs

**On every `git commit`:**

| Hook | Scope |
|------|-------|
| trailing-whitespace, end-of-file-fixer, check-yaml, check-added-large-files, check-merge-conflict, detect-private-key | All files |
| ruff-format, ruff (check + fix) | Python (runners, scripts) |
| gofmt, go vet, golangci-lint | Go (backend, operator, public-api — per-module) |
| eslint | Frontend TypeScript/JavaScript |
| branch-protection | Blocks commits to main/master/production |

**On every `git push`:**

| Hook | Scope |
|------|-------|
| push-protection | Blocks pushes to main/master/production |

### Run Manually

```bash
make lint                                        # All hooks, all files
pre-commit run gofmt-check --all-files           # Single hook
pre-commit run --files path/to/file.go           # Single file
```

### Skip Hooks

```bash
git commit --no-verify    # Skip pre-commit hooks
git push --no-verify      # Skip pre-push hooks
```

### Notes

- Go and ESLint wrappers (`scripts/pre-commit/`) skip gracefully if the toolchain is not installed
- `tsc --noEmit` and `npm run build` are **not** included (slow; CI gates on them)
- Branch/push protection scripts remain in `scripts/git-hooks/` and are invoked by pre-commit

## Testing

- **Frontend unit tests**: `cd components/frontend && npx vitest run --coverage` (466 tests, ~74% coverage). See `components/frontend/README.md`.
- **E2E tests**: `cd e2e && npx cypress run --browser chrome` (58 tests, mock SDK). See `e2e/README.md`.
- **Runner tests**: `cd components/runners/ambient-runner && python -m pytest tests/`
- **Backend tests**: `cd components/backend && make test`. See `components/backend/TEST_GUIDE.md`.

## Local Development

### Prerequisites

- Go 1.24+ (backend/operator)
- Node.js 20+ and npm (frontend)
- Python 3.11+ (runner)
- Podman or Docker (container builds)
- Kind and kubectl (local cluster)

### Quick Start

```shell
make kind-up          # Create Kind cluster and deploy all components
make local-up         # Start local development environment
make local-down       # Tear down local environment
make local-clean      # Full cleanup
make local-rebuild    # Rebuild and redeploy
```

### Troubleshooting

```shell
# Check pod status
kubectl get pods -n ambient-code

# View pod logs
kubectl logs <pod-name> -n ambient-code

# Complete reset
kind delete cluster --name ambient-code && make kind-up
```

## Observability

- **Langfuse integration** for LLM tracing with privacy-preserving defaults — see `docs/internal/observability/observability-langfuse.md`
- **Operator metrics** with Grafana dashboards — see `docs/internal/observability/operator-metrics-visualization.md`
- Overview: `docs/internal/observability/README.md`

## More Info

See [BOOKMARKS.md](BOOKMARKS.md) for architecture decisions, development context, code patterns, and component-specific guides.
