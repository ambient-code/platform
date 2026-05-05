# Ambient Code Platform: Security Specification

**Status:** Draft
**Authors:** Platform Team
**Last Updated:** 2026-05-05

## Summary

The Ambient Code Platform runs agentic AI sessions inside Kubernetes. Each session is a
pod that executes an LLM-powered runner, accesses external services (Vertex AI, GitHub,
Jira), and stores results via the API server.

This specification defines who can do what. Six identity boundaries govern the platform:
an SRE-managed Control Plane that reconciles state across namespaces, per-session
ServiceAccounts that isolate runner pods from each other, user SSO tokens that scope
runner authorization to the creating user, per-project LLM credentials (Vertex AI),
per-project integration credentials (GitHub/GitLab/Jira/etc.), and a namespace-scoped
build agent SA for OpenShift CI/CD workflows.

**The critical gap today:** all runner sessions in a namespace share a ServiceAccount with
unscoped Secret access. Any session can read another session's runner tokens. This spec
closes that gap with per-session Roles restricted by `resourceNames`.

## Accounts and Tokens

| Identity | Type | Owner | Scope | Lifetime | Purpose |
|----------|------|-------|-------|----------|---------|
| `ambient-control-plane` | K8s ServiceAccount | SRE | Cluster (ClusterRole) | Long-lived (token Secret) | Watches API server, reconciles sessions/projects to K8s, writes status back |
| `ambient-control-plane` OIDC token | OAuth2 client_credentials | SRE | API server | Auto-refreshed (30s buffer) | CP authenticates to API server for session/credential CRUD |
| `backend-api` | K8s ServiceAccount | SRE | Cluster (ClusterRole) | Pod lifetime | Backend API: manages CRs, mints session tokens, validates user tokens |
| `frontend` | K8s ServiceAccount | SRE | Cluster (ClusterRole) | Pod lifetime | Frontend: TokenReview and SubjectAccessReview only |
| `ambient-session-<name>` | K8s ServiceAccount | SRE (created by operator) | Namespace (Role) | Session lifetime | Per-session runner identity; scoped to own secrets and session CR |
| Runner bot token | K8s TokenRequest | SRE (minted by operator) | Session-specific | Mounted + refreshed by kubelet | Runner authenticates to K8s API and API server for status/credential ops |
| Runner AGUI token | UUID | SRE (generated per session) | Session-specific | Session lifetime | Authenticates inbound AG-UI requests to runner pod (bearer validation) |
| CP RSA-encrypted session token | RSA + OIDC exchange | SRE | Session-specific | On-demand (per request) | Runner fetches API token from CP `/token` endpoint using encrypted session ID |
| User SSO token | OIDC (Red Hat SSO) | User | User's RBAC scope | SSO session TTL | User authenticates to frontend/backend; propagated as `caller_token` to runner |
| `Credential(provider=vertex)` | GCP service account key | User | Project | Until rotated | Vertex AI LLM inference; stored in API server, materialized as K8s Secret per namespace |
| `Credential(provider=github)` | PAT or GitHub App token | User | Project | Until rotated | Git operations; fetched at runtime, written to ephemeral `/tmp/`, cleared per turn |
| `Credential(provider=gitlab)` | PAT | User | Project | Until rotated | GitLab repository access |
| `Credential(provider=jira)` | API token | User | Project | Until rotated | Jira issue tracking integration |
| `Credential(provider=google)` | OAuth2 token | User | Project | Until rotated | Google Workspace integrations |
| `Credential(provider=kubeconfig)` | Kubeconfig | User | Project | Until rotated | Cross-cluster Kubernetes operations |
| `ambient-agent` (proposed) | K8s ServiceAccount | SRE | Single namespace (Role) | Long-lived | OpenShift build agent: BuildConfig, ImageStream, deploy within one namespace |

## Overview

This document defines the security boundaries for the Ambient Code Platform. It addresses
identity isolation, credential scoping, and the principle of least privilege across the
control plane, runner sessions, and user-facing integrations.

---

## 1. OpenShift Namespace-Scoped Build Agent Service Account

### Purpose

A dedicated ServiceAccount for automated build-and-deploy workflows within a single
OpenShift namespace. This SA enables agentic workflows to build container images via
`BuildConfig`, push to the internal registry, and deploy workloads without requiring
cluster-admin privileges.

### Scope

- Bound to a single namespace (e.g., `ambient-sandbox`)
- Cannot access other namespaces, nodes, or cluster-scoped resources
- Cannot create or modify CRDs, ClusterRoles, or ClusterRoleBindings

### Permissions

| API Group | Resources | Verbs |
|-----------|-----------|-------|
| `build.openshift.io` | `buildconfigs`, `buildconfigs/instantiate`, `builds`, `builds/log` | get, list, watch, create, update, patch, delete |
| `image.openshift.io` | `imagestreams`, `imagestreamtags`, `imagestreamimages` | get, list, watch, create, update, patch, delete |
| `apps` | `deployments`, `statefulsets`, `replicasets` | get, list, watch, create, update, patch, delete |
| `""` (core) | `pods`, `pods/log`, `services`, `configmaps`, `secrets`, `persistentvolumeclaims`, `serviceaccounts`, `events` | get, list, watch, create, update, patch, delete |
| `route.openshift.io` | `routes` | get, list, watch, create, update, patch, delete |
| `batch` | `jobs`, `cronjobs` | get, list, watch, create, update, patch, delete |
| `networking.k8s.io` | `networkpolicies` | get, list, watch, create, update, patch, delete |
| `rbac.authorization.k8s.io` | `roles`, `rolebindings` | get, list, watch, create, update, patch, delete |

Additionally requires the built-in `system:image-builder` role for internal registry push access.

### Rationale

Agentic workflows need to build and deploy without human intervention, but must not
escalate beyond the target namespace. This SA provides the minimal surface area for a
CI/CD agent to operate a full build-test-deploy cycle while remaining invisible to the
rest of the cluster.

---

## 2. Security Boundaries

### 2.1 SRE Boundary: Control Plane Service Account

**Identity:** `ambient-control-plane` ServiceAccount (cluster-scoped RBAC)

**Role:** The single SRE-owned identity that bridges the API server and Kubernetes. The
Control Plane runs as an informer/watcher that:

- Watches the API server (PostgreSQL-backed) for new/modified session and project resources
- Reconciles desired state to Kubernetes (creates Pods, Services, Secrets, ServiceAccounts
  in project namespaces)
- Writes reconciled status back to the API server (phase transitions, runner pod names,
  credential resolution results)

**Current Implementation:**
- SA defined in `components/manifests/base/rbac/control-plane-sa.yaml`
- ClusterRole in `components/manifests/base/rbac/control-plane-clusterrole.yaml`
- Long-lived token via companion Secret (`kubernetes.io/service-account-token`)
- Authenticates to API server via OIDC `client_credentials` flow or static token

**Security Properties:**
- This SA must never be exposed to user workloads
- Runner containers must not mount or inherit the CP token
- The CP creates per-session SAs with scoped tokens (see 2.5) rather than sharing its own

### 2.2 User Boundary: Vertex AI Credentials

**Identity:** Per-project `Credential(provider=vertex)` stored in the API server (PostgreSQL)

**Role:** Provides Vertex AI / Google Cloud authentication for LLM inference. Each project
can bind its own Vertex credential, allowing different teams to use different GCP projects
or service accounts.

**Flow:**
1. User creates `Credential(provider=vertex)` in their project via the API
2. On runner pod provisioning, the Control Plane calls `resolveCredentialIDs()` to look up
   the project's Vertex credential
3. The CP writes the Vertex service account key into a Kubernetes Secret
   (`ambient-vertex`) in the project namespace
4. The runner pod mounts this secret and uses it for `GOOGLE_APPLICATION_CREDENTIALS`

**Security Properties:**
- Vertex credentials are scoped per project, not shared globally
- Credential tokens are write-only in the API (never returned in GET responses)
- The runner fetches credentials at runtime via authenticated API calls, not baked into
  the container image
- Credential rotation requires only updating the `Credential` resource; the CP
  re-resolves on next session provisioning

### 2.3 User Boundary: Red Hat SSO / User Token Propagation

**Identity:** The authenticated user's own SSO token, propagated into the runner

**Role:** Ensures the runner operates with the creating user's authorization context, not
an elevated service identity. The runner's API calls (session status updates, credential
fetches, AG-UI event streaming) are scoped to what the user is allowed to access.

**Flow:**
1. User authenticates via Red Hat SSO (OIDC)
2. The backend mints a per-session K8s ServiceAccount token annotated with the user's
   identity (`ambient-code.io/created-by-user-id`)
3. The runner resolves its bot token via the CP token endpoint (OIDC `client_credentials`
   exchange, encrypted with CP's RSA public key)
4. When a human interacts via AG-UI, their bearer token is passed through as
   `context.caller_token` — the runner uses this token first, falling back to the bot
   token only if expired
5. Backend RBAC enforcement (`enforceCredentialRBAC()`) validates the caller is either the
   session owner or a bot acting on behalf of the owner

**Security Properties:**
- Runner cannot access resources the user cannot access
- Cross-user credential access is blocked (403)
- The bot token is scoped to the specific session's namespace and resources
- Token refresh uses RSA-encrypted session ID exchange, not stored credentials

### 2.4 User Boundary: Integration Credentials

**Identity:** Per-project `Credential(provider=*)` resources — `github`, `gitlab`, `jira`,
`google`, `kubeconfig`, and future providers

**Role:** Users bind external integration credentials to their project. The runner fetches
these at runtime to perform git operations, issue tracking, and other integrations.

**Current Providers (OpenAPI enum):**
- `github` — PAT or GitHub App token for repository access
- `gitlab` — PAT for GitLab repository access
- `jira` — API token for issue tracking
- `google` — OAuth2 token for Google integrations
- `kubeconfig` — Kubernetes config for cross-cluster operations

**Flow:**
1. User creates `Credential(provider=github, ...)` in their project
2. Runner fetches credential token at runtime:
   `GET /api/ambient/v1/projects/{project}/credentials/{id}/token`
3. Token is written to ephemeral files (`/tmp/`) for git credential helper and CLI
   wrappers
4. Credentials are cleared from environment after each turn
   (`clear_runtime_credentials()`)

**Security Properties:**
- Credentials are project-scoped — projects cannot access each other's credentials
- RBAC permissions: `credential:token` required for token fetch (separate from
  `credential:read`)
- Backend validates caller hostname is cluster-internal (`.svc.cluster.local`, `localhost`)
  to prevent token exfiltration
- Credential tokens are write-only in the API (presenter strips `Token` field from GET
  responses)

#### 2.4a Dynamic MCP Credential Watching and Pod Lifecycle

**Current State:** MCP servers run as sidecars (`ambient-mcp` container) injected by the
Control Plane when provisioning runner pods. Credentials for MCP servers are stored in a
shared Secret (`mcp-server-credentials`) keyed by `serverName:userID`.

**Target State:** If the Control Plane continuously watches `Credential` resources for
changes, MCP configurations can be dynamically applied without restarting the runner:

- **Sidecar mode (current):** MCP runs as a sidecar container alongside the runner in the
  same Pod. Suitable for lightweight, session-scoped MCP servers. Limited to the pod's
  lifecycle — cannot be independently restarted or scaled.
- **Pod mode (proposed):** MCP runs as an independent Pod in the project namespace with its
  own lifecycle (readiness probes, independent restarts, resource limits). Required when
  MCP servers need:
  - Independent scaling or resource allocation
  - Persistence across session restarts
  - Shared access across multiple sessions in the same project
  - Long-running connections (databases, message queues) that survive session recycling

**Credential Watch Flow:**
1. Control Plane watches `Credential` resources on the API server (via informer or
   polling)
2. On credential create/update/delete, CP evaluates which sessions or MCP pods are
   affected
3. For sidecar mode: CP triggers a pod rolling restart with updated environment
4. For pod mode: CP creates/updates/deletes MCP Pods with the new credential configuration
5. MCP pods authenticate to external services using the bound credential tokens

### 2.5 SRE Boundary: Per-Session Service Accounts

**Problem Statement:**

Today, all runner sessions within a project namespace share access to each other's
resources. A compromised or misbehaving session can:
- Read other sessions' runner tokens from Kubernetes Secrets
- Access other sessions' mounted credentials
- Interfere with other sessions' pods via the Kubernetes API
- Exfiltrate data from other sessions' PVCs

This is a significant security gap. The shared access model was an expedient choice during
early development but violates the principle of least privilege.

**Current State (partially implemented):**

The operator already creates per-session ServiceAccounts
(`ambient-session-<sessionName>`) with scoped Roles:

```
SA:    ambient-session-<sessionName>
Role:  get/list/watch/create/update/patch AgenticSessions
       create SelfSubjectAccessReviews
       get Secrets
       get/list/update MLflow Experiments
```

However, the Role's `get Secrets` permission is not scoped to the session's own secrets.
Any session SA can read any Secret in the namespace, including other sessions' runner
tokens.

**Target State:**

Each session must have a ServiceAccount that can only access its own resources:

| Resource | Allowed Names | Verbs |
|----------|--------------|-------|
| Secrets | `ambient-runner-token-<sessionName>`, `ambient-vertex` (read-only) | get |
| Pods | Labeled `ambient-code/session=<sessionName>` | get, list, watch |
| AgenticSessions | `<sessionName>` | get, update (status only) |
| SelfSubjectAccessReviews | (any) | create |

**Implementation Requirements:**
1. **Per-session Role:** The operator must generate a Role per session with `resourceNames`
   restrictions on Secrets and AgenticSessions
2. **Label-based pod access:** Use label selectors in NetworkPolicies to restrict inter-pod
   communication to same-session containers only
3. **Secret naming convention:** All session-scoped secrets must follow the pattern
   `ambient-runner-token-<sessionName>` or `<sessionName>-*` to enable `resourceNames`
   restrictions
4. **Shared secrets (read-only):** Project-wide secrets (`ambient-vertex`,
   `ambient-runner-secrets`) should be mounted as read-only volumes rather than accessed
   via the Kubernetes API
5. **NetworkPolicy per session:** Extend the existing `ensureRunnerNetworkPolicy()` to
   create per-session policies that restrict ingress/egress to only the session's own pods
   and the control plane

**Migration Path:**
1. Deploy per-session Roles with `resourceNames` restrictions (backward compatible —
   existing sessions continue to work with broader access until recreated)
2. Update the operator's `regenerateRunnerToken()` to bind the session SA to the new
   scoped Role instead of the namespace-wide Role
3. Audit existing sessions for cross-session secret access (add audit logging for Secret
   GET operations with caller identity)
4. Enforce per-session NetworkPolicies in new sessions; backfill existing sessions during
   next maintenance window

---

## 3. Security Boundary Summary

```
+------------------------------------------------------------------+
|                        Cluster                                   |
|                                                                  |
|  +---------------------------+  +-----------------------------+  |
|  | ambient-code namespace    |  | project namespace           |  |
|  |                           |  |                             |  |
|  | [Control Plane]           |  |  [Session A Pod]            |  |
|  |  SA: ambient-control-plane|  |   SA: ambient-session-aaa   |  |
|  |  - watches API server     |  |   - own secrets only        |  |
|  |  - reconciles to K8s     |  |   - own session CR only     |  |
|  |  - writes status back    |  |   - user's SSO token        |  |
|  |                           |  |   - project vertex cred     |  |
|  | [API Server]              |  |   +------------------+      |  |
|  |  SA: (pod identity)       |  |   | MCP sidecar/pod  |      |  |
|  |  - PostgreSQL backend     |  |   | - integration     |      |  |
|  |  - Credential store      |  |   | - creds from API  |      |  |
|  |  - RBAC enforcement      |  |   +------------------+      |  |
|  |                           |  |                             |  |
|  | [Backend]                 |  |  [Session B Pod]            |  |
|  |  SA: backend-api          |  |   SA: ambient-session-bbb   |  |
|  |  - user token passthrough|  |   - ISOLATED from A         |  |
|  |  - credential RBAC       |  |   - own secrets only        |  |
|  +---------------------------+  +-----------------------------+  |
|                                                                  |
+------------------------------------------------------------------+
```

**Key Invariants:**
1. No runner session can access another session's secrets or tokens
2. No runner session can operate beyond the user's own authorization scope
3. Integration credentials are project-scoped and fetched at runtime, never baked in
4. The Control Plane SA is the only identity that spans namespaces
5. MCP lifecycle (sidecar vs. pod) is determined by operational requirements, not security
   compromise

---

## References

- [Security Standards](../../security-standards.md)
- [User Token Authentication ADR](../adr/0002-user-token-authentication.md)
- [Credential API OpenAPI Spec](../../../components/ambient-api-server/openapi/openapi.credentials.yaml)
- [Control Plane RBAC](../../../components/manifests/base/rbac/control-plane-clusterrole.yaml)
- [Operator Session Handler](../../../components/operator/internal/handlers/sessions.go)
