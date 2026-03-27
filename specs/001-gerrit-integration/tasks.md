# Tasks: Gerrit Integration Connector

**Input**: Design documents from `/specs/001-gerrit-integration/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in the feature specification. Test tasks are omitted.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Backend**: `components/backend/`
- **Frontend**: `components/frontend/src/`
- **Runner**: `components/runners/ambient-runner/`

---

## Phase 1: Setup

**Purpose**: Bundle the Gerrit MCP server into the runner image and establish the foundation for integration work.

- [x] T001 Add Gerrit MCP server clone and build steps to runner Dockerfile at `components/runners/ambient-runner/Dockerfile` — clone `https://gerrit.googlesource.com/gerrit-mcp-server` at pinned commit `5666642afe1a5217e2529225d4bd9c9df6310bd6` (2026-03-23, master), run `./build-gerrit.sh`, copy built venv + source to final image under `/opt/gerrit-mcp-server/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Backend credential storage and validation — MUST be complete before frontend or runner work.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T002 Create `ValidateGerritToken()` function in `components/backend/handlers/integration_validation.go` — accepts URL, authMethod, username, httpToken, gitcookiesContent; makes `GET /a/accounts/self` request to the Gerrit instance using HTTP Basic Auth or gitcookies Cookie header; strips `)]}'` XSSI prefix from response; returns (bool, error) following the same pattern as `ValidateJiraToken()` and `ValidateGitLabToken()`
- [x] T003 Create `components/backend/handlers/gerrit_auth.go` with GerritCredentials struct (UserID, InstanceName, URL, AuthMethod, Username, HTTPToken, GitcookiesContent, UpdatedAt), K8s Secret helpers (store/get/delete/list using `gerrit-credentials` secret with `{instanceName}:{userID}` key format and label `ambient-code.io/provider: gerrit`), and five HTTP handlers: `ConnectGerrit` (POST, validates via `ValidateGerritToken` then stores), `TestGerritConnection` (POST, validates without storing), `GetGerritStatus` (GET with `:instanceName` param), `DisconnectGerrit` (DELETE with `:instanceName` param), `ListGerritInstances` (GET, returns all instances for user) — follow `jira_auth.go` patterns for K8s client usage, conflict retry, and error handling
- [x] T004 Add session-scoped Gerrit credential fetch handler `GetGerritCredentialsForSession` in `components/backend/handlers/runtime_credentials.go` — follows the same RBAC pattern as existing session credential endpoints; fetches ALL Gerrit instances for the effective user and returns them as a JSON array of instances with fields: instanceName, url, authMethod, username, httpToken, gitcookiesContent
- [x] T005 Add Gerrit status to unified integrations status in `components/backend/handlers/integrations_status.go` — add `getGerritStatusForUser()` helper that lists all Gerrit instances for the user; include `gerrit` field in the response object with an `instances` array of `{connected, instanceName, url, authMethod, updatedAt}`
- [x] T006 Register all Gerrit routes in `components/backend/routes.go` — cluster-level: `POST /api/auth/gerrit/connect`, `POST /api/auth/gerrit/test`, `GET /api/auth/gerrit/instances`, `GET /api/auth/gerrit/:instanceName/status`, `DELETE /api/auth/gerrit/:instanceName/disconnect`; session-scoped: `GET /api/projects/:projectName/agentic-sessions/:sessionName/credentials/gerrit`

**Checkpoint**: Backend API is complete — Gerrit credentials can be stored, validated, listed, and fetched via session endpoints.

---

## Phase 3: User Story 1 — Connect a Gerrit Instance (Priority: P1) MVP

**Goal**: Users can connect and disconnect Gerrit instances from the Integrations page with credential validation.

**Independent Test**: Navigate to Integrations page, enter Gerrit credentials (HTTP basic or gitcookies), click Connect, verify "Connected" status appears. Disconnect and verify status clears.

### Implementation for User Story 1

- [x] T007 [P] [US1] Create API client functions in `components/frontend/src/services/api/gerrit-auth.ts` — export `connectGerrit(data: GerritConnectRequest)`, `testGerritConnection(data: GerritConnectRequest)`, `getGerritInstances()`, `getGerritInstanceStatus(instanceName: string)`, `disconnectGerrit(instanceName: string)` using the existing `apiClient` pattern from `jira-auth.ts`
- [x] T008 [P] [US1] Add Gerrit types to `components/frontend/src/services/api/integrations.ts` — add `GerritAuthMethod`, `GerritInstanceStatus`, and `gerrit: { instances: GerritInstanceStatus[] }` field to the existing `IntegrationsStatus` type
- [x] T009 [US1] Create React Query hooks in `components/frontend/src/services/queries/use-gerrit.ts` — export `useConnectGerrit()`, `useDisconnectGerrit()`, `useTestGerritConnection()` mutations with query invalidation on `['integrations', 'status']` and `['gerrit', 'instances']` keys; export `useGerritInstances()` query hook — follow `use-jira.ts` patterns
- [x] T010 [US1] Create `components/frontend/src/components/gerrit-connection-card.tsx` — GerritConnectionCard component with: auth method toggle (radio/select: "HTTP Credentials" vs "Gitcookies"), conditional form fields (URL + username + HTTP token for http_basic; URL + gitcookies textarea for git_cookies), instance name input field, Connect/Test/Disconnect buttons, connection status indicator with server URL display, error/success toast messages — follow `jira-connection-card.tsx` layout and state patterns
- [x] T011 [US1] Add GerritConnectionCard to integrations page grid in `components/frontend/src/app/integrations/IntegrationsClient.tsx` — import and render `<GerritConnectionCard>` in the grid layout alongside existing cards, passing `status={integrations?.gerrit}` and `onRefresh={refetch}` props

**Checkpoint**: User Story 1 complete — users can connect/disconnect Gerrit instances via the UI with real-time validation and status display.

---

## Phase 4: User Story 2 — Use Gerrit Tools in Agentic Sessions (Priority: P1)

**Goal**: Sessions automatically have access to Gerrit MCP tools when the user has connected credentials.

**Independent Test**: Connect Gerrit credentials, create a session, ask the agent to "list my open changes on Gerrit" and verify it returns results.

### Implementation for User Story 2

- [x] T012 [P] [US2] Add Gerrit MCP server entry to `components/runners/ambient-runner/.mcp.json` — add `"gerrit"` server with `command` pointing to bundled Python venv (`/opt/gerrit-mcp-server/.venv/bin/python`), `args` set to `["/opt/gerrit-mcp-server/gerrit_mcp_server/main.py", "stdio"]`, and `env` block with `PYTHONPATH: "/opt/gerrit-mcp-server/"` and `GERRIT_CONFIG_PATH: "${GERRIT_CONFIG_PATH}"`
- [x] T013 [P] [US2] Add `fetch_gerrit_credentials()` function to `components/runners/ambient-runner/ambient_runner/platform/auth.py` — call `_fetch_credential(context, "gerrit")` to get all Gerrit instances from the backend session credential endpoint; return the parsed instances list
- [x] T014 [US2] Add Gerrit config generation to `components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py` — implement `generate_gerrit_config(instances)` that: (1) creates `/tmp/gerrit-mcp/` directory, (2) for each instance with `git_cookies` auth, writes gitcookies content to `/tmp/gerrit-mcp/.gitcookies` temp file, (3) builds `gerrit_config.json` with `gerrit_hosts` array mapping each instance to the Gerrit MCP server's native config format (name, external_url, authentication with type/username/auth_token or gitcookies_path), (4) writes config to `/tmp/gerrit-mcp/gerrit_config.json`, (5) sets `GERRIT_CONFIG_PATH` env var; call this from `build_mcp_servers()` before loading MCP config
- [x] T015 [US2] Add Gerrit auth check to `check_mcp_authentication()` in `components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py` — add case for server name `"gerrit"`: check if `GERRIT_CONFIG_PATH` is set and the config file exists; if yes return `(True, "Gerrit credentials configured")`; if not, attempt backend API fallback (same pattern as Jira); return `(False, "Gerrit not configured")` if no credentials found. Note: mid-session credential expiry is handled gracefully by the Gerrit MCP server itself (returns auth errors to the agent), so no additional error handling is needed in the runner
- [x] T016 [US2] Add Gerrit to `populate_runtime_credentials()` in `components/runners/ambient-runner/ambient_runner/platform/auth.py` — add `fetch_gerrit_credentials(context)` to the `asyncio.gather()` call; if instances are returned, call `generate_gerrit_config(instances)` to write the config file before MCP server startup
- [x] T017 [US2] Add Gerrit credential clearing to `clear_runtime_credentials()` in `components/runners/ambient-runner/ambient_runner/platform/auth.py` — remove `GERRIT_CONFIG_PATH` env var and delete `/tmp/gerrit-mcp/` directory (config file + gitcookies) after turn completes

**Checkpoint**: User Story 2 complete — sessions with connected Gerrit credentials can query changes, view diffs, and post comments via the 21 Gerrit MCP tools.

---

## Phase 5: User Story 3 — View Gerrit Integration Status Per Session (Priority: P2)

**Goal**: Users can see Gerrit connection status in the per-session integrations panel.

**Independent Test**: Open a session's settings/integrations panel, verify Gerrit shows "Connected" with URL (or "Not connected" with link to Integrations page).

### Implementation for User Story 3

- [x] T018 [US3] Add Gerrit status display to the session integrations panel in `components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/settings/integrations-panel.tsx` — add a Gerrit section that reads from the session's integration status; display "Connected" with instance URL(s) when Gerrit credentials exist, or "Not connected" with a link to `/integrations` when absent — follow the same display pattern used for Jira/GitLab in this panel

**Checkpoint**: User Story 3 complete — session-level Gerrit status is visible.

---

## Phase 6: User Story 4 — Connect Multiple Gerrit Instances (Priority: P3)

**Goal**: Users can connect multiple Gerrit instances and all are available in every session.

**Independent Test**: Connect two different Gerrit servers (e.g., openstack + android), start a session, verify the agent can interact with both.

### Implementation for User Story 4

- [x] T019 [US4] Update `GerritConnectionCard` in `components/frontend/src/components/gerrit-connection-card.tsx` to display a list of connected instances with individual disconnect buttons, and an "Add Instance" button that shows the connection form — when multiple instances are connected, show each as a row with instance name, URL, auth method, and a disconnect action
- [x] T020 [US4] Verify multi-instance `gerrit_config.json` generation in runner — ensure `generate_gerrit_config()` correctly maps multiple instances to multiple `gerrit_hosts` entries; the first instance's URL becomes `default_gerrit_base_url`; gitcookies from multiple instances are concatenated into a single `.gitcookies` file (one line per host)

**Checkpoint**: User Story 4 complete — multiple Gerrit instances are manageable and all available in sessions.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories.

- [x] T021 [P] Verify backend compiles cleanly: run `cd components/backend && gofmt -l . && go vet ./... && go build ./...`
- [x] T022 [P] Verify frontend builds cleanly: run `cd components/frontend && npm run build` — ensure zero errors and zero warnings
- [x] T023 [P] Verify runner installs cleanly: run `cd components/runners/ambient-runner && uv venv && uv pip install -e .`
- [ ] T024 Run end-to-end manual validation per quickstart.md: connect Gerrit instance via UI, verify status, create session, query changes via agent, disconnect and verify cleanup

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - US1 and US2 are both P1 but US2 depends on US1 (runner needs credentials connected via UI)
  - US3 can proceed in parallel with US2 (different files)
  - US4 extends US1 (multi-instance UI) and US2 (multi-instance config generation)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) — no other story dependencies
- **User Story 2 (P1)**: Can start after Foundational (Phase 2) — independent of US1 at code level but requires US1 for end-to-end testing
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) — independent of US1/US2
- **User Story 4 (P3)**: Extends US1 (UI) and US2 (runner config) — start after both are complete

### Within Each User Story

- API client before React Query hooks before components (frontend)
- Credential fetching before config generation (runner)
- Core implementation before integration with existing pages

### Parallel Opportunities

- T002 (validation) can run in parallel with T001 (Dockerfile) — different components
- T007 and T008 (frontend API + types) can run in parallel — different files
- T012 (.mcp.json) can run in parallel with T013 (auth.py) — different files
- T018 (US3 session panel) can run in parallel with T012-T017 (US2 runner work)
- T021, T022, T023 (build verifications) can all run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch API client and types in parallel:
Task: "T007 [P] [US1] Create API client in gerrit-auth.ts"
Task: "T008 [P] [US1] Add Gerrit types to integrations.ts"

# Then sequentially:
Task: "T009 [US1] Create React Query hooks in use-gerrit.ts"
Task: "T010 [US1] Create GerritConnectionCard component"
Task: "T011 [US1] Add card to IntegrationsClient.tsx grid"
```

## Parallel Example: User Story 2

```bash
# Launch MCP config and auth in parallel:
Task: "T012 [P] [US2] Add Gerrit entry to .mcp.json"
Task: "T013 [P] [US2] Add fetch_gerrit_credentials() to auth.py"

# Then sequentially:
Task: "T014 [US2] Add Gerrit config generation to mcp.py"
Task: "T015 [US2] Add Gerrit auth check to mcp.py"
Task: "T016 [US2] Add Gerrit to populate_runtime_credentials()"
Task: "T017 [US2] Add Gerrit to clear_runtime_credentials()"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (Dockerfile bundling)
2. Complete Phase 2: Foundational (backend API)
3. Complete Phase 3: User Story 1 (frontend connect/disconnect)
4. **STOP and VALIDATE**: Test connecting a real Gerrit instance
5. Deploy/demo if ready

### Incremental Delivery

1. Setup + Foundational -> Backend API ready
2. Add User Story 1 -> Users can connect Gerrit (MVP!)
3. Add User Story 2 -> Sessions can use Gerrit tools (core value!)
4. Add User Story 3 -> Session-level status visibility
5. Add User Story 4 -> Multi-instance support for power users
6. Polish -> Build verification and manual E2E validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- The Gerrit MCP server is an external dependency — pin to a specific commit hash in the Dockerfile for reproducibility
- Gerrit REST API responses have a `)]}'` XSSI prefix that must be stripped in validation code
