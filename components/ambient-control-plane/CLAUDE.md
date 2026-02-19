# CLAUDE.md — ambient-control-plane

## Project Intent

The **ambient-control-plane** is a Go program that watches the ambient-api-server REST API for changes and reconciles desired state — exactly like kube-controller-manager watches kube-apiserver. It polls API endpoints, diffs against an in-memory cache, synthesizes ADDED/MODIFIED/DELETED events, and dispatches them to resource-specific reconcilers.

## Architecture

```
ambient-api-server (REST API)
        │
        │ poll (GET /api/ambient-api-server/v1/{resource})
        ▼
   ┌─────────┐
   │ Informer│──── diff against cache ──→ ResourceEvent
   └─────────┘                                  │
        │                                       ▼
        │                              ┌──────────────┐
        └──────────────────────────────│  Reconcilers │
                                       └──────────────┘
                                       Session | Workflow | Task
```

No Kubernetes dependency — pure HTTP client against the ambient-api-server OpenAPI.

## Quick Reference

```bash
make binary          # Build the binary
make run             # Build and run
make test            # Run tests with race detector
make lint            # gofmt -l + go vet
make fmt             # Auto-format
make tidy            # go mod tidy
```

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `AMBIENT_API_SERVER_URL` | `http://localhost:8000` | Base URL of ambient-api-server |
| `AMBIENT_API_TOKEN` | (empty) | Bearer token for API authentication |
| `POLL_INTERVAL_SECONDS` | `5` | Seconds between list/diff cycles |
| `WORKER_COUNT` | `2` | Number of reconciler workers |
| `LOG_LEVEL` | `info` | zerolog level (debug, info, warn, error) |

## Package Layout

```
cmd/ambient-control-plane/main.go   Entrypoint, signal handling, client setup
internal/config/config.go           Env-based configuration loading
internal/informer/informer.go       Poll-and-diff engine with event synthesis
internal/reconciler/reconciler.go   Reconciler interface + per-resource impls
```

## Key Patterns

- **Poll-and-diff informer**: No watch/SSE on the API server, so the informer polls list endpoints and compares `updated_at` timestamps against an in-memory cache to detect changes.
- **Event dispatch**: Handlers registered per resource string (e.g. `"sessions"`). The informer calls all handlers for a resource when an event fires.
- **Graceful shutdown**: `signal.NotifyContext(SIGINT, SIGTERM)` → context cancellation propagates to informer loop.
- **OpenAPI generated client**: Imports `ambient-api-server/pkg/api/openapi` via local `replace` directive in `go.mod`.

## Cross-Session Coordination

**Read `../working.md` at the start of every session.** This is the shared coordination document between ambient-api-server and ambient-control-plane Claude sessions. Tag entries with `[CP]`, update your status, and check for announcements/requests from the API side before starting work.

## Loadable Context (for Claude Code sessions)

| Topic | File |
|---|---|
| Cross-session coordination protocol | `../working.md` |
| API surface (endpoints, models, pagination, auth) | `docs/api-surface.md` |
| Architecture deep-dive (informer internals, reconciler contract, extension guide) | `docs/architecture.md` |

## Dependencies

- `github.com/ambient/platform/components/ambient-api-server` — OpenAPI-generated Go client (local replace)
- `github.com/rs/zerolog` — Structured logging

## Go Standards

- `go fmt ./...` enforced
- `go vet ./...` required
- Table-driven tests with subtests
- No `panic()` in production code
- No `interface{}` in new code — use generics or concrete types
