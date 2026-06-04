# OpenShell Sandbox Isolation Specification

**Date:** 2026-06-04
**Status:** Implemented — validated end-to-end on ROSA OpenShift (kernel 5.14.0-570.99.1.el9_6)
**Related:** `specs/agents/runner.spec.md` § OpenShell Sandbox Isolation, `specs/control-plane/control-plane.spec.md`

---

## Purpose

This specification defines the requirements for sandboxing the Claude Code agent
subprocess using NVIDIA OpenShell's Supervisor binary. The sandbox prevents a
compromised or misbehaving agent from accessing credentials, filesystem regions,
network endpoints, or syscalls outside its declared policy.

---

## Requirements

### Requirement: Sandbox Activation

The sandbox SHALL be activated when the control plane environment variable
`OPENSHELL_ENABLED` is set to `true`. When not enabled, the runner SHALL launch
Claude Code directly without any sandbox wrapper.

#### Scenario: Sandbox enabled

- GIVEN the CP config has `OpenShellEnabled = true`
- WHEN a session pod is provisioned
- THEN the runner container SHALL have `OPENSHELL_ENABLED=true` in its environment
- AND the Claude CLI SHALL be launched through the OpenShell Supervisor wrapper

#### Scenario: Sandbox disabled (default)

- GIVEN the CP config has `OpenShellEnabled = false` (or unset)
- WHEN a session pod is provisioned
- THEN the runner container SHALL NOT have OpenShell environment variables
- AND the Claude CLI SHALL be launched directly by the Claude Agent SDK

---

### Requirement: File Mode Operation

The Supervisor SHALL operate in file mode using local policy files. The system
SHALL NOT require an OpenShell Gateway service.

#### Scenario: Policy file delivery

- GIVEN an `openshell-policy` ConfigMap exists in the CP namespace
- WHEN a session is provisioned in a runner namespace
- THEN the reconciler SHALL copy the ConfigMap to the runner namespace
- AND mount it as a read-only volume at `/etc/openshell`

#### Scenario: Policy file format

- GIVEN the ConfigMap contains `policy.rego` and `policy.yaml`
- WHEN the Supervisor starts
- THEN it SHALL load the Rego rules from `--policy-rules`
- AND load the YAML data from `--policy-data`
- AND validate the policy before spawning the child process

---

### Requirement: Network Namespace Isolation

The agent subprocess SHALL run in a separate Linux network namespace. All network
traffic from the agent SHALL route through the Supervisor's TLS proxy.

#### Scenario: Network namespace creation

- GIVEN the Supervisor starts with network policy configured
- WHEN it creates the sandbox environment
- THEN it SHALL create a new network namespace with a veth pair
- AND the host side SHALL listen on `10.200.0.1:3128` (HTTP CONNECT proxy)
- AND the sandbox side SHALL have `10.200.0.2/24` with default route via `10.200.0.1`
- AND the child process SHALL have `HTTPS_PROXY`, `HTTP_PROXY`, `ALL_PROXY` set to `http://10.200.0.1:3128`

#### Scenario: Blocked endpoint

- GIVEN an endpoint is NOT listed in any `network_policies` entry
- WHEN the agent attempts to connect to that endpoint
- THEN the proxy SHALL refuse the connection
- AND the agent SHALL receive a connection error

#### Scenario: Allowed endpoint

- GIVEN an endpoint IS listed in a `network_policies` entry
- AND the requesting binary matches the policy's `binaries` list
- WHEN the agent connects to that endpoint
- THEN the proxy SHALL establish an HTTP CONNECT tunnel
- AND perform TLS termination with the ephemeral per-sandbox CA
- AND forward the request to the upstream server

---

### Requirement: TLS Proxy

The Supervisor SHALL generate an ephemeral CA certificate per sandbox lifetime and
inject it into the child process via `SSL_CERT_FILE`, `NODE_EXTRA_CA_CERTS`, and
`GIT_SSL_CAINFO` environment variables.

#### Scenario: TLS trust chain

- GIVEN the Supervisor generates an ephemeral CA at startup
- WHEN the agent makes an HTTPS request through the proxy
- THEN the proxy SHALL issue a per-hostname leaf certificate signed by the ephemeral CA
- AND the agent's TLS client SHALL trust the certificate via the injected CA bundle
- AND the proxy SHALL verify upstream certificates against the system CA store

---

### Requirement: Filesystem Isolation (Landlock LSM)

The agent subprocess SHALL be confined to a filesystem allowlist enforced by
Landlock LSM.

#### Scenario: Read-only paths

- GIVEN the policy declares `/usr`, `/lib`, `/proc`, `/dev/urandom`, `/app`, `/etc`, `/var/log`, `/home/sandbox` as read-only
- WHEN the agent attempts to write to any of these paths
- THEN the write SHALL be denied by the kernel

#### Scenario: Read-write paths

- GIVEN the policy declares `/workspace`, `/tmp`, `/dev/null`, `/app/.claude` as read-write
- WHEN the agent writes to these paths
- THEN the write SHALL succeed

#### Scenario: Undeclared paths

- GIVEN a path is not listed in either read-only or read-write lists
- WHEN the agent attempts to access that path
- THEN access SHALL be denied by the kernel

#### Scenario: Landlock compatibility

- GIVEN the kernel supports Landlock ABI v2 or higher
- WHEN the Supervisor applies the Landlock ruleset
- THEN it SHALL apply all rules
- AND report the number of rules applied and skipped

- GIVEN the kernel does NOT support Landlock
- AND the policy has `landlock.compatibility: best_effort`
- WHEN the Supervisor attempts to apply Landlock
- THEN it SHALL log a warning and continue without filesystem isolation

---

### Requirement: Process Privilege Drop

The Supervisor SHALL drop privileges before executing the agent binary.

#### Scenario: Privilege drop sequence

- GIVEN the Supervisor starts as root (UID 0)
- WHEN it forks the child process
- THEN the pre_exec closure SHALL call `setgroups`, `setgid`, `setuid` to switch to the `sandbox` user
- AND set `RLIMIT_CORE` to 0 (no core dumps)
- AND set `PR_SET_DUMPABLE` to 0 (blocks ptrace attach)
- AND set `PR_SET_NO_NEW_PRIVS` to 1 (no setuid escalation)

#### Scenario: Privilege drop verification

- GIVEN the child has called `setuid(sandbox_uid)`
- WHEN the Supervisor verifies the drop
- THEN it SHALL attempt `setuid(0)` and confirm it returns `EPERM`

---

### Requirement: Syscall Filtering (seccomp-BPF)

The agent subprocess SHALL have a seccomp-BPF filter applied that blocks
dangerous syscalls.

#### Scenario: Blocked syscalls

- GIVEN the seccomp filter is applied
- WHEN the agent attempts `ptrace`, `memfd_create`, or `io_uring_setup`
- THEN the syscall SHALL be blocked

#### Scenario: Blocked socket domains

- GIVEN the seccomp filter is applied
- WHEN the agent attempts to create sockets with `AF_PACKET`, `AF_NETLINK`, or `AF_BLUETOOTH`
- THEN the socket creation SHALL be blocked

---

### Requirement: Container Security Context

The reconciler SHALL configure the runner container's security context based on
the `OpenShellEnabled` flag.

#### Scenario: OpenShell enabled

- GIVEN `OpenShellEnabled = true`
- WHEN the reconciler builds the pod spec
- THEN the container security context SHALL include:
  - `allowPrivilegeEscalation: true`
  - `runAsUser: 0`
  - `runAsNonRoot: false`
  - `capabilities.drop: [ALL]`
  - `capabilities.add: [NET_ADMIN, SYS_ADMIN, SYS_PTRACE, SETUID, SETGID, CHOWN, DAC_OVERRIDE]`
- AND the pod-level security context SHALL include `seccompProfile.type: Unconfined`

#### Scenario: OpenShell disabled

- GIVEN `OpenShellEnabled = false`
- WHEN the reconciler builds the pod spec
- THEN the container security context SHALL include:
  - `allowPrivilegeEscalation: false`
  - `capabilities.drop: [ALL]`
- AND the pod-level security context SHALL NOT override seccomp

---

### Requirement: Policy ConfigMap Propagation

The reconciler SHALL propagate the OpenShell policy ConfigMap from the control
plane namespace to each runner namespace.

#### Scenario: ConfigMap already exists

- GIVEN the policy ConfigMap already exists in the runner namespace
- WHEN the reconciler provisions a session
- THEN it SHALL skip the copy
- AND proceed with pod creation

#### Scenario: ConfigMap does not exist

- GIVEN the policy ConfigMap does NOT exist in the runner namespace
- AND the ConfigMap exists in the CP namespace
- WHEN the reconciler provisions a session
- THEN it SHALL create a copy in the runner namespace
- AND the copy SHALL contain the same `data` keys as the source

---

### Requirement: Runner Image Prerequisites

The runner container image SHALL include all dependencies required for sandbox
operation.

#### Scenario: Image contents

- GIVEN the runner Dockerfile
- WHEN the image is built
- THEN it SHALL contain:
  - `/openshell-sandbox` binary (pinned to a specific version)
  - `iproute` package (provides `ip netns` for network namespace management)
  - A `sandbox` user and group (for privilege drop target)
  - `/var/run/netns` directory with mode 777 (for network namespace mount points)
  - `/workspace` directory owned by `sandbox:sandbox`
  - `/usr/local/bin/claude` symlink to the bundled Claude CLI binary
  - `/app/openshell-claude-wrapper.sh` wrapper script

---

### Requirement: Wrapper Script Dispatch

The wrapper script SHALL dispatch to the Supervisor or directly to Claude based
on the `OPENSHELL_ENABLED` environment variable.

#### Scenario: OpenShell enabled

- GIVEN `OPENSHELL_ENABLED=true`
- WHEN the wrapper script executes
- THEN it SHALL exec the Supervisor with `--policy-rules`, `--policy-data`, `--log-level` flags
- AND pass the Claude binary path and all arguments after `--`

#### Scenario: OpenShell disabled

- GIVEN `OPENSHELL_ENABLED` is unset or not `true`
- WHEN the wrapper script executes
- THEN it SHALL exec the Claude binary directly

---

## Operational Notes

### Supervisor Log Messages (OCSF Format)

The Supervisor emits structured logs in OCSF (Open Cybersecurity Schema Framework) format:

| Log Entry | Severity | Meaning |
|-----------|----------|---------|
| `CONFIG:LOADING` | INFO | Loading policy from local files |
| `CONFIG:VALIDATED` | INFO | Sandbox user validated in image |
| `CONFIG:ENABLED` | INFO | TLS termination enabled, ephemeral CA generated |
| `CONFIG:CREATING` | INFO | Creating network namespace |
| `CONFIG:CREATED` | INFO | Network namespace created with IP addresses |
| `CONFIG:DEGRADED` | MEDIUM | `nft` not found; bypass detection rules not installed |
| `CONFIG:PROBED` | INFO | Landlock availability probed |
| `CONFIG:BUILT` | INFO | Landlock ruleset built with rule counts |
| `NET:LISTEN` | INFO | Proxy listening on address |
| `PROC:LAUNCH` | INFO | Child process spawned |
| `CONFIG:CLEANED_UP` | INFO | Network namespace cleaned up |

### Debugging

Set `OPENSHELL_LOG_LEVEL=debug` in the wrapper script or environment to enable
verbose Supervisor logging. Debug output includes individual Landlock rule
applications, `ip` command invocations, and certificate processing details.

### OpenShift Cluster Setup

1. Create a custom SCC named `openshell-sandbox` with the required capabilities
2. Bind the SCC to the runner service account via a ClusterRoleBinding or
   namespace-scoped RoleBinding with `system:openshift:scc:openshell-sandbox`
3. Verify with `oc get pod <pod> -o jsonpath='{.metadata.annotations.openshift\.io/scc}'`
   — it should show `openshell-sandbox`
