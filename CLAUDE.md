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
- `components/ambient-api-server/` - Go REST API microservice (rh-trex-ai framework), PostgreSQL-backed
- `components/ambient-sdk/` - Go + Python client SDK for the platform's public REST API
- `components/open-webui-llm/` - Open WebUI LLM integration
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
make kind-rebuild              # Rebuild images + redeploy to running cluster
make kind-login                # Set kubectl context + configure acpctl
make dev-bootstrap             # Bootstrap developer workspace
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

## Critical Conventions

Cross-cutting rules that apply across ALL components. Component-specific conventions live in
component DEVELOPMENT.md files (see [BOOKMARKS.md](BOOKMARKS.md) > Component Development Guides).

- **User token auth required**: All user-facing API ops use `GetK8sClientsForRequest(c)`, never the backend service account
- **No tokens in logs/errors/responses**: Use `len(token)` for logging, generic messages to users
- **OwnerReferences on all child resources**: Jobs, Secrets, PVCs must have controller owner refs
- **No `panic()` in production**: Return explicit `fmt.Errorf` with context
- **No `any` types in frontend**: Use proper types, `unknown`, or generic constraints
- **Feature flags strongly recommended**: Gate new features behind Unleash flags. Use `/unleash-flag` to set up
- **No new CRDs**: Existing CRDs (AgenticSession, ProjectSettings) are grandfathered. For new persistent storage, confirm with the user whether to use repo files or PostgreSQL — do not default to K8s custom resources
- **Conventional commits**: Squashed on merge to `main`
- **Design for extensibility before adding items**: When building infrastructure that will have
  things added to it (menus, config schemas, API surfaces), build the extensibility mechanism
  first — conditional rendering, feature-flag gating, discovery. Retrofitting causes rework.
- **Verify contracts and references**: Before building on an assumption (env var exists, path is
  correct, URL is reachable), verify the contract. After moving anything, grep scripts, workflows,
  manifests, and configs — not just source code.
- **CI/CD security**: Never use `pull_request_target` (grants write access to forked PR code).
  Never hardcode tokens — use `actions/create-github-app-token`. For automated pipelines:
  discovery → validation → PR → auto-merge.
- **Full-stack awareness**: Before building a new pipeline, check if an existing one can be
  reused. Auth/credential/API changes must update ALL consumers (backend, CLI, SDK, runner,
  sidecar) in the same PR.
- **Separate configuration from code**: Config changes must not require code changes. Externalize
  via env vars, ConfigMaps, manifests, or feature flags. If a value varies across environments
  or changes over time, it's config, not code.
- **Image references must match across the stack**: After changing an image name or tag, grep all overlays, workflows, and ConfigMaps
- **Reconcile, don't create-or-skip**: Use update-or-create patterns, not create-and-ignore-`AlreadyExists`
- **Never silently swallow partial failures**: Every error path must propagate or be collected, not discarded
- **Namespace-scope shared state keys**: Cache keys and status entries must include namespace/project prefix
- **Restricted SecurityContext on all containers**: `runAsNonRoot`, drop `ALL` capabilities, `readOnlyRootFilesystem`

Component-specific conventions:
- Backend: [DEVELOPMENT.md](components/backend/DEVELOPMENT.md), [ERROR_PATTERNS.md](components/backend/ERROR_PATTERNS.md), [K8S_CLIENT_PATTERNS.md](components/backend/K8S_CLIENT_PATTERNS.md)
- Frontend: [DEVELOPMENT.md](components/frontend/DEVELOPMENT.md), [REACT_QUERY_PATTERNS.md](components/frontend/REACT_QUERY_PATTERNS.md)
- Operator: [DEVELOPMENT.md](components/operator/DEVELOPMENT.md)
- Security: [security-standards.md](docs/security-standards.md)

## Pre-commit Hooks

Configured in `.pre-commit-config.yaml`. Install: `make setup-hooks`. Run all: `make lint`.

- Go and ESLint wrappers (`scripts/pre-commit/`) skip gracefully if the toolchain is not installed
- `tsc --noEmit` and `npm run build` are **not** in pre-commit (slow; CI gates on them)
- Branch/push protection blocks commits and pushes to main/master/production

## Testing

- **Frontend unit tests**: `cd components/frontend && npx vitest run --coverage`. See `components/frontend/README.md`.
- **E2E tests**: `cd e2e && npx cypress run --browser chrome`. See `e2e/README.md`.
- **Runner tests**: `cd components/runners/ambient-runner && python -m pytest tests/`
- **Backend tests**: `cd components/backend && make test`. See `components/backend/TEST_GUIDE.md`.

## Convention Authority

This file and [BOOKMARKS.md](BOOKMARKS.md) are the authoritative source of project conventions. The [ACP Constitution](.specify/memory/constitution.md) covers spec-kit-specific governance (commit discipline thresholds, context engineering, amendment process) but defers to this file for shared conventions. If they conflict, this file wins. Spec-kit is optional tooling, not mandatory.
