# CLAUDE.md — ambient-control-plane

## Project Intent

The **ambient-control-plane** is a Go program that watches the ambient-api-server via gRPC streams and reconciles desired state into Kubernetes — exactly like kube-controller-manager watches kube-apiserver. It performs an initial list-sync via the Go SDK, then subscribes to gRPC watch streams for real-time change events, dispatching them to resource-specific reconcilers.

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

Three operating modes via `MODE` env var:
- **`kube`** (default) — Reconciles into Kubernetes (CRs, Namespaces, RoleBindings)
- **`local`** — Spawns runner processes directly, no K8s dependency
- **`test`** — Tally reconcilers only, no side effects

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
| `AMBIENT_API_TOKEN` | **(required)** | Bearer token for API authentication |
| `AMBIENT_API_PROJECT` | `default` | Default project for SDK client |
| `AMBIENT_GRPC_SERVER_ADDR` | `localhost:8001` | gRPC server address |
| `AMBIENT_GRPC_USE_TLS` | `false` | Enable TLS for gRPC connection |
| `MODE` | `kube` | Operating mode: `kube`, `local`, or `test` |
| `LOG_LEVEL` | `info` | zerolog level (debug, info, warn, error) |
| `KUBECONFIG` | (empty) | Path to kubeconfig (kube mode) |
| `NAMESPACE` | `ambient-code` | Default K8s namespace (kube mode) |

#### Local Mode Variables

| Variable | Default | Description |
|---|---|---|
| `LOCAL_WORKSPACE_ROOT` | `~/.ambient/workspaces` | Root directory for runner workspaces |
| `LOCAL_PROXY_ADDR` | `127.0.0.1:9080` | AG-UI proxy listen address |
| `CORS_ALLOWED_ORIGIN` | `http://localhost:3000` | CORS origin for AG-UI proxy |
| `LOCAL_RUNNER_COMMAND` | `python local_entry.py` | Command to run for each session |
| `LOCAL_PORT_RANGE` | `9100-9199` | Port range for runner processes |
| `LOCAL_MAX_SESSIONS` | `10` | Maximum concurrent sessions |
| `BOSS_URL` | (empty) | Boss coordinator URL |
| `BOSS_SPACE` | `default` | Boss coordinator space |

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

## Key Patterns

- **gRPC watch + list-sync informer**: Initial sync via paginated SDK list calls, then gRPC watch streams for real-time updates. Async dispatch via buffered channel (256).
- **Event dispatch**: Handlers registered per resource string (e.g. `"sessions"`). The informer calls all handlers for a resource when an event fires.
- **Write-back echo detection**: `lastWritebackAt sync.Map` compares `UpdatedAt` timestamps to prevent infinite update loops when the control plane writes status back to the API server.
- **Graceful shutdown**: `signal.NotifyContext(SIGINT, SIGTERM)` → context cancellation propagates to informer loop and process manager.
- **Go SDK client**: Imports `ambient-sdk/go-sdk/client` for API access.

## Dependencies

- `github.com/ambient-code/platform/components/ambient-sdk/go-sdk` — Go SDK client
- `github.com/ambient/platform/components/ambient-api-server/pkg/api/grpc` — gRPC proto definitions
- `github.com/rs/zerolog` — Structured logging
- `k8s.io/client-go`, `k8s.io/apimachinery` — Kubernetes dynamic client (kube mode)
- `google.golang.org/grpc` — gRPC client

## Go Standards

- `go fmt ./...` enforced
- `go vet ./...` required
- Table-driven tests with subtests
- No `panic()` in production code
- No `interface{}` in new code — use generics or concrete types
