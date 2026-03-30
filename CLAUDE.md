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
make build-all                # Build all container images
make deploy                   # Deploy to cluster
make test                     # Run tests
make lint                     # Lint code
make kind-up                  # Start local Kind cluster
make test-e2e-local           # Run E2E tests against Kind
make benchmark                # Run component benchmark harness
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
make lint                                    # All hooks, all files
pre-commit run gofmt-check --all-files       # Single hook
pre-commit run --files path/to/file.go       # Single file
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

## More Info

See [BOOKMARKS.md](BOOKMARKS.md) for architecture decisions, development context, code patterns, and component-specific guides.
