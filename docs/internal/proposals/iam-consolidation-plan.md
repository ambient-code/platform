# Ambient IAM — Three Improvement Plans

---

## 1. Consolidate Around RH SSO

### The Goal

One issuer. Every token in the system — user, runner, access key, service — comes from or is
validated by RH SSO. No RSA keypairs, no K8s SA minting loops, no two-step exchanges.

### What Changes (by identity type)

#### A. Human users — no change
Already through RH SSO OAuth proxy. OCP issues `sha256~` tokens; the proxy validates them and
injects `X-Forwarded-*` headers. The backend and ambient-api-server already validate JWTs against
the RH SSO JWKS endpoint. This path is already right.

#### B. Access keys — replace K8s SAs with RH SSO service accounts

**Current:** Backend creates `ambient-key-<name>-<uid>` K8s ServiceAccount, creates RoleBinding,
calls `TokenRequest` API, returns JWT. User stores the JWT and sends it as Bearer on every call.
Tracking is done via a `last-used-at` annotation.

**Target:** Backend calls the Keycloak Admin REST API to create a **confidential client** (service
account) in RH SSO. Client credentials (`client_id` / `client_secret`) are returned to the user
once. User calls RH SSO token endpoint to get a short-lived OIDC access token, sends it as Bearer.

- Revocation: delete the client in Keycloak → all future token requests fail immediately
- Role assignment: Keycloak client roles map to `project:admin/edit/view`
- Token introspection: any component can call `/introspect` to verify a key is still active
- No K8s SA objects, no K8s RoleBindings for access keys, no `TokenRequest` calls

#### C. Runner pods — replace K8s SA + RSA exchange with OIDC Token Exchange (RFC 8693)

**Current:**
1. Operator creates `ambient-session-<id>` SA
2. Operator calls `TokenRequest` → stores JWT in Secret `ambient-runner-token-<id>`
3. Pod mounts the Secret, sends JWT to backend
4. Pod also calls control plane `/token` with RSA-encrypted session ID
5. Control plane decrypts with RSA-4096 private key, returns OIDC token
6. Operator refreshes the K8s JWT every 45 minutes

**Target:**
1. OCP automatically projects a short-lived K8s SA token into every pod at
   `/var/run/secrets/kubernetes.io/serviceaccount/token` (standard, no setup needed)
2. On startup, runner calls RH SSO token exchange endpoint:
   ```
   POST /auth/realms/redhat-external/protocol/openid-connect/token
   grant_type=urn:ietf:params:oauth:grant-type:token-exchange
   subject_token=<k8s-projected-sa-token>
   subject_token_type=urn:ietf:params:oauth:token-type:jwt
   client_id=ambient-runner-exchange
   client_secret=<secret>
   requested_token_type=urn:ietf:params:oauth:token-type:access_token
   audience=ambient-platform
   ```
3. RH SSO validates the K8s JWT against the cluster JWKS, issues a scoped OIDC token with custom
   claims: `session_id`, `project`, `role=agent:runner`
4. Runner uses this OIDC token for all downstream API calls (backend, ambient-api-server)
5. Token expiry is handled by standard OIDC refresh (no operator refresh loop needed)

The entire control plane token server, RSA keypair bootstrap, and the 45-minute refresh loop in
the operator go away.

#### D. Service-to-service (control plane, backend SA) — already right or align

- Control plane already uses OIDC client credentials ✓
- Backend SA (`backend-api`) should get its own Keycloak confidential client and use client
  credentials for outbound calls to ambient-api-server (currently uses in-cluster SA token)

---

### RH SSO: What Needs to Be Registered

#### Clients (Confidential, Service Account enabled)

| Client ID | Grant Type | Purpose | Roles Needed |
|---|---|---|---|
| `ambient-control-plane` | client_credentials | Already exists. Control plane → API server | `platform:admin` or equivalent |
| `ambient-backend` | client_credentials | Backend → ambient-api-server auth | `platform:admin` or equivalent |
| `ambient-runner-exchange` | token_exchange | Accept K8s JWT, issue scoped runner token | `token-exchange` permission on realm |
| `ambient-key-manager` | client_credentials | Keycloak Admin API — create/delete access key clients | `manage-clients`, `view-clients` realm roles |
| `ambient-key-<project>-<name>` (dynamic) | client_credentials | Per user-created access key | Project-scoped role (admin/edit/view) |

#### Realm Configuration

| Setting | Value | Why |
|---|---|---|
| Token Exchange feature | Enabled | Required for RFC 8693 runner flow |
| K8s cluster as Identity Provider | Add cluster OIDC endpoint | So RH SSO can validate K8s-issued JWTs |
| Cluster JWKS URL | `https://api.<cluster-domain>:6443/openid/v1/jwks` | RH SSO fetches this to verify runner tokens |
| Client roles | `project:admin`, `project:editor`, `project:viewer` | Assigned to access key clients |
| Custom claim mapper | `session_id`, `project`, `role` on runner tokens | Downstream components read these claims |

#### K8s Secrets Required (in `ambient-code` namespace)

| Secret Name | Keys | Purpose |
|---|---|---|
| `ambient-sso-admin-credentials` | `client_id`, `client_secret` | Keycloak Admin API for access key lifecycle |
| `ambient-runner-exchange-credentials` | `client_id`, `client_secret` | Runner token exchange client |
| `ambient-backend-oidc` | `client_id`, `client_secret` | Backend service-to-service auth |

#### Environment Variables (changes/additions)

| Component | Variable | Value |
|---|---|---|
| Backend | `SSO_ADMIN_CLIENT_ID` | `ambient-key-manager` |
| Backend | `SSO_ADMIN_CLIENT_SECRET` | from Secret |
| Backend | `SSO_REALM_URL` | `https://sso.redhat.com/auth/realms/redhat-external` |
| Control plane | `OIDC_CLIENT_ID` | `ambient-control-plane` (existing) |
| Control plane | `OIDC_CLIENT_SECRET` | from Secret (existing) |
| Runner | `SSO_TOKEN_EXCHANGE_URL` | RH SSO token endpoint |
| Runner | `SSO_EXCHANGE_CLIENT_ID` | `ambient-runner-exchange` |
| Runner | `SSO_EXCHANGE_CLIENT_SECRET` | from Secret |

---

### What Gets Deleted

| Component | What Goes Away |
|---|---|
| Operator | SA creation code for `ambient-session-*` |
| Operator | `TokenRequest` minting code |
| Operator | 45-minute token refresh loop |
| Operator | Secret `ambient-runner-token-*` creation |
| Control plane | Entire `internal/tokenserver/` package |
| Control plane | Entire `internal/keypair/` package |
| Control plane | Secret `ambient-cp-token-keypair` |
| Control plane | `CPTokenListenAddr`, `CPTokenURL`, `ProjectKubeTokenFile` config fields |
| Backend | SA creation in `CreateProjectKey()` |
| Backend | `DeleteProjectKey()` SA/RoleBinding deletion |
| Backend | `ListProjectKeys()` SA label selector query |
| Backend | `updateAccessKeyLastUsedAnnotation()` |
| Manifests | All `ambient-key-*` ClusterRole bindings (no longer static) |

### What Gets Added

| Component | What's New |
|---|---|
| Backend | Keycloak Admin API client (`pkg/keycloak/`) |
| Backend | `CreateProjectKey()` → create Keycloak client, assign roles, return `client_id`+`client_secret` |
| Backend | `DeleteProjectKey()` → delete Keycloak client |
| Backend | `ListProjectKeys()` → list Keycloak clients with `ambient-key-` prefix |
| Runner | OIDC token exchange on startup (call SSO, cache token, refresh before expiry) |
| ambient-api-server | No change — already validates RH SSO JWTs |

---

### Migration Path

1. **Register all clients in RH SSO** and validate token exchange with the cluster
2. **Deploy runner with dual-mode**: try exchange first, fall back to RSA for existing sessions
3. **Deploy operator without SA creation** for new sessions only (existing sessions unaffected)
4. **Once all active sessions are on new path**: remove RSA exchange from control plane
5. **Migrate access keys**: for each existing K8s SA access key, create Keycloak client,
   notify users to re-issue credentials (old K8s SA tokens expire naturally)
6. **Remove old K8s SAs and Secrets**: `kubectl delete sa -l app=ambient-access-key -A`

---

---

## 2. DB RBAC as Source of Truth — Options

### The Problem to Solve

Today a project admin must grant access in two independent systems: K8s RoleBindings (for the
backend/K8s API layer) and DB role_bindings (for ambient-api-server). They're not synced. You
can grant someone in one and forget the other. Neither system knows the other exists.

### Constraint You Can't Remove

K8s enforces RBAC natively for K8s API operations. When the backend does `SSAR` to check if a
user can `list agenticsessions`, K8s itself makes that decision using RoleBindings. You cannot
bypass this without rewriting how K8s works. So K8s RBAC **for K8s operations** always exists.

The question is: *who is the write plane* — where does an admin go to say "give Alice access to
project X", and how does that propagate.

---

### Option A: DB Drives K8s (Reconciliation) — Recommended

**DB is the write plane. K8s RoleBindings are a derived artifact.**

When a user is added to a project in ambient-api-server's `role_bindings` table, a new reconciler
(in the control plane or operator) watches for those changes and creates/deletes the corresponding
K8s RoleBinding automatically.

```
Admin calls: POST /api/ambient/v1/role_bindings
  { user_id: "alice", role_id: "project:editor", scope: "project", scope_id: "my-project" }
    ↓
ambient-api-server writes to role_bindings table
    ↓
Reconciler watches role_bindings (polling or change-data-capture)
    ↓
Reconciler creates K8s RoleBinding in namespace "my-project":
  subject: alice  →  ClusterRole: ambient-project-edit
    ↓
Backend SSAR continues to work unchanged
```

**Role mapping table** (DB role → K8s ClusterRole):

| DB Role | K8s ClusterRole |
|---|---|
| `project:owner` | `ambient-project-admin` |
| `project:editor` | `ambient-project-edit` |
| `project:viewer` | `ambient-project-view` |
| `platform:admin` | cluster-admin or custom |
| `agent:runner` | (no K8s ClusterRole needed — runner uses token exchange) |
| Fine-grained (`credential:token-reader`, etc.) | (DB RBAC only, no K8s mapping needed) |

**What changes:**
- New reconciler in control plane: watches `role_bindings` table, syncs K8s RoleBindings
- Backend permissions handler (`/api/projects/:name/permissions`) delegates writes to
  ambient-api-server instead of directly creating K8s RoleBindings
- Frontend permissions UI calls ambient-api-server instead of backend
- Backend SSAR, middleware — no change

**Tradeoffs:**
- Eventual consistency: DB write → K8s propagation has a lag (aim for < 5s)
- Reconciler needs K8s admin permissions to create RoleBindings
- Fine-grained DB permissions (`credential:token`) have no K8s equivalent — they're DB-only
  and that's fine (ambient-api-server enforces them directly)

---

### Option B: ambient-api-server as Authorization Service

**DB is authoritative. Backend calls ambient-api-server for every authz decision.**

Backend replaces `SelfSubjectAccessReview` calls with HTTP calls to a new ambient-api-server
endpoint: `POST /api/ambient/v1/authz/check`.

```
Backend request arrives with user token
    ↓
Backend extracts user identity from JWT claims (preferred_username)
    ↓
Backend calls: POST /api/ambient/v1/authz/check
  { user: "alice", resource: "agenticsessions", action: "list", project: "my-project" }
    ↓
ambient-api-server queries role_bindings → roles → permissions
Returns: { allowed: true }
    ↓
Backend proceeds (or returns 403)
```

Backend caches results for 30 seconds (same as current SSAR cache).

**What changes:**
- Backend: `globalSSARCache` logic remains, but calls ambient-api-server instead of K8s API
- ambient-api-server: new `/authz/check` endpoint
- K8s RoleBindings: can be removed for project-level user bindings (only system SAs need them)
- The K8s ClusterRoles `ambient-project-admin/edit/view` can be retired for user access

**Tradeoffs:**
- Backend takes a **hard synchronous dependency** on ambient-api-server. If ambient-api-server
  is down, the backend cannot authorize any request.
- Risk of circular dependency if ambient-api-server itself calls backend for anything.
- Eliminates K8s audit trail for user actions (SSAR no longer used).
- K8s RBAC for K8s operations still required for system SAs (operator, control plane, etc.)
- Net result: simpler for users, harder operationally.

---

### Option C: Explicit Split (No Single Source)

Accept that two systems exist but make the split **intentional and documented**:

- **K8s RBAC** owns: "can this identity access this namespace at all" (coarse gate)
- **DB RBAC** owns: "can this identity do this specific action on this resource" (fine-grained)

Both are authoritative for their domain. No overlap. Documented contract.

The only change: everywhere a human admin today has to grant in both systems, replace with a
single API call that writes to both atomically. The backend's permissions handler writes a K8s
RoleBinding **and** a DB role_binding in the same request.

**Tradeoffs:**
- Still two systems, but the dual-write is explicit and visible
- No reconciler needed, no new service dependencies
- Easiest to implement
- Doesn't actually solve the sync problem — just moves the two-write burden to the backend

---

### Recommendation

**Option A** is the right call. It gives you a single human-facing write plane (the DB) while
keeping K8s RBAC functioning as it does today. The backend changes minimally. The reconciler is
a small, focused component (< 200 lines of controller-runtime code).

The reconciler fits naturally in the control plane, which already has K8s admin permissions and
watches resources for reconciliation. Add it alongside the existing project namespace reconciler.

**One thing to decide:** what to do with fine-grained permissions like `credential:token-reader`
that have no K8s equivalent. The answer is: leave them DB-only. K8s RBAC enforces the coarse
gate (can you access the project). DB RBAC enforces the fine-grained gate (can you read a token
within the project). This split is actually correct — they serve different enforcement points.

---

---

## 3. Extend the Credentials Table

### The Goal

Move all provider OAuth tokens (GitHub, GitLab, Google, Jira, Gerrit, CodeRabbit) from the
scattered K8s Secrets in the backend namespace into the `credentials` table in ambient-api-server.
Single audit trail, single access control model, single API.

### Schema Change

Add `user_id` and `scope` columns (one migration):

```sql
ALTER TABLE credentials
  ADD COLUMN user_id  TEXT,
  ADD COLUMN scope    TEXT NOT NULL DEFAULT 'project';

-- scope = 'project': project_id set, user_id null  (existing behavior)
-- scope = 'user':    user_id set,    project_id may be null
```

New unique index for user credentials:
```sql
CREATE UNIQUE INDEX credentials_user_provider_url
  ON credentials (user_id, provider, url)
  WHERE scope = 'user' AND deleted_at IS NULL;
```

### New Routes (ambient-api-server)

```
GET    /api/ambient/v1/users/me/credentials
POST   /api/ambient/v1/users/me/credentials
DELETE /api/ambient/v1/users/me/credentials/{id}
GET    /api/ambient/v1/users/me/credentials/{id}/token
```

The `/me` route resolves `user_id` from the JWT `preferred_username` claim — no user ID in URL.

### What Moves From K8s Secrets to DB

| K8s Secret (backend namespace) | → DB credential |
|---|---|
| `gitlab-user-tokens` (key: userID) | `scope=user, provider=gitlab, token=<PAT>, url=<instance_url>` |
| `google-creds-{hash}` | `scope=user, provider=google, token=<access_token>`, refresh token in `annotations` JSON |
| `jira-creds-{hash}` | `scope=user, provider=jira, token=<api_token>, url=<jira_url>, email=<email>` |
| `gerrit-creds-{hash}` | `scope=user, provider=gerrit, token=<http_token>, url=<instance_url>` (one row per instance) |
| `coderabbit-creds-{hash}` | `scope=user, provider=coderabbit, token=<api_key>` |
| `{session}-{provider}-oauth` | `scope=project, provider=<provider>` + session ID in `labels` JSON |

### What Stays Where It Is

| Token | Stays Because |
|---|---|
| `oauth-callbacks` Secret | Transient state (UUID-keyed, short TTL) — a K8s Secret or Redis is fine |
| GitHub App installation tokens | Never stored; minted on demand from private key |
| Runner pod K8s JWT | Changes to OIDC exchange (see plan 1) — not a credential to store |

### What Changes in the Backend

Every `StoreX()` / `GetX()` / `DeleteX()` function in `handlers/oauth.go` and `handlers/secrets.go`
becomes an API call to ambient-api-server instead of a K8s Secret operation:

```
StoreGitLabToken(userID, token)  →  POST /users/me/credentials {provider: "gitlab", token: ...}
GetGitLabToken(userID)           →  GET  /users/me/credentials?provider=gitlab + /token
DeleteGitLabToken(userID)        →  DELETE /users/me/credentials/{id}
```

The backend's K8s Secret operations for OAuth credentials reduce to zero.

### RBAC for User Credentials (DB RBAC)

New permission: `user_credential:token` (fetch raw token for my own credential)
New built-in role: `user:self` — every authenticated user gets this automatically (bound at login)

Permissions:
```json
["user_credential:read", "user_credential:list", "user_credential:create",
 "user_credential:update", "user_credential:delete", "user_credential:token"]
```

Users can only see and fetch their own credentials (enforced by `user_id = JWT.sub` filter,
not just RBAC — defense in depth).

### Encryption (later)

When ready, add a `kek_id` column (key-encryption-key ID) and encrypt `token` with AES-256-GCM
using a DEK wrapped by the KEK. The KMS can be OCP's built-in etcd encryption, Vault, or RHKMS.
The schema is designed so this is an additive change — no routes change, only the service layer.

---

### Migration Order

1. Deploy ambient-api-server schema migration (additive — no downtime)
2. Deploy new `/users/me/credentials` routes
3. Deploy backend with dual-write: write to both K8s Secret AND DB (dark launch)
4. Validate reads from DB return correct data
5. Flip backend to read from DB (write to K8s Secret removed)
6. Clean up orphaned K8s Secrets
