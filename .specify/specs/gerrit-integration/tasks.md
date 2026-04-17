# Tasks: Gerrit Integration Connector

**Input**: Design documents from `/specs/gerrit-integration/`
**Prerequisites**: plan.md, spec.md

## Phase 1: Feature Flag

- [ ] T001 [US6] Add `gerrit.enabled` feature flag to `components/manifests/base/core/flags.json` with `scope:workspace` tag

---

## Phase 2: Backend Core (Blocking)

- [ ] T002 [US1] Create `components/backend/handlers/gerrit_auth.go` — GerritCredentials struct, SSRF URL validation (validateGerritURL, isPrivateOrBlocked, ssrfSafeTransport), K8s Secret CRUD (store/get/list/delete with 3x conflict retry), ConnectGerrit handler, GetGerritStatus handler, DisconnectGerrit handler, ListGerritInstances handler. Follow jira_auth.go pattern. Use K8sClient for Secret ops, GetK8sClientsForRequest for auth.
- [ ] T003 [US1] Add Gerrit validation to `components/backend/handlers/integration_validation.go` — ValidateGerritToken (GET /a/accounts/self, 15s timeout, SSRF-safe transport, HTTP Basic and gitcookies support), parseGitcookies (tab-delimited format, subdomain flag logic), TestGerritConnection handler. Add `var validateGerritTokenFn = ValidateGerritToken` for test mockability.
- [ ] T004 [US6] Add `getGerritStatusForUser` to `components/backend/handlers/integrations_status.go` — return instances array, add to GetIntegrationsStatus response under "gerrit" key
- [ ] T005 [US5] Add `GetGerritCredentialsForSession` to `components/backend/handlers/runtime_credentials.go` — RBAC via enforceCredentialRBAC, returns all instances with auth details
- [ ] T006 Register Gerrit routes in `components/backend/routes.go` — POST connect, POST test, GET instances, GET :instanceName/status, DELETE :instanceName/disconnect, GET session credentials

**Checkpoint**: Backend API functional

---

## Phase 3: Backend Tests

- [ ] T007 [US1] Create `components/backend/handlers/gerrit_auth_test.go` — Ginkgo v2 suite with test_constants labels. Cover: auth token required, user context validation, instance name validation (valid/invalid), HTTPS enforcement, private IP rejection (loopback, metadata, CGNAT, RFC ranges), mixed credential rejection, valid HTTP Basic flow, valid gitcookies flow, per-user Secret isolation, disconnect, list sorted, DNS rebinding edge cases. Use HTTPTestUtils and K8sTestUtils, mock validateGerritTokenFn.

**Checkpoint**: `cd components/backend && make test` passes

---

## Phase 4: Frontend

- [ ] T008 [P] [US1] Create `components/frontend/src/services/api/gerrit-auth.ts` — GerritAuthMethod type, GerritConnectRequest (discriminated union), GerritTestRequest, GerritTestResponse, GerritInstanceStatus, GerritInstancesResponse types. API functions: connectGerrit, testGerritConnection, getGerritInstances, getGerritInstanceStatus, disconnectGerrit
- [ ] T009 [P] [US1] Create `components/frontend/src/services/queries/use-gerrit.ts` — useGerritInstances (queryKey ['gerrit','instances']), useConnectGerrit (invalidates ['integrations','status'] + ['gerrit','instances']), useDisconnectGerrit (same invalidation), useTestGerritConnection (no invalidation)
- [ ] T010 [P] [US1] Create Next.js proxy routes: `components/frontend/src/app/api/auth/gerrit/connect/route.ts`, `test/route.ts` (15s AbortSignal.timeout), `instances/route.ts`, `[instanceName]/status/route.ts`, `[instanceName]/disconnect/route.ts`. Follow jira route pattern with buildForwardHeadersAsync.
- [ ] T011 [US1] Create `components/frontend/src/components/gerrit-connection-card.tsx` — multi-instance card. Instance list with green status indicators. Add form: instance name input (auto-lowercase), URL input, auth method radio (http_basic/git_cookies), conditional fields (username+token with show/hide toggle OR gitcookies textarea). Clear other auth fields on radio switch. Test button, Save button. Client-side validation (name min 2 chars, required fields). Gate with useWorkspaceFlag(projectName, 'gerrit.enabled').
- [ ] T012 [US6] Add GerritConnectionCard to `components/frontend/src/app/integrations/IntegrationsClient.tsx` and add gerrit to IntegrationsStatus type in `components/frontend/src/services/api/integrations.ts`

**Checkpoint**: `cd components/frontend && npm run build` passes with 0 errors, 0 warnings

---

## Phase 5: Runner Integration

- [ ] T013 [US5] Add `fetch_gerrit_credentials(context)` to `components/runners/ambient-runner/ambient_runner/platform/auth.py` — calls _fetch_credential(context, "gerrit"), returns list of instance dicts. Export from `__init__.py`.
- [ ] T014 [US5] Update `populate_runtime_credentials` in auth.py — add gerrit to asyncio.gather. On success: call generate_gerrit_config. On PermissionError: add to auth_failures. On other error: log warning, preserve stale config.
- [ ] T015 [US5] Add `generate_gerrit_config(instances)` to `components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py` — creates /tmp/gerrit-mcp/, writes gerrit_config.json (gerrit_hosts array with name, external_url, authentication per instance), writes combined .gitcookies for git_cookies instances (0o600), sets GERRIT_CONFIG_PATH env var. Clean up stale config on each call. Handle empty list (clear env var).
- [ ] T016 [P] [US5] Add gerrit MCP server entry to `components/runners/ambient-runner/.mcp.json`

**Checkpoint**: `cd components/runners/ambient-runner && python -m pytest tests/` passes

---

## Phase 6: Documentation + Polish

- [ ] T017 [P] Create `docs/gerrit-integration.md` — overview, auth methods, multi-instance usage, instance naming rules, API reference, security (SSRF, credential storage, rotation), troubleshooting
- [ ] T018 Run `make lint` — all pre-commit hooks pass

---

## Dependencies & Execution Order

- **Phase 1** (flag): No deps, start immediately
- **Phase 2** (backend): Depends on Phase 1 (flag exists)
- **Phase 3** (tests): Depends on Phase 2
- **Phase 4** (frontend): Depends on Phase 2 (backend API exists); T008/T009/T010 can run in parallel; T011 depends on T008+T009; T012 depends on T011
- **Phase 5** (runner): Depends on Phase 2 (credential endpoint exists); T013-T016 are sequential except T016
- **Phase 6** (docs): Depends on all prior phases
