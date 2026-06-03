# Adapting ambient-runner to Use OpenShell

> Analysis date: 2026-06-03
> Companion doc: [OpenShell Security Model Analysis](openshell-security-analysis.md)
> Target component: `components/runners/ambient-runner/ambient_runner/`

---

## Current Runner Credential Model (The Problem)

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

---

## OpenShell Integration Strategies

### Strategy 1: OpenShell Supervisor wrapping Claude CLI (Recommended)

Replace the runner container's direct credential injection with OpenShell's Supervisor wrapping the Claude CLI subprocess. The Supervisor is **not** a sidecar container — it is a binary invoked by `bridge.py` that fork/execs the Claude CLI, applying Landlock, seccomp, and netns isolation in the `pre_exec` closure (after fork, before exec). This gives the Supervisor control over the agent's process setup, which a separate sidecar container cannot achieve.

#### What Changes

| Component | Current | With OpenShell |
|-----------|---------|---------------|
| `auth.py:populate_runtime_credentials()` | Sets `os.environ["GITHUB_TOKEN"] = real_token` | Sets `os.environ["GITHUB_TOKEN"] = "openshell:resolve:env:GITHUB_TOKEN"` |
| Token files (`/tmp/.ambient_*`) | Contain real tokens | Contain placeholder strings |
| Git credential helper | Reads real token from file | Reads placeholder; OpenShell proxy rewrites on outbound |
| `gh` wrapper | Exports real `GH_TOKEN` | Exports placeholder; proxy rewrites |
| Network egress | Direct to `api.github.com`, etc. | Via OpenShell HTTP CONNECT proxy at `10.200.0.1:3128` |
| `secret_redaction.py` | Primary defense for output stream | Redundant but kept as defense-in-depth |
| `_grpc_client.py` | Direct gRPC to API server | Whitelisted in network policy (intra-cluster) |
| Claude CLI subprocess | Full env access with real secrets | Runs in sandbox netns with placeholders only |

#### Implementation Steps

**1. New OpenShell provider type**

Register Ambient's credential store as an OpenShell provider. The Operator creates a provider config that maps each credential type (github, gitlab, jira, etc.) to the corresponding backend API credential endpoint. Two options:

- OpenShell's Gateway calls the Ambient backend to fetch the real token on demand
- The Operator pre-populates the provider at pod creation time (simpler, no Gateway dependency)

**2. Modify `platform/auth.py`**

Replace `populate_runtime_credentials()` with a version that writes placeholders instead of real values:

```python
# Before (current)
os.environ["GITHUB_TOKEN"] = github_creds["token"]  # real secret
_GITHUB_TOKEN_FILE.write_text(github_creds["token"])  # real secret on disk

# After (with OpenShell)
os.environ["GITHUB_TOKEN"] = "openshell:resolve:env:GITHUB_TOKEN"  # placeholder
_GITHUB_TOKEN_FILE.write_text("openshell:resolve:env:GITHUB_TOKEN")  # placeholder
# Real secret held only in Supervisor memory → proxy rewrites on outbound
```

The same pattern applies to all HTTP-based credential types: `GITLAB_TOKEN`, `JIRA_API_TOKEN`, `ANTHROPIC_API_KEY`, `CODERABBIT_API_KEY`.

> **HTTP-only limitation:** The placeholder/proxy pattern works at the HTTP layer only. The proxy rewrites `Authorization: Bearer openshell:resolve:env:GITHUB_TOKEN` in HTTP requests, but cannot intercept credential usage in non-HTTP contexts. The git credential helper and `gh` wrapper work because git/gh ultimately make HTTPS requests that pass through the proxy. However, SSH-based git auth, kubeconfig client certificates, and any non-HTTP protocol would receive the placeholder string verbatim. Future credential types using non-HTTP protocols will need a different isolation approach (e.g., agent-side socket forwarding or dedicated MCP tools).
>
> Current credential types and their compatibility:
> - `GITHUB_TOKEN` — HTTP-based, works with proxy rewrite
> - `GITLAB_TOKEN` — HTTP-based, works with proxy rewrite
> - `JIRA_API_TOKEN` — HTTP-based, works with proxy rewrite
> - `ANTHROPIC_API_KEY` — HTTP-based, works with proxy rewrite
> - `CODERABBIT_API_KEY` — HTTP-based, works with proxy rewrite
> - `KUBECONFIG` — **Mixed**: API server calls are HTTPS (works), but client certificate auth embeds certs in the kubeconfig file (placeholder won't work for cert-based auth). Token-based kubeconfig auth works.

**3. Modify the Dockerfile**

Add OpenShell Supervisor binary. The runner (uvicorn) starts normally; the Supervisor is invoked by `bridge.py` when launching the Claude CLI subprocess:

```dockerfile
# Add OpenShell binary
COPY --from=openshell/supervisor:latest /usr/bin/openshell-sandbox /usr/bin/openshell-sandbox

# Entrypoint unchanged — uvicorn runs unsandboxed:
CMD ["/bin/bash", "-c", "umask 0022 && cd /app/ambient-runner && uvicorn main:app --host 0.0.0.0 --port 8001"]
```

The Supervisor wraps only the Claude CLI subprocess (launched from `bridges/claude/bridge.py`), applying Landlock + seccomp + netns to the agent process. The runner itself (FastAPI/uvicorn, gRPC client, credential fetching) runs outside the sandbox boundary.

> **Capability requirement:** The Supervisor needs `NET_ADMIN` capability to create the network namespace (`unshare(CLONE_NEWNET)`) and set up the veth pair that routes agent traffic through `10.200.0.1:3128`. Without `CLONE_NEWNET`, placeholders will be sent as-is to upstream APIs — the proxy has no way to intercept requests outside its network namespace. The Operator must add `NET_ADMIN` to the runner container's `securityContext.capabilities.add`.

**4. Network policy via OpenShell**

Replace the K8s `NetworkPolicy` with OpenShell's per-sandbox network namespace + OPA policy:

```yaml
network_policies:
  ambient_backend:
    name: ambient-backend-access
    endpoints:
      - host: backend-service.ambient-code.svc.cluster.local
        port: 8080
        protocol: rest
        access: read-write
    binaries:
      - { path: /usr/bin/python3 }

  ambient_grpc:
    name: ambient-grpc-access
    endpoints:
      - host: ambient-api-server.ambient-code.svc.cluster.local
        port: 9000
        protocol: connect
        access: read-write
    binaries:
      - { path: /usr/bin/python3 }

  github_api:
    name: github-api-access
    endpoints:
      - host: api.github.com
        port: 443
        protocol: rest
        access: read-write

  anthropic_api:
    name: anthropic-api-access
    endpoints:
      - host: api.anthropic.com
        port: 443
        protocol: rest
        access: read-write

  gitlab_api:
    name: gitlab-api-access
    endpoints:
      - host: "*.gitlab.com"
        port: 443
        protocol: rest
        access: read-write
```

**5. `_grpc_client.py` — No changes needed**

The gRPC channel to the API server is established by the runner process, which runs outside the OpenShell sandbox boundary. Since only the Claude CLI subprocess is sandboxed, the gRPC client is unaffected.

**6. Modify `bridges/claude/bridge.py`**

The bridge launches Claude CLI via the Supervisor binary instead of directly. The Supervisor fork/execs the agent process, applying sandbox restrictions in the `pre_exec` closure:

```python
# Before (current)
subprocess.Popen(["claude", "--sdk", ...], env=agent_env)

# After (with OpenShell)
subprocess.Popen(
    ["openshell-sandbox", "--provider", "ambient", "--", "claude", "--sdk", ...],
    env=agent_env
)
```

The Supervisor owns the agent's process lifecycle — it creates the netns, applies Landlock/seccomp, drops privileges, then execs the Claude CLI. `HTTP_PROXY`/`HTTPS_PROXY` are injected automatically by the Supervisor into the sandboxed process environment.

**7. Operator changes**

The Operator (`components/operator/`) configures OpenShell provider + policy per session Job:

- Register the Ambient provider via OpenShell's **gRPC-only** Gateway API (`openshell.v1.OpenShell` service — `CreateProvider`, `SetClusterInference`). There are no REST equivalents; the Gateway multiplexes gRPC and HTTP on port 8080, but provider/inference management is exclusively gRPC. Proto definitions: `proto/openshell.proto`, `proto/inference.proto` in the OpenShell upstream repo.
- Add `NET_ADMIN` capability to the runner container's `securityContext` (required for Supervisor to create network namespace)
- Generate per-session OPA policies based on the session's credential bindings
- Pass the policy YAML as a volume mount

#### Files to Modify

| File | Change |
|------|--------|
| `platform/auth.py` | `populate_runtime_credentials()` writes placeholders, not real tokens |
| `platform/auth.py` | Token files (`/tmp/.ambient_*`) get placeholder values |
| `platform/auth.py` | `install_git_credential_helper()` — helper returns placeholder; proxy rewrites |
| `platform/auth.py` | `install_gh_wrapper()` — wrapper exports placeholder `GH_TOKEN` |
| `_grpc_client.py` | No changes needed — gRPC runs in runner process, outside Claude subprocess sandbox boundary |
| `Dockerfile` | Add OpenShell Supervisor binary (entrypoint unchanged) |
| `bridges/claude/bridge.py` | Launch Claude CLI via `openshell-sandbox` binary; Supervisor fork/execs with sandbox pre_exec |
| `middleware/secret_redaction.py` | Keep as defense-in-depth (now truly redundant) |
| `components/operator/` | Configure OpenShell provider via gRPC Gateway API; add `NET_ADMIN` capability; generate per-session OPA policies |

---

### Strategy 2: OpenShell as Pod Runtime (Operator-Level)

The Operator spawns Jobs using an OpenShell-managed container runtime instead of raw K8s containers. The integration moves up a level — runner code doesn't change, but the Operator configures OpenShell as the execution environment.

**Pros:** Zero runner code changes.

**Cons:** Requires OpenShell's Kubernetes compute driver to be production-ready (currently alpha). Heavier Operator changes. Less control over per-session policy granularity from the runner's perspective.

---

### Strategy 3: OpenShell Provider Bridge (Minimal, Credential-Only)

Adopt only the credential placeholder/proxy pattern without the full sandbox. Write a thin Python adapter that:

1. Starts a local HTTP CONNECT proxy in the runner pod
2. Holds real secrets in proxy memory (separate process, higher privilege)
3. Injects placeholders into `os.environ`
4. Rewrites placeholders to real values on outbound requests

**Pros:** No Rust dependency, no kernel features (Landlock/seccomp) needed. Works on any kernel version. Smallest change surface.

**Cons:** No Landlock/seccomp/netns isolation — only credential isolation. Agent can still bypass the proxy if it makes raw socket calls (no network namespace enforcement). No L7 inspection or OPA policy evaluation.

---

## Strategy Comparison

| Criterion | Strategy 1 (Sidecar) | Strategy 2 (Pod Runtime) | Strategy 3 (Proxy Only) |
|-----------|---------------------|------------------------|------------------------|
| Credential isolation | Full (placeholder/proxy) | Full (placeholder/proxy) | Partial (no netns enforcement) |
| Network isolation | Full (netns + iptables) | Full (netns + iptables) | None |
| Filesystem isolation | Landlock LSM | Landlock LSM | None |
| Syscall filtering | seccomp-BPF | seccomp-BPF | None |
| L7 inspection (OPA) | Yes | Yes | No |
| Runner code changes | Moderate (`auth.py`, `bridge.py`, `Dockerfile`) | None | Small (new proxy module) |
| Operator changes | Moderate (provider + policy config) | Heavy (new compute driver) | None |
| Kernel requirements | Linux 5.13+ (Landlock) | Linux 5.13+ (Landlock) | None |
| OpenShell maturity dependency | Supervisor (stable) | K8s driver (alpha) | None (custom code) |
| Container capability requirement | `NET_ADMIN` (for netns setup) | Depends on runtime | None |
| Gateway API protocol | gRPC only (`openshell.v1.OpenShell`) | gRPC only | N/A |
| Credential protocol support | HTTP-only (placeholder/proxy rewrite) | HTTP-only | HTTP-only |
| Defense depth | 5 layers | 5 layers | 1 layer |

---

## Recommendation

**Strategy 1 (Sidecar Supervisor)** is the right path. It provides:

- Agent never sees real secrets (even `/proc/self/environ` inspection fails)
- L7 inspection via OPA policies (audit which APIs the agent calls)
- Landlock + seccomp hardening within the container
- Binary identity via SHA256 TOFU (only known binaries can make network calls)
- The existing `secret_redaction.py` becomes a true defense-in-depth layer rather than the primary defense

The critical architectural insight: OpenShell's credential proxy pattern eliminates the single point of failure in the current design. Today, `populate_runtime_credentials()` puts real secrets into a space the agent fully controls. OpenShell moves real secrets into Supervisor memory — a separate privilege domain the agent cannot access.

### Prerequisite: Kernel Version

OpenShell's Landlock LSM requires Linux 5.13+. The runner containers run on UBI 10 (RHEL 10), which ships kernel 6.x — this is satisfied. OpenShell's `best_effort` Landlock mode also provides graceful degradation if the kernel lacks support.

### Migration Path

1. **Phase 1 — Credential proxy only (Strategy 3):** Ship a Python-only credential proxy as a proof of concept. Validates the placeholder/rewrite pattern works with git credential helper, `gh` wrapper, and Claude CLI without requiring OpenShell binary.

2. **Phase 2 — Sidecar Supervisor (Strategy 1):** Add OpenShell Supervisor binary, network namespace isolation, Landlock, and seccomp. This is the production target.

3. **Phase 3 — OPA policies:** Add L7 inspection with per-session OPA policies generated by the Operator from the session's credential bindings and project settings.
