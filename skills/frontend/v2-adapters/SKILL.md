---
name: frontend-v2-adapters
description: >
  Build v2 frontend adapters that call the ambient-api-server via the
  @ambient-platform/sdk instead of the legacy K8s backend. Covers the full
  frontend API surface: root-level catch-all proxy replacing ~92 individual
  route files, 3 SDK-native v2 adapters (Sessions, Projects, ScheduledSessions),
  and 27 passthrough domains routed through the API server's proxy plugin.
  Use when creating adapters under services/adapters/v2/, wiring the SDK client,
  or mapping API server responses to canonical frontend types.
---

# v2 Adapter Implementation

Replace the frontend's direct BACKEND_URL proxying with a single catch-all that routes all traffic through the ambient-api-server. Build SDK-native v2 adapters for 3 domains; leave 27 passthrough domains on v1 adapters (they work unchanged because the API server proxies non-native paths to the legacy backend).

**PR policy:** Do NOT create any PR until the user explicitly approves.

Load before any code changes:
- `specs/frontend/v2-adapters.spec.md` (behavioral contract)
- `specs/standards/frontend/conventions.spec.md`

## User Input

```text
$ARGUMENTS
```

## Prerequisites (ALL DONE)

- PR #1511 — `/api/ambient/v1/[...path]` catch-all (MERGED)
- PR #1523 — SDK generator fix for typed actions (MERGED)
- PR #1524 — SDK regeneration with ScheduledSession support (MERGED)

SDK already supports: optional token, relative baseUrl, display_name on Project, ScheduledSession CRUD + suspend/resume/trigger/runs actions.

## Architecture

```
SDK-native:   hook → port → v2 adapter → SDK → fetch('/api/ambient/v1/...') → catch-all → API server
Passthrough:  hook → port → v1 adapter → API svc → fetch('/api/...') → catch-all → API server → backend
```

## Workstream A: Root-Level Catch-All

**New:** `src/app/api/[...path]/route.ts` — forwards ALL methods to `API_SERVER_URL`.

Based on the existing ambient catch-all pattern. Path reconstruction: `${API_SERVER_URL}/api/${pathStr}` (Next.js consumes the `/api/` prefix). Preserve query strings, auth headers (via `buildForwardHeadersAsync`), Accept/Content-Type. Stream SSE/NDJSON via TransformStream. Return 502 on failure.

**Delete** the existing ambient catch-all (`ambient/v1/[...path]/route.ts`) — root catch-all subsumes it.

**Delete ~92 route files** that are simple BACKEND_URL proxies.

**Keep 7 routes** with special logic (Next.js gives specific routes priority):
- `/api/me/` — reads forwarded auth headers
- `/api/config/loading-tips/` — reads env var
- `/api/feature-flags/` + `client/register/` + `client/metrics/` — Unleash proxy
- `workspace/upload/` — file compression, SSRF protection
- `workspace/[...path]/` — MIME detection, inline rendering

**Update workspace routes** from `BACKEND_URL` to `API_SERVER_URL`.

## Workstream B: SDK Utilities

Four files in `services/adapters/v2/`:

**`client.ts`** — SDK client factory. `new AmbientClient({ baseUrl: '/', project })`. No token (proxy handles auth). New client per-call (project varies).

**`pagination.ts`** — Convert `PaginationParams` (limit/offset) → SDK `ListOptions` (page/size). Transform SDK list → `PaginatedResult<T>`.

**`json.ts`** — `parseJsonField<T>(value, fallback)`: handles JSONB fields that arrive as strings or pre-parsed objects.

**`errors.ts`** — Catch `AmbientAPIError` → rethrow as `ApiClientError(reason, code)`.

## Workstream C: SDK-Native v2 Adapters

### Projects (simplest)

5 methods via SDK, 3 NOT_IMPLEMENTED. Read the port at `services/ports/projects.ts` and canonical `Project` type at `types/api/projects.ts`. Map SDK flat snake_case → canonical camelCase. Parse JSONB `labels`/`annotations`.

### Sessions (most complex)

8 via SDK, 3 via v1 API fallback (clone, pod-events, export), 1 NOT_IMPLEMENTED (Google Drive).

SDK `Session` is flat with JSONB strings → canonical `AgenticSession` has nested metadata/spec/status. Parse all JSONB fields. Use `kube_cr_uid` for `metadata.uid` (not database `id`). Nest `llm_*` into `llmSettings`. Resolve `workflow_id` → `activeWorkflow`.

**Hybrid fallback:** For methods without SDK support, import the v1 API service. The v1 API hits the catch-all → API server → backend proxy.

**Name→ID:** For API-server-created sessions, name=id. Legacy K8s sessions need search by `kube_cr_name`. KSUID fast-path (26 alphanum chars → try direct get first).

### ScheduledSessions (medium)

All 9 methods via SDK. Key: Agent resolution. API server stores agent config on separate `Agent` resource.

- **Read:** `agent_id` → fetch Agent → reconstruct `sessionTemplate`
- **Create:** Create Agent from `sessionTemplate` → create ScheduledSession
- **Update:** Patch Agent and/or ScheduledSession
- **Delete:** Delete ScheduledSession + orphaned Agent cleanup
- **Actions:** `suspend()`, `resume()`, `trigger()`, `runs()` — direct SDK calls

Batch Agent resolution on list (dedup `agent_id`s).

## Workstream D: Wire Up

Change 3 barrel exports to `./v2/`. Bump `BACKEND_VERSION` to `'v2'`. 27 passthrough re-exports unchanged.

## Testing

Mock SDK with `vi.fn()`. Test response transformation, JSONB parsing, request mapping, pagination, NOT_IMPLEMENTED, ScheduledSession Agent resolution/decomposition/cleanup.

## Key Files

| Category | Path (relative to `components/`) |
|----------|------|
| SDK types | `ambient-sdk/ts-sdk/src/{session,project,scheduled_session,agent}.ts` |
| SDK APIs | `ambient-sdk/ts-sdk/src/{session_api,project_api,scheduled_session_api,agent_api}.ts` |
| Canonical types | `frontend/src/types/api/{sessions,projects,scheduled-sessions}.ts` |
| Ports | `frontend/src/services/ports/{sessions,projects,scheduled-sessions}.ts` |
| v1 adapters | `frontend/src/services/adapters/v1/{sessions,projects,scheduled-sessions}.ts` |
| Existing catch-all | `frontend/src/app/api/ambient/v1/[...path]/route.ts` |
| Re-exports | `frontend/src/services/adapters/{sessions,projects,scheduled-sessions}.ts` |
| API errors | `frontend/src/services/api/errors.ts` |
| Config | `frontend/src/lib/config.ts` |
| Auth | `frontend/src/lib/auth.ts` |
