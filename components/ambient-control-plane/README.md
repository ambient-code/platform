# ambient-control-plane

Kubernetes-native controller for the Ambient Code Platform. Watches the `ambient-api-server` via gRPC streams and reconciles desired state into Kubernetes — the same relationship as `kube-controller-manager` to `kube-apiserver`.

## Architecture

```
ambient-api-server (REST + gRPC)
        │
        │ initial sync: SDK list calls (paginated)
        │ live updates: gRPC watch streams
        ▼
   ┌─────────┐
   │ Informer│──── cache + event synthesis ──→ ResourceEvent
   └─────────┘                                      │
        │                                           ▼
        │                              ┌──────────────────┐
        └──────────────────────────────│    Reconcilers    │
                                       └──────────────────┘
                                       Session | Project | ProjectSettings
```

| Kubernetes | ambient-control-plane |
|---|---|
| kube-apiserver | ambient-api-server |
| kube-controller-manager | ambient-control-plane |
| Watch streams (HTTP/2 chunked) | gRPC watch streams |
| CustomResource CRDs | Go SDK types |
| client-go informers | `internal/informer` package |
| controller reconcile loops | `internal/reconciler` package |

## Operating Modes

Three modes via `MODE` env var:

| Mode | Description | Dependencies |
|---|---|---|
| `kube` (default) | Reconciles into Kubernetes (CRs, Namespaces, RoleBindings) | K8s cluster |
| `local` | Spawns runner processes directly, AG-UI proxy | Filesystem only |
| `test` | Tally reconcilers, counts events, no side effects | None |

## Package Layout

```
cmd/ambient-control-plane/main.go        Entrypoint, signal handling, client setup
internal/config/config.go                Env-based configuration loading
internal/informer/informer.go            gRPC watch + list-sync engine with event synthesis
internal/watcher/watcher.go              gRPC stream management with reconnect backoff
internal/reconciler/reconciler.go        K8s reconcilers (Session, Project, ProjectSettings)
internal/reconciler/local_session.go     Local mode session reconciler (process spawning)
internal/reconciler/tally.go             Test mode tally reconciler
internal/kubeclient/kubeclient.go        Kubernetes dynamic client wrapper
internal/process/manager.go              Runner process lifecycle management
internal/proxy/agui_proxy.go             AG-UI reverse proxy for local mode
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `AMBIENT_API_SERVER_URL` | `http://localhost:8000` | Base URL of ambient-api-server |
| `AMBIENT_API_TOKEN` | **(required)** | Bearer token for API authentication |
| `AMBIENT_API_PROJECT` | `default` | Default project for SDK client |
| `AMBIENT_GRPC_SERVER_ADDR` | `localhost:8001` | gRPC server address |
| `AMBIENT_GRPC_USE_TLS` | `false` | Enable TLS for gRPC connection |
| `MODE` | `kube` | Operating mode: `kube`, `local`, or `test` |
| `LOG_LEVEL` | `info` | zerolog level (debug, info, warn, error) |
| `KUBECONFIG` | (empty) | Path to kubeconfig (kube mode) |
| `NAMESPACE` | `ambient-code` | Default K8s namespace (kube mode) |

### Local Mode Variables

| Variable | Default | Description |
|---|---|---|
| `LOCAL_WORKSPACE_ROOT` | `~/.ambient/workspaces` | Root directory for runner workspaces |
| `LOCAL_PROXY_ADDR` | `127.0.0.1:9080` | AG-UI proxy listen address |
| `CORS_ALLOWED_ORIGIN` | `http://localhost:3000` | CORS origin for AG-UI proxy |
| `LOCAL_RUNNER_COMMAND` | `python local_entry.py` | Command to run for each session |
| `LOCAL_PORT_RANGE` | `9100-9199` | Port range for runner processes |
| `LOCAL_MAX_SESSIONS` | `10` | Maximum concurrent sessions |

## Quick Start

```bash
make binary          # Build the binary
make run             # Build and run
make test            # Run tests with race detector
make lint            # gofmt -l + go vet
make fmt             # Auto-format
make tidy            # go mod tidy
```

## Reconcilers (kube mode)

| Reconciler | Resource | SDK Type | K8s Resources |
|---|---|---|---|
| `SessionReconciler` | `sessions` | `types.Session` | AgenticSession CRs |
| `ProjectReconciler` | `projects` | `types.Project` | Namespaces |
| `ProjectSettingsReconciler` | `project_settings` | `types.ProjectSettings` | RoleBindings |

## Key Design Decisions

- **Write-back echo detection**: When a reconciler writes status back to the API server, `UpdatedAt` is stored. On the next watch event, matching timestamps are skipped to prevent infinite update loops.
- **Buffered dispatch channel** (capacity 256): Watch handlers block on send (`dispatchBlocking`), ensuring no events are lost under backpressure.
- **Cache protected by `sync.RWMutex`**: Write lock held during cache mutations; released before dispatching to prevent deadlock.
- **Graceful shutdown**: `signal.NotifyContext(SIGINT, SIGTERM)` propagates cancellation to the informer loop and process manager.

## Known Limitations

- **List-then-watch gap**: Resources created between initial sync and gRPC stream establishment may be missed until the next watch event.
- **`any` type in events**: `ResourceEvent.Object` uses `any`, requiring type assertions in reconcilers. Generics would be more robust.
- **In-memory cache only**: Cache is rebuilt on each restart from a full initial sync.
- **Write-back echo is timestamp-based**: Relies on `UpdatedAt` microsecond equality. A resource-version approach would be more robust.
