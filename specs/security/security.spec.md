# Security Specification

## Purpose

The Ambient Code Platform runs agentic AI sessions inside Kubernetes. Each session is a
pod that executes an LLM-powered runner, accesses external services (Vertex AI, GitHub,
Jira), and stores results via the API server.

This specification defines who can do what. Six identity boundaries govern the platform:
an SRE-managed Control Plane that reconciles state across Projects, per-session
ServiceAccounts that isolate runner pods from each other, user SSO tokens that scope
runner authorization to the creating user, per-Project LLM credentials (Vertex AI),
per-Project integration credentials (GitHub/GitLab/Jira/etc.), and a Project-scoped
build agent SA for OpenShift CI/CD workflows.

**Critical gap:** all runner sessions in a Project share a ServiceAccount with unscoped
Secret access. Any session can read another session's runner tokens. This spec closes
that gap with per-session Roles restricted by `resourceNames`.

**Terminology:** Each Project is realized as a single Kubernetes namespace. This spec
uses "Project" for the Ambient isolation boundary and "namespace" only when referring
to the Kubernetes primitive directly.

## Accounts and Tokens

### Control Plane Identities

| Identity | Type | Owner | Scope | Lifetime | Purpose |
|----------|------|-------|-------|----------|---------|
| `ambient-control-plane` | K8s ServiceAccount | SRE | Cluster (ClusterRole) | Long-lived (token Secret) | Watches API server, reconciles sessions/projects to K8s, writes status back |
| `ambient-control-plane` OIDC token | OAuth2 client_credentials | SRE | API server | Auto-refreshed (30s buffer) | CP authenticates to API server for session/credential CRUD |

### Platform Service Identities

| Identity | Type | Owner | Scope | Lifetime | Purpose |
|----------|------|-------|-------|----------|---------|
| `backend-api` | K8s ServiceAccount | SRE | Cluster (ClusterRole) | Pod lifetime | Backend API: manages CRs, mints session tokens, validates user tokens |
| `frontend` | K8s ServiceAccount | SRE | Cluster (ClusterRole) | Pod lifetime | Frontend: TokenReview and SubjectAccessReview only |

### Session Runtime Identities

| Identity | Type | Owner | Scope | Lifetime | Purpose |
|----------|------|-------|-------|----------|---------|
| `ambient-session-<name>` | K8s ServiceAccount | SRE (created by operator) | Project (Role) | Session lifetime | Per-session runner identity; scoped to own secrets and session CR |
| Runner bot token | K8s TokenRequest | SRE (minted by operator) | Session-specific | Mounted + refreshed by kubelet | Runner authenticates to K8s API and API server for status/credential ops |
| Runner AGUI token | UUID | SRE (generated per session) | Session-specific | Session lifetime | Authenticates inbound AG-UI requests to runner pod (bearer validation) |
| CP RSA-encrypted session token | RSA + OIDC exchange | SRE | Session-specific | On-demand (per request) | Runner fetches API token from CP `/token` endpoint using encrypted session ID |

### User Authentication

| Identity | Type | Owner | Scope | Lifetime | Purpose |
|----------|------|-------|-------|----------|---------|
| User SSO token | OIDC (Red Hat SSO) | User | User's RBAC scope | SSO session TTL | User authenticates to frontend/backend; propagated as `caller_token` to runner |

### Project Credentials

| Identity | Type | Owner | Scope | Lifetime | Purpose |
|----------|------|-------|-------|----------|---------|
| `Credential(provider=vertex)` | GCP service account key | User | Project | Until rotated | Vertex AI LLM inference; stored in API server, materialized as K8s Secret per Project |
| `Credential(provider=github)` | PAT or GitHub App token | User | Project | Until rotated | Git operations; fetched at runtime, written to ephemeral storage, cleared per turn |
| `Credential(provider=gitlab)` | PAT | User | Project | Until rotated | GitLab repository access |
| `Credential(provider=jira)` | API token | User | Project | Until rotated | Jira issue tracking integration |
| `Credential(provider=google)` | OAuth2 token | User | Project | Until rotated | Google Workspace integrations |
| `Credential(provider=kubeconfig)` | Kubeconfig | User | Project | Until rotated | Cross-cluster Kubernetes operations |

### Build Agent Identity (Proposed)

| Identity | Type | Owner | Scope | Lifetime | Purpose |
|----------|------|-------|-------|----------|---------|
| `ambient-agent` | K8s ServiceAccount | SRE | Single Project (Role) | Long-lived | OpenShift build agent: BuildConfig, ImageStream, deploy within one Project |

## Requirements

### Requirement: Control Plane Identity Isolation

The Control Plane SA SHALL be the only identity that spans Projects. Runner containers
MUST NOT mount or inherit the CP token. The CP SHALL create per-session SAs with scoped
tokens rather than sharing its own.

#### Scenario: Runner cannot access CP token

- GIVEN a runner pod in a Project
- WHEN the pod enumerates available ServiceAccount tokens
- THEN no CP token is present in the pod's filesystem or environment

#### Scenario: CP reconciles across Projects

- GIVEN the Control Plane is running
- WHEN a new session is created in any Project
- THEN the CP reconciles the session to Kubernetes resources in that Project's namespace
- AND uses its own cluster-scoped SA for cross-Project operations

### Requirement: Vertex AI Credential Scoping

Vertex AI credentials SHALL be scoped per Project. The credential token MUST be
write-only in the API (never returned in GET responses). The runner SHALL fetch
credentials at runtime via authenticated API calls.

#### Scenario: Credential write-only enforcement

- GIVEN a user creates a `Credential(provider=vertex)` in their Project
- WHEN another user calls `GET /credentials/{id}`
- THEN the response contains metadata but the `token` field is absent

#### Scenario: Credential materialization

- GIVEN a Project has a Vertex credential
- WHEN a runner pod is provisioned in that Project
- THEN the CP resolves the credential and writes the service account key into a K8s Secret
- AND the runner pod mounts this secret for `GOOGLE_APPLICATION_CREDENTIALS`

#### Scenario: Credential rotation

- GIVEN a Vertex credential is updated via the API
- WHEN the next session is provisioned in that Project
- THEN the CP re-resolves the credential and writes the updated key

### Requirement: User Token Propagation

The runner SHALL operate with the creating user's authorization context. The runner
MUST NOT access resources the creating user cannot access.

#### Scenario: User SSO token passed to runner

- GIVEN a user authenticates via SSO and creates a session
- WHEN a human interacts via AG-UI
- THEN their bearer token is passed through as `caller_token`
- AND the runner uses this token for API calls, falling back to the bot token only if expired

#### Scenario: Cross-user credential access blocked

- GIVEN user A creates a session
- WHEN user B's token is used to access user A's session credentials
- THEN the backend returns 403 Forbidden

#### Scenario: Bot token scoped to session

- GIVEN a runner pod with a bot token
- WHEN the bot token is used for API calls
- THEN access is restricted to the specific session's resources within the Project

### Requirement: Integration Credential Isolation

Integration credentials SHALL be Project-scoped. Projects MUST NOT access each other's
credentials. Credential tokens SHALL be write-only in the API.

#### Scenario: Cross-Project credential access blocked

- GIVEN Project A has a GitHub credential
- WHEN a runner in Project B attempts to fetch that credential
- THEN the request is denied

#### Scenario: Runner fetches credential at runtime

- GIVEN a Project has a GitHub credential
- WHEN a runner pod in that Project requests the credential token
- THEN the token is returned via the restricted endpoint
- AND the runner writes it to ephemeral storage
- AND the credential is cleared after each turn

#### Scenario: Token fetch restricted to cluster-internal callers

- GIVEN a valid credential token request
- WHEN the caller is not cluster-internal
- THEN the request is denied to prevent token exfiltration

### Requirement: MCP Credential Lifecycle

MCP server credentials SHALL follow the same Project-scoped isolation as other
integration credentials. The Control Plane SHOULD support dynamic credential updates
without requiring full pod restarts.

#### Scenario: Sidecar mode credential update

- GIVEN an MCP sidecar running alongside a runner
- WHEN the Project's MCP credentials are updated
- THEN the CP triggers a pod rolling restart with updated environment

#### Scenario: Pod mode credential update (proposed)

- GIVEN an MCP server running as an independent Pod
- WHEN the Project's MCP credentials are updated
- THEN the CP updates the MCP Pod configuration without affecting the runner

### Requirement: Per-Session Service Account Isolation

Each session MUST have a ServiceAccount that can only access its own resources.
Sessions MUST NOT be able to read other sessions' runner tokens from K8s Secrets.

#### Scenario: Session cannot read another session's secrets

- GIVEN Session A and Session B running in the same Project
- WHEN Session A attempts to read Session B's runner token Secret
- THEN the request is denied by RBAC (`resourceNames` restriction)

#### Scenario: Per-session Role restricts Secret access

- GIVEN a new session is created
- WHEN the operator provisions the session SA
- THEN the Role restricts Secret access to `ambient-runner-token-<sessionName>` and shared read-only secrets
- AND AgenticSession access is restricted to `<sessionName>`

#### Scenario: NetworkPolicy isolates session pods

- GIVEN a session pod is running
- WHEN another session's pod attempts to connect
- THEN the NetworkPolicy blocks the traffic
- AND only the session's own pods and the Control Plane can communicate

#### Scenario: Shared secrets mounted read-only

- GIVEN Project-wide secrets exist (e.g., Vertex credentials)
- WHEN a session pod needs access
- THEN the secrets are mounted as read-only volumes
- AND they are not accessible via the K8s API from the session SA

### Requirement: Per-Session SA Target State

Each session SA SHALL be restricted to the following resources:

| Resource | Allowed Names | Verbs |
|----------|--------------|-------|
| Secrets | `ambient-runner-token-<sessionName>`, shared secrets (read-only mount) | get |
| Pods | Labeled `ambient-code/session=<sessionName>` | get, list, watch |
| AgenticSessions | `<sessionName>` | get, update (status only) |
| SelfSubjectAccessReviews | (any) | create |

### Requirement: Build Agent SA Scoping (OpenShift)

The build agent SA SHALL be bound to a single Project. It MUST NOT access other
Projects, nodes, or cluster-scoped resources. It MUST NOT create or modify CRDs,
ClusterRoles, or ClusterRoleBindings.

#### Scenario: Build agent deploys within Project

- GIVEN a build agent SA bound to a Project
- WHEN the agent triggers a BuildConfig
- THEN the build runs within that Project's namespace
- AND images are pushed to the internal registry via `system:image-builder`

#### Scenario: Build agent cannot escalate

- GIVEN a build agent SA bound to a Project
- WHEN the agent attempts to create a ClusterRole
- THEN the request is denied

### Requirement: Build Agent Permissions

The build agent SA SHALL have the following permissions within its Project:

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

## Credential Authorization Model

This section defines how credentials are authorized at runtime. For credential Kind schemas,
API endpoints, and provider enum definitions, see the
[Ambient Data Model Spec](../api/ambient-model.spec.md).

### Requirement: Project-Scoped Credential Sharing

Credentials SHALL belong to a Project. All agents in the Project SHALL share the Project's
credentials automatically — no explicit sharing or per-credential RoleBindings needed. At
session start, the resolver SHALL list all credentials in the agent's Project and return
the matching credential for each requested provider.

This follows the Kubernetes resource model:

| Ambient | Kubernetes Analogy | Relationship |
|---------|-------------------|--------------|
| Project | Namespace | Isolation boundary, owns child resources |
| Agent | Deployment | Mutable definition, runs workloads |
| Session | Pod | Ephemeral execution, created from Agent |
| Credential | Secret | Project-scoped secret, available to all workloads in the Project |

Named patterns:
- **Project Robot Account** — credential created in a Project; all agents use it automatically.
- **Multi-Project credential** — create the same credential (same PAT) in multiple Projects. Each Project gets its own Credential record.
- **No credential** — Projects without credentials run sessions without provider integrations.

#### Scenario: All agents share Project credentials

- GIVEN a Project has a GitHub credential
- WHEN any agent in that Project starts a session
- THEN the runner can fetch the GitHub credential token

#### Scenario: Cross-Project credential isolation

- GIVEN Project A and Project B each have credentials
- WHEN an agent in Project A requests credentials
- THEN only Project A's credentials are returned

### Requirement: Token Reader Role Grant

The `credential:token-reader` role SHALL be granted to the runner service account by the
platform at session start. It MUST NOT be granted via user-facing `POST /role_bindings`.
It is a platform-internal binding managed by the operator.

Credential CRUD SHALL be governed by the caller's project-level role — `project:owner` and
`project:editor` can create/update/delete credentials; `project:viewer` can list/read
metadata.

#### Scenario: Runner can fetch token

- GIVEN a runner SA with `credential:token-reader` bound at session start
- WHEN the runner calls `GET /projects/{id}/credentials/{cred_id}/token`
- THEN the raw token is returned

#### Scenario: Human user cannot fetch token

- GIVEN a human user without `credential:token-reader`
- WHEN they call `GET /projects/{id}/credentials/{cred_id}/token`
- THEN the request is denied with 403

### Requirement: Proxy Authentication

All backend paths not mapped to a native `/api/ambient/v1/...` endpoint SHALL be forwarded
verbatim to the backend service. The API server SHALL authenticate the caller, inject
service credentials, then proxy the request — preserving method, path, query string, body,
and response status.

Runner-internal endpoints (called by runner pods at runtime):
- `POST /api/projects/{p}/agentic-sessions/{s}/github/token` — get a GitHub token for a session
- `GET /api/projects/{p}/agentic-sessions/{s}/credentials/{provider}` — fetch credential by provider
- `POST /api/projects/{p}/agentic-sessions/{s}/runner/feedback` — runner feedback

These endpoints MUST validate the caller is cluster-internal to prevent token exfiltration.

#### Scenario: External caller blocked from runner endpoints

- GIVEN an external client with a valid token
- WHEN they call a runner-internal endpoint
- THEN the request is denied because the caller is not cluster-internal

## Security Boundary Summary

```
+------------------------------------------------------------------+
|                        Cluster                                   |
|                                                                  |
|  +---------------------------+  +-----------------------------+  |
|  | ambient-code (platform)   |  | Project A                   |  |
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
3. Integration credentials are Project-scoped and fetched at runtime, never baked in
4. The Control Plane SA is the only identity that spans Projects
5. MCP lifecycle (sidecar vs. pod) is determined by operational requirements, not security compromise

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| Agent ownership via RBAC, not a hardcoded FK | Ownership is expressed as a RoleBinding (`scope=agent`, `scope_id=agent_id`). Enables multi-owner and delegated ownership consistently across all Kinds. |
| Credential is Project-scoped, like a Kubernetes Secret | Credentials live inside a Project. All agents in the Project share them automatically. Duplication across Projects is intentional and explicit — each Project gets its own Credential record, same as Kubernetes Secrets in different Namespaces. |
| Credential token is write-only | Prevents token exfiltration via the standard REST API. Raw token only surfaced to runners via the runtime credentials path, not to end users. |
| Four-scope RBAC (`global`, `project`, `agent`, `session`) | Credential access is implicit via Project membership — no dedicated `credential` scope needed. Simpler model with fewer moving parts. |
| Credential CRUD governed by project roles | `project:owner` and `project:editor` can manage credentials. No separate `credential:owner` / `credential:reader` roles — project roles cover it. |
| `agent:runner` role | Pods get minimum viable credential: read agent definition, push session messages, send inbox. |
| Union-only permissions | No deny rules — simpler mental model for fleet operators. |
| Token stored in database, encrypted at rest | Single authoritative store. A future Vault integration can be adopted by pointing the DB row at a Vault path without changing the API surface. |
| `google` token serialized as a string | Service Account JSON is serialized into the single `token` field. Keeps the schema uniform across all providers. |
| No validation on creation | First-use error is acceptable. Avoids a network call to the provider at creation time and the failure modes that come with it. |
| Credential rotation is user-managed | Users update the token via `PATCH` or `acpctl credential update`. No platform-side rotation or expiry tracking. |
| No migration utility for existing K8s Secrets | Users re-enter credentials via the new API. The old Secret-based path is removed when the new API is live. |
| Dedicated tokens, not personal credentials | Users are expected to create dedicated Robot Accounts or PATs for each Project, not share their personal credentials. Each Project gets its own Credential records. |

## References

- [Ambient Data Model Spec](../api/ambient-model.spec.md) — Credential/RBAC schemas, endpoints, provider enum
- [Security Standards](../standards/security/security.spec.md)
- [User Token Authentication ADR](../../docs/internal/adr/0002-user-token-authentication.md)
