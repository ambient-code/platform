# Tasks: Add Context7/DeepWiki MCP Tools + Generic MCP Credential Framework

## Status

| Field       | Value                                  |
|-------------|----------------------------------------|
| Branch      | 1-add-mcp-tools                        |
| Jira        | RHOAIENG-52212                         |
| Spec        | [spec.md](spec.md)                     |
| Plan        | [plan.md](plan.md)                     |
| Total Tasks | 21                                     |

## User Story Mapping

| Story | Description | Spec Reference | Priority |
|-------|-------------|----------------|----------|
| US1   | Agent looks up library documentation via Context7 | Scenario 1, FR-1 | P1 |
| US2   | Agent researches upstream dependency via DeepWiki | Scenario 2, FR-2 | P1 |
| US3   | Existing workflows remain unaffected | Scenario 3, FR-5 | P1 |
| US4   | Admin adds authenticated MCP server with zero bespoke code | Scenario 4, FR-6–FR-10 | P1 |
| US5   | User without credentials has graceful degradation | Scenario 5, FR-8.3 | P2 |

---

## Phase 1: Setup & Prerequisites

**Goal**: Confirm SDK support and verify the change won't break existing functionality.

- [x] T001 Verify Claude Agent SDK supports HTTP-type MCP transport (`"type": "http"` with `"url"`) by checking SDK version/docs or running a local test

**Completion criteria**: Confirmed HTTP MCP transport works, or identified fallback.

---

## Phase 2: Context7 + DeepWiki (US1, US2)

**Goal**: Add documentation tools to the platform. This is the immediate deliverable.

- [x] T002 [US1] [US2] Add `context7` and `deepwiki` server entries to `components/runners/ambient-runner/.mcp.json` — preserve all existing entries, add new entries with `"type": "http"` and `"url"`

**Completion criteria**: `.mcp.json` valid JSON with all 5 servers. `load_mcp_config()` returns all 5. `build_allowed_tools()` includes `mcp__context7__*` and `mcp__deepwiki__*`.

---

## Phase 3: Backend — Generic MCP Credential CRUD (US4)

**Goal**: Build the generic credential storage and API endpoints.

- [x] T003 [US4] Create `components/backend/handlers/mcp_credentials.go` with `MCPServerCredentials` struct (userId, serverName, fields map[string]string, updatedAt) and K8s Secret CRUD functions: `storeMCPCredentials()`, `getMCPCredentials()`, `deleteMCPCredentials()` — using Secret name `mcp-server-credentials`, key format `{serverName}:{sanitizedUserID}`, 3-retry optimistic locking pattern from `jira_auth.go`
- [x] T004 [US4] Add HTTP handlers in `components/backend/handlers/mcp_credentials.go`: `ConnectMCPServer` (POST), `GetMCPServerStatus` (GET), `DisconnectMCPServer` (DELETE) — with user token auth via `GetK8sClientsForRequest`, userID validation via `isValidUserID`, and server name validation (lowercase alphanumeric + hyphens, max 63 chars)
- [x] T005 [US4] Add runtime fetch handler `GetMCPCredentialsForSession` in `components/backend/handlers/mcp_credentials.go` — with session ownership check pattern from `runtime_credentials.go` (read session CR → extract `spec.userContext.userId` → verify caller owns session)
- [x] T006 [US4] Register all generic MCP credential routes in `components/backend/routes.go`: cluster-level auth group for connect/status/disconnect (`/auth/mcp/:serverName/*`), project group for runtime fetch (`/agentic-sessions/:sessionName/credentials/mcp/:serverName`)
- [x] T007 [P] [US4] Add MCP server credentials to integrations status aggregator in `components/backend/handlers/integrations_status.go` — return `mcpServers` field in status response with per-server connected/disconnected state

**Completion criteria**: Backend compiles. CRUD operations work against K8s Secret. Routes registered. Auth enforced on all endpoints.

---

## Phase 4: Runner — Generic Credential Fetching & Auth Check (US4, US5)

**Goal**: Runner automatically fetches and injects credentials for MCP servers that need them.

- [x] T008 [US4] Add generic MCP credential fetch function `_fetch_mcp_credentials(context, server_name)` in `components/runners/ambient-runner/ambient_runner/platform/auth.py` that calls `GET /credentials/mcp/{serverName}` using existing `_fetch_credential()` pattern
- [x] T009 [US4] Add `populate_mcp_server_credentials(context, mcp_servers)` function in `components/runners/ambient-runner/ambient_runner/platform/auth.py` — iterates MCP servers with `env` blocks containing `${MCP_*}` references, fetches credentials, maps fields to env vars using convention `MCP_{SERVER_NAME}_{FIELD_NAME}`
- [x] T010 [US4] Integrate `populate_mcp_server_credentials()` into bridge setup in `components/runners/ambient-runner/ambient_runner/bridges/claude/bridge.py` — call after `load_mcp_config()` but before `expand_env_vars()` in `_setup_platform()`
- [x] T011 [US4] [US5] Extend `check_mcp_authentication()` in `components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py` with generic fallback: for unrecognized server names, check if `MCP_{SERVER_NAME}_*` env vars are populated; return `(True, ...)` if found, `(None, None)` if not
- [x] T012 [US5] Verify graceful degradation: when credentials are missing for a server, the runner logs a warning but the session starts normally and other tools work

**Completion criteria**: Runner fetches generic credentials. Env vars populated before MCP config expansion. Missing credentials don't block sessions.

---

## Phase 5: Frontend — Generic MCP Credential UI (US4)

**Goal**: Users can manage MCP server credentials through the Integrations page.

- [x] T013 [P] [US4] Create API client `components/frontend/src/services/api/mcp-credentials.ts` with functions: `connectMCPServer(serverName, fields)`, `getMCPServerStatus(serverName)`, `disconnectMCPServer(serverName)`
- [x] T014 [P] [US4] Create React Query hooks `components/frontend/src/services/queries/use-mcp-credentials.ts` with: `useMCPServerStatus(serverName)`, `useConnectMCPServer()`, `useDisconnectMCPServer()` — following pattern from `use-jira.ts`
- [x] T015 [US4] Create generic credential card component `components/frontend/src/components/mcp-credential-card.tsx` — takes serverName and field definitions as props, renders dynamic form, shows connection status, handles connect/disconnect
- [x] T016 [US4] Add "MCP Servers" section to `components/frontend/src/app/integrations/IntegrationsClient.tsx` — renders `MCPCredentialCard` for each server that accepts credentials (empty initially since Context7/DeepWiki don't need auth, but infrastructure ready)

**Completion criteria**: Frontend builds with 0 errors, 0 warnings. MCP Servers section renders on Integrations page. No `any` types.

---

## Phase 6: Verification (US1–US5)

**Goal**: End-to-end verification of both parts.

- [ ] T017 [US1] Start a session and verify Context7 tools (`resolve-library-id`, `query-docs`) work without approval prompts
- [ ] T018 [US2] Start a session and verify DeepWiki tools (`read_wiki_structure`, `read_wiki_contents`, `ask_question`, `list_available_repos`) work without approval prompts
- [ ] T019 [US3] Verify existing MCP tools (webfetch, Jira, Google Workspace) still function; run `make test-e2e-local` for regression check
- [ ] T020 [US4] End-to-end test of generic credential framework: add a test MCP server entry to `.mcp.json` with `env` block referencing `${MCP_TESTSERVER_APIKEY}`, store credentials via POST `/auth/mcp/test-server/connect`, start a session, verify env var is populated and MCP status shows "configured"
- [ ] T021 Verify network egress from staging/production cluster to `mcp.context7.com:443` and `mcp.deepwiki.com:443`

---

## Dependencies

```
T001 → T002 (SDK verified before config change)
T002 has no dependency on T003–T016 (can ship independently)
T003 → T004, T005 (CRUD before handlers)
T004 + T005 → T006 (handlers before routes)
T006 → T007 (routes before status aggregator)
T008 → T009 (fetch function before population logic)
T009 → T010 (population logic before bridge integration)
T010 + T011 → T012 (integration before graceful degradation test)
T013 + T014 → T015 (API + hooks before card component)
T015 → T016 (card before integrations page)
T017–T021 after all implementation tasks
```

## Parallel Execution Opportunities

| Parallel Group | Tasks | Reason |
|----------------|-------|--------|
| Backend + Runner + Frontend | T003–T007, T008–T012, T013–T016 | Independent components, different languages |
| Verification | T017, T018, T019, T020, T021 | Independent test scenarios |
| Network check | T021 | Independent of code changes |

## Implementation Strategy

**MVP (shippable independently)**: T001 + T002 — Context7/DeepWiki in `.mcp.json`. Single JSON edit. Can merge immediately.

**Full delivery**: T001–T021 — Documentation tools + generic credential framework.

**Suggested execution order**:
1. T001 (gate: verify SDK HTTP transport)
2. T002 (immediate value: documentation tools live)
3. T003–T007 in sequence (backend credential framework)
4. T008–T012 in sequence (runner credential delivery) — can parallel with frontend
5. T013–T016 in sequence (frontend credential UI) — can parallel with runner
6. T017–T021 in parallel (verification)

## Summary

| Metric | Value |
|--------|-------|
| Total tasks | 21 |
| Part 1 (Context7/DeepWiki) | 2 tasks (T001–T002) |
| Part 2 Backend | 5 tasks (T003–T007) |
| Part 2 Runner | 5 tasks (T008–T012) |
| Part 2 Frontend | 4 tasks (T013–T016) |
| Verification | 5 tasks (T017–T021) |
| Per-story: US1 | 2 tasks |
| Per-story: US2 | 2 tasks |
| Per-story: US3 | 1 task |
| Per-story: US4 | 15 tasks |
| Per-story: US5 | 2 tasks |
| MVP scope | T001 + T002 (shippable independently) |
| Parallel opportunities | Backend/Runner/Frontend can develop simultaneously |
