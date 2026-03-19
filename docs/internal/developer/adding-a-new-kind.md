# Adding a New Kind

Step-by-step developer guide for adding a new resource Kind to the Ambient platform. A Kind is a full CRUD resource backed by a database table, exposed via REST API, generated into the SDK, and surfaced through the CLI.

**Reference diagram:** [`docs/internal/architecture/diagrams/agent-build.mmd`](../architecture/diagrams/agent-build.mmd)

---

## Overview

Adding a Kind touches four components in order:

```
API Server (generator) → SDK (make generate-sdk) → CLI (commands) → Tests
```

Proto/gRPC is an optional fifth step, required only when adding a streaming or bidirectional endpoint.

---

## Phase 0 — Design

Before writing any code, update the data model spec.

**File:** `docs/internal/design/ambient-data-model.md`

1. Add the entity to the ERD with all fields, types, and FK relationships.
2. Define the REST endpoints (which of GET/POST/PATCH/DELETE the Kind supports).
3. Document request and response shapes in the API Reference section.

This document is the source of truth for what gets built. The generator output and OpenAPI spec must match it.

---

## Phase 1 — API Server

**Working directory:** `components/ambient-api-server/`

### Step 1 — Generate the plugin

```bash
go run ./scripts/generator.go \
  --kind MyKind \
  --fields "name:string:required,description:string,priority:int" \
  --project ambient-api-server \
  --repo github.com/ambient-code/platform/components \
  --library github.com/openshift-online/rh-trex-ai
```

Field type modifiers: `:required` (non-nullable, included in required array), `:optional` (nullable, default).
Supported types: `string`, `int`, `int64`, `bool`, `float`, `time`.

This creates:

```
plugins/myKind/
  model.go        # Gorm model + patch request struct
  dao.go          # Data access (Get, Create, Replace, Delete, FindByIDs, All)
  handler.go      # HTTP handlers (Create, Get, List, Patch, Delete)
  service.go      # Business logic + OnUpsert / OnDelete event handlers
  presenter.go    # OpenAPI ↔ model conversion
  migration.go    # Gormigrate migration with AutoMigrate
  plugin.go       # init() — registers routes, service, controller, presenter paths, migration
  mock_dao.go     # Mock DAO for unit tests
  *_test.go       # Integration test + factory

openapi/openapi.myKind.yaml   # Paths + schemas for MyKind
```

### Step 2 — Wire into the main OpenAPI spec

Edit `openapi/openapi.yaml` and add `$ref` entries for the new paths:

```yaml
/api/ambient/v1/my_kinds:
  $ref: 'openapi.myKind.yaml#/paths/~1api~1ambient~1v1~1my_kinds'
/api/ambient/v1/my_kinds/{id}:
  $ref: 'openapi.myKind.yaml#/paths/~1api~1ambient~1v1~1my_kinds~1{id}'
```

### Step 3 — Regenerate the OpenAPI Go client

```bash
make generate
```

This builds a container image using `Dockerfile.openapi`, runs the OpenAPI generator inside it, and copies the output to `pkg/api/openapi/`. Never edit files in `pkg/api/openapi/` by hand.

### Step 4 — Register the plugin

Add a side-effect import to `cmd/ambient-api-server/main.go`:

```go
_ "github.com/ambient-code/platform/components/ambient-api-server/plugins/myKind"
```

The `init()` in `plugin.go` registers routes, the service locator, the controller, presenter paths, and the migration automatically.

### Step 5 — Build and test

```bash
go build ./...
AMBIENT_ENV=integration_testing go test -p 1 -v ./...
```

Integration tests use `testcontainers-go` to spin up PostgreSQL — no external DB needed.

---

## Phase 1b — Proto / gRPC (streaming endpoints only)

Skip this phase unless the Kind needs a streaming or bidirectional gRPC endpoint (e.g., server-sent events via gRPC).

**Working directory:** `components/ambient-api-server/`

### Step 1 — Edit the proto definition

Add RPC methods and message types to the relevant file in `proto/ambient/v1/`:

```protobuf
rpc WatchMyKinds(WatchMyKindsRequest) returns (stream MyKindEvent) {}

message WatchMyKindsRequest {
  string project_id = 1;
}

message MyKindEvent {
  string type = 1;
  MyKind my_kind = 2;
}
```

### Step 2 — Regenerate the gRPC code

```bash
make proto
# runs: cd proto && buf generate
# reads buf.gen.yaml — remote plugins buf.build/protocolbuffers/go + buf.build/grpc/go
# output: pkg/api/grpc/ambient/v1/*.pb.go and *_grpc.pb.go
```

### Step 3 — Implement the gRPC handler

Add the server-side implementation in `plugins/sessions/grpc_handler.go` or create a new handler file in the plugin package.

---

## Phase 2 — SDK

**Working directory:** `components/ambient-sdk/`

No code changes are required. The SDK generator auto-discovers all `openapi/openapi.*.yaml` files in the API server's openapi directory.

```bash
make generate-sdk
```

This builds the generator binary and runs it against the API server's `openapi.yaml`. It produces:

| Output | Location |
|--------|----------|
| Go types + builders | `go-sdk/types/my_kind.go` |
| Go API client | `go-sdk/client/my_kind_api.go` (Create, Get, List, Update, Delete, ListAll) |
| Python client | `python-sdk/ambient_platform/_my_kind_api.py` |
| TypeScript client | `ts-sdk/src/my_kind_api.ts` |

Verify the output builds:

```bash
cd go-sdk && go build ./...
python -c "from ambient_platform import *"
```

---

## Phase 3 — CLI

**Working directory:** `components/ambient-cli/`

### Step 1 — Add command files

Create `cmd/acpctl/mykind/` following the pattern in `cmd/acpctl/session/`:

```
cmd/acpctl/mykind/
  mykind.go    # NewCmd() returning the parent cobra.Command
  list.go      # acpctl get mykinds
  get.go       # acpctl get mykind <id>
  create.go    # acpctl create mykind --name foo
  delete.go    # acpctl delete mykind <id>
```

### Step 2 — Register the subcommand

In `cmd/acpctl/root.go`:

```go
rootCmd.AddCommand(mykind.NewCmd())
```

### Step 3 — Build and smoke test

```bash
go build -o acpctl ./cmd/acpctl/

acpctl get mykinds
acpctl get mykind <id>
acpctl create mykind --name test
acpctl delete mykind <id>
```

---

## Phase 4 — Integration Tests

Run all three components against a live API server:

```bash
# Terminal 1 — start the API server (no auth, dev mode)
cd components/ambient-api-server
make run-no-auth

# Terminal 2 — run all tests
cd components/ambient-api-server
AMBIENT_ENV=integration_testing go test -p 1 -v ./...

cd components/ambient-sdk/go-sdk
go run examples/main.go

cd components/ambient-cli
./acpctl get mykinds
./acpctl create mykind --name test
```

All three must pass before the Kind is considered complete.

---

## Checklist

- [ ] ERD and API Reference updated in `ambient-data-model.md`
- [ ] Plugin generated: `plugins/myKind/` files present
- [ ] `openapi/openapi.myKind.yaml` generated
- [ ] `openapi/openapi.yaml` updated with `$ref` entries
- [ ] `make generate` ran, `pkg/api/openapi/` regenerated
- [ ] `cmd/ambient-api-server/main.go` import added
- [ ] `go build ./...` passes in `ambient-api-server`
- [ ] `go test ./...` passes in `ambient-api-server`
- [ ] Proto updated and `make proto` ran (if streaming)
- [ ] `make generate-sdk` ran, new types appear in all three SDKs
- [ ] CLI commands implemented and registered
- [ ] Full integration test pass (API + SDK + CLI)
