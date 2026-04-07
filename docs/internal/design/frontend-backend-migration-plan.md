# Frontend/Backend Migration Plan: ambient-api-server as Primary API

**Date:** 2026-04-07
**Status:** Draft
**Scope:** Replace `components/backend` session and project CRUD with `ambient-api-server`; keep `components/backend` for K8s-native plumbing

---

## Background

The frontend (`components/frontend`) talks exclusively to `components/backend` (Gin/Go), which talks directly to K8s via user-scoped clients. `ambient-api-server` (rh-trex-ai/PostgreSQL) was built in parallel as the new coordination layer and is already used by `acpctl` and the control plane.

The goal is to make `ambient-api-server` the authoritative source of truth for sessions and projects, with `components/backend` retained only for operations that require direct K8s access (workspace, git, OAuth, secrets, AG-UI proxy, etc.).

---

## Key Insight: Identity Model is Compatible

The feared identity mismatch does not exist in practice:

- `ambient-api-server` `project_id` is a free-form string — the control plane already stores the K8s namespace name there (e.g. `ambient-dev`, `credential-test4`)
- `ambient-api-server` `kube_cr_name` stores the session's K8s resource name, which is the lowercased KSUID
- `components/backend` already uses namespace name as project identity and CR name as session identity
- `agent_id` is nullable — sessions can exist without agents, matching the existing frontend model exactly

The two systems use the same human-readable namespace name as project identity. No translation layer is required.

---

## What Moves vs. What Stays

### Moves to ambient-api-server (source of truth)

| Frontend call | Current backend handler | Target |
|---|---|---|
| `GET /projects` | `ListProjects` | `GET /api/ambient/v1/projects` |
| `POST /projects` | `CreateProject` | `POST /api/ambient/v1/projects` |
| `GET /projects/{name}` | `GetProject` | `GET /api/ambient/v1/projects/{id}` |
| `PUT /projects/{name}` | `UpdateProject` | `PATCH /api/ambient/v1/projects/{id}` |
| `DELETE /projects/{name}` | `DeleteProject` | `DELETE /api/ambient/v1/projects/{id}` |
| `GET /projects/{p}/agentic-sessions` | `ListSessions` | `GET /api/ambient/v1/sessions?search=project_id='...'` |
| `POST /projects/{p}/agentic-sessions` | `CreateSession` | `POST /api/ambient/v1/sessions` |
| `GET /projects/{p}/agentic-sessions/{s}` | `GetSession` | `GET /api/ambient/v1/sessions/{id}` (by kube_cr_name) |
| `PUT /projects/{p}/agentic-sessions/{s}` | `UpdateSession` | `PATCH /api/ambient/v1/sessions/{id}` |
| `POST /projects/{p}/agentic-sessions/{s}/start` | `StartSession` | `POST /api/ambient/v1/sessions/{id}/start` |
| `POST /projects/{p}/agentic-sessions/{s}/stop` | `StopSession` | `POST /api/ambient/v1/sessions/{id}/stop` |
| `DELETE /projects/{p}/agentic-sessions/{s}` | `DeleteSession` | `DELETE /api/ambient/v1/sessions/{id}` |
| `GET /me` | middleware/auth | `GET /api/ambient/v1/users/me` (or existing `/me`) |

### Stays in components/backend (K8s-native plumbing)

These cannot move without replicating K8s control-plane logic into `ambient-api-server`, which is out of scope:

| Category | Endpoints |
|---|---|
| AG-UI SSE proxy | `/agui/events`, `/agui/run`, `/agui/interrupt`, `/agui/feedback`, `/agui/tasks/*`, `/agui/capabilities` |
| Workspace file ops | `/workspace`, `/workspace/*path` |
| Git operations | `/git/status`, `/git/configure-remote`, `/git/list-branches`, `/git/merge-status` |
| GitHub push/PR/diff | `/github/diff`, `/github/push`, `/github/abandon` |
| Pod events | `/pod-events` |
| OAuth flows | `/auth/github/*`, `/auth/gitlab/*`, `/auth/google/*`, `/auth/jira/*`, `/auth/mcp/*` |
| Secrets | `/secrets`, `/runner-secrets`, `/integration-secrets` |
| Permissions/RBAC | `/permissions/*` |
| API keys | `/keys/*` |
| Models | `/models` |
| Runner types | `/runner-types` |
| Feature flags | `/feature-flags/*` |
| Scheduled sessions | `/scheduled-sessions/*` |
| Repo browser | `/repo/tree`, `/repo/blob`, `/repo/branches` |
| LDAP | `/ldap/*` |
| Cluster info | `/cluster-info` |
| OOTB workflows | `/workflows/ootb` |
| MCP status | `/mcp/status`, `/mcp/invoke` |
| Runtime credentials | `/credentials/*` |
| Session export | `/export` |
| Display name | `/displayname` |
| Clone | `/clone` |

---

## Required Changes

### 1. ambient-api-server — Session model additions

**Add `runner_type` field** to the Session model. This is the only field present in the `AgenticSession` CR that is missing from `ambient-api-server`.

Files to change:
- `plugins/sessions/model.go` — add `RunnerType *string`
- `plugins/sessions/handler.go` — include in `Patch` handler
- `plugins/sessions/presenter.go` — include in `ConvertSession` / `PresentSession`
- `openapi/openapi.sessions.yaml` — add `runner_type` to schema
- `plugins/sessions/migration.go` — add `ALTER TABLE sessions ADD COLUMN runner_type VARCHAR(255)`
- Regenerate OpenAPI client (`model_session.go`)

**Add `kube_cr_name` search support** to the session list endpoint so `components/backend` can resolve session name → KSUID by K8s name.

Files to change:
- `plugins/sessions/dao.go` — `AllByKubeCrName(ctx, name)` or make the generic search support `kube_cr_name` column
- `plugins/sessions/handler.go` — pass through `?search=kube_cr_name='...'` or add a dedicated lookup path

### 2. ambient-api-server — Project model additions

**Verify `name` is queryable and unique.** The frontend uses human-readable project names as identifiers. `ambient-api-server` uses KSUIDs as primary keys but the `name` field exists. The backend needs to resolve `name` → `id` on each request.

Options:
- **A (preferred):** Add `GET /api/ambient/v1/projects?search=name='foo'` support and use name as a stable lookup key. One extra query per request, cached in the backend.
- **B:** Require the frontend to use KSUIDs everywhere — this is a larger frontend change.

Option A is preferred to minimize frontend changes.

Files to change:
- `plugins/projects/dao.go` — ensure `name` column is indexed and searchable
- `plugins/projects/migration.go` — add unique index on `name` if not present

### 3. components/backend — Proxy layer for migrated endpoints

`components/backend` becomes a thin proxy for the migrated endpoints, forwarding to `ambient-api-server` with the user's token. It retains full ownership of all K8s-native routes.

**New env var:** `AMBIENT_API_SERVER_URL` — the internal URL of `ambient-api-server` (already set in the deployment).

**Proxy pattern for each migrated handler:**

```go
// Example: ListSessions proxy
func ListSessions(c *gin.Context) {
    projectName := c.Param("projectName")
    token := extractUserToken(c)
    resp, err := ambientAPIClient.Get(
        fmt.Sprintf("/api/ambient/v1/sessions?search=project_id='%s'", projectName),
        token,
    )
    // forward response body and status code as-is
    c.DataFromReader(resp.StatusCode, resp.ContentLength, "application/json", resp.Body, nil)
}
```

The backend does NOT transform the response — it forwards it verbatim. The frontend will need minor field-name adjustments (see §4).

**Session name → KSUID resolution:**

For routes that take `:sessionName` (the K8s CR name), the backend resolves to KSUID before forwarding:

```go
func resolveSessionID(c *gin.Context, projectName, sessionName, token string) (string, error) {
    resp, err := ambientAPIClient.Get(
        fmt.Sprintf("/api/ambient/v1/sessions?search=kube_cr_name='%s' and project_id='%s'", sessionName, projectName),
        token,
    )
    // extract items[0].id
}
```

This lookup can be cached in-process with a short TTL (30s) keyed on `projectName/sessionName` to avoid per-request overhead.

### 4. Frontend — Field name adjustments

The `ambient-api-server` session response shape differs slightly from what the frontend currently expects. These are the known deltas:

| Frontend expects | ambient-api-server returns | Action |
|---|---|---|
| `displayName` | `name` | Map `name` → `displayName` in presenter, or update frontend |
| `runnerType` | `runner_type` (to be added) | Add field, map snake_case → camelCase |
| `agentStatus` | not present | Derived field — compute from `phase` in the frontend or backend proxy |
| `userContext` | not present | K8s-only concept — drop or move to `annotations` |
| `scheduledSession` | not present | Stays in `components/backend` scheduled-sessions resource |
| `repoUrl` (single) | `repo_url` ✅ | camelCase conversion in frontend client |
| `repos` (JSON string) | `repos` ✅ | Same |
| `llmModel` | `llm_model` ✅ | camelCase conversion |
| `phase` | `phase` ✅ | Same |
| `startTime` | `start_time` ✅ | camelCase conversion |
| `completionTime` | `completion_time` ✅ | camelCase conversion |
| `parentSessionId` | `parent_session_id` ✅ | camelCase conversion |

The frontend `services/api/sessions.ts` already sends/receives JSON — updating field names is a localized change in the type definitions (`src/types/api/sessions.ts`) and the service layer.

Alternatively, `components/backend` can apply a response transformation shim that renames fields before returning to the frontend, keeping the frontend unchanged during the transition.

### 5. Authentication pass-through

`components/backend` currently validates the user's OIDC token via middleware and uses it to construct user-scoped K8s clients. For the proxied endpoints, it needs to forward the same token to `ambient-api-server` as a `Bearer` token in the `Authorization` header.

`ambient-api-server` already validates OIDC tokens via JWK cert URL. No changes needed on the `ambient-api-server` auth side — the token flows through unchanged.

### 6. Project creation — dual write during transition

Project creation in `components/backend` currently creates a K8s namespace + RoleBindings. This K8s provisioning must continue. The session record creation must also happen in `ambient-api-server`.

**During migration:** `CreateProject` in `components/backend` does both:
1. Creates K8s namespace (existing behavior)
2. `POST /api/ambient/v1/projects` to `ambient-api-server` with `project_id = namespace_name`

**After migration is complete:** The control plane already watches `ambient-api-server` projects and provisions namespaces. The backend's direct K8s namespace creation can be removed once the control plane is the sole provisioner.

### 7. Session creation — dual write during transition

`CreateSession` in `components/backend` currently creates an `AgenticSession` CR directly. The control plane watches the CR and creates the pod.

With `ambient-api-server` as source of truth:
1. `POST /api/ambient/v1/sessions` → `ambient-api-server` stores the row, returns KSUID
2. The control plane watches `ambient-api-server` for new pending sessions and creates the CR + pod (this is already the control plane's job)
3. The CR name = `kube_cr_name` = the KSUID stored in the session row

**During migration:** The backend may still need to create the CR directly if the control plane is not yet deployed in all environments. The dual-write pattern (write to `ambient-api-server` AND create CR) is the safe transition path.

---

## Migration Phases

### Phase 1 — ambient-api-server model completeness
- Add `runner_type` to Session model + migration
- Add `kube_cr_name` search to session list
- Verify `name` is uniquely indexed on projects
- Add `GET /api/ambient/v1/sessions/{id}/start` and `stop` endpoints if not present
- **No frontend or backend changes yet**

### Phase 2 — Backend proxy layer
- Add `AMBIENT_API_SERVER_URL` to `components/backend` config
- Implement thin proxy client in backend
- Implement `resolveSessionID` cache
- Migrate `ListSessions`, `GetSession` handlers to proxy — read-only, low risk
- Verify frontend behavior is unchanged (response shape shim if needed)

### Phase 3 — Write path migration
- Migrate `CreateSession`, `UpdateSession`, `PatchSession` to proxy
- Migrate `StartSession`, `StopSession`, `DeleteSession` to proxy
- Dual-write CR creation until control plane is sole provisioner in all envs

### Phase 4 — Project CRUD migration
- Migrate `ListProjects`, `GetProject` to proxy
- Migrate `CreateProject` to dual-write (namespace + API)
- Migrate `UpdateProject`, `DeleteProject` to proxy

### Phase 5 — Frontend direct access (optional)
- Frontend calls `ambient-api-server` directly for migrated endpoints, bypassing `components/backend` entirely
- `components/backend` becomes a pure K8s-native sidecar
- Requires CORS config and token forwarding from the frontend

### Phase 6 — Backend cleanup
- Remove migrated handlers from `components/backend`
- Remove K8s namespace creation from `CreateProject` (control plane owns provisioning)
- Remove direct CR creation from `CreateSession`

---

## Risk Areas

| Risk | Mitigation |
|---|---|
| Session name resolution adds latency | In-process cache with 30s TTL keyed on `project/session` name |
| Project name not unique in ambient-api-server | Add unique index in migration; enforce in `CreateProject` handler |
| dual-write consistency (CR + DB) | Use the DB as leader; if DB write fails, abort before CR creation |
| Frontend field name breakage | Response shim in backend proxy during transition; frontend type update in Phase 5 |
| `agentStatus` derived field missing | Compute from `phase` in the proxy response shim; identical mapping already done in the frontend today |
| Control plane not deployed in all envs | Keep CR creation in backend as fallback; gate on `AMBIENT_CP_ENABLED` env var |
| Auth token forwarding scope | `ambient-api-server` JWK validation must accept the same OIDC issuer as `components/backend` — already configured in production |

---

## Files Affected (Summary)

### ambient-api-server
- `plugins/sessions/model.go` — add `runner_type`
- `plugins/sessions/migration.go` — DB migration
- `plugins/sessions/handler.go` — `runner_type` in patch; `kube_cr_name` search
- `plugins/sessions/presenter.go` — `runner_type` in convert/present
- `plugins/sessions/dao.go` — `AllByKubeCrName` or search support
- `openapi/openapi.sessions.yaml` — schema update
- `plugins/projects/migration.go` — unique index on `name`

### components/backend
- `handlers/sessions.go` — proxy `List`, `Get`, `Create`, `Update`, `Start`, `Stop`, `Delete`
- `handlers/projects.go` — proxy `List`, `Get`, `Create`, `Update`, `Delete`; retain namespace provisioning
- `server/server.go` — add `AMBIENT_API_SERVER_URL` config
- new file: `handlers/ambient_proxy.go` — shared proxy client + `resolveSessionID` cache

### components/frontend
- `src/types/api/sessions.ts` — field name updates (snake_case → camelCase, add `runner_type`)
- `src/services/api/sessions.ts` — adjust field mapping if not using shim
- `src/services/api/projects.ts` — adjust if project shape changes

---

## Non-Goals

- Migrating AG-UI proxy, workspace, git, OAuth, secrets, permissions, scheduled sessions, LDAP, feature flags, models, runner types to `ambient-api-server` — these are K8s-native and belong in `components/backend`
- Replacing `components/backend` entirely — it remains the K8s control surface
- Changing the frontend URL structure — all existing frontend routes remain valid
