# NVIDIA OpenShell — Security Model Analysis

> Research date: 2026-06-03
> Source: https://github.com/NVIDIA/OpenShell (commit f954e592)
> Implementation status: **Integrated** — Supervisor v0.0.56, file mode, validated on ROSA OpenShift (kernel 5.14+)
> Implementation record: [openshell-runner-adaptation.md](openshell-runner-adaptation.md)

## Overview

OpenShell is a Rust-based sandbox runtime for AI agents. It wraps tools like Claude Code, Codex, or Copilot in a hardened execution environment with defense-in-depth isolation. It provides a security cage that the agent runs inside, enforcing filesystem, network, process, and credential policies via declarative YAML.

- **Language**: Rust (core runtime), Python (SDK/bindings)
- **License**: Apache 2.0
- **Status**: Alpha (single-developer mode)
- **Compute drivers**: Docker, Podman, Kubernetes, MicroVM

## Architecture (Three Components)

| Component | Role |
|-----------|------|
| **Gateway** | Authenticated control plane (gRPC + mTLS). Stores providers, policies, sandbox state in a database. |
| **Supervisor** (`openshell-sandbox`) | The security boundary. Runs inside the container alongside the agent. Enforces policy, manages credentials, runs the proxy. |
| **CLI/SDK/TUI** | User-facing. Creates sandboxes, manages providers, attaches to sessions. |

## How It Keeps Credentials From Agents — The Placeholder/Proxy Pattern

OpenShell uses a credential proxy rewrite architecture that ensures agents never see real secrets.

### Flow

1. **User registers a provider** (e.g., `openshell provider create claude --from-env`). The real `ANTHROPIC_API_KEY` is stored in the Gateway database.

2. **Sandbox is created** with `--provider claude`. At startup, the Supervisor calls the Gateway's `GetSandboxProviderEnvironment` gRPC endpoint to fetch credentials.

3. **Real secrets stay in Supervisor memory only.** The Supervisor injects placeholder values into the agent's environment:
   ```
   ANTHROPIC_API_KEY=openshell:resolve:env:ANTHROPIC_API_KEY
   ```
   The agent process never sees the real token.

4. **When the agent makes an API call** (e.g., to `api.anthropic.com`), the request goes through the sandbox's HTTP CONNECT proxy. The proxy rewrites the placeholder back to the real secret in the outbound request before forwarding it upstream.

5. **E2E tests verify** that raw secrets are never present in the child process environment — only the `openshell:resolve:env:*` placeholders.

Even if the agent dumps its own environment variables, reads `/proc/self/environ`, or logs its env, it only sees placeholder strings. The real credentials exist exclusively in the Supervisor process memory space, which runs at higher privilege.

### Supported Provider Types

| Type | Environment Variables Injected | Typical Use |
|------|-------------------------------|-------------|
| `claude` | `ANTHROPIC_API_KEY`, `CLAUDE_API_KEY` | Claude Code, Anthropic API |
| `github` | `GITHUB_TOKEN`, `GH_TOKEN` | GitHub API, `gh` CLI |
| `gitlab` | `GITLAB_TOKEN`, `GLAB_TOKEN`, `CI_JOB_TOKEN` | GitLab API, `glab` CLI |
| `nvidia` | `NVIDIA_API_KEY` | NVIDIA API Catalog |
| `openai` | `OPENAI_API_KEY` | OpenAI SDK |
| `copilot` | `COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, `GITHUB_TOKEN` | GitHub Copilot |
| `generic` | User-defined | Any service with custom credentials |

## Five Isolation Layers

### 1. Network Isolation (Linux Network Namespaces + veth Pairs)

The agent runs in a separate network namespace with only one reachable IP: the proxy at `10.200.0.1`.

- All outbound traffic MUST pass through the HTTP CONNECT proxy
- iptables rules LOG + REJECT any bypass attempts (direct connections ignoring proxy env vars)
- A bypass monitor watches `dmesg` for these events in real-time
- SSRF protection: DNS results are validated against RFC 1918/loopback ranges before establishing upstream connections

**IP addressing**:

| Component | IP Address | Purpose |
|-----------|-----------|---------|
| Host veth interface | `10.200.0.1/24` | Proxy listener endpoint |
| Sandbox veth interface | `10.200.0.2/24` | Sandboxed process gateway |
| Default route (in sandbox namespace) | `via 10.200.0.1` | Routes all traffic to proxy |

**Bypass detection iptables rules** (OUTPUT chain in sandbox namespace):

| Priority | Rule | Target | Purpose |
|----------|------|--------|---------|
| 1 | `-d 10.200.0.1/32 -p tcp --dport 3128` | `ACCEPT` | Allow connections to proxy |
| 2 | `-o lo` | `ACCEPT` | Allow loopback traffic |
| 3 | `-m conntrack --ctstate ESTABLISHED,RELATED` | `ACCEPT` | Allow response packets |
| 4 | `-p tcp --syn ... --log-prefix "openshell:bypass:{ns}:"` | `LOG` | Log TCP bypass attempts |
| 5 | `-p tcp` | `REJECT` | Reject TCP bypass attempts |
| 6 | `-p udp ... --log-prefix "openshell:bypass:{ns}:"` | `LOG` | Log UDP bypass attempts |
| 7 | `-p udp` | `REJECT` | Reject UDP bypass attempts |

### 2. Process Isolation (Pre-exec Enforcement)

After fork but before exec, a strict enforcement sequence runs:

1. `setpgid` — isolate process group
2. `setns` — enter network namespace
3. `harden_child_process` — apply resource limits
4. `drop_privileges` — switch to unprivileged user
5. `sandbox::apply` — Landlock + seccomp

**Resource hardening**:

| Mechanism | Setting | Purpose |
|-----------|---------|---------|
| `RLIMIT_CORE` | 0 | No core dumps (prevents sensitive memory leaks) |
| `RLIMIT_NPROC` | 512 | Prevents fork bombs |
| `PR_SET_DUMPABLE` | 0 | Blocks ptrace attach |
| `PR_SET_NO_NEW_PRIVS` | 1 | No setuid escalation |

**Privilege drop verification**: After `setuid`/`setgid` to the unprivileged user, the supervisor attempts `setuid(0)` and confirms it returns `EPERM` — proving privileges cannot be regained.

**Environment sanitization**: `scrub_sensitive_env` removes `OPENSHELL_SSH_HANDSHAKE_SECRET` before spawning the child process.

### 3. Filesystem Isolation (Landlock LSM)

Landlock provides kernel-level filesystem access control via explicit allowlists.

**Two-phase application**:
1. **Phase 1 (as root)**: `landlock::prepare()` opens `PathFd`s for all allowed paths while the supervisor still has root privileges
2. **Phase 2 (unprivileged)**: Inside the `pre_exec` closure, after `drop_privileges()`, `restrict_self()` applies the Landlock ruleset — this does not require root

**Compatibility modes**:
- `best_effort` — logs a warning and continues without filesystem isolation if the kernel lacks Landlock
- `hard_requirement` — aborts sandbox startup if Landlock cannot be enforced

Credentials are never written to the sandbox filesystem. They exist only in Supervisor memory.

### 4. Syscall Filtering (seccomp-BPF)

Three layers of seccomp filters:

| Filter | Target | Blocked Syscalls |
|--------|--------|-----------------|
| **Supervisor Prelude** | Supervisor process | `mount`, `umount2`, `pivot_root`, `init_module`, `delete_module`, `bpf`, `perf_event_open`, `userfaultfd` |
| **Clone3 Filter** | Child process | `clone3` → `ENOSYS` (forces glibc fallback to `clone` where flags can be filtered) |
| **Main Runtime Filter** | Child process | `ptrace`, `memfd_create`, `io_uring_setup`; socket domains `AF_PACKET`, `AF_NETLINK`, `AF_BLUETOOTH`, `AF_VSOCK` |

In `NetworkMode::Block`, additionally blocks `AF_INET` and `AF_INET6` to deny all network access.

### 5. L7 Protocol Inspection (OPA + MITM Proxy)

For endpoints configured with `protocol: rest`, the proxy performs deep application-layer inspection:

1. **TLS termination**: The Supervisor generates an ephemeral per-sandbox CA at startup. The CA cert is injected into the child process via `SSL_CERT_FILE`. The proxy issues on-the-fly certificates for each intercepted hostname (cached up to 256 entries).

2. **Request parsing**: Each HTTP request is parsed. Paths are canonicalized to prevent `../` and `%2e%2e` traversal bypasses.

3. **OPA evaluation**: Each request is evaluated against Rego rules with input containing `(host, port, binary_path, http_method, canonicalized_path)`.

4. **Credential redaction**: Placeholders in request URIs are resolved by the proxy, but the **redacted** path is sent to OPA to prevent secrets leaking into policy logs.

5. **Enforcement modes**: `Audit` (log but forward) or `Enforce` (block with 403 Forbidden).

**Example L7 policy** (GitHub API read-only):

```yaml
network_policies:
  github_api:
    name: github-api-readonly
    endpoints:
      - host: api.github.com
        port: 443
        protocol: rest
        access: read-only
    binaries:
      - { path: /usr/bin/curl }
```

## Policy Engine (OPA/Rego)

- Policies are YAML-defined, compiled to OPA Rego rules evaluated by `regorus` (pure-Rust Rego evaluator)
- In production (gRPC mode), policies are fetched from the Gateway and hot-reloaded every 30 seconds
- Binary identity: the proxy resolves which binary is making each request via `/proc` inspection + SHA256 TOFU (Trust On First Use) cache — if a binary is modified after first use, the connection is denied
- Formal verification available via `openshell-prover` crate

**Key OPA rules**:

| Rule | Returns | Purpose |
|------|---------|---------|
| `allow_network` | `bool` | L4 allow/deny decision |
| `network_action` | `"allow"` or `"deny"` | L4 routing decision |
| `deny_reason` | `string` | Human-readable denial reason |
| `matched_network_policy` | `string` | Matched policy rule name |
| `matched_endpoint_config` | `object` | L7 inspection configuration |
| `allow_request` | `bool` | Per-request HTTP allow/deny |
| `request_deny_reason` | `string` | L7-specific denial reason |

## PKI and TLS

### Gateway mTLS (Default)

- Three-tier PKI: Cluster CA → Server cert (Gateway TLS termination) → Client cert (CLI + sandbox pods)
- Shared client cert for CLI and all sandbox pods (individual sandbox identity via `x-sandbox-id` gRPC header)
- Ephemeral CA key: used only during generation, not stored in Kubernetes
- Long-lived certificates (effectively never expire)
- Reconciliation: bootstrap checks existing K8s secrets, validates PEM markers, regenerates if malformed, triggers rollout restart on rotation

### Sandbox MITM CA (L7 Inspection)

- Generated fresh per sandbox lifecycle — completely separate from the cluster PKI
- `SandboxCa::generate()` creates a self-signed root with `KeyCertSign` and `CrlSign` usages
- `CertCache` maintains per-hostname leaf certificates (up to 256)
- Upstream verification uses Mozilla root store + system CA paths
- CA cert written to `/etc/openshell-tls/ca-cert.pem`, injected via `SSL_CERT_FILE` and `NODE_EXTRA_CA_CERTS`

### Authentication Modes

| Feature | mTLS | Cloudflare JWT |
|---------|------|----------------|
| **Transport** | HTTPS (mTLS) | HTTPS (Plain) or HTTP |
| **Credential** | Client Certificate | JWT Bearer Token |
| **Gateway Flag** | Default | `allow_unauthenticated=true` |
| **CLI Command** | `gateway add --local` | `gateway login` |

## Audit Logging (OCSF)

OpenShell uses the Open Cybersecurity Schema Framework (OCSF) standard for structured security logging:

- Network decisions (allow/deny) with matched policy name and denial reason
- Process lifecycle events
- Security findings (e.g., Landlock unavailable on kernel)
- Bypass detection events with remediation hints
- Configuration changes

## Relevance to Ambient

> **Status: Implemented.** The Supervisor (file mode, v0.0.56) is integrated into the
> runner. See [openshell-runner-adaptation.md](openshell-runner-adaptation.md) for
> full implementation details.

OpenShell operates at a different layer than Ambient but is directly complementary:

- **Ambient** orchestrates *which* agents run, *when*, *where*, and *with what prompt/context*
- **OpenShell** provides the *sandbox runtime* that those agents execute inside

The runner now uses OpenShell's Supervisor to add intra-container isolation
(Landlock, seccomp, network namespace, L7 proxy) that is significantly more
granular than container-level SecurityContext and NetworkPolicy alone.

### Integration Points (Implemented)

| Ambient Component | OpenShell Equivalent | Integration Status |
|-------------------|---------------------|-------------------|
| Runner container (SecurityContext) | Supervisor (Landlock + seccomp + netns) | **Implemented** — Supervisor wraps Claude CLI; 7 capabilities granted to runner |
| Runner NetworkPolicy | Network namespace + proxy + OPA | **Implemented** — per-binary network ACLs via Rego policy; TLS proxy enforces endpoint allowlist |
| K8s Secret env var injection | Provider placeholder/proxy rewrite | **Deferred** — LLM credentials still in runner env; integration credentials isolated via MCP sidecars |
| Runner pod RBAC | Binary identity + TOFU cache | **Implemented** — policy `binaries` list restricts which executables can access each endpoint |

### What We Learned During Implementation

Key divergences from this analysis that were discovered during implementation:

1. **File mode eliminates Gateway dependency.** The Supervisor reads policy from
   local files (`--policy-rules`, `--policy-data`). No gRPC Gateway, no mTLS PKI,
   no provider registration. Policy is distributed via K8s ConfigMap.

2. **7 capabilities required, not just NET_ADMIN.** The Supervisor's `pre_exec`
   closure calls `setgroups`/`setgid`/`setuid` (requires SETUID, SETGID), `chown`
   (requires CHOWN), mount operations (requires SYS_ADMIN), and process inspection
   (requires SYS_PTRACE).

3. **Landlock ABI compatibility.** The Supervisor detects the kernel's Landlock ABI
   version at runtime (`abi:v5` on kernel 5.14+) and applies rules compatible with
   that version. The `best_effort` mode ensures graceful degradation.

4. **OCSF logging is production-ready.** The structured log format provides clear
   diagnostics for each sandbox setup phase, making production troubleshooting
   straightforward.
