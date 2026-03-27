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

---

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
go run ./scripts/generator.go \
  --kind Agent \
  --fields "project_id:string:required,name:string:required,prompt:string" \
  --project ambient \
  --repo github.com/ambient-code/platform/components \
  --library github.com/openshift-online/rh-trex-ai
```

After generation, always check and fix:

1. **Directory naming** — generator creates `{kindLowerPlural}` (e.g. `inboxMessages`, not `inbox`). Rename manually if the desired package name differs.
2. **Middleware import** — `RegisterRoutes` uses `auth.JWTMiddleware` in generated code. Replace with `environments.JWTMiddleware`. Every time. No exceptions.
3. **Nested route variable names** — `mux.Vars` key must match the route variable. Nested routes use `{pa_id}` or `{msg_id}`, not `{id}`. Generated handlers always use `{id}` — fix them.
4. **Integration tests for nested routes** — generated tests call flat client methods that don't exist for nested resources. Stub with `t.Skip("nested route — hand-write test")`.

---

## OpenAPI Fragments

Each resource has its own fragment `openapi/openapi.<kind>.yaml`. The main `openapi/openapi.yaml` references all fragments. Edit the fragment, never the merged file.

**Schema placement rules (critical for generator):**

The SDK generator's `inferResourceName` scans each sub-spec file and selects schemas alphabetically. The **first candidate alphabetically must use `allOf`** (primary resource schema extending `ObjectReference`). If it doesn't, the parse fails silently and generates wrong client code.

- **Primary resource schemas** (have `allOf`) → in the sub-spec file `openapi.<kind>.yaml`
- **Auxiliary schemas** (request bodies, response envelopes, view models) → in `openapi.yaml` main `components/schemas`

Auxiliary schemas that do **not** end in `List`, `PatchRequest`, or `StatusPatchRequest` must live in the main file:
- `IgniteRequest`, `IgniteResponse` → `openapi.yaml`
- `ProjectHome`, `ProjectHomeAgent` → `openapi.yaml`
- Any ad-hoc view model or request body → `openapi.yaml`

Safe in sub-spec files:
- `ProjectAgent` (has `allOf`)
- `ProjectAgentList`, `ProjectAgentPatchRequest` (filtered by suffix)

---

## gRPC

- Proto definitions: `proto/ambient/v1/*.proto`
- Generated stubs: `pkg/api/grpc/ambient/v1/` (committed — regenerate with `make proto`)
- gRPC server registered alongside HTTP in `cmd/ambient-api-server/environments/`
- Auth: `pkg/middleware/bearer_token_grpc.go` — same JWT/token flow as HTTP

**Regenerate protos:**
```bash
cd components/ambient-api-server && make proto
```

**gRPC presenter completeness rule:** `grpc_presenter.go` `sessionToProto()` (and equivalent for other Kinds) must map **every** field that exists in both the DB model and proto message. Missing fields cause downstream consumers (CP, operator) to receive zero values silently — no compile error, no runtime error, just wrong data.

When adding a new field to a model:
1. Add to `model.go` struct
2. Add to proto definition in `proto/ambient/v1/<kind>.proto`
3. Run `make proto`
4. Add mapping in `grpc_presenter.go`
5. Add mapping in `presenter.go` (REST)

**proto field addition workflow:**
```bash
# 1. Edit .proto
# 2. Regenerate
make proto
# 3. Verify *.pb.go changed
git diff pkg/api/grpc/
# 4. Wire through presenter
```

Do not edit `*.pb.go` directly — it is always overwritten by `make proto`.

---

## HTTP Handler Patterns

### Handler error pattern

```go
if errors.IsNotFound(err) {
    return nil, errors.NewNotFoundError("session %s not found", id)
}
if err != nil {
    return nil, errors.NewInternalServerError("failed to create session: %v", err)
}
```

Use `pkg/errors` types — they map to correct HTTP status codes automatically. No `panic()` — ever.

### Nested resource handler scoping

For resources nested under a parent (e.g. `InboxMessage` under `Agent`):

- `handler.go` List must inject the parent ID into the TSL search filter — never return cross-parent data
- `handler.go` Create must set the parent ID from the URL path variable, ignoring any `parent_id` in the request body (prevents body spoofing)

Example for InboxMessage scoped to Agent:
```go
// List — always scope by agent
listArgs.Search = fmt.Sprintf("agent_id = '%s'", mux.Vars(r)["pa_id"])

// Create — always set from URL
inboxMessage.AgentId = mux.Vars(r)["pa_id"]
```

`listArgs.Search` is `string`, not `*string` — use empty-string checks, not nil checks.

### HTTP status for ignite endpoints

- `Ignite` returns HTTP 201 on new session creation
- `Ignite` returns HTTP 200 on re-ignite (session already active)
- SDK `doMultiStatus()` accepts both — use it for any endpoint that can return either

### PATCH request scope

Limit `PatchRequest` structs to only the fields that users are permitted to change. For `InboxMessage`, only `Read *bool` is permitted. No other fields. Prevents privilege escalation via PATCH.

---

## Migrations

Migrations live in `plugins/<kind>/migration.go`. They use `gormigrate`:

```go
func Migration() *gormigrate.Migration {
    return &gormigrate.Migration{
        ID: "202601010001",
        Migrate: func(db *gorm.DB) error {
            return db.AutoMigrate(&MyKind{})
        },
        Rollback: func(db *gorm.DB) error {
            return db.Migrator().DropTable("my_kinds")
        },
    }
}
```

Migration rules:
- IDs are timestamps — use `YYYYMMDDNNNN` format, unique across all plugins
- Migrations are **additive only** in production — no column drops, no renames
- Register in `cmd/ambient-api-server/environments/db_migrations.go`

---

## Authentication & Authorization

- JWT validation: `pkg/middleware/bearer_token.go`
- gRPC JWT: `pkg/middleware/bearer_token_grpc.go`
- RBAC: `pkg/rbac/middleware.go` + `pkg/rbac/permissions.go`
- Caller context (username, roles) propagated via `pkg/middleware/caller_context.go`
- `AMBIENT_ENV=development` disables auth (local dev only)
- **Never log token values** — log `len(token)` if you need to debug

---

## Testing

- **Table-driven tests required** for all service/handler logic
- `AMBIENT_ENV=integration_testing` spins up ephemeral Postgres via testcontainers-go
- Requires podman socket: `systemctl --user start podman.socket`
- `DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock`
- Mock DAOs (`mock_dao.go`) for unit tests without DB
- Presenter nil safety: nil-guard each nullable field independently — `UpdatedAt` and `CreatedAt` can be nil independently; treating them as a pair causes panics

---

## Build Commands

```bash
cd components/ambient-api-server

make generate          # Regenerate OpenAPI Go client from openapi/*.yaml
make binary            # Compile the ambient-api-server binary
make test              # Integration tests — spins up testcontainer PostgreSQL
make test-integration  # Run only ./test/integration/... package
make proto             # Regenerate gRPC stubs from proto/
make proto-lint        # Lint proto definitions
go fmt ./...           # Format Go source
golangci-lint run      # Lint
```

> `make generate` must be run after any change to `openapi/*.yaml`. It emits to `pkg/api/openapi/` — never edit that directory manually.

---

## Pre-Commit Checklist

- [ ] `go fmt ./...` applied
- [ ] `go vet ./...` passes
- [ ] `golangci-lint run` passes
- [ ] No `panic()` — use `errors.NewInternalServerError`
- [ ] Proto changes → `make proto` run, `*.pb.go` committed
- [ ] OpenAPI changes → `make generate` run, `pkg/api/openapi/` never edited directly
- [ ] New plugin follows 8-file structure
- [ ] Generator middleware import fixed (`environments.JWTMiddleware`)
- [ ] Nested route `mux.Vars` keys match route variable names
- [ ] gRPC presenter maps all new fields
- [ ] Table-driven tests for new logic
- [ ] DB migrations additive only (no column drops in production)
- [ ] Auxiliary DTO schemas in `openapi.yaml`, not sub-spec files
