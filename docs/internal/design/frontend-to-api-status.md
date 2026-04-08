# Frontend → API Migration Status

Mapping of all 90 Next.js API route handlers to their backend targets.
Identifies which routes have migrated to `ambient-api-server`, which must stay on the V1 backend, and which are candidates for migration.

**Last updated:** 2026-03-18 (all known bugs resolved)

---

## Environment Variables

| Variable | Default | Used By |
|---|---|---|
| `BACKEND_URL` | `http://localhost:8080/api` | V1 backend (K8s-backed Gin server) |
| `AMBIENT_API_URL` | `http://localhost:8000` | ambient-api-server (PostgreSQL REST) |

---

## ✅ Migrated to ambient-api-server

All of these route handlers call `ambient-api-client` functions or `AMBIENT_API_URL` directly.

| Next.js Route | Methods | Ambient Endpoint | Error Handling |
|---|---|---|---|
| `/api/agents` | GET | `GET /api/ambient/v1/agents?search=project_id='...'` | No |
| `/api/project-documents` | GET | `GET /api/ambient/v1/project_documents?search=project_id='...'` | No |
| `/api/projects` | GET, POST | `GET /api/ambient/v1/projects`, `POST /api/ambient/v1/projects` | Yes |
| `/api/projects/[name]` | GET, PUT, DELETE | `GET /api/ambient/v1/projects?search=name='...'`, `PATCH /api/ambient/v1/projects/:id`, `DELETE /api/ambient/v1/projects/:id` | Yes |
| `/api/projects/[name]/agentic-sessions` | GET, POST | `GET /api/ambient/v1/sessions?search=project_id='...'`, `POST /api/ambient/v1/sessions` | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]` | GET, PUT, DELETE | `GET /api/ambient/v1/sessions/:id`, `PATCH /api/ambient/v1/sessions/:id`, `DELETE /api/ambient/v1/sessions/:id` | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/agui/events` | GET | `GET /api/ambient/v1/sessions/:id/ag_ui` (SSE) | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/agui/history` | GET | `GET /api/ambient/v1/ag_ui_events?search=session_id='...'` | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/agui/interrupt` | POST | `POST /api/ambient/v1/sessions/:id/ag_ui` | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/agui/run` | POST | `POST /api/ambient/v1/sessions/:id/ag_ui` | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/agui/runs` | GET | `GET /api/ambient/v1/ag_ui_events?search=session_id='...'` (deduped run IDs) | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/displayname` | PUT | `PATCH /api/ambient/v1/sessions/:id` (`name` field) | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/start` | POST | `POST /api/ambient/v1/sessions/:id/start` | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/stop` | POST | `POST /api/ambient/v1/sessions/:id/stop` | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/watch` | GET | `GET /api/ambient/v1/sessions/:id/messages` (SSE) | Yes |
| `/api/session-check-ins` | GET | `GET /api/ambient/v1/session_check_ins?search=session_id='...'` | No |

> **Session ID resolution pattern:** Routes that receive `sessionName` (the K8s CR name or ambient UUID) resolve to the ambient session UUID via `getAmbientSessionByCrName`, which first tries a direct `GET /sessions/{id}` lookup, then falls back to `GET /sessions?search=kube_cr_name='...'`.

> **Error handling:** All migrated routes now have try/catch. An upstream error returns a structured JSON error response.

---

## ❌ Must Stay on V1 Backend

No equivalent exists in ambient-api-server. These require K8s access, runner pod filesystem, or external service integrations only the V1 backend provides.

### Auth & Integrations

| Next.js Route | Methods | Reason | Error Handling |
|---|---|---|---|
| `/api/auth/github/disconnect` | POST | OAuth flows, K8s secrets | Yes |
| `/api/auth/github/install` | POST | OAuth flows, K8s secrets | Yes |
| `/api/auth/github/pat` | POST, DELETE | PAT management, K8s secrets | Yes |
| `/api/auth/github/pat/status` | GET | K8s secret lookup | Yes |
| `/api/auth/github/status` | GET | K8s secret lookup | Yes |
| `/api/auth/github/user/callback` | GET | OAuth redirect handling | Yes |
| `/api/auth/gitlab/connect` | POST | OAuth flows, K8s secrets | Yes |
| `/api/auth/gitlab/disconnect` | DELETE | K8s secrets | Yes |
| `/api/auth/gitlab/status` | GET | K8s secret lookup | Yes |
| `/api/auth/google/connect` | POST | OAuth flows, K8s secrets | Yes |
| `/api/auth/google/disconnect` | POST | K8s secrets | Yes |
| `/api/auth/google/status` | GET | K8s secret lookup | Yes |
| `/api/auth/integrations/status` | GET | Integration secret lookups in K8s | Yes |
| `/api/auth/jira/connect` | POST | OAuth flows, K8s secrets | Yes |
| `/api/auth/jira/disconnect` | DELETE | K8s secrets | Yes |
| `/api/auth/jira/status` | GET | K8s secret lookup | Yes |

### Project Sub-resources (K8s-backed)

| Next.js Route | Methods | Reason | Error Handling |
|---|---|---|---|
| `/api/projects/[name]/access` | GET | K8s RBAC | Yes |
| `/api/projects/[name]/feature-flags` | GET | Unleash feature flags | Yes |
| `/api/projects/[name]/feature-flags/[flagName]` | GET | Unleash feature flags | Yes |
| `/api/projects/[name]/feature-flags/[flagName]/disable` | POST | Unleash feature flags | Yes |
| `/api/projects/[name]/feature-flags/[flagName]/enable` | POST | Unleash feature flags | Yes |
| `/api/projects/[name]/feature-flags/[flagName]/override` | PUT, DELETE | Unleash feature flags | Yes |
| `/api/projects/[name]/feature-flags/evaluate/[flagName]` | GET | Unleash feature flags | Yes |
| `/api/projects/[name]/integration-secrets` | GET, PUT | K8s secrets | Yes |
| `/api/projects/[name]/integration-status` | GET | K8s secret status | Yes |
| `/api/projects/[name]/keys` | GET, POST | K8s secret-backed API keys | Yes |
| `/api/projects/[name]/keys/[keyId]` | DELETE | K8s secret-backed API keys | Yes |
| `/api/projects/[name]/models` | GET | Runner pod inference | Yes |
| `/api/projects/[name]/permissions` | GET, POST | K8s RBAC | Yes |
| `/api/projects/[name]/permissions/[subjectType]/[subjectName]` | DELETE | K8s RBAC | Yes |
| `/api/projects/[name]/repo/blob` | GET | Git via runner pod | Yes |
| `/api/projects/[name]/repo/tree` | GET | Git via runner pod | Yes |
| `/api/projects/[name]/runner-secrets` | GET, PUT | K8s secrets | Yes |
| `/api/projects/[name]/runner-secrets/config` | GET, PUT | K8s secrets | Yes |
| `/api/projects/[name]/runner-types` | GET | K8s runner type discovery | Yes |
| `/api/projects/[name]/secrets` | GET | K8s secrets | Yes |
| `/api/projects/[name]/settings` | GET, PUT | K8s ProjectSettings CRD | Yes |
| `/api/projects/[name]/users/forks` | GET, POST | K8s namespace forks | Yes |

### Session Sub-resources (Runner Pod / K8s)

| Next.js Route | Methods | Reason | Error Handling |
|---|---|---|---|
| `/api/projects/[name]/agentic-sessions/[sessionName]/agui/capabilities` | GET | Runner capability check | Yes (returns `{capabilities:[]}`) |
| `/api/projects/[name]/agentic-sessions/[sessionName]/agui/feedback` | POST | AG-UI state in runner | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/clone` | POST | Session CR clone | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/credentials/github` | GET | K8s secrets in runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/credentials/gitlab` | GET | K8s secrets in runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/credentials/google` | GET | K8s secrets in runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/credentials/jira` | GET | K8s secrets in runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/export` | GET | Session CR export | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/git/configure-remote` | POST | Git ops via runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/git/merge-status` | GET | Git ops via runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/git/status` | GET | Git ops via runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/mcp/invoke` | POST | MCP via runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/mcp/status` | GET | MCP via runner pod | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/oauth/google/url` | GET | OAuth via runner | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/pod-events` | GET | K8s pod events | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/repos` | POST | Runner repo management | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/repos/[repoName]` | DELETE | Runner repo management | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/repos/status` | GET | Runner repo management | Partial (non-ok only) |
| `/api/projects/[name]/agentic-sessions/[sessionName]/workflow` | POST | Runner workflow ops | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/workflow/metadata` | GET | Runner workflow ops | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/workspace` | GET | Runner pod filesystem | Yes (returns `{items:[]}`) |
| `/api/projects/[name]/agentic-sessions/[sessionName]/workspace/[...path]` | GET, PUT, DELETE | Runner pod filesystem | Yes |
| `/api/projects/[name]/agentic-sessions/[sessionName]/workspace/upload` | POST | Runner pod filesystem | Yes (extensive validation) |

### Global

| Next.js Route | Methods | Reason | Error Handling |
|---|---|---|---|
| `/api/cluster-info` | GET | K8s cluster metadata | Yes |
| `/api/runner-types` | GET | K8s runner type discovery | Yes (returns `[]`) |
| `/api/workflows/ootb` | GET | K8s workflow definitions | Yes (returns `[]`) |

---

## Internal Only (No Outbound Fetch)

| Next.js Route | Methods | Source | Error Handling |
|---|---|---|---|
| `/api/config/loading-tips` | GET | Reads `LOADING_TIPS` env var | Partial |
| `/api/feature-flags` | GET | Proxies to `UNLEASH_URL` | Yes (returns `{toggles:[]}`) |
| `/api/feature-flags/client/metrics` | POST | Proxies to `UNLEASH_URL` | Yes (always 202) |
| `/api/feature-flags/client/register` | POST | Proxies to `UNLEASH_URL` | Yes (always 202) |
| `/api/me` | GET | Reads forwarded auth headers | Yes (returns `{authenticated:false}`) |
| `/api/version` | GET | Reads `VTEAM_VERSION` env var | No (no fetch, no throw) |

---

## Known Bugs & Anomalies

All previously identified bugs have been resolved. No open anomalies.

| Route | Issue | Status |
|---|---|---|
| `credentials/*` (all 4) | Path params NOT `encodeURIComponent`'d | ✅ Fixed |
| `/api/projects/[name]/settings` | Used manual header copy instead of `buildForwardHeadersAsync` | ✅ Fixed |
| `/api/workflows/ootb` | Only forwarded `Authorization` header | ✅ Fixed |
| All V1 routes without try/catch | `ECONNREFUSED` → unhandled 500 | ✅ Fixed (all return structured errors) |
| `start`, `stop`, `watch`, `agui/interrupt` | Migrated to ambient but no try/catch | ✅ Fixed |

---

## ambient-api-server Endpoint Reference

Full endpoint list for `/api/ambient/v1`:

| Method | Path | Notes |
|---|---|---|
| GET/POST | `/sessions` | List/create |
| GET/PATCH/DELETE | `/sessions/{id}` | By UUID |
| POST | `/sessions/{id}/start` | Transition to Pending |
| POST | `/sessions/{id}/stop` | Transition to Stopping |
| GET/POST | `/sessions/{id}/messages` | SSE stream with `Accept: text/event-stream` |
| GET | `/sessions/{id}/ag_ui` | SSE-only AG-UI event stream |
| POST | `/sessions/{id}/ag_ui` | Send user turn |
| GET/POST | `/sessions/{id}/checkin` | Latest check-in |
| GET | `/sessions/{id}/checkins` | All check-ins |
| GET/POST | `/projects` | List/create |
| GET/PATCH/DELETE | `/projects/{id}` | By UUID |
| GET | `/projects/{id}/agents` | Agents for project |
| GET | `/projects/{id}/home` | SSE project home feed |
| GET | `/projects/{id}/home/snapshot` | Project home snapshot |
| GET/POST/PATCH/DELETE | `/project_settings` | Project settings CRUD |
| GET/POST/PATCH/DELETE | `/agents` | Agent CRUD |
| POST | `/agents/{id}/start` | Start agent |
| GET | `/agents/{id}/start` | Preview start context |
| GET | `/agents/{id}/sessions` | Agent sessions |
| GET | `/agents/{id}/checkins` | Agent check-ins |
| GET/POST | `/agents/{id}/inbox` | Agent inbox |
| PATCH/DELETE | `/agents/{id}/inbox/{msg_id}` | Inbox message ops |
| GET/POST/PATCH/DELETE | `/session_check_ins` | Check-in CRUD |
| GET/POST/PATCH/DELETE | `/project_documents` | Document CRUD |
| GET | `/projects/{id}/documents` | Documents by project |
| PUT | `/projects/{id}/documents/{slug}` | Upsert by slug |
| GET/POST/PATCH/DELETE | `/users` | User CRUD |
| GET/POST/PATCH/DELETE | `/roles` | Role CRUD |
| GET/POST/PATCH/DELETE | `/role_bindings` | Role binding CRUD |
| GET/POST/PATCH/DELETE | `/ag_ui_events` | AG-UI event CRUD |
| GET | `/openapi` | OpenAPI spec |
