# OpenShell Integration Plan — Option C (Supervisor Binary in Runner Image)

**Branch:** `spec/runner-openshell-desired-state`
**Date:** 2026-06-04
**Status:** Implementation ready

## Architecture

```
Runner Container (Pod)
├── uvicorn main:app (AG-UI server, runs as UID 1001)
│   └── bridge.py → ClaudeAgentAdapter(cli_path="/app/openshell-claude-wrapper.sh")
│       └── Wrapper script execs:
│           openshell-sandbox \
│             --policy-rules /etc/openshell/policy.rego \
│             --policy-data /etc/openshell/policy.yaml \
│             -- claude "$@"
│
├── /openshell-sandbox (14.9 MB static binary, from supervisor image)
├── /app/openshell-claude-wrapper.sh (wrapper script)
├── /etc/openshell/policy.rego (OPA Rego rules, mounted from ConfigMap)
└── /etc/openshell/policy.yaml (policy data, mounted from ConfigMap)
```

The OpenShell Supervisor creates a **sandboxed network namespace** around the Claude CLI subprocess.
All agent network traffic routes through an HTTP CONNECT proxy at `10.200.0.1:3128`.
The Supervisor applies **Landlock LSM** (filesystem), **seccomp-BPF** (syscalls), and **OPA L7 inspection** (HTTP allow/deny).

No Gateway needed. No CRDs. Pure Go control plane + static binary in runner image.

## Changes Required

### 1. Runner Dockerfile (`components/runners/ambient-runner/Dockerfile`)

```dockerfile
# After system packages, before WORKDIR:
# Add iproute2 for network namespace setup (ip netns, ip link)
RUN dnf install -y iproute && dnf clean all

# Copy OpenShell supervisor binary from official image
COPY --from=ghcr.io/nvidia/openshell/supervisor:latest /openshell-sandbox /openshell-sandbox

# Add sandbox user/group (OpenShell requires process.run_as_user = "sandbox")
RUN groupadd -r sandbox && useradd -r -g sandbox -s /bin/bash sandbox

# Copy wrapper script
COPY openshell-claude-wrapper.sh /app/openshell-claude-wrapper.sh
RUN chmod +x /app/openshell-claude-wrapper.sh /openshell-sandbox
```

### 2. Wrapper Script (`components/runners/ambient-runner/openshell-claude-wrapper.sh`)

```bash
#!/bin/bash
set -euo pipefail

if [[ "${OPENSHELL_ENABLED:-}" == "true" ]]; then
  exec /openshell-sandbox \
    --policy-rules "${OPENSHELL_POLICY_RULES:-/etc/openshell/policy.rego}" \
    --policy-data "${OPENSHELL_POLICY_DATA:-/etc/openshell/policy.yaml}" \
    --log-level "${OPENSHELL_LOG_LEVEL:-warn}" \
    -- claude "$@"
else
  exec claude "$@"
fi
```

### 3. Bridge Modification (`components/runners/ambient-runner/ambient_runner/bridges/claude/bridge.py`)

In `_ensure_adapter()` at line ~749, after building `options` dict:

```python
if os.getenv("OPENSHELL_ENABLED") == "true":
    options["cli_path"] = "/app/openshell-claude-wrapper.sh"
```

`cli_path` is already in `_SDK_OPTIONS_DENYLIST` (line 55), so users can't override it.

### 4. Control Plane Config (`components/ambient-control-plane/internal/config/config.go`)

Add to `ControlPlaneConfig` struct:
```go
OpenShellEnabled    bool
OpenShellPolicyName string  // ConfigMap name for policy files
```

Add to `Load()`:
```go
OpenShellEnabled:    os.Getenv("OPENSHELL_ENABLED") == "true",
OpenShellPolicyName: envOrDefault("OPENSHELL_POLICY_CONFIGMAP", "openshell-policy"),
```

### 5. Control Plane Reconciler (`components/ambient-control-plane/internal/reconciler/kube_reconciler.go`)

#### 5a. SecurityContext — Add NET_ADMIN capability

In `ensurePod()` at line ~525, when `OpenShellEnabled`:
```go
"securityContext": map[string]interface{}{
    "allowPrivilegeEscalation": false,
    "capabilities": map[string]interface{}{
        "drop": []interface{}{"ALL"},
        "add":  []interface{}{"NET_ADMIN"},  // OpenShell network namespace
    },
},
```

#### 5b. Volumes — Add policy ConfigMap

In `buildVolumes()`:
```go
if r.cfg.OpenShellEnabled {
    vols = append(vols, map[string]interface{}{
        "name": "openshell-policy",
        "configMap": map[string]interface{}{
            "name": r.cfg.OpenShellPolicyName,
        },
    })
}
```

#### 5c. Volume Mounts — Add /etc/openshell

In `buildVolumeMounts()`:
```go
if r.cfg.OpenShellEnabled {
    mounts = append(mounts, map[string]interface{}{
        "name":      "openshell-policy",
        "mountPath": "/etc/openshell",
        "readOnly":  true,
    })
}
```

#### 5d. Environment Variables

In `buildEnv()`:
```go
if r.cfg.OpenShellEnabled {
    env = append(env,
        envVar("OPENSHELL_ENABLED", "true"),
        envVar("OPENSHELL_POLICY_RULES", "/etc/openshell/policy.rego"),
        envVar("OPENSHELL_POLICY_DATA", "/etc/openshell/policy.yaml"),
    )
}
```

### 6. OpenShell Policy Files (ConfigMap)

#### 6a. Rego Rules (`policy.rego`)

Standard OpenShell policy package with `allow_network` and `allow_request` rules.
Controls L4 (connection-level) and L7 (HTTP request-level) access.

#### 6b. Policy Data (`policy.yaml`)

```yaml
version: 1
process:
  run_as_user: sandbox
filesystem_policy:
  writable_paths:
    - /workspace
    - /tmp
    - /app/.claude
  readable_paths:
    - /
landlock:
  best_effort: true
network_policies:
  - name: anthropic-api
    endpoints:
      - api.anthropic.com:443
    binaries:
      - /usr/local/bin/claude
  - name: github-api
    endpoints:
      - api.github.com:443
      - github.com:443
    binaries:
      - /usr/bin/git
      - /usr/local/bin/gh
  - name: npm-registry
    endpoints:
      - registry.npmjs.org:443
    binaries:
      - /usr/bin/npm
      - /usr/bin/node
```

### 7. Deployment ConfigMap

Create `openshell-policy` ConfigMap in the CP namespace containing the Rego and YAML policy files.
The CP will reference this ConfigMap when building runner pod specs.

## Deployment Steps

1. Build modified runner image → `quay.io/ambient_code/vteam_claude_runner:openshell-v1`
2. Build modified CP image → `quay.io/ambient_code/vteam_control_plane:openshell-v1`
3. Create namespace `ambient-openshell` on ROSA cluster
4. Create `openshell-policy` ConfigMap in `ambient-openshell`
5. Deploy PostgreSQL, api-server, control-plane (adapted from `install-standard.sh`)
6. Set `OPENSHELL_ENABLED=true` on CP deployment
7. Create session → verify runner pod has NET_ADMIN, policy mount, wrapper CLI

## Security Considerations

- **NET_ADMIN is required** for `unshare(CLONE_NEWNET)` — the supervisor creates an isolated network namespace
- **Landlock** provides kernel-enforced filesystem isolation — agent cannot read sidecar creds from `/proc`
- **seccomp-BPF** blocks `ptrace`, `memfd_create`, raw sockets — prevents privilege escalation
- **OPA L7** provides per-binary HTTP allow/deny — only `claude` can reach Anthropic API
- **Credential proxy** (Phase 2, requires Gateway) — replaces real tokens with opaque placeholders

## Phase 2 (Future): Credential Proxy via OpenShell Gateway

Phase 1 (this plan) provides kernel-level isolation without Gateway.
Phase 2 would add OpenShell Gateway as an init container or sidecar for credential proxy:
- Provider registration per session
- Token placeholder rewriting at the proxy layer
- Removes need for sidecar credential containers entirely

## Testing Checklist

- [ ] Runner image builds with supervisor binary
- [ ] Wrapper script correctly dispatches (OPENSHELL_ENABLED=true → supervisor, false → direct)
- [ ] CP adds NET_ADMIN capability when OpenShellEnabled
- [ ] Policy ConfigMap mounted at /etc/openshell
- [ ] Claude CLI subprocess runs inside sandboxed netns
- [ ] Agent can reach Anthropic API through proxy
- [ ] Agent cannot reach sidecar ports (8091-8094)
- [ ] Landlock prevents filesystem access outside /workspace, /tmp
