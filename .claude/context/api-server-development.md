# API Server Development Context

**When to load:** Working on `components/ambient-api-server/` — REST handlers, gRPC, plugins, OpenAPI, or DB migrations

## Quick Reference

- **Language:** Go 1.21+
- **Framework:** rh-trex-ai (upstream), Gin HTTP, gRPC
- **DB:** PostgreSQL via GORM + testcontainers-go for tests
- **Proto:** `proto/ambient/v1/` → `pkg/api/grpc/ambient/v1/`
- **OpenAPI:** `openapi/openapi.yaml` → `pkg/api/openapi/` (never edit generated files)
- **Entry:** `cmd/ambient-api-server/main.go`
- **Environments:** `AMBIENT_ENV=development|integration_testing|production`

## Plugin Architecture

Every resource kind lives in `plugins/<kind>/`:

```
plugins/sessions/
  plugin.go       # registers routes + gRPC handlers
  model.go        # DB struct (GORM)
  handler.go      # HTTP handlers (REST)
  service.go      # business logic
  dao.go          # DB access layer
  presenter.go    # model → API response
  migration.go    # DB schema migration
  mock_dao.go     # test mock
  *_test.go       # table-driven tests
```

Generate a new plugin:
```bash
cd components/ambient-api-server
go run ./scripts/generator.go --kind MyResource --fields "name:string,status:string"
```

## gRPC

- Proto definitions: `proto/ambient/v1/*.proto`
- Generated stubs: `pkg/api/grpc/ambient/v1/` (committed, regenerate with `make generate`)
- gRPC server registered alongside HTTP in `cmd/ambient-api-server/environments/`
- Auth: `pkg/middleware/bearer_token_grpc.go` — same JWT/token flow as HTTP
- Watch streams: server-side streaming RPCs for session events

**Regenerate protos:**
```bash
cd components/ambient-api-server && make generate
```

## OpenAPI / SDK Generation Pipeline

```
openapi/openapi.yaml
  └─ make generate (in ambient-api-server)
       └─ pkg/api/openapi/*.go          (Go server stubs — DO NOT EDIT)
  └─ ambient-sdk/generator/
       └─ go-sdk/client/*.go            (generated Go SDK)
       └─ python-sdk/ambient_platform/  (generated Python SDK)
       └─ ts-sdk/src/                   (generated TypeScript SDK)
```

**Never edit `pkg/api/openapi/` directly.** Edit `openapi/openapi.yaml`, then `make generate`.

## Key Commands

```bash
cd components/ambient-api-server
make binary          # Build binary
make run             # Run locally (no auth, development env)
make run-no-auth     # Explicit no-auth mode
make test            # Run all tests (requires podman socket)
make generate        # Regenerate protos + OpenAPI stubs
make db/setup        # Start test Postgres container
make db/teardown     # Stop test Postgres container
make lint            # gofmt + go vet + golangci-lint
```

## Authentication & Authorization

- JWT validation: `pkg/middleware/bearer_token.go`
- gRPC JWT: `pkg/middleware/bearer_token_grpc.go`
- RBAC: `pkg/rbac/middleware.go` + `pkg/rbac/permissions.go`
- Caller context (username, roles) propagated via `pkg/middleware/caller_context.go`
- `AMBIENT_ENV=development` disables auth (local dev only)

## Error Handling

```go
// Plugin handlers follow this pattern
if err != nil {
    return nil, errors.NewInternalServerError("failed to create session: %v", err)
}
if !found {
    return nil, errors.NewNotFoundError("session %s not found", id)
}
```

Use `pkg/errors` types — they map to correct HTTP status codes automatically.

## Testing

- **Table-driven tests required** for all service/handler logic
- `AMBIENT_ENV=integration_testing` spins up ephemeral Postgres via testcontainers-go
- Requires podman socket: `systemctl --user start podman.socket`
- `DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock`
- Mock DAOs (`mock_dao.go`) for unit tests without DB

## Pre-Commit Checklist

- [ ] `go fmt ./...` applied
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` passes
- [ ] No `panic()` — use `errors.NewInternalServerError`
- [ ] Proto changes → `make generate` run
- [ ] OpenAPI changes → `make generate` run, never edit `pkg/api/openapi/` directly
- [ ] New plugin follows 8-file structure
- [ ] Table-driven tests for new logic
- [ ] DB migrations additive only (no column drops in production)
