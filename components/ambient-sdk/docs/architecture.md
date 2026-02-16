# SDK Architecture

## Design Philosophy

The Ambient Platform SDK is an **HTTP-first, zero-Kubernetes** client library. It deliberately hides the platform's Kubernetes internals behind simple REST semantics so that consumers never need `kubectl`, `client-go`, or cluster credentials.

### Core Principles

1. **Pure HTTP** — Standard REST calls over HTTPS. No CRD watchers, no informers, no leader election.
2. **Minimal Dependencies** — Go SDK uses only the standard library. Python SDK uses only `httpx`.
3. **Type Safety** — Strongly-typed request/response structures in both languages with compile-time (Go) and runtime (Python) validation.
4. **Secure by Default** — Token validation on construction, automatic log redaction, sanitized error surfaces.
5. **API-First** — The API server's `openapi.yaml` is the single source of truth. SDK types derive from it.

## Platform Integration

```
                    ┌──────────────┐
                    │   Frontend   │
                    │   (NextJS)   │
                    └──────┬───────┘
                           │
┌──────────────┐    ┌──────▼───────┐    ┌───────────────┐    ┌────────────┐
│  ambient-sdk │───►│  API Server  │───►│ Control Plane │───►│  Operator  │
│  (Go/Python) │    │  (Go + Gin)  │    │ (Reconciler)  │    │ (K8s Ctrl) │
└──────────────┘    └──────────────┘    └───────────────┘    └─────┬──────┘
                                                                   │
                                                             ┌─────▼──────┐
                                                             │   Runner   │
                                                             │ (Claude CLI)│
                                                             └────────────┘
```

### Data Flow

1. **SDK** sends `POST /v1/sessions` with task, model, and repos to the API server.
2. **API Server** creates an `AgenticSession` Custom Resource in the target namespace.
3. **Control Plane** detects the new CR via polling (`GET /api/ambient-api-server/v1/sessions`).
4. **Operator** watches CRs and spawns a Kubernetes Job.
5. **Runner** pod executes Claude Code CLI, writes results back to the CR status.
6. **SDK** polls `GET /v1/sessions/{id}` until status is `completed` or `failed`.

### Contract Boundary

The SDK's contract is defined entirely by the API server's OpenAPI spec (`../ambient-api-server/openapi/`). Everything below the API server is opaque:

| Visible to SDK | Hidden from SDK |
|---|---|
| `/v1/sessions` endpoints | AgenticSession CRD schema |
| Bearer token + project header | Kubernetes RBAC policies |
| Session status lifecycle | Job scheduling, pod creation |
| JSON request/response shapes | CR spec/status fields |

## SDK Structure

### Go SDK (`go-sdk/`)

```
client/client.go   — HTTP client, request execution, log sanitization
types/types.go     — API types, SecureToken, input validators
examples/main.go   — Working lifecycle demo
```

The Go client is a single struct wrapping `*http.Client` with:
- `SecureToken` for type-safe, log-safe token handling
- `slog`-based structured logging with `ReplaceAttr` sanitizer
- Context-aware methods (`CreateSession`, `GetSession`, `ListSessions`, `WaitForCompletion`)

### Python SDK (`python-sdk/`)

```
ambient_platform/client.py      — AmbientClient with httpx
ambient_platform/types.py       — Dataclasses matching OpenAPI
ambient_platform/exceptions.py  — Typed exception hierarchy
examples/main.py                — Working lifecycle demo
```

The Python client is a class wrapping `httpx.Client` with:
- Input validation on construction (`_validate_token`, `_validate_project`, `_validate_base_url`)
- `from_env()` factory for environment-based setup
- Context manager support for automatic resource cleanup
- Structured exceptions: `AmbientAPIError` → `AuthenticationError`, `SessionNotFoundError`, `AmbientConnectionError`

## Session Lifecycle

```
  POST /v1/sessions
        │
        ▼
   ┌─────────┐     ┌─────────┐     ┌───────────┐
   │ pending  │────►│ running │────►│ completed │
   └─────────┘     └────┬────┘     └───────────┘
                        │
                        ▼
                   ┌─────────┐
                   │ failed  │
                   └─────────┘
```

- **pending**: CR created, waiting for operator to schedule a Job
- **running**: Job pod is executing the Claude Code CLI
- **completed**: Task finished successfully; `result` field populated
- **failed**: Task failed; `error` field populated

## Cross-Component Coordination

The file `../working.md` serves as a coordination protocol between Claude sessions working on different components. Key rules:

- Read before writing
- Append, don't overwrite
- Tag entries with `[API]` or `[CP]`
- Contracts section defines the agreed API surface

The SDK depends on the contracts in `working.md` — particularly the session list endpoint and authentication scheme.

## Future Roadmap

| Phase | Status | Description |
|---|---|---|
| Phase 1: HTTP-Only Go + Python | Done | Core session CRUD with polling |
| Phase 2: TypeScript SDK | Planned | Generated types from OpenAPI, React Query integration |
| Phase 3: Advanced Features | Planned | OpenTelemetry instrumentation, SDK-based testing utilities |
