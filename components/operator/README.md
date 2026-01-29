# Agentic Operator

Kubernetes operator watching Custom Resources and managing AgenticSession Job lifecycle.

## Features

- **Controller-runtime based** - Uses work queues with rate limiting for scalable processing
- **Concurrent reconciliation** - Processes multiple sessions in parallel (configurable)
- **Event deduplication** - Multiple rapid events are coalesced into single reconciles
- **Automatic retries** - Failed reconciles are requeued with exponential backoff
- Watches AgenticSession CRs and spawns Jobs with runner pods
- Updates CR status based on Job completion
- Handles timeout and cleanup
- Idempotent reconciliation

## Configuration

### Command Line Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--max-concurrent-reconciles` | 10 | Maximum parallel session reconciliations |
| `--metrics-bind-address` | :8080 | Prometheus metrics endpoint |
| `--health-probe-bind-address` | :8081 | Health/readiness probe endpoint |
| `--leader-elect` | false | Enable leader election for HA |
| `--legacy-watch` | false | Use old watch-based implementation (debugging) |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MAX_CONCURRENT_RECONCILES` | 10 | Override max concurrent reconciles |
| `DEV_MODE` | false | Enable development logging |
| `NAMESPACE` | default | Operator namespace |
| `BACKEND_NAMESPACE` | (same as NAMESPACE) | Backend API namespace |
| `AMBIENT_CODE_RUNNER_IMAGE` | quay.io/ambient_code/vteam_claude_runner:latest | Runner image |

### Performance Tuning

For high-throughput environments:

```yaml
args:
  - --max-concurrent-reconciles=20  # Increase parallelism
  - --leader-elect=false
```

For HA deployments:

```yaml
spec:
  replicas: 2
  template:
    spec:
      containers:
      - args:
        - --leader-elect=true  # Only one active controller
```

## Development

### Prerequisites

- Go 1.24+
- kubectl
- Kubernetes cluster access
- CRDs installed in cluster

### Quick Start

```bash
cd components/operator

# Build
go build -o operator .

# Run locally (requires k8s access and CRDs installed)
go run .

# Run with legacy watch mode (for debugging)
go run . --legacy-watch
```

### Build

```bash
# Build binary
go build -o operator .

# Build container image
docker build -t operator .
# or
podman build -t operator .
```

### Testing

```bash
# Run tests
go test ./... -v

# Run tests with coverage
go test ./... -v -cover
```

### Linting

```bash
# Format code
gofmt -l .

# Run go vet
go vet ./...

# Run golangci-lint
golangci-lint run
```

**Pre-commit checklist**:
```bash
# Run all linting checks
gofmt -l .             # Should output nothing
go vet ./...
golangci-lint run

# Auto-format code
gofmt -w .
```

## Architecture

### Package Structure

```
operator/
├── internal/
│   ├── config/        # K8s client init, config loading
│   ├── controller/    # Controller-runtime reconcilers (NEW)
│   │   ├── agenticsession_controller.go  # Main reconciler with work queue
│   │   └── reconcile_phases.go           # Phase-specific reconciliation logic
│   ├── types/         # GVR definitions, resource helpers
│   ├── handlers/      # Handler logic called from controllers
│   │   ├── sessions.go      # Session management logic
│   │   ├── reconciler.go    # Exported functions for controller
│   │   ├── namespaces.go    # Namespace watcher
│   │   └── projectsettings.go  # ProjectSettings watcher
│   └── services/      # Reusable services (PVC provisioning, etc.)
└── main.go            # Manager setup and controller registration
```

### Controller-Runtime Benefits

The operator uses [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) which provides:

1. **Work Queue** - Events are added to a queue and processed asynchronously
2. **Rate Limiting** - Exponential backoff prevents API server overload
3. **Deduplication** - Multiple rapid events = single reconcile
4. **Concurrency** - Multiple reconcilers process sessions in parallel
5. **Predicates** - Filter events to reduce unnecessary reconciles

### Key Patterns

See `CLAUDE.md` in project root for:
- Reconciliation pattern with Result and error handling
- Status updates (UpdateStatus subresource)
- Error handling and requeuing
- Phase-based state machine

## Prometheus Metrics

The operator exposes metrics at `:8080/metrics` for monitoring session lifecycle and performance.

### Available Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `ambient_session_startup_duration_seconds` | Histogram | `namespace` | Time from Pending to Running |
| `ambient_sessions_total` | Counter | `namespace` | Total sessions created |
| `ambient_sessions_completed_total` | Counter | `namespace`, `final_phase` | Sessions reaching terminal states (Stopped, Failed, Completed) |
| `ambient_session_phase_transitions_total` | Counter | `namespace`, `from_phase`, `to_phase` | Phase transition counts |
| `ambient_reconcile_duration_seconds` | Histogram | `phase`, `success` | Reconcile loop timing |
| `ambient_pod_creation_duration_seconds` | Histogram | `namespace` | Pod creation timing |
| `ambient_token_provision_duration_seconds` | Histogram | `namespace` | Runner token provisioning time |
| `ambient_session_errors_total` | Counter | `namespace`, `phase`, `error_type` | Error tracking |

### Example PromQL Queries

**95th percentile startup time:**
```promql
histogram_quantile(0.95, sum(rate(ambient_session_startup_duration_seconds_bucket[5m])) by (le, namespace))
```

**Average startup time:**
```promql
sum(rate(ambient_session_startup_duration_seconds_sum[5m])) / sum(rate(ambient_session_startup_duration_seconds_count[5m]))
```

**Sessions started per hour:**
```promql
sum(increase(ambient_sessions_total[1h])) by (namespace)
```

**Phase transitions per minute:**
```promql
sum(rate(ambient_session_phase_transitions_total[1m])) by (from_phase, to_phase)
```

**Error rate:**
```promql
sum(rate(ambient_session_errors_total[5m])) by (phase, error_type)
```

**Reconcile success rate:**
```promql
sum(rate(ambient_reconcile_duration_seconds_count{success="true"}[5m])) / sum(rate(ambient_reconcile_duration_seconds_count[5m]))
```

### OpenShift User Workload Monitoring

To enable metrics scraping in OpenShift:

1. Enable user workload monitoring (done once per cluster):
```bash
oc -n openshift-monitoring edit configmap cluster-monitoring-config
# Add: enableUserWorkload: true
```

2. Apply the ServiceMonitor (included in manifests):
```bash
oc apply -f components/manifests/base/operator-metrics-service.yaml
```

3. Access metrics in OpenShift Console → Observe → Metrics

## Reference Files

- `internal/controller/agenticsession_controller.go` - Main reconciler
- `internal/controller/reconcile_phases.go` - Phase handlers
- `internal/controller/metrics.go` - Prometheus metric definitions
- `internal/handlers/reconciler.go` - Exported handler functions
- `internal/handlers/sessions.go` - Core session management logic
- `internal/config/config.go` - K8s client initialization
- `internal/types/resources.go` - GVR definitions
