# Backend API

Go-based REST API for the Ambient Code Platform, managing Kubernetes Custom Resources with multi-tenant project isolation.

## Features

- **Project-scoped endpoints**: `/api/projects/:project/*` for namespaced resources
- **Multi-tenant isolation**: Each project maps to a Kubernetes namespace
- **WebSocket support**: Real-time session updates
- **Git operations**: Repository cloning, forking, PR creation
- **RBAC integration**: OpenShift OAuth for authentication

## Development

### Prerequisites

- Go 1.21+
- kubectl
- Docker or Podman
- Access to Kubernetes cluster (for integration tests)

### Quick Start

```bash
cd components/backend

# Install dependencies
make deps

# Run locally
make run

# Run with hot-reload (requires: go install github.com/cosmtrek/air@latest)
make dev
```

### Local development authentication (DISABLE_AUTH removed)

The backend **does not** support authentication bypass via `DISABLE_AUTH` (or any env-var based bypass).
All authenticated endpoints require a valid Kubernetes/OpenShift token passed via:

- `Authorization: Bearer <token>` (preferred)
- `X-Forwarded-Access-Token: <token>` (when running behind an auth proxy)

#### Option A: OpenShift / CRC (recommended for this repo)

```bash
# Login and obtain a user token
oc login ...
export OC_TOKEN="$(oc whoami -t)"

# Example request
curl -H "Authorization: Bearer ${OC_TOKEN}" \
  http://localhost:8080/health
```

#### Option B: kind/minikube (ServiceAccount token for local dev)

```bash
export DEV_NS=ambient-code
kubectl create namespace "${DEV_NS}" 2>/dev/null || true

kubectl -n "${DEV_NS}" create serviceaccount backend-dev 2>/dev/null || true

# Minimal example permissions (adjust as needed)
kubectl -n "${DEV_NS}" create role backend-dev \
  --verb=get,list,watch,create,update,patch,delete \
  --resource=secrets,configmaps,services,pods,rolebindings 2>/dev/null || true

kubectl -n "${DEV_NS}" create rolebinding backend-dev \
  --role=backend-dev \
  --serviceaccount="${DEV_NS}:backend-dev" 2>/dev/null || true

export DEV_TOKEN="$(kubectl -n "${DEV_NS}" create token backend-dev)"

curl -H "Authorization: Bearer ${DEV_TOKEN}" \
  http://localhost:8080/health
```

> Tip: If you’re running the backend in-cluster (e.g. via `make dev-start`), your client must still send a token; the backend will never fall back to the in-cluster service account for user-initiated operations.

### Build

```bash
# Build binary
make build

# Build container image
make build CONTAINER_ENGINE=docker  # or podman
```

### Testing

```bash
make test              # Unit + contract tests
make test-unit         # Unit tests only
make test-contract     # Contract tests only
make test-integration  # Integration tests (requires k8s cluster)
make test-permissions  # RBAC/permission tests
make test-coverage     # Generate coverage report
```

For integration tests, set environment variables:
```bash
export TEST_NAMESPACE=test-namespace
export CLEANUP_RESOURCES=true
make test-integration
```

### Linting

```bash
make fmt               # Format code
make vet               # Run go vet
make lint              # golangci-lint (install with make install-tools)
```

**Pre-commit checklist**:
```bash
# Run all linting checks
gofmt -l .             # Should output nothing
go vet ./...
golangci-lint run

# Auto-format code
gofmt -w .
```

### Dependencies

```bash
make deps              # Download dependencies
make deps-update       # Update dependencies
make deps-verify       # Verify dependencies
```

### Environment Check

```bash
make check-env         # Verify Go, kubectl, docker installed
```

## Architecture

See `CLAUDE.md` in project root for:
- Critical development rules
- Kubernetes client patterns
- Error handling patterns
- Security patterns
- API design patterns

## Reference Files

- `handlers/sessions.go` - AgenticSession lifecycle, user/SA client usage
- `handlers/middleware.go` - Auth patterns, token extraction, RBAC
- `handlers/helpers.go` - Utility functions (StringPtr, BoolPtr)
- `types/common.go` - Type definitions
- `server/server.go` - Server setup, middleware chain, token redaction
- `routes.go` - HTTP route definitions and registration
