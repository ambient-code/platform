# OpenShell Runner Adaptation — Implementation Record

> Initial analysis: 2026-06-03
> Implementation completed: 2026-06-04
> Companion doc: [OpenShell Security Model Analysis](openshell-security-analysis.md)
> Formal spec: `specs/security/openshell-sandbox.spec.md`
> Target components: `components/runners/ambient-runner/`, `components/ambient-control-plane/`

---

## Summary

The runner now wraps the Claude Code subprocess inside NVIDIA OpenShell's
Supervisor binary (`openshell-sandbox` v0.0.56) in **file mode** — no OpenShell
Gateway required. Five defense-in-depth isolation layers are applied: network
namespace, TLS proxy with L7 OPA inspection, Landlock filesystem sandbox,
seccomp-BPF syscall filtering, and privilege drop to an unprivileged user.

The implementation was validated end-to-end on ROSA OpenShift
(kernel 5.14.0-570.99.1.el9_6). All five layers confirmed operational.

---

## Strategy Selection

Three strategies were evaluated (see original analysis below). **Strategy 1
(Supervisor wrapping Claude CLI)** was selected and implemented. The key insight
was that file mode eliminates the Gateway dependency entirely — the Supervisor
reads policy from local Rego + YAML files, and the control plane distributes
these via ConfigMap propagation. No gRPC provider registration is needed.

The credential placeholder/proxy pattern (Phase 2 in the original analysis) is
deferred. The current implementation provides network, filesystem, syscall, and
process isolation without changing credential flow. LLM credentials (Vertex AI
service account) remain in the runner environment because they are necessary for
inference and the SDK loads them before the sandbox starts.

---

## What Was Actually Built

### Architecture (Implemented)

```
Runner Pod (FastAPI + uvicorn) — runs UNSANDBOXED as UID 0
  │
  ├── bridge.py: _ensure_adapter() sets cli_path when OPENSHELL_ENABLED=true
  │     options["cli_path"] = "/app/openshell-claude-wrapper.sh"
  │
  └── Claude Agent SDK spawns wrapper as subprocess
        │
        └── /app/openshell-claude-wrapper.sh
              │  reads: OPENSHELL_ENABLED, OPENSHELL_POLICY_RULES, OPENSHELL_POLICY_DATA
              │
              └── exec /openshell-sandbox \
                    --policy-rules /etc/openshell/policy.rego \
                    --policy-data /etc/openshell/policy.yaml \
                    --log-level warn \
                    -- /usr/local/bin/claude "$@"
                        │
                        ├── Supervisor (Rust, runs as root):
                        │   1. Load + validate OPA policy
                        │   2. Create network namespace (ip netns add sandbox-{uuid})
                        │   3. Create veth pair (10.200.0.1 ↔ 10.200.0.2)
                        │   4. Start TLS proxy on 10.200.0.1:3128
                        │   5. Generate ephemeral CA for MITM TLS
                        │   6. Prepare Landlock PathFds while still root
                        │
                        ├── fork()
                        │   pre_exec closure (child process, before exec):
                        │     1. setns(CLONE_NEWNET) → enter sandbox network namespace
                        │     2. setgroups/setgid/setuid → drop to sandbox:sandbox
                        │     3. RLIMIT_CORE=0, PR_SET_DUMPABLE=0, PR_SET_NO_NEW_PRIVS=1
                        │     4. landlock_restrict_self → 12 filesystem rules (8 RO, 4 RW)
                        │     5. seccomp::apply → block ptrace, memfd_create, raw sockets
                        │
                        └── exec(/usr/local/bin/claude) → runs as sandbox user
                              env injected by supervisor:
                                HTTPS_PROXY=http://10.200.0.1:3128
                                HTTP_PROXY=http://10.200.0.1:3128
                                ALL_PROXY=http://10.200.0.1:3128
                                SSL_CERT_FILE=/etc/openshell-tls/ca-bundle.pem
                                NODE_EXTRA_CA_CERTS=/etc/openshell-tls/openshell-ca.pem
                                NODE_USE_ENV_PROXY=1
                                GIT_SSL_CAINFO=/etc/openshell-tls/ca-bundle.pem
```

### Files Changed

| File | Component | What Changed |
|------|-----------|-------------|
| `Dockerfile` | Runner | Pin supervisor v0.0.56 from `ghcr.io/nvidia/openshell/supervisor:0.0.56`; add `iproute` (provides `ip netns`); create `sandbox` user/group; pre-create `/workspace` owned by sandbox; symlink bundled claude binary to `/usr/local/bin/claude`; set `/home/sandbox` to 755; create `/var/run/netns` with 777 |
| `openshell-claude-wrapper.sh` | Runner | New file — shell script that dispatches to supervisor or direct claude based on `OPENSHELL_ENABLED` env var |
| `bridges/claude/bridge.py` | Runner | 1 line added in `_ensure_adapter()`: `options["cli_path"] = "/app/openshell-claude-wrapper.sh"` when `OPENSHELL_ENABLED == "true"` |
| `.openshell-ref/policy.rego` | Runner | Official OPA Rego policy from OpenShell repo (`package openshell.sandbox`, ~741 lines) |
| `.openshell-ref/policy.yaml` | Runner | Policy data: filesystem allowlists, Landlock config, process identity, network ACLs for 6 endpoint groups |
| `internal/reconciler/kube_reconciler.go` | Control Plane | `buildRunnerSecurityContext()`: 7 capabilities when OpenShell enabled; `buildVolumes()`: openshell-policy ConfigMap volume; `buildVolumeMounts()`: mount at `/etc/openshell`; `buildEnv()`: inject OPENSHELL_* env vars; `ensureOpenShellPolicy()`: copy ConfigMap from CP to runner namespace; `ensurePod()`: pod-level seccompProfile Unconfined |
| `internal/config/config.go` | Control Plane | `OpenShellEnabled` (from `OPENSHELL_ENABLED` env) and `OpenShellPolicyName` (from `OPENSHELL_POLICY_CONFIGMAP`, default `openshell-policy`) |
| `internal/kubeclient/kubeclient.go` | Control Plane | Added `ConfigMapGVR`, `GetConfigMap()`, `CreateConfigMap()` |
| `cmd/ambient-control-plane/main.go` | Control Plane | Thread `OpenShellEnabled` and `OpenShellPolicyName` from config into reconciler |

### What Did NOT Change

| Component | Why |
|-----------|-----|
| `platform/auth.py` | Credential placeholder/proxy is deferred; real tokens still injected into runner env |
| `_grpc_client.py` | gRPC client runs in runner process, outside sandbox boundary |
| `middleware/secret_redaction.py` | Retained as-is; still provides output stream defense-in-depth |
| `bridges/claude/session.py` | SessionWorker lifecycle unchanged; supervisor is transparent to the SDK |
| `components/operator/` | OpenShell integration is in the CP, NOT the operator |

---

## Critical Implementation Details

### Why 7 Capabilities (Not Just NET_ADMIN)

The original analysis estimated only `NET_ADMIN` was needed. In practice, the
Supervisor's `pre_exec` closure requires significantly more:

| Capability | Discovery Method | Required For |
|------------|-----------------|-------------|
| `NET_ADMIN` | Expected (documented) | `ip netns add`, veth pair setup, routing |
| `SYS_ADMIN` | EPERM on `mount --make-shared /var/run/netns` | Mount propagation for netns mount points; `nsenter` for in-namespace `ip` commands |
| `SYS_PTRACE` | Exit code 133 (SIGTRAP) when ptrace wrapper attempted | Binary identity verification via `/proc` inspection |
| `SETUID` | `setgroups(1, ...) = EPERM` in forked child | `drop_privileges()` calls `setgroups` before `setgid`/`setuid` to switch from root to sandbox |
| `SETGID` | Same as SETUID — discovered together | `drop_privileges()` calls `setgid(sandbox_gid)` |
| `CHOWN` | Supervisor sets ownership on `/workspace` and `/tmp` | `chown sandbox:sandbox` on read-write directories before privilege drop |
| `DAC_OVERRIDE` | Directory access failures during privilege transition | Access directories that don't have world-readable permissions |

### Why `runAsUser: 0`

The Supervisor MUST start as root because:
1. Network namespace creation requires `CAP_NET_ADMIN` + effective UID 0
2. Mount operations on `/var/run/netns` require `CAP_SYS_ADMIN`
3. The `drop_privileges()` call in `pre_exec` transitions from root → sandbox user

After fork, the child process runs as `sandbox:sandbox` (non-root). The
Supervisor parent process remains as root for the TLS proxy.

### Why `seccompProfile: Unconfined`

The Supervisor applies its own three-layer seccomp-BPF filter to the child
process. If the container-level seccomp profile (from the CRI runtime) is more
restrictive, it can interfere with the Supervisor's own syscalls for namespace
setup, mount operations, and process management. Setting `Unconfined` at the pod
level delegates seccomp entirely to the Supervisor.

### Why File Mode (No Gateway)

The original analysis assumed Gateway mode with gRPC provider registration. File
mode was chosen instead because:

1. No additional service to deploy and operate
2. Policy is static per deployment — it doesn't change per-session
3. ConfigMap propagation is a native K8s pattern the CP already uses
4. The Supervisor loads policy from `--policy-rules` and `--policy-data` flags
5. Eliminates the mTLS PKI bootstrap that Gateway mode requires

### The `/usr/local/bin/claude` Symlink

The Claude Agent SDK bundles its CLI binary at a version-dependent path:
```
/usr/local/lib/python3.12/site-packages/claude_agent_sdk/_bundled/claude
```

This path changes with Python version and SDK version. The policy's `binaries`
list needs a stable path to identify which binary is making network requests.
The Dockerfile creates a symlink at build time:

```dockerfile
BUNDLED=$(python3 -c 'import claude_agent_sdk; from pathlib import Path; print(Path(claude_agent_sdk.__file__).parent / "_bundled" / "claude")') && \
ln -sf "$BUNDLED" /usr/local/bin/claude
```

The wrapper script and policy both reference `/usr/local/bin/claude`.

---

## Debugging Journey

### Error Progression (Chronological)

| # | Error | Root Cause | Fix |
|---|-------|-----------|-----|
| 1 | SCC `restricted` blocking `NET_ADMIN` | Default OpenShift SCC doesn't allow custom capabilities | Created custom SCC `openshell-sandbox` |
| 2 | ConfigMap not found in runner namespace | Policy ConfigMap exists only in CP namespace | Added `ensureOpenShellPolicy()` to reconciler |
| 3 | Invalid Rego policy format | Initial policy was hand-written; supervisor expects official format | Replaced with official Rego from OpenShell repo |
| 4 | `EPERM` on network namespace creation | Missing mount propagation for `/var/run/netns` | Added `SYS_ADMIN` capability, `allowPrivilegeEscalation: true`, `runAsUser: 0` |
| 5 | `EINVAL` from unknown syscall | Initially misattributed to `landlock_restrict_self(fd, flags=1)` needing kernel 6.10+ | Extensive ptrace debugging proved `landlock_restrict_self` was NEVER called; actual cause was `setgroups` EPERM |
| 6 | `setgroups(1, ...) = EPERM` | Missing `SETUID`, `SETGID`, `CHOWN`, `DAC_OVERRIDE` capabilities | Added all 4 capabilities to reconciler and SCC |
| 7 | `Permission denied (os error 13)` launching claude | `claude` binary not in PATH inside sandbox | Added `/usr/local/bin/claude` symlink in Dockerfile |

### The EINVAL Misdiagnosis

The most significant debugging challenge was error #5. The Supervisor logged
`EINVAL` and the initial hypothesis was that `landlock_restrict_self(fd, flags=1)`
was failing because the `LANDLOCK_RESTRICT_SELF_LOG` flag (bit 0) requires
kernel 6.10+.

Nine custom C ptrace tracer programs were built and injected into the container
to intercept every syscall from every thread. The definitive finding:
**syscall 446 (`landlock_restrict_self`) was never called by any traced process**.
The EINVAL errors were all from `prctl(23, ...)` in forked `ip`/`nsenter`
subprocesses — non-fatal background noise.

The actual failing syscall was `setgroups(1, [sandbox_gid]) = -1 (EPERM)` in the
child process after fork, during the `drop_privileges()` sequence. The fix was
adding `SETUID`, `SETGID`, `CHOWN`, and `DAC_OVERRIDE` capabilities.

---

## Verified End-to-End Results

### Sandbox Layer Confirmation (from Supervisor logs)

```
CONFIG:LOADING  [INFO] Loading OPA policy engine from local files
CONFIG:VALIDATED [INFO] Validated 'sandbox' user exists in image
CONFIG:ENABLED  [INFO] TLS termination enabled: ephemeral CA generated
CONFIG:CREATING [INFO] Creating network namespace [ns:sandbox-*]
CONFIG:CREATED  [INFO] Network namespace created [host_ip:10.200.0.1 sandbox_ip:10.200.0.2]
CONFIG:PROBED   [INFO] Landlock filesystem sandbox available [abi:v5 compat:BestEffort ro:8 rw:4]
CONFIG:BUILT    [INFO] Landlock ruleset built [rules_applied:12 skipped:0]
PROC:LAUNCH     [INFO] /usr/local/bin/claude(pid)
```

### Network Policy Enforcement (from curl tests inside sandbox)

| Target | HTTP Status | Policy Match |
|--------|-------------|-------------|
| `api.anthropic.com` | 404 (connected) | `anthropic-api` |
| `us-east5-aiplatform.googleapis.com` | 404 (connected) | `vertex-ai` |
| `oauth2.googleapis.com` | 404 (connected) | `vertex-ai` |
| `api.github.com` | 200 (connected) | `github` |
| `evil.com` | 000 (refused) | No match — **blocked** |

### Sandbox Environment (injected by Supervisor)

```
ALL_PROXY=http://10.200.0.1:3128
HTTPS_PROXY=http://10.200.0.1:3128
HTTP_PROXY=http://10.200.0.1:3128
NO_PROXY=127.0.0.1,localhost,::1
SSL_CERT_FILE=/etc/openshell-tls/ca-bundle.pem
NODE_EXTRA_CA_CERTS=/etc/openshell-tls/openshell-ca.pem
NODE_USE_ENV_PROXY=1
GIT_SSL_CAINFO=/etc/openshell-tls/ca-bundle.pem
DENO_CERT=/etc/openshell-tls/openshell-ca.pem
```

---

## OpenShift SCC Reference

The custom SCC required on OpenShift clusters:

```yaml
apiVersion: security.openshift.io/v1
kind: SecurityContextConstraints
metadata:
  name: openshell-sandbox
allowHostDirVolumePlugin: false
allowHostIPC: false
allowHostNetwork: false
allowHostPID: false
allowHostPorts: false
allowPrivilegeEscalation: true
allowPrivilegedContainer: false
allowedCapabilities:
  - NET_ADMIN
  - SYS_ADMIN
  - SYS_PTRACE
  - SETUID
  - SETGID
  - CHOWN
  - DAC_OVERRIDE
defaultAddCapabilities: null
fsGroup:
  type: RunAsAny
readOnlyRootFilesystem: false
requiredDropCapabilities:
  - KILL
  - MKNOD
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: MustRunAs
seccompProfiles:
  - '*'
supplementalGroups:
  type: RunAsAny
volumes:
  - configMap
  - downwardAPI
  - emptyDir
  - persistentVolumeClaim
  - projected
  - secret
```

---

## Known Warnings (Non-Fatal)

| Warning | Source | Impact |
|---------|--------|--------|
| `nft not found; bypass detection rules will not be installed` | Supervisor | `nftables` not in runner image; bypass detection iptables rules not installed. Network namespace still enforces routing. |
| `runtime cgroup pids.max is unlimited` | Supervisor | No PID limit configured at container/cgroup level. Fork bomb protection relies on `RLIMIT_NPROC=512` set by supervisor. |
| `Failed to delete network namespace` | Supervisor | Cleanup race on fast shutdown. Harmless; pod restart clears all. |

---

## Future Work

### Phase 2: Credential Placeholder/Proxy

Replace real integration tokens with `openshell:resolve:env:*` placeholders in
the runner environment. The Supervisor's TLS proxy would rewrite placeholders to
real values on outbound HTTP requests. This eliminates LLM credential exposure
in the agent's `/proc/self/environ`.

### Phase 3: Per-Session OPA Policies

Generate per-session policy data from the project's credential bindings. A session
with only GitHub credentials would get a tighter network policy than one with
GitHub + GitLab + Jira.

### Phase 4: nftables Bypass Detection

Add the `nftables` package to the runner image to enable the Supervisor's bypass
detection iptables rules (LOG + REJECT for direct connections that skip the proxy).

---

## Original Analysis (Preserved for Context)

The sections below are the original pre-implementation analysis from 2026-06-03.
They are preserved for historical context. The actual implementation diverged
from the original analysis in several ways (file mode instead of Gateway mode,
7 capabilities instead of 1, no auth.py changes in Phase 1).

<details>
<summary>Original: Current Runner Credential Model (The Problem)</summary>

The runner puts **real secrets directly into `os.environ`** and the agent's process memory. If the agent inspects its own environment, it sees real credentials.

### How Secrets Flow Today

| Mechanism | File | What Happens |
|-----------|------|-------------|
| `populate_runtime_credentials()` | `platform/auth.py` | Fetches real tokens from backend API, writes them into `os.environ`: `GITHUB_TOKEN`, `GITLAB_TOKEN`, `JIRA_API_TOKEN`, `ANTHROPIC_API_KEY`, `CODERABBIT_API_KEY`, etc. |
| Token files on disk | `platform/auth.py` | Writes real tokens to `/tmp/.ambient_github_token`, `/tmp/.ambient_gitlab_token`, `/tmp/.ambient_kubeconfig` for the git credential helper and `gh` wrapper |
| Git credential helper | `platform/auth.py` | Shell script at `/tmp/git-credential-ambient` reads the real token from temp file and pipes it to git |
| `gh` CLI wrapper | `platform/auth.py` | Shell script reads real GitHub token from file, exports `GH_TOKEN`, then exec's the real `gh` |
| Secret redaction middleware | `middleware/secret_redaction.py` | Post-hoc defense: scrubs secrets from *outbound AG-UI events* only — the agent process still has full access to real secrets in memory and on disk |

### The Gap

```
Agent reads /proc/self/environ     → sees GITHUB_TOKEN=ghp_real_secret
Agent runs: cat /tmp/.ambient_*    → sees real tokens
Agent runs: echo $ANTHROPIC_API_KEY → sees real API key
```

The redaction middleware protects the *output stream* (events sent to the frontend), not the agent itself. A compromised or misbehaving agent has unrestricted access to all credentials.

</details>

<details>
<summary>Original: Strategy Comparison</summary>

| Criterion | Strategy 1 (Supervisor) | Strategy 2 (Pod Runtime) | Strategy 3 (Proxy Only) |
|-----------|---------------------|------------------------|------------------------|
| Credential isolation | Full (placeholder/proxy) | Full (placeholder/proxy) | Partial (no netns enforcement) |
| Network isolation | Full (netns + iptables) | Full (netns + iptables) | None |
| Filesystem isolation | Landlock LSM | Landlock LSM | None |
| Syscall filtering | seccomp-BPF | seccomp-BPF | None |
| L7 inspection (OPA) | Yes | Yes | No |
| Runner code changes | Moderate | None | Small |
| Kernel requirements | Linux 5.13+ | Linux 5.13+ | None |
| Defense depth | 5 layers | 5 layers | 1 layer |

**Strategy 1 was selected.**

</details>
