# Architecture Deep-Dive

## Design Philosophy

The ambient-control-plane follows the Kubernetes controller pattern adapted for a plain REST API:

| Kubernetes | ambient-control-plane |
|---|---|
| kube-apiserver | ambient-api-server |
| kube-controller-manager | ambient-control-plane |
| Watch streams (HTTP/2 chunked) | Poll-and-diff with in-memory cache |
| CustomResource CRDs | OpenAPI-generated model types |
| client-go informers | `internal/informer` package |
| controller reconcile loops | `internal/reconciler` package |

The key difference: ambient-api-server has no watch/SSE endpoint, so the informer synthesizes events by polling list endpoints and diffing against cached state.

## Component Responsibilities

### `cmd/ambient-control-plane/main.go`

Entry point. Responsibilities:

1. Load configuration from environment variables
2. Build the OpenAPI client with server URL, HTTP timeout, and optional bearer token
3. Create the informer with the configured poll interval
4. Instantiate and register all reconcilers
5. Start the informer loop
6. Handle graceful shutdown via `signal.NotifyContext(SIGINT, SIGTERM)`

The `run()` function returns an error, and `main()` prints it and exits non-zero. When the context is cancelled (shutdown signal), `run()` returns `nil` — this is intentional so a clean shutdown is not treated as an error.

```go
err = inf.Run(ctx)
if err != nil && ctx.Err() != nil {
    logger.Info().Msg("shutdown complete")
    return nil
}
return err
```

### `internal/config/config.go`

Pure environment-variable-based configuration. No config files, no flags. Returns a `ControlPlaneConfig` struct or an error if any env var has an invalid value.

| Field | Env Var | Type | Default |
|---|---|---|---|
| `APIServerURL` | `AMBIENT_API_SERVER_URL` | string | `http://localhost:8000` |
| `APIToken` | `AMBIENT_API_TOKEN` | string | (empty) |
| `PollInterval` | `POLL_INTERVAL_SECONDS` | duration | 5s |
| `WorkerCount` | `WORKER_COUNT` | int | 2 |
| `LogLevel` | `LOG_LEVEL` | string | `info` |

### `internal/informer/informer.go`

The informer is the core engine. It maintains three independent in-memory caches (one per resource type) and a handler registry.

#### Data Structures

```go
type Informer struct {
    client        *openapi.APIClient
    pollInterval  time.Duration
    handlers      map[string][]EventHandler      // resource → handlers
    mu            sync.RWMutex                    // protects handlers map
    logger        zerolog.Logger
    sessionCache  map[string]openapi.Session      // id → last-known state
    workflowCache map[string]openapi.Workflow
    taskCache     map[string]openapi.Task
}
```

#### Event Types

```go
type ResourceEvent struct {
    Type      EventType      // ADDED | MODIFIED | DELETED
    Resource  string         // "sessions" | "workflows" | "tasks"
    Object    interface{}    // current state (typed as openapi.Session etc.)
    OldObject interface{}    // previous state (only set for MODIFIED)
}
```

#### Poll-and-Diff Algorithm

Each sync cycle (per resource type) follows this algorithm:

```
1. GET /api/ambient-api-server/v1/{resource} → list all items
2. For each item in the API response:
   a. If item.id NOT in cache → ADDED event, add to cache
   b. If item.id in cache AND item.updated_at != cached.updated_at → MODIFIED event, update cache
   c. If item.id in cache AND timestamps match → no event (unchanged)
3. For each item in cache NOT in the API response → DELETED event, remove from cache
```

This approach correctly handles:
- New resources appearing between polls
- Resources being updated (detected via `updated_at` timestamp)
- Resources being deleted from the API server
- First sync populates the cache and fires ADDED events for all existing resources

#### Sync Order

Resources sync in a fixed order: sessions → workflows → tasks. If any sync fails, the remaining resource types are skipped for that cycle. The next cycle retries all.

#### Handler Dispatch

Handlers are called synchronously in registration order. A failing handler is logged but does not prevent other handlers from executing or block future events.

```go
func (inf *Informer) dispatch(ctx context.Context, event ResourceEvent) {
    inf.mu.RLock()
    handlers := inf.handlers[event.Resource]
    inf.mu.RUnlock()
    for _, handler := range handlers {
        if err := handler(ctx, event); err != nil {
            inf.logger.Error().Err(err).Str("resource", event.Resource).Msg("handler failed")
        }
    }
}
```

### `internal/reconciler/reconciler.go`

#### Reconciler Interface

```go
type Reconciler interface {
    Resource() string
    Reconcile(ctx context.Context, event informer.ResourceEvent) error
}
```

`Resource()` returns the resource string that matches the informer's handler registration (e.g. `"sessions"`). `Reconcile()` receives every event for that resource.

#### Current Implementations

Three reconcilers exist as skeleton implementations:

| Reconciler | Resource | OpenAPI Type |
|---|---|---|
| `SessionReconciler` | `"sessions"` | `openapi.Session` |
| `WorkflowReconciler` | `"workflows"` | `openapi.Workflow` |
| `TaskReconciler` | `"tasks"` | `openapi.Task` |

Each reconciler:
1. Type-asserts `event.Object` to the correct OpenAPI model
2. Switches on `event.Type` (ADDED/MODIFIED/DELETED)
3. Logs the event with resource ID and name
4. Returns `nil` (no business logic yet)

#### Registration Pattern

In `main.go`, reconcilers are instantiated and registered via a helper:

```go
func registerReconciler(inf *informer.Informer, rec reconciler.Reconciler) {
    inf.RegisterHandler(rec.Resource(), rec.Reconcile)
}
```

## Adding a New Resource Reconciler

To watch a new API resource (e.g. Agents):

1. **Add a cache field** to `Informer` struct in `informer.go`:
   ```go
   agentCache map[string]openapi.Agent
   ```

2. **Initialize the cache** in `New()`:
   ```go
   agentCache: make(map[string]openapi.Agent),
   ```

3. **Add a sync method** following the `syncSessions` pattern — list from API, diff against cache, dispatch events.

4. **Call the sync method** from `syncAll()`.

5. **Create a reconciler** in `reconciler.go`:
   ```go
   type AgentReconciler struct { ... }
   func (r *AgentReconciler) Resource() string { return "agents" }
   func (r *AgentReconciler) Reconcile(ctx context.Context, event informer.ResourceEvent) error { ... }
   ```

6. **Register in main.go**:
   ```go
   agentReconciler := reconciler.NewAgentReconciler(apiClient, logger)
   registerReconciler(inf, agentReconciler)
   ```

## Known Limitations

- **No pagination in sync**: The informer calls list endpoints without pagination parameters, relying on the default page size (100). Resources beyond page 1 are not synced.
- **No watch/SSE**: Changes are detected only at poll boundaries. Latency = 0 to `POLL_INTERVAL_SECONDS`.
- **Sequential sync**: Resource types sync sequentially within a cycle. A slow or failing API call delays all subsequent syncs.
- **In-memory cache only**: Cache is lost on restart. The first sync after restart fires ADDED events for all existing resources.
- **No retry/backoff**: Failed sync cycles are logged but not retried with exponential backoff.
- **`interface{}` in events**: `ResourceEvent.Object` is `interface{}`, requiring type assertions in reconcilers. Future improvement: use generics.
- **WorkerCount unused**: The `WorkerCount` config field is loaded but not yet used — reconcilers run synchronously in the informer goroutine.

## Concurrency Model

Currently single-goroutine: the informer's `Run()` loop does all polling, diffing, and handler dispatch in one goroutine. The `sync.RWMutex` on the handler map exists to allow safe handler registration before `Run()` is called, but during operation there is no concurrent access.

Future improvement: use a work queue with `WorkerCount` goroutines consuming events, decoupling poll speed from reconciliation latency.
