# SDK Development Context

**When to load:** Working on `components/ambient-sdk/` — Go SDK, Python SDK, TypeScript SDK, or the generator

## Quick Reference

- **Generator:** `ambient-sdk/generator/` (Go binary) — reads `openapi.yaml`, writes SDK source
- **Go SDK:** `ambient-sdk/go-sdk/` — idiomatic Go client with iterators, watch streams
- **Python SDK:** `ambient-sdk/python-sdk/` — async-friendly, gRPC client included
- **TypeScript SDK:** `ambient-sdk/ts-sdk/` — typed fetch client for browser/Node
- **Source of truth:** `components/ambient-api-server/openapi/openapi.yaml`

## The Generation Pipeline

```
openapi/openapi.yaml  (edit this)
    │
    └── ambient-sdk/generator/main.go
            │
            ├── go-sdk/client/*.go       (generated)
            ├── go-sdk/types/*.go        (generated)
            ├── python-sdk/ambient_platform/*.py  (generated)
            └── ts-sdk/src/*.ts          (generated)
```

**Full regen (all SDKs):**
```bash
cd components/ambient-sdk
make generate          # runs generator against ../ambient-api-server/openapi/openapi.yaml
```

**Single SDK regen:**
```bash
cd components/ambient-sdk/generator
./generator --lang go     --spec ../../ambient-api-server/openapi/openapi.yaml --out ../go-sdk/
./generator --lang python --spec ../../ambient-api-server/openapi/openapi.yaml --out ../python-sdk/
./generator --lang ts     --spec ../../ambient-api-server/openapi/openapi.yaml --out ../ts-sdk/
```

## Go SDK

- **Client:** `go-sdk/client/client.go` — `NewClient(baseURL, token string)`
- **Resource APIs:** `*_api.go` per resource (sessions, projects, users, agents, roles, role_bindings)
- **Iterators:** `iterator.go` — `List()` returns lazy iterator, not full slice
- **Watch:** `session_watch.go`, `session_messages.go` — gRPC streaming watches
- **Types:** `go-sdk/types/*.go` — all model types (generated, do not edit)

**Usage pattern:**
```go
client := client.NewClient("http://localhost:13595", token)
sessions, err := client.Sessions.List(ctx, projectName, client.ListOptions{})
watch, err := client.Sessions.Watch(ctx, projectName, sessionName)
```

## Python SDK

- **Client:** `python-sdk/ambient_platform/client.py`
- **gRPC client:** `_grpc_client.py` — for watch streams
- **Session messages:** `_session_messages_api.py` — AG-UI event streaming
- **Tests:** `python-sdk/tests/`

```bash
cd components/ambient-sdk/python-sdk
uv venv && uv pip install -e .
python -m pytest tests/
```

## TypeScript SDK

- **Client:** `ts-sdk/src/client.ts`
- **Types:** one file per resource (e.g. `session.ts`, `project.ts`)
- **Tests:** `ts-sdk/tests/`

```bash
cd components/ambient-sdk/ts-sdk
npm install && npm test
```

## Adding a New Resource to the SDK

1. Add resource to `components/ambient-api-server/openapi/openapi.yaml`
2. Run `make generate` in `ambient-api-server` (updates Go server stubs)
3. Run `make generate` in `ambient-sdk` (updates all SDK clients)
4. Verify generated files compile: `cd go-sdk && go build ./...`
5. Write examples in `go-sdk/examples/` or `python-sdk/examples/`

## Generator Rules (Standing — Do Not Violate)

### `inferResourceName` — sub-spec schema placement

The generator's `inferResourceName` function scans each `openapi.*.yaml` sub-spec file and selects schemas alphabetically. The **first candidate alphabetically must use `allOf`** (i.e. be a primary resource schema extending `ObjectReference`). If the first candidate lacks `allOf`, the entire parse fails.

**Rule:** Auxiliary DTO schemas — request bodies, response envelopes, view models — that do **not** end in `List`, `PatchRequest`, or `StatusPatchRequest` must live in `openapi.yaml` main `components/schemas`, **not** in sub-spec `openapi.*.yaml` files.

**Examples of schemas that must go in `openapi.yaml`:**
- `IgniteRequest`, `IgniteResponse`
- `ProjectHome`, `ProjectHomeAgent`
- Any ad-hoc view model or request body

**Examples of schemas that belong in sub-spec files:**
- `ProjectAgent` (primary resource — has `allOf`)
- `ProjectAgentList`, `ProjectAgentPatchRequest` (filtered by suffix, safe)

Violating this rule causes a silent parse failure where the generator picks the wrong schema as the primary resource and generates incorrect client code.

## Generator Templates

Templates live in `generator/templates/<lang>/`. Each `.tmpl` file uses Go `text/template`:
- `client.go.tmpl` — top-level client struct with all resource APIs
- `types.go.tmpl` — model struct definitions from OpenAPI schemas
- `*_api.go.tmpl` — per-resource CRUD + list + watch methods

Edit templates to change generated code patterns — **do not edit generated files directly**.

## Pre-Commit Checklist

- [ ] `openapi.yaml` is the source of change — not generated files
- [ ] `make generate` run after any `openapi.yaml` change
- [ ] Go SDK: `cd go-sdk && go build ./... && go vet ./...`
- [ ] Python SDK: `python -m pytest tests/`
- [ ] TypeScript SDK: `npm test`
- [ ] Generator templates updated if new patterns needed
- [ ] Examples updated to reflect new capabilities
