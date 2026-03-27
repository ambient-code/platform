# SDK Development Context

**When to load:** Working on `components/ambient-sdk/` — Go SDK, Python SDK, TypeScript SDK, or the generator

## Quick Reference

- **Generator:** `ambient-sdk/generator/` (Go binary) — reads `openapi.yaml`, writes SDK source
- **Go SDK:** `ambient-sdk/go-sdk/` — idiomatic Go client with iterators, watch streams
- **Python SDK:** `ambient-sdk/python-sdk/` — async-friendly, gRPC client included
- **TypeScript SDK:** `ambient-sdk/ts-sdk/` — typed fetch client for browser/Node
- **Source of truth:** `components/ambient-api-server/openapi/openapi.yaml`

---

## The Generation Pipeline

```
openapi/openapi.yaml  (edit this)
    │
    └── ambient-sdk/generator/main.go
            │
            ├── go-sdk/client/*.go       (generated — do not edit)
            ├── go-sdk/types/*.go        (generated — do not edit)
            ├── python-sdk/ambient_platform/*.py  (generated — do not edit)
            └── ts-sdk/src/*.ts          (generated — do not edit)
```

**Full regen (all SDKs):**
```bash
cd components/ambient-sdk
make generate-sdk      # runs generator against ../ambient-api-server/openapi/openapi.yaml
```

**Verify all SDKs compile after generation:**
```bash
cd components/ambient-sdk
make verify-sdk        # generate-sdk + compile-check all three outputs
```

Always run `make generate` in `ambient-api-server` first — the SDK generator reads the merged `openapi.yaml` output from that step.

---

## Go SDK

- **Client:** `go-sdk/client/client.go` — `NewClient(baseURL, token string)`
- **Resource APIs:** `*_api.go` per resource — CRUD + list + watch
- **Iterators:** `iterator.go` — `List()` returns lazy iterator, not full slice
- **Watch / streams:** `session_watch.go`, `session_messages.go` — gRPC streaming

**Usage pattern:**
```go
client := client.NewClient("http://localhost:13595", token)
sessions, err := client.Sessions().List(ctx, listOptions)
msgs, err := client.Sessions().WatchMessages(ctx, sessionID, afterSeq)
```

### Extension files for nested resources

The generator uses the **first path segment** of a resource's routes as the base path for all generated client methods. For resources nested under `/projects/{id}/agents/...`, the generator emits `/projects` as the base path — wrong for all nested operations.

**Fix:** write hand-crafted extension files that add the correct nested methods:

- `go-sdk/client/agent_extensions.go` — non-CRUD methods (`GetInProject`, `ListInboxInProject`, `SendInboxInProject`, `Ignite`, etc.)
- These live alongside generated files but are never overwritten by the generator

**Rule:** Any method that uses a nested URL must live in an `*_extensions.go` file, not in generated code. Before implementing a new CLI command that calls a nested API endpoint, check whether the extension method exists. If not, add it first.

```go
// agent_extensions.go — example pattern
func (a *AgentAPI) GetInProject(ctx context.Context, projectID, agentID string) (*types.Agent, error) {
    path := fmt.Sprintf("/api/ambient/v1/%s/agents/%s",
        url.PathEscape(projectID), url.PathEscape(agentID))
    return a.client.doGet(ctx, path, &types.Agent{})
}
```

**URL encoding rule:** All nested resource URLs must use `url.PathEscape` (Go) / `encodeURIComponent` (TS) on every path segment. Not just the leaf — every segment.

### SSE / streaming endpoints

The SDK's `do()` and `doMultiStatus()` methods unmarshal the response body and close the connection. For SSE streams, you need the body open and streaming.

SSE endpoints use a separate pattern — return `io.ReadCloser`, caller closes it:

```go
// session_messages.go
func (a *SessionAPI) StreamEvents(ctx context.Context, sessionID string) (io.ReadCloser, error) {
    path := fmt.Sprintf("/api/ambient/v1/sessions/%s/events", url.PathEscape(sessionID))
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.client.baseURL+path, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Authorization", "Bearer "+a.client.token)
    req.Header.Set("Accept", "text/event-stream")
    resp, err := a.client.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    if resp.StatusCode != http.StatusOK {
        resp.Body.Close()
        return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }
    return resp.Body, nil
}
```

`StreamEvents` needs access to `a.client.baseURL`, `a.client.token`, and `a.client.httpClient` — all unexported. Since extension files are in the same `client` package, this works without accessors. Do not try to fit SSE into `do()`.

---

## Python SDK

- **Client:** `python-sdk/ambient_platform/client.py`
- **gRPC client:** `_grpc_client.py` — for watch streams
- **Session messages:** `_session_messages_api.py` — AG-UI event streaming
- **Tests:** `python-sdk/tests/`

**Nested resource pitfall (Python):** Same issue as Go — generator emits wrong base path. Python extension classes live alongside generated files. Follow the same pattern: subclass or extend the generated class with hand-written methods for nested routes.

```bash
cd components/ambient-sdk/python-sdk
uv venv && uv pip install -e .
python -m pytest tests/
```

---

## TypeScript SDK

- **Client:** `ts-sdk/src/client.ts`
- **Types:** one file per resource (e.g. `session.ts`, `project.ts`)
- **Tests:** `ts-sdk/tests/`

**Nested resource pitfall (TS):** Same issue — complete rewrites required for nested resources:
- `ts-sdk/src/project_agent_api.ts` — complete rewrite with correct nested paths
- `ts-sdk/src/inbox_message_api.ts` — complete rewrite

After fixing the generator to handle nested paths correctly, delete these extension files and re-verify.

**URL encoding rule (TS):** Use `encodeURIComponent` on every path segment.

```bash
cd components/ambient-sdk/ts-sdk
npm install && npm test
```

---

## Generator `inferResourceName` Rule

The generator's `inferResourceName` function scans each `openapi.*.yaml` sub-spec file and selects schemas alphabetically. The **first candidate alphabetically must use `allOf`** (primary resource schema). If the first candidate lacks `allOf`, the entire parse fails silently — it picks the wrong schema and generates incorrect client code with no error message.

**Schemas that break the generator if put in sub-spec files:**
- `IgniteRequest`, `IgniteResponse` — alphabetically before `InboxMessage` (the primary resource)
- Any view model or request DTO that sorts before the primary resource name

**Prevention:** Keep all auxiliary schemas in `openapi.yaml` main `components/schemas`. See `api-server-development.md` for the full rule.

---

## Generator Templates

Templates live in `generator/templates/<lang>/`. Each `.tmpl` file uses Go `text/template`:
- `client.go.tmpl` — top-level client struct with all resource APIs
- `types.go.tmpl` — model struct definitions from OpenAPI schemas
- `*_api.go.tmpl` — per-resource CRUD + list + watch methods

Edit templates to change generated code patterns — **do not edit generated files directly**.

**Generator directory naming:** Generator creates directory named `{kindLowerPlural}`. For `InboxMessage` → `inboxMessages`. If the desired package name differs (e.g. `inbox`), copy and rename manually.

---

## Build Commands

```bash
cd components/ambient-sdk

make build-generator   # Compile the SDK generator binary
make generate-sdk      # Run generator → Go + Python + TypeScript SDKs
make verify-sdk        # generate-sdk + compile-check all three outputs
```

---

## Pre-Commit Checklist

- [ ] `openapi.yaml` is the source of change — not generated files
- [ ] `make generate` run in api-server first (updates merged openapi.yaml)
- [ ] `make generate-sdk` run in ambient-sdk
- [ ] Go SDK: `cd go-sdk && go build ./... && go vet ./...`
- [ ] Python SDK: `python -m pytest tests/`
- [ ] TypeScript SDK: `npm test`
- [ ] Nested resource methods in `*_extensions.go`, not generated files
- [ ] URL encoding: `url.PathEscape` (Go) / `encodeURIComponent` (TS) on all segments
- [ ] SSE endpoints return `io.ReadCloser`, not unmarshaled result
- [ ] Generator templates updated if new patterns needed
- [ ] Examples updated to reflect new capabilities
