# ADR-0008: Pluggable Execution Environments for Agentic Sessions

**Status:** Proposed
**Date:** 2026-04-24
**Deciders:** Platform Architecture Team
**Technical Story:** Enable e2e testing workflows that require nested containers or full OS environments (KubeVirt, kubernetes-mcp-server eval testing)

## Context and Problem Statement

ACP's execution model is "one Kubernetes Pod per session" (ADR-0001, Agent Runtime Registry design). The registry controls what goes into the pod, and the operator creates pods directly via the K8s API. This model is sufficient for AI agent sessions that need a sandboxed filesystem and network access.

However, certain workloads require capabilities that are unavailable in a non-root, capability-dropped pod:

- **Nested container runtimes** (Docker/Podman) for creating kind clusters
- **Full OS environments** for installing system packages, running systemd services
- **Nested virtualization** for KubeVirt e2e testing against real VMs

A concrete example: the [kubevirt-ai-helpers eval workflow](https://github.com/lyarwood/kubevirt-ai-helpers/tree/main/workflows/create-eval-from-docs#future-work-in-session-eval-testing) generates evaluation tasks that must be tested outside the ACP session because runner pods cannot create kind clusters. The workarounds (pre-provisioned cluster, sidecar, external kubeconfig) all require out-of-band setup and break the self-contained session model.

The current operator implementation has no abstraction separating "session lifecycle management" from "execution resource management":

| Layer | Coupling to Pods |
|-------|-----------------|
| `operator/internal/handlers/sessions.go` | Builds `corev1.Pod` directly (~600 lines) |
| `operator/internal/controller/reconcile_phases.go` | Switches on `pod.Status.Phase`, inspects `ContainerStatuses` by name |
| `operator/internal/handlers/sessions.go` (service) | Creates `ClusterIP` Service with pod label selectors |
| CRD status conditions | Named `PodCreated`, `PodScheduled`, `RunnerStarted` |

The upper layers are already execution-agnostic:

| Layer | Why it works as-is |
|-------|-------------------|
| PlatformBridge (Python, ADR-0006) | `run()`, `set_context()`, `shutdown()` never reference pods |
| AgenticSession CRD spec | Defines repos, prompt, env vars, LLM settings — no execution fields |
| Credential refresh | Runner fetches creds dynamically from Backend API via HTTP |
| Backend AG-UI proxy | Routes via K8s Service DNS — doesn't care what backs the Service |

## Decision Drivers

* **Testing coverage:** Pod sandbox restrictions block entire categories of e2e testing (nested containers, kind clusters, KubeVirt workflows)
* **Self-contained sessions:** Workarounds requiring out-of-band cluster provisioning break the session model
* **Backward compatibility:** Existing pod-based sessions must continue working unchanged
* **Incremental adoption:** Must be feature-gated and opt-in, not a platform-wide migration
* **Minimal blast radius:** Upper layers (bridge, backend, frontend) should require zero changes
* **Registry-driven:** Consistent with the Agent Runtime Registry design principle of "registry controls what the session runs on"

## Considered Options

1. **Pluggable `ExecutionEnvironment` interface with KubeVirt VMs**
2. **Privileged pods with Docker-in-Docker**
3. **Sidecar kind cluster**
4. **External cluster injection via kubeconfig**
5. **Kata Containers / gVisor runtime class**

## Decision Outcome

Chosen option: "Pluggable `ExecutionEnvironment` interface with KubeVirt VMs", because it provides true OS-level isolation without compromising pod security standards, is backward compatible via the registry's existing extensibility model, and solves the full problem (nested containers, system packages, nested virtualization) rather than a subset.

### Consequences

**Positive:**

* Unlocks e2e testing for projects requiring nested containers (KubeVirt, kubernetes-mcp-server)
* No changes to upper layers — PlatformBridge, backend proxy, frontend are unaffected
* Pod-based sessions continue working unchanged (default `executionKind`)
* Extracts a clean interface from the monolithic pod builder, improving operator maintainability
* Feature-gated and opt-in — no risk to existing workloads

**Negative:**

* Operator gains an optional KubeVirt client dependency
* KubeVirt must be installed on the target cluster for VM sessions
* VM sessions have higher startup latency (~30-120s vs ~5-15s for pods)
* Cloud-init bootstrap adds a new maintenance surface (VM image, systemd units, cloud-init templates)
* Operator reconciler complexity increases (two execution paths)

**Risks:**

* VM startup latency may be unacceptable for interactive sessions — mitigated by pre-baked VM images and optional warm pools (future)
* Cloud-init credential injection is less secure than K8s secret volume mounts — mitigated by using KubeVirt's `accessCredentials` or `virtio-fs` passthrough for secrets
* KubeVirt API changes between versions may require client updates — mitigated by importing the KubeVirt client module with version pinning

## Detailed Design

### ExecutionEnvironment Interface

Introduce an interface in the operator that abstracts execution resource lifecycle:

```go
// ExecutionEnvironment abstracts Pod vs VM creation and monitoring.
type ExecutionEnvironment interface {
    Create(ctx context.Context, session *unstructured.Unstructured, runtime *AgentRuntimeSpec) error
    GetStatus(ctx context.Context, namespace, name string) (*ExecutionStatus, error)
    Delete(ctx context.Context, namespace, name string) error
    EnsureService(ctx context.Context, namespace, name string, port int32) error
}

type ExecutionStatus struct {
    Phase        string       // "Pending", "Running", "Succeeded", "Failed"
    Ready        bool
    StartedAt    *metav1.Time
    TerminatedAt *metav1.Time
    Message      string
}
```

Two implementations:
- `PodExecutionEnvironment` — refactors the existing pod creation logic from `sessions.go` into this interface, preserving all current behavior
- `KubeVirtExecutionEnvironment` — creates `VirtualMachine` CR + cloud-init `Secret` + K8s `Service`

### Registry Schema Extension

Add an `executionKind` field to `AgentRuntimeSpec`. The existing `provider` field is reserved for AI vendor dispatch (`"anthropic"`, `"google"`) and is not repurposed.

```jsonc
// agent-registry-configmap.yaml → runtimes.json
{
  "id": "claude-kubevirt",
  "displayName": "Claude Code (VM)",
  "description": "Claude with full OS environment — nested containers, kind clusters, KubeVirt e2e",
  "framework": "claude-agent-sdk",
  "provider": "anthropic",

  // ─── NEW: Execution Kind ───
  "executionKind": "kubevirt-vm",   // "pod" (default) or "kubevirt-vm"

  "container": {
    "image": "quay.io/ambient_code/ambient_runner:latest",
    "port": 8001,
    "env": {
      "RUNNER_TYPE": "claude-agent-sdk",
      "RUNNER_STATE_DIR": ".claude"
    }
  },

  "sandbox": {
    "stateDir": ".claude",
    "persistence": "s3",
    "workspaceSize": "50Gi",
    "seed": { "cloneRepos": true, "hydrateState": true }
  },

  // ─── NEW: VM Config (only when executionKind == "kubevirt-vm") ───
  "vmConfig": {
    "containerDiskImage": "quay.io/ambient_code/fedora-runner:latest",
    "memory": "4Gi",
    "cpu": 2,
    "cloudInitTemplate": "ambient-runner-cloud-init"
  },

  "auth": {
    "requiredSecretKeys": ["ANTHROPIC_API_KEY"],
    "secretKeyLogic": "any",
    "vertexSupported": true
  },
  "defaultModel": "claude-sonnet-4-5",
  "models": [
    { "value": "claude-sonnet-4-5", "label": "Claude Sonnet 4.5" }
  ],
  "featureGate": "runner.kubevirt-vm.enabled"
}
```

Go type additions in `registry.go`:

```go
type AgentRuntimeSpec struct {
    // ... existing fields unchanged ...

    ExecutionKind string        `json:"executionKind,omitempty"` // "pod" (default) or "kubevirt-vm"
    VMConfig      *VMConfigSpec `json:"vmConfig,omitempty"`
}

type VMConfigSpec struct {
    ContainerDiskImage string `json:"containerDiskImage"`
    Memory             string `json:"memory"`
    CPU                int    `json:"cpu"`
    CloudInitTemplate  string `json:"cloudInitTemplate"`
}
```

### Operator Dispatch

In the session handler, select the execution environment based on `executionKind`:

```go
func getExecutionEnvironment(runtime *AgentRuntimeSpec, clients *ClientSet) ExecutionEnvironment {
    switch runtime.ExecutionKind {
    case "kubevirt-vm":
        return NewKubeVirtExecutionEnvironment(clients.KubeVirtClient, runtime)
    default: // "pod" or empty — backward compatible
        return NewPodExecutionEnvironment(clients.K8sClient, runtime)
    }
}
```

### Reconciler Generalization

Replace pod-specific status checks with the interface:

```go
// Before (reconcile_phases.go):
pod := &corev1.Pod{}
err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: ns}, pod)
UpdateSessionFromPodStatus(ctx, session, pod)

// After:
env := getExecutionEnvironment(runtime, r.clients)
status, err := env.GetStatus(ctx, ns, sessionName)
UpdateSessionFromExecutionStatus(ctx, session, status)
```

### VM Session Lifecycle

```
Pending  → Operator creates VirtualMachine CR + cloud-init Secret
Creating → Operator watches VMI status.phase (Scheduling → Running)
           Cloud-init bootstraps runner process inside VM
Running  → AG-UI server listens on port 8001
           K8s Service routes traffic to VMI's virt-launcher Pod IP
           State-sync runs as systemd timer inside VM
Stopping → Operator sets VM spec.running=false or deletes VM
Stopped  → State persisted to S3, VM resources cleaned up
```

### Cloud-Init Bootstrap

The operator renders a cloud-init `userData` script that:

1. Writes session env vars to `/etc/environment.d/ambient.conf`
2. Writes credentials to `/var/run/secrets/ambient/` (from cloud-init `write_files` or `accessCredentials`)
3. Hydrates workspace from S3 if resuming (`seed.hydrateState`)
4. Starts the ambient-runner AG-UI server as a systemd service on the registry-defined port
5. Starts state-sync as a systemd timer (reusing existing sync scripts)

The `containerDiskImage` in `vmConfig` is a pre-baked VM image containing:
- Python runtime + ambient-runner package
- S3 sync tooling (rclone or aws-cli)
- Docker/Podman (the whole point — enabling nested containers)
- systemd service/timer unit files for runner and state-sync

### Networking

KubeVirt VMIs run inside `virt-launcher` pods. The operator creates a K8s Service that selects the VMI's virt-launcher pod using standard label selectors:

```go
svc := &corev1.Service{
    Spec: corev1.ServiceSpec{
        Selector: map[string]string{
            "kubevirt.io/domain": sessionName + "-runner",
        },
        Ports: []corev1.ServicePort{
            {Name: "agui", Port: port, TargetPort: intstr.FromInt(int(port))},
        },
    },
}
```

The backend's `getRunnerEndpoint()` already resolves via Service DNS (`session-{name}.{namespace}.svc.cluster.local`) and requires no changes.

## Options Not Chosen

### Option 2: Privileged Pods with Docker-in-Docker

Run session pods with elevated privileges to enable Docker-in-Docker.

**Rejected because:**
- Requires `privileged: true` or `SYS_ADMIN` capability, violating pod security standards
- DinD is fragile (cgroup nesting, storage driver conflicts) and a known security anti-pattern
- Does not solve nested virtualization (KubeVirt-in-pod is not feasible)
- Breaks multi-tenant isolation guarantees (ADR-0001)

### Option 3: Sidecar Kind Cluster

Deploy a kind cluster controller as a sidecar container in the session pod.

**Rejected because:**
- kind requires a container runtime (Docker/containerd) — same root/capability problem as option 2
- Sidecar lifecycle management adds complexity without solving the fundamental constraint
- Does not provide a full OS environment

### Option 4: External Cluster Injection via Kubeconfig

Accept a user-provided kubeconfig for a pre-provisioned KubeVirt-enabled cluster.

**Viable as a stopgap but rejected as the primary solution because:**
- Requires out-of-band cluster provisioning — breaks self-contained session model
- Security concerns with user-managed kubeconfig credentials
- No standard lifecycle management (who creates/destroys the cluster?)
- Already documented as a workaround in kubevirt-ai-helpers

### Option 5: Kata Containers / gVisor Runtime Class

Use an alternative OCI runtime that provides stronger isolation.

**Rejected because:**
- Still container-scoped — no nested virtualization support
- Kata uses lightweight VMs but constrains the workload to a single container image
- Does not provide Docker/Podman inside the sandbox
- gVisor intercepts syscalls, which may break kind and KubeVirt tooling

## Implementation Phases

### Phase 1: Extract ExecutionEnvironment Interface

Refactor `sessions.go` and `reconcile_phases.go`:
- Extract pod creation into `PodExecutionEnvironment`
- Extract pod status monitoring into `ExecutionStatus` mapping
- Replace direct pod API calls in reconciler with interface calls
- All existing tests must pass — zero behavioral change

### Phase 2: KubeVirt ExecutionEnvironment

Implement `KubeVirtExecutionEnvironment`:
- VirtualMachine CR creation with cloud-init
- VMI status monitoring mapped to `ExecutionStatus`
- Service creation targeting virt-launcher pod
- Feature-gated behind `runner.kubevirt-vm.enabled`

### Phase 3: VM Image and Cloud-Init

Build the pre-baked VM containerDisk image:
- Fedora/Ubuntu base with Python, ambient-runner, Docker, S3 tooling
- systemd unit files for runner and state-sync
- Cloud-init template for credential and env var injection
- CI pipeline for image builds

### Phase 4: Validation

- End-to-end: create a VM session, run a prompt, verify AG-UI connectivity
- State-sync: verify workspace persistence across VM restart
- Target workload: run kubevirt-ai-helpers eval workflow inside a VM session, confirm kind cluster creation works
- Latency: measure VM session startup vs pod session startup

## Key Files

* `components/operator/internal/handlers/sessions.go` — Pod creation logic, primary refactor target
* `components/operator/internal/controller/reconcile_phases.go` — Phase state machine, pod status checks
* `components/operator/internal/handlers/reconciler.go` — `UpdateSessionFromPodStatus`, generalize to `UpdateSessionFromExecutionStatus`
* `components/operator/internal/handlers/registry.go` — `AgentRuntimeSpec` type, add `ExecutionKind` and `VMConfigSpec`
* `components/manifests/base/core/agent-registry-configmap.yaml` — Registry ConfigMap, add VM runner entry
* `components/manifests/base/crds/agenticsessions-crd.yaml` — CRD, may need optional `spec.executionRuntime` field

## Validation

How do we know this decision was correct?

* VM session can create a kind cluster and run kubectl commands inside it
* kubevirt-ai-helpers eval workflow runs end-to-end inside a VM session without out-of-band setup
* Pod-based sessions are unaffected — no regression in startup time, success rate, or behavior
* Operator maintainability improves — pod creation logic is isolated behind a clean interface

## Links

* [ADR-0001: Kubernetes-Native Architecture](0001-kubernetes-native-architecture.md)
* [ADR-0006: Ambient Runner SDK Architecture](0006-ambient-runner-sdk-architecture.md)
* [Agent Runtime Registry Design](../design/agent-runtime-registry-plan.md)
* [kubevirt-ai-helpers: Future Work — In-Session Eval Testing](https://github.com/lyarwood/kubevirt-ai-helpers/tree/main/workflows/create-eval-from-docs#future-work-in-session-eval-testing)
* [KubeVirt User Guide: VirtualMachine](https://kubevirt.io/user-guide/virtual_machines/virtual_machines/)
* [KubeVirt User Guide: Cloud-Init](https://kubevirt.io/user-guide/virtual_machines/startup_scripts/#cloud-init)
