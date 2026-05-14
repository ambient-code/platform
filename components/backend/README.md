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

### Authentication

The backend supports two auth modes, controlled by the `sso-authentication` Unleash feature flag. This is an **infrastructure flag** — it is not visible in the workspace settings UI and is not user-configurable. It is enabled per-environment by the ops team during SSO migration.

**SSO mode (flag on):** The backend validates JWTs from Keycloak against the JWKS endpoint, extracts identity from OIDC claims (`email`, `preferred_username`, `groups`), and uses K8s impersonation for all API calls. API keys (K8s ServiceAccount tokens) are accepted via TokenReview fallback.

**Legacy mode (flag off):** The backend reads `X-Forwarded-Access-Token` or `Authorization: Bearer` headers and uses the raw token as the K8s bearer token (OAuth proxy flow).

In the Kind dev cluster, legacy mode is the default. Toggle SSO on/off with `make kind-sso-toggle` (affects both frontend and backend).

#### Local development (Kind)

`make kind-up` deploys Keycloak automatically. The backend is configured with:
- `SSO_ISSUER_URL` — points to the in-cluster Keycloak
- `SSO_AUDIENCE` — `ambient-frontend`

To test backend endpoints directly with a Keycloak JWT:

```bash
# Get a JWT from Keycloak (from within the cluster)
JWT=$(kubectl run -n ambient-code jwt-dev --rm -i --restart=Never --quiet \
  --image=curlimages/curl -- sh -c \
  ‘curl -sf -X POST http://keycloak-service:8080/realms/ambient-code/protocol/openid-connect/token \
    -d client_id=ambient-frontend \
    -d client_secret=dev-secret-do-not-use-in-prod \
    -d grant_type=password \
    -d username=developer \
    -d password=developer \
    -d scope=openid’ 2>/dev/null | jq -r ‘.access_token’)

curl -H "Authorization: Bearer $JWT" http://localhost:12646/api/projects
```

K8s ServiceAccount tokens also work (dual-path auth):

```bash
TOKEN=$(kubectl get secret test-user-token -n ambient-code \
  -o jsonpath=’{.data.token}’ | base64 -d)
curl -H "Authorization: Bearer $TOKEN" http://localhost:12646/api/projects
```

#### Unit tests note

Unit tests use:

- `go test -tags=test ./handlers`
- `SetValidTestToken(...)` (see `components/backend/tests/test_utils/http_utils.go`)

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

### Feature flags (Unleash)

See [docs/feature-flags](../../docs/feature-flags/README.md) for env vars, handler usage, and examples.

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
- `handlers/featureflags.go` - Feature flag helpers (see docs/feature-flags/)
- `featureflags/featureflags.go` - Unleash client init
- `types/common.go` - Type definitions
- `server/server.go` - Server setup, middleware chain, token redaction
- `routes.go` - HTTP route definitions and registration
