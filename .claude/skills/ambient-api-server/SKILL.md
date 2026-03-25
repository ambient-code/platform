# Skill: ambient-api-server

**Activates when:** Working in `components/ambient-api-server/` or asked about API server, REST endpoints, gRPC, OpenAPI, or plugins.

---

## What This Skill Knows

You are working on the **Ambient API Server** — a Go REST + gRPC server built on the `rh-trex-ai` framework. It serves the Ambient Platform's resource API under `/api/ambient/v1/`.

### Plugin System (critical)

Every resource kind is a self-contained plugin in `plugins/<kind>/`. The 8-file structure is mandatory:

```
plugins/<kind>/
  plugin.go      # route + gRPC registration
  model.go       # GORM DB struct
  handler.go     # HTTP handlers
  service.go     # business logic
  dao.go         # DB layer
  presenter.go   # model → API response
  migration.go   # DB schema
  mock_dao.go    # test mock
  *_test.go      # table-driven tests
```

Generate scaffold: `go run ./scripts/generator.go --kind Foo --fields "name:string,status:string"`

### OpenAPI is the source of truth

```
openapi/openapi.yaml
  → make generate → pkg/api/openapi/  (DO NOT EDIT)
  → ambient-sdk make generate → all SDKs
```

Never edit `pkg/api/openapi/` files — they are generated from `openapi/openapi.yaml`.

### gRPC

- Protos: `proto/ambient/v1/`
- Stubs: `pkg/api/grpc/ambient/v1/` (generated)
- Auth: `pkg/middleware/bearer_token_grpc.go`
- Watch streams: server-side streaming RPCs — emit AG-UI events to clients
- Ownership bypass: when JWT username not in context (test tokens), skip per-user ownership check

### Environments

| `AMBIENT_ENV` | Auth | DB | Use for |
|---|---|---|---|
| `development` | disabled | localhost:5432 | local `make run` |
| `integration_testing` | mock | testcontainer (ephemeral) | `make test` |
| `production` | JWT required | cluster Postgres | deployed |

### Test setup

```bash
systemctl --user start podman.socket
export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock
cd components/ambient-api-server && make test
```

---

## Runbook: Add a New API Resource

1. Define schema in `openapi/openapi.yaml`
2. Run `make generate` (updates Go stubs)
3. `go run ./scripts/generator.go --kind MyResource --fields "..."`
4. Implement `service.go` and `dao.go` business logic
5. Write table-driven tests in `*_test.go`
6. Run `make test` — all green
7. Run `make generate` in `ambient-sdk` to propagate to SDKs
8. Run `make build` in `ambient-cli` to update CLI

## Runbook: Add a gRPC Watch Stream

1. Add streaming RPC to `proto/ambient/v1/<resource>.proto`
2. `make generate` → updates stubs in `pkg/api/grpc/`
3. Implement handler in `plugins/<kind>/plugin.go` — register gRPC server
4. Implement fan-out: in-memory subscriber map keyed by session ID
5. Feed events from runner gRPC push → fan-out → all subscribers
6. Skip ownership check for non-JWT tokens
7. Test with: `acpctl session messages -f --project <p> <session>`

---

## Common Pitfalls

- **Never edit `pkg/api/openapi/`** — run `make generate` instead
- **`panic()` is forbidden** — use `errors.NewInternalServerError(...)`
- **DB migrations are additive** — never drop columns in production overlays
- **Test token ownership** — `WatchSessionMessages` must skip username check when JWT username is absent from context
