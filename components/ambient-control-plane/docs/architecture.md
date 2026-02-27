# Architecture Deep-Dive

## Design Philosophy

The ambient-control-plane follows the Kubernetes controller pattern, bridging the ambient-api-server with Kubernetes:

| Kubernetes | ambient-control-plane |
|---|---|
| kube-apiserver | ambient-api-server |
| kube-controller-manager | ambient-control-plane |
| Watch streams (HTTP/2 chunked) | gRPC watch streams |
| CustomResource CRDs | Go SDK types |
| client-go informers | `internal/informer` package |
| controller reconcile loops | `internal/reconciler` package |

## Operating Modes

The control plane supports three modes via the `MODE` environment variable:

| Mode | Description | Dependencies |
|---|---|---|
| `kube` (default) | Reconciles into Kubernetes (CRs, Namespaces, RoleBindings) | K8s cluster |
| `local` | Spawns runner processes directly, AG-UI proxy | Filesystem only |
| `test` | Tally reconcilers, counts events, no side effects | None |

## Component Responsibilities

### `cmd/ambient-control-plane/main.go`

Entry point. Responsibilities:

1. Load configuration from environment variables (validates `AMBIENT_API_TOKEN`)
2. Build the Go SDK client with server URL, token, and project
3. Establish gRPC connection (with optional TLS)
4. Create the informer with watch manager
5. Instantiate and register mode-specific reconcilers
6. Start the informer run loop
7. Handle graceful shutdown via `signal.NotifyContext(SIGINT, SIGTERM)`

### `internal/config/config.go`

Pure environment-variable-based configuration. No config files, no flags. Returns a `ControlPlaneConfig` struct or an error if `AMBIENT_API_TOKEN` is missing.

See CLAUDE.md for the full environment variable reference.

### `internal/watcher/watcher.go`

Manages gRPC watch streams with automatic reconnection. Each resource type gets its own goroutine running a `watchLoop` that:

1. Opens a gRPC watch stream
2. Receives events and dispatches to registered handlers
3. On stream error, applies exponential backoff (capped at 30s) and reconnects

### `internal/informer/informer.go`

The informer is the core engine. It maintains three in-memory caches (sessions, projects, project_settings) and a handler registry.

#### Startup Sequence

```
1. Initial sync: paginated SDK list calls populate caches, fire ADDED events (blocking dispatch)
2. Start dispatch loop goroutine (reads from buffered event channel)
3. Wire gRPC watch handlers
4. Start gRPC watch streams (one goroutine per resource)
```

#### Event Flow

```
gRPC watch event → handleXxxWatch() → cache update (under mutex) → dispatchBlocking → event channel → dispatchLoop → handlers
```

All watch handlers use `dispatchBlocking` to ensure no events are lost under backpressure.

#### Cache Consistency

Watch handlers hold `inf.mu.Lock()` during cache mutations and build the event while holding the lock. The lock is released before dispatching to prevent deadlock (dispatch → handler → RLock would deadlock with Lock).

### `internal/reconciler/reconciler.go`

#### Reconciler Interface

```go
type Reconciler interface {
    Resource() string
    Reconcile(ctx context.Context, event informer.ResourceEvent) error
}
```

#### Implementations (kube mode)

| Reconciler | Resource | SDK Type | K8s Resources |
|---|---|---|---|
| `SessionReconciler` | `sessions` | `types.Session` | AgenticSession CRs |
| `ProjectReconciler` | `projects` | `types.Project` | Namespaces |
| `ProjectSettingsReconciler` | `project_settings` | `types.ProjectSettings` | RoleBindings |

#### Write-Back Echo Detection

When a reconciler writes status back to the API server, the response's `UpdatedAt` timestamp is stored. On the next watch event, if `session.UpdatedAt` matches the stored timestamp, the event is skipped to prevent infinite update loops.

### `internal/kubeclient/kubeclient.go`

Kubernetes dynamic client wrapper. Provides typed methods for AgenticSession CRDs, Namespaces, and RoleBindings using `k8s.io/client-go/dynamic`.

### `internal/process/manager.go` (local mode)

Runner process lifecycle management:
- Port pool allocation with availability checking
- Workspace directory creation per session
- Environment variable allowlisting (security: only approved vars inherited)
- Process group management (`Setpgid` + kill `-pgid`)
- SIGTERM → SIGKILL escalation with configurable grace period
- Stderr ring buffer for last N lines

### `internal/proxy/agui_proxy.go` (local mode)

Reverse proxy routing AG-UI protocol requests to runner processes by session ID. Supports SSE streaming, health checks, and standard HTTP proxying with configurable CORS.

## Known Limitations

- **List-then-watch gap**: Resources created between initial sync completing and gRPC streams establishing may be missed until a subsequent watch event.
- **`any` type in events**: `ResourceEvent.Object` and `WatchEvent.Object` use `any`, requiring type assertions in reconcilers. Future improvement: use generics.
- **In-memory cache only**: Cache is lost on restart. The first sync after restart fires ADDED events for all existing resources.
- **Write-back echo is timestamp-based**: Relies on `UpdatedAt` microsecond truncation equality. A resource-version approach would be more robust.

## Concurrency Model

The informer uses a multi-goroutine architecture:
- One goroutine per gRPC watch stream (3 total: sessions, projects, project_settings)
- One dispatch loop goroutine consuming from a buffered channel (capacity 256)
- Watch handlers block on channel send (`dispatchBlocking`) ensuring no event loss
- Cache access protected by `sync.RWMutex` (write lock for mutations, read lock for handler dispatch)
