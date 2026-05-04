---
name: frontend-v2-adapters
description: >
  Build v2 frontend adapters that call the ambient-api-server via the
  @ambient-platform/sdk instead of the legacy K8s backend. Use when
  creating adapters under services/adapters/v2/, wiring the SDK client,
  adding proxy routes for /api/ambient/v1/, or mapping API server responses
  to canonical frontend types. Triggers on: "v2 adapter", "api server adapter",
  "migrate domain to v2", "SDK adapter", "ambient-api-server frontend".
---

# v2 Adapter Implementation

Build adapters in `services/adapters/v2/` that implement the same port interfaces as v1, but call the ambient-api-server through the `@ambient-platform/sdk`.

Behavioral contract: `specs/frontend/v2-adapters.spec.md`. Load `specs/standards/frontend/conventions.spec.md` and `specs/standards/frontend/react-query.spec.md` before any code changes.

## User Input

```text
$ARGUMENTS
```

## Architecture

```
hook → port → v2 adapter → SDK client → fetch('/api/ambient/v1/...') → Next.js proxy → API server
```

The SDK provides typed methods and types. A catch-all Next.js route proxies to the API server with auth headers. v2 adapters transform SDK responses into canonical frontend types.

## Scope

This skill covers **Sessions**, **Projects**, and **ScheduledSessions** — the three domains to migrate. ScheduledSessions has a full CRUD plugin in the API server (`plugins/scheduledSessions/`) with suspend/resume/trigger/runs endpoints. The TS SDK currently lacks ScheduledSession types — add to the OpenAPI spec and regenerate as a prerequisite (see SDK Prerequisites for ScheduledSessions below). **SessionEvents** (AG-UI streaming) is a separate port (`SessionEventsPort`) that can be migrated independently — see the spec for details.

## SDK Setup

The SDK lives at `components/ambient-sdk/ts-sdk/`. It's auto-generated from the OpenAPI spec.

**Generator changes required.** The SDK constructor currently requires `baseUrl` (non-empty) and `token` (validated for length/format). For browser use through the Next.js proxy, update the generator templates in `components/ambient-sdk/generator/` to:

1. Make `token` optional in `AmbientClientConfig` — skip the `Authorization` header when absent
2. Accept relative `baseUrl` values (empty string or `'/'`) — skip `new URL()` validation for relative URLs, or use `'/'` as the base
3. Add `display_name` to the SDK `Project` type — the API server model has `display_name` but the current SDK generator drops it

Regenerate with `make generate-sdk`.

**Frontend client** at `services/api/v2/client.ts`:
```typescript
import { AmbientClient } from '@ambient-platform/sdk'

export function getClient(projectName?: string): AmbientClient {
  return new AmbientClient({ baseUrl: '/', project: projectName })
}
```

`baseUrl: '/'` sends requests to the same origin. No token — the proxy handles auth via OAuth headers. A new client is created per-call because port methods pass `projectName` per-call and the SDK sets project context globally via `X-Ambient-Project` on the client config. The client is stateless (no connection pool), so per-request creation has negligible overhead.

## Proxy Route

Create `src/app/api/ambient/v1/[...path]/route.ts` — a catch-all that forwards to `API_SERVER_URL` using `buildForwardHeadersAsync()` from `src/lib/auth.ts`. Handle GET, POST, PATCH, DELETE. Forward query strings and request bodies. Return the API server's response status and body.

## Adapter Pattern

Each v2 adapter in `services/adapters/v2/{domain}.ts` follows this shape:

1. Import the SDK client and port type
2. Call SDK methods (`ambientClient.sessions.list()`, `.get()`, etc.)
3. Transform SDK types to canonical frontend types
4. Return through the port interface

Use a factory function (`create{Domain}V2Adapter()`) with a default export, matching the v1 pattern.

**Return type normalization.** Some port methods return non-resource types that differ from SDK return types:

| Port method | Port returns | SDK returns | Normalization |
|---|---|---|---|
| `stopSession` | `string` | `Session` | Call `.stop()`, return `session.phase` |
| `startSession` | `{ message: string }` | `Session` | Call `.start()`, return `{ message: 'Session started' }` |
| `deleteSession` | `void` | `void` | Direct (no conversion) |
| `deleteProject` | `string` | `void` | Call `.delete()`, return `'Project deleted'` |

## Name → ID Resolution

Port methods identify sessions by K8s name (`sessionName`), but SDK methods (`get`, `update`, `stop`, `start`, `delete`) take a database ID. For API-server-created sessions, `kube_cr_name` equals the database `id` (auto-set on creation), so the name IS the ID.

For sessions that originated from the K8s CRD backend (legacy), `kube_cr_name` is the K8s resource name and `id` is a different KSUID. These require a lookup:

```typescript
async function resolveSessionId(client: AmbientClient, sessionName: string): Promise<string> {
  const escaped = sessionName.replace(/'/g, "''")
  const result = await client.sessions.list({ search: `kube_cr_name = '${escaped}'`, size: 1 })
  if (!result.items.length) throw new ApiClientError(`Session '${sessionName}' not found`, 'NOT_FOUND')
  return result.items[0].id
}
```

**Security:** The `search` parameter is interpolated into a TSL query. Escape single quotes in `sessionName` to prevent TSL injection. The API server's TSL parser is the final defense, but the adapter should not rely on it.

Optimization: if `sessionName` looks like a KSUID (26 alphanumeric chars), try `client.sessions.get(sessionName)` first and fall back to search only on 404. Most v2-created sessions will hit the fast path.

## Session Mapping

SDK `Session` (flat, snake_case, JSONB as strings) → canonical `AgenticSession` (nested, camelCase, typed objects).

**metadata**: `kube_cr_name` → `name`, `kube_namespace` → `namespace`, `kube_cr_uid` → `uid`, `created_at` → `creationTimestamp`, `labels`/`annotations` → parse JSON to `Record<string, string>`.

Note: use `kube_cr_uid` (the Kubernetes UID) for `metadata.uid`, not the database row `id`. The database `id` is a KSUID for internal use; `kube_cr_uid` is the K8s resource UID that components expect.

**spec**: `name` → `displayName`, `prompt` → `initialPrompt`, `llm_model`/`llm_temperature`/`llm_max_tokens` → nest into `llmSettings`, `repos` → parse JSON to `SessionRepo[]`, `timeout` → direct, `workflow_id` → resolve to `activeWorkflow` (see Workflow Resolution below), `environment_variables` → parse JSON to `Record<string, string>`.

Note: `mainRepoIndex` has no SDK source — default to `undefined`. The `botAccount` and `resourceOverrides` types exist in `types/api/sessions.ts` but are NOT fields on `AgenticSessionSpec` — do not map them.

**status**: `phase` → direct, `start_time` → `startTime`, `completion_time` → `completionTime`, `conditions` → parse JSON to `SessionCondition[]`, `reconciled_repos` → parse JSON to `ReconciledRepo[]`, `reconciled_workflow` → parse JSON to `ReconciledWorkflow`, `sdk_session_id` → `sdkSessionId`, `sdk_restart_count` → `sdkRestartCount`.

**Fields without SDK source** — these canonical `AgenticSessionStatus` fields have no corresponding SDK field. Default all to `undefined` (they are optional in the type):
- `observedGeneration`
- `lastActivityTime`
- `agentStatus`
- `stoppedReason`
- `jobName`
- `runnerPodName`

**Unmapped SDK fields** — these SDK fields have no canonical target and are dropped during transformation: `agent_id`, `assigned_user_id`, `created_by_user_id`, `parent_session_id`, `project_id`, `repo_url`, `triggered_by_user_id`, `interactive`.

**autoBranch**: derive from `repos` if the backend computes it, or default to `undefined`.

Use a safe JSON parser for all JSONB string fields. Collection fields may arrive as JSON strings or already-typed objects depending on the API server version — handle both:
```typescript
function parseJsonField<T>(value: unknown, fallback: T): T {
  if (value == null) return fallback
  if (typeof value !== 'string') return value as T
  try { return JSON.parse(value) } catch { return fallback }
}
```

## Session Request Mapping

Canonical `CreateAgenticSessionRequest` → SDK `SessionCreateRequest`. The reverse of response mapping: flatten nested objects, serialize collections to JSON strings.

| Canonical (camelCase) | SDK (snake_case) | Transform |
|---|---|---|
| `initialPrompt` | `prompt` | direct |
| `llmSettings.model` | `llm_model` | flatten |
| `llmSettings.temperature` | `llm_temperature` | flatten |
| `llmSettings.maxTokens` | `llm_max_tokens` | flatten |
| `displayName` | `name` | direct; SDK `name` is **required** — if `displayName` is absent, generate a default (e.g., timestamp-based or UUID) |
| `timeout` | `timeout` | direct |
| `repos` | `repos` | `JSON.stringify()` |
| `environmentVariables` | `environment_variables` | `JSON.stringify()` |
| `activeWorkflow` | `workflow_id` | see Workflow Resolution below |
| `labels` | `labels` | `JSON.stringify()` |
| `annotations` | `annotations` | `JSON.stringify()` |
| `parent_session_id` | `parent_session_id` | direct (canonical type uses snake_case for this field) |

**Fields without SDK target** — these `CreateAgenticSessionRequest` fields have no corresponding SDK field. Drop silently:
- `stopOnRunFinished` — feature-flag gated, not yet wired in API server
- `sdkOptions` — feature-flag gated, not yet wired
- `runnerType` — internal orchestration concern, not exposed via API
- `userContext` — legacy backend enriches from auth context; API server handles differently
- `inactivityTimeout` — control-plane applies default, not an API server field

For updates, use `SessionPatchRequest` with the same field mapping. Only include changed fields — SDK PATCH semantics send only non-undefined fields.

## Workflow Resolution

The canonical type uses `activeWorkflow: { gitUrl, branch, path? }`. The SDK uses `workflow_id: string`.

**Response (workflow_id → activeWorkflow):** If `workflow_id` is present, call `client.workflows.get(workflow_id)` to retrieve `git_url`, `branch`, and `path`. Cache workflow lookups within a request to avoid repeated calls for list operations. If the workflow lookup fails (deleted workflow), set `activeWorkflow` to `undefined`.

**Request (activeWorkflow → workflow_id):** If `activeWorkflow.gitUrl` is provided, call `client.workflows.list({ search: "git_url = '...'" })` to resolve the ID. If no match, the adapter should create the workflow or throw — this is a design decision for the implementer. Document the chosen behavior.

For list responses with many sessions sharing the same `workflow_id`, batch-resolve: collect unique workflow IDs, fetch all in one `list({ search: "id in ('...', '...')" })` call, then map. TSL `in` queries are supported — confirmed by API server integration tests.

## Project Mapping

SDK `Project` → canonical `Project`. `name` → `name` (identifier), `display_name` → `displayName` (once SDK is fixed to include this field; until then, default to `name`), `description` → `description`, `labels`/`annotations` → parse JSON, `status` → direct (default `'Active'`), `created_at` → `creationTimestamp`, `id` → `uid`. Defaults: `isOpenShift` → `false`, `namespace` → same as `name`.

**SDK gap:** The API server model has `display_name` (see `plugins/projects/model.go:11`) but the SDK `Project` type currently omits it. Fix the SDK generator to include this field. Until fixed, derive `displayName` from `name`.

## Project Request Mapping

Canonical `CreateProjectRequest` / `UpdateProjectRequest` → SDK `ProjectCreateRequest` / `ProjectPatchRequest`.

| Canonical | SDK | Transform |
|---|---|---|
| `name` | `name` | direct (project identifier) |
| `displayName` | `display_name` | direct (requires SDK generator fix) |
| `description` | `description` | direct |
| `labels` | `labels` | `JSON.stringify()` |
| `annotations` | `annotations` | `JSON.stringify()` |

## ScheduledSession Architecture

The API server decomposes what the CRD bundles into separate resources:

- **Agent config** (model, repos, workflow, env vars, prompt) → `Agent` resource
- **Schedule config** (cron, timezone, enabled) → `ScheduledSession` resource
- **Execution config** (timeout, inactivityTimeout, stopOnRunFinished, runnerType) → ScheduledSession fields (prerequisite: add these to API server model — they are session execution params, not agent config)

The v2 adapter bridges this transparently. Consumers continue to work with `sessionTemplate` (the CRD-shaped bundled config) — the adapter creates/resolves Agents behind the port boundary.

```
Consumer → ScheduledSessionsPort → v2 adapter → { Agent API + ScheduledSession API }
```

## ScheduledSession Mapping (Response)

SDK `ScheduledSession` → canonical `ScheduledSession`. The adapter must resolve `agent_id` to reconstruct `sessionTemplate`.

**Step 1: Resolve Agent** — fetch Agent by `agent_id` via `client.agents.get(agentId)`. Cache Agent lookups within a request to avoid repeated calls for list operations (collect unique `agent_id` values, batch-fetch).

**Step 2: Reconstruct sessionTemplate from Agent:**

| Agent field | sessionTemplate field | Transform |
|---|---|---|
| `prompt` | `initialPrompt` | direct |
| `llm_model` | `llmSettings.model` | nest |
| `llm_temperature` | `llmSettings.temperature` | nest |
| `llm_max_tokens` | `llmSettings.maxTokens` | nest |
| `repo_url` | `repos[0].url` | wrap in single-element array (if non-null) |
| `workflow_id` | `activeWorkflow` | resolve workflow → `{ gitUrl, branch, path }` (reuse Workflow Resolution) |
| `environment_variables` | `environmentVariables` | parse JSON to `Record<string, string>` |

**Step 3: Map direct ScheduledSession fields:**

| SDK field | Canonical field | Transform |
|---|---|---|
| `name` | `name` | direct |
| `name` | `displayName` | fallback (until `display_name` is added to model) |
| `schedule` | `schedule` | direct |
| `enabled` | `suspend` | invert (`!enabled`) |
| `last_run_at` | `lastScheduleTime` | direct |
| `created_at` | `creationTimestamp` | direct |

**Step 4: Fields without SDK source — default to safe values:**
- `namespace` → project name
- `activeCount` → `0` (no API server equivalent; runtime state)
- `reuseLastSession` → `false` (no API server equivalent)
- `labels`, `annotations` → parse JSON or `{}`

**When `agent_id` is absent** (once made optional on the API server): use `session_prompt` → `sessionTemplate.initialPrompt`, leave other sessionTemplate fields at defaults.

## ScheduledSession Request Mapping (Create)

Canonical `CreateScheduledSessionRequest` → implicit Agent creation + SDK ScheduledSession creation. The adapter decomposes `sessionTemplate` into agent-level and schedule-level config.

**Step 1: Create Agent from sessionTemplate**

| sessionTemplate field | Agent field | Transform |
|---|---|---|
| `initialPrompt` | `prompt` | direct |
| `llmSettings.model` | `llm_model` | flatten |
| `llmSettings.temperature` | `llm_temperature` | flatten |
| `llmSettings.maxTokens` | `llm_max_tokens` | flatten |
| `repos[0].url` | `repo_url` | extract first repo URL |
| `activeWorkflow` | `workflow_id` | resolve `{ gitUrl, branch, path }` → workflow ID (reuse Workflow Resolution) |
| `environmentVariables` | `environment_variables` | `JSON.stringify()` |

Agent `name`: generate from scheduled session name (e.g., `{displayName}-agent` or `scheduled-{timestamp}`).

**Step 2: Create ScheduledSession**

| Canonical field | SDK field | Transform |
|---|---|---|
| `displayName` | `name` | direct |
| `schedule` | `schedule` | direct |
| `suspend` | `enabled` | invert (`!suspend`) |
| (from step 1) | `agent_id` | reference created Agent |
| `sessionTemplate.initialPrompt` | `session_prompt` | direct (prompt override) |

**Fields without SDK target** — drop silently (execution params, prerequisite to add to API server model):
- `sessionTemplate.timeout` → future `timeout` field on ScheduledSession
- `sessionTemplate.inactivityTimeout` → future `inactivity_timeout` field
- `sessionTemplate.stopOnRunFinished` → future `stop_on_run_finished` field
- `sessionTemplate.runnerType` → future `runner_type` field
- `reuseLastSession` → no API server equivalent

## ScheduledSession Request Mapping (Update)

For updates that change `sessionTemplate` fields: update the referenced Agent via `client.agents.update(agentId, patch)` with the changed agent-level fields. For updates that only change `schedule`, `suspend`, or `displayName`: patch the ScheduledSession only. If both change, patch both.

## ScheduledSession Delete + Agent Cleanup

On delete: delete the ScheduledSession via `client.scheduledSessions.delete(id)`. For the auto-created Agent, check if any other ScheduledSessions reference the same `agent_id` (query `client.scheduledSessions.list({ search: "agent_id = '...'" })`). If no other references exist, delete the Agent. If other references exist, leave it.

No cascade delete exists at the DB level — the adapter handles cleanup explicitly.

## ScheduledSessionsPort Method Status

All 9 port methods are implementable via the SDK. No `NOT_IMPLEMENTED` throws needed.

| Method | v2 Strategy | Notes |
|---|---|---|
| `listScheduledSessions` | Implement | `client.scheduledSessions.list()` + batch Agent resolution |
| `getScheduledSession` | Implement | `client.scheduledSessions.get()` + Agent resolution |
| `createScheduledSession` | Implement | Create Agent → create ScheduledSession (see Request Mapping) |
| `updateScheduledSession` | Implement | Patch Agent and/or ScheduledSession (see Update mapping) |
| `deleteScheduledSession` | Implement | Delete ScheduledSession + conditional Agent cleanup |
| `suspendScheduledSession` | Implement | `client.scheduledSessions.suspend()` |
| `resumeScheduledSession` | Implement | `client.scheduledSessions.resume()` |
| `triggerScheduledSession` | Implement | `client.scheduledSessions.trigger()` |
| `listScheduledSessionRuns` | Implement | `client.scheduledSessions.runs()` (returns sessions — reuse Session Mapping) |

## SDK Prerequisites for ScheduledSessions

1. **OpenAPI spec**: Add ScheduledSession schemas and paths to `openapi.yaml` — the plugin exists but is not registered in the OpenAPI spec (no paths, no schemas). Add under the `# AUTO-ADD NEW PATHS` and `# AUTO-ADD NEW SCHEMAS` markers.
2. **TS SDK regeneration**: Run `make generate-sdk` after OpenAPI update to get TS types and client methods.
3. **`agent_id` optionality**: Update API server validation to make `agent_id` optional on ScheduledSession create — sessions should be able to exist without an agent.
4. **Execution fields**: Add `timeout`, `inactivity_timeout`, `stop_on_run_finished`, `runner_type` to the ScheduledSession model — these are session execution params, not agent config.

## Pagination

Create `services/adapters/v2/pagination.ts`. The SDK returns `{ kind, page, size, total, items }`. Transform to `PaginatedResult<T>`:

- `totalCount` = `total`
- `hasMore` = `page * size < total`
- `nextPage` = fetch with `page + 1`, or `undefined` when `!hasMore`
- Apply a `transform` function to each item during pagination

This differs from the v1 helper which uses offset-based math.

**Input parameter conversion.** Consumers pass `PaginationParams` (`limit`, `offset`). The SDK uses `ListOptions` (`page`, `size`). The page-based API can only start at page boundaries, so convert by snapping to the nearest page:

```typescript
function toListOptions(params?: PaginationParams): ListOptions | undefined {
  if (!params) return undefined
  const size = params.limit ?? 20
  const page = params.offset ? Math.floor(params.offset / size) + 1 : 1
  return { page, size, search: params.search }
}
```

**Limitation:** arbitrary offsets that don't align to page boundaries (e.g., `offset=15, limit=20`) snap to the nearest page start. This is acceptable because all existing consumers use page-aligned offsets (offset increments by `limit`). `params.continue` (K8s continuation token) has no SDK equivalent — ignored in v2.

## Error Mapping

The SDK throws `AmbientAPIError` with `statusCode`, `code`, `reason`, `operationId`. Catch and rethrow as `ApiClientError(reason, code, { operationId })`. Backend-specific fields don't leak to consumers.

## Port Methods Without SDK Support

Not all port methods have API server equivalents. These methods throw `ApiClientError` with code `'NOT_IMPLEMENTED'` until the API server adds support.

**SessionsPort** (6 of 12 methods affected):

| Method | Strategy | Rationale |
|---|---|---|
| `cloneSession` | throw `NOT_IMPLEMENTED` | No clone endpoint in API server |
| `getSessionPodEvents` | throw `NOT_IMPLEMENTED` | K8s-only — no Postgres equivalent |
| `updateSessionDisplayName` | **Implement** via `client.sessions.update(id, { name })` | Maps to `SessionPatchRequest.name` |
| `getSessionExport` | throw `NOT_IMPLEMENTED` | No export endpoint in API server |
| `switchSessionModel` | **Implement** via `client.sessions.update(id, { llm_model })` | Maps to `SessionPatchRequest.llm_model` |
| `saveToGoogleDrive` | throw `NOT_IMPLEMENTED` | External integration, not API server concern |

Net: 3 methods implementable via SDK patch, 3 throw `NOT_IMPLEMENTED`.

**ProjectsPort** (3 of 8 methods affected):

| Method | Strategy | Rationale |
|---|---|---|
| `getProjectIntegrationStatus` | throw `NOT_IMPLEMENTED` | Sub-resource not modeled in API server |
| `getProjectMcpServers` | throw `NOT_IMPLEMENTED` | Stored in `ProjectSettings`, separate domain |
| `updateProjectMcpServers` | throw `NOT_IMPLEMENTED` | Stored in `ProjectSettings`, separate domain |

Note: `ProjectSettings` has its own SDK API (`client.projectSettings`). MCP server methods could be wired through that API in a future iteration.

**ScheduledSessionsPort**: All 9 methods are implementable — see ScheduledSessionsPort Method Status above.

## Cache Isolation

`BACKEND_VERSION` in `services/queries/query-keys.ts` prefixes all cache keys. Bump to `'v2'` when migrating domains to prevent stale v1 data from being served.

## Migration

To switch a domain from v1 to v2, change the barrel export:

```typescript
// services/adapters/sessions.ts
// Before: export * from './v1/sessions'
export * from './v2/sessions'
```

No hook or component changes needed.

## Testing

Mock SDK client methods with `vi.fn()`. Provide SDK-shaped responses (flat, snake_case, JSONB as strings). Assert canonical types come out (nested, camelCase, parsed). Test: pagination, error mapping, JSONB parse failures (malformed strings → fallback defaults), unmapped fields are dropped, missing fields get defaults, request mapping serializes collections, unsupported methods throw `NOT_IMPLEMENTED`, return type normalization produces correct shapes, name→ID resolution with KSUID fast path and search fallback.

**ScheduledSession-specific tests:**
- Agent creation from sessionTemplate decomposition (create path)
- Agent resolution on read (reconstruct sessionTemplate from Agent config)
- Batch Agent resolution for list operations (dedup by `agent_id`)
- Update routing: sessionTemplate change → Agent patch, schedule-only change → ScheduledSession patch only
- Delete with Agent cleanup check (no other references → delete Agent; other references → keep)
- `enabled` ↔ `suspend` inversion in both directions
- Missing `agent_id` fallback (`session_prompt` → `sessionTemplate.initialPrompt`, defaults for rest)

## Key Files

- SDK types: `components/ambient-sdk/ts-sdk/src/{session,project,base}.ts`
- SDK client: `components/ambient-sdk/ts-sdk/src/{client,session_api,project_api}.ts`
- SDK generator: `components/ambient-sdk/generator/`
- Canonical types: `components/frontend/src/types/api/sessions.ts`, `components/frontend/src/types/api/scheduled-sessions.ts`
- Ports: `components/frontend/src/services/ports/{sessions,projects,scheduled-sessions,types}.ts`
- v1 reference: `components/frontend/src/services/adapters/v1/sessions.ts`, `components/frontend/src/services/adapters/v1/scheduled-sessions.ts`
- Auth: `components/frontend/src/lib/auth.ts`
- Query keys: `components/frontend/src/services/queries/query-keys.ts`
- API server ScheduledSession plugin: `components/ambient-api-server/plugins/scheduledSessions/`
- API server Agent plugin: `components/ambient-api-server/plugins/agents/`
- Go SDK ScheduledSession types: `components/ambient-sdk/go-sdk/types/scheduled_session.go`
- Migration roadmap: `components/ambient-api-server/DATA_MODEL_COMPARISON.md`
