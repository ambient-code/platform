# Implementation Plan: Gerrit Integration Connector

**Branch**: `001-gerrit-integration` | **Date**: 2026-04-17 | **Spec**: [spec.md](spec.md)

## Summary

Add Gerrit as a native integration following the established Jira/CodeRabbit pattern. Multi-instance support with HTTP Basic and gitcookies auth methods, SSRF protection, and MCP server config generation for agentic sessions. Gated behind a feature flag.

## Technical Context

**Language/Version**: Go 1.23 (backend), TypeScript/Next.js (frontend), Python 3.12 (runner)
**Primary Dependencies**: Gin (HTTP), client-go (K8s), React Query, Shadcn/ui
**Storage**: Kubernetes Secrets (per-user credential storage)
**Testing**: Ginkgo v2 (backend), Vitest (frontend), pytest (runner)
**Target Platform**: Kubernetes cluster
**Constraints**: Follow existing integration patterns (Jira as canonical reference)

## Project Structure

### Files to Create

```text
components/backend/handlers/gerrit_auth.go              # Handlers: Connect, Status, Disconnect, List
components/backend/handlers/gerrit_auth_test.go          # Ginkgo v2 test suite
components/frontend/src/components/gerrit-connection-card.tsx
components/frontend/src/services/api/gerrit-auth.ts
components/frontend/src/services/queries/use-gerrit.ts
components/frontend/src/app/api/auth/gerrit/connect/route.ts
components/frontend/src/app/api/auth/gerrit/test/route.ts
components/frontend/src/app/api/auth/gerrit/instances/route.ts
components/frontend/src/app/api/auth/gerrit/[instanceName]/status/route.ts
components/frontend/src/app/api/auth/gerrit/[instanceName]/disconnect/route.ts
docs/gerrit-integration.md
```

### Files to Modify

```text
components/backend/handlers/integration_validation.go    # Add ValidateGerritToken, parseGitcookies, TestGerritConnection
components/backend/handlers/integrations_status.go       # Add getGerritStatusForUser to GetIntegrationsStatus
components/backend/handlers/runtime_credentials.go       # Add GetGerritCredentialsForSession
components/backend/routes.go                             # Register Gerrit routes
components/frontend/src/app/integrations/IntegrationsClient.tsx  # Add GerritConnectionCard
components/frontend/src/services/api/integrations.ts     # Add gerrit to IntegrationsStatus type
components/runners/ambient-runner/ambient_runner/platform/auth.py        # Add fetch_gerrit_credentials
components/runners/ambient-runner/ambient_runner/platform/__init__.py    # Export fetch_gerrit_credentials
components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py   # Add generate_gerrit_config
components/runners/ambient-runner/.mcp.json              # Add gerrit MCP server entry
components/manifests/base/core/flags.json                # Add gerrit.enabled feature flag
```

## Implementation Phases

### Phase 1: Feature Flag + Backend Handlers

**Reference**: `jira_auth.go` (269 lines), `integration_validation.go` (229 lines)

1. Add `gerrit.enabled` flag to `flags.json` with `scope:workspace` tag
2. Create `gerrit_auth.go` with:
   - `GerritCredentials` struct (UserID, InstanceName, URL, AuthMethod, Username, HTTPToken, GitcookiesContent, UpdatedAt)
   - `ConnectGerrit` handler — validates instance name (regex), validates URL (SSRF), validates auth method exclusivity, validates credentials, stores in Secret
   - `GetGerritStatus` handler — single instance status lookup
   - `DisconnectGerrit` handler — remove instance from Secret with conflict retry
   - `ListGerritInstances` handler — all instances sorted by name
   - `storeGerritCredentials` — K8s Secret CRUD with 3x conflict retry (follows Jira pattern)
   - `getGerritCredentials`, `listGerritCredentials`, `deleteGerritCredentials` — Secret data access
   - SSRF: `validateGerritURL` (scheme + DNS resolution + IP range check), `isPrivateOrBlocked` (all RFC ranges), `ssrfSafeTransport` (custom dialer re-validates at connection time)
3. Add to `integration_validation.go`:
   - `ValidateGerritToken(ctx, url, authMethod, username, httpToken, gitcookiesContent) (bool, error)` — validates against `/a/accounts/self` with 15s timeout and SSRF-safe transport
   - `parseGitcookies(gerritURL, content) (string, error)` — parses tab-delimited format, subdomain flag logic
   - `TestGerritConnection` handler — validates without storing
4. Add to `integrations_status.go`: `getGerritStatusForUser` returning instances array
5. Add to `runtime_credentials.go`: `GetGerritCredentialsForSession` with RBAC via `enforceCredentialRBAC`
6. Register routes in `routes.go`

**K8s client usage**: `K8sClient` (service account) for Secret CRUD. `GetK8sClientsForRequest(c)` for user auth validation. Follows K8S_CLIENT_PATTERNS.md.

### Phase 2: Backend Tests

**Reference**: Existing Ginkgo test suites in handlers/

1. Create `gerrit_auth_test.go` with Ginkgo v2:
   - Auth token required checks
   - Instance name validation (valid/invalid patterns)
   - URL validation (HTTPS enforcement, private IP rejection)
   - Mixed credential rejection
   - Valid HTTP Basic credential flow
   - Valid gitcookies credential flow
   - Per-user Secret isolation
   - Disconnect and list operations
   - SSRF edge cases (loopback, metadata, CGNAT, DNS rebinding)
2. Use `test_utils.HTTPTestUtils` and `test_utils.K8sTestUtils`
3. Mock validation via package-level var: `var validateGerritTokenFn = ValidateGerritToken`

### Phase 3: Frontend

**Reference**: `jira-connection-card.tsx` (263 lines), `jira-auth.ts` (35 lines), `use-jira.ts` (31 lines)

1. Create `gerrit-auth.ts` — types (GerritAuthMethod, GerritConnectRequest as discriminated union, GerritInstanceStatus, etc.) and API functions
2. Create `use-gerrit.ts` — React Query hooks: `useGerritInstances`, `useConnectGerrit`, `useDisconnectGerrit`, `useTestGerritConnection` with cache invalidation on `['integrations', 'status']` and `['gerrit', 'instances']`
3. Create Next.js proxy routes (5 files) — follow Jira route pattern with `buildForwardHeadersAsync`; test route gets 15s `AbortSignal.timeout()`
4. Create `gerrit-connection-card.tsx` — multi-instance card:
   - Instance list with green status indicators
   - Add form with: instance name, URL, auth method radio (http_basic/git_cookies), conditional fields
   - Clear other auth method fields when switching radio buttons
   - Test and Save buttons, show/hide token toggle
   - Client-side validation: instance name min 2 chars, URL required, auth fields required
   - Gate with `useWorkspaceFlag(projectName, 'gerrit.enabled')`
5. Add to `IntegrationsClient.tsx` — import and render GerritConnectionCard
6. Update `IntegrationsStatus` type to include gerrit

### Phase 4: Runner Integration

**Reference**: `auth.py` (fetch_jira_credentials pattern), `mcp.py`

1. Add `fetch_gerrit_credentials(context)` to `auth.py`:
   - Calls `_fetch_credential(context, "gerrit")`
   - Returns list of instance dicts
   - Handles PermissionError (auth failure) vs other errors (network)
2. Update `populate_runtime_credentials`:
   - Add gerrit to `asyncio.gather` alongside google/jira/gitlab/github
   - On success: call `generate_gerrit_config(instances)`
   - On PermissionError: add to `auth_failures`, clear config
   - On other error: log warning, preserve stale config
3. Add `generate_gerrit_config(instances)` to `mcp.py`:
   - Creates `/tmp/gerrit-mcp/` directory
   - Writes `gerrit_config.json` with `gerrit_hosts` array
   - For git_cookies instances: writes combined `.gitcookies` file (0o600)
   - Sets `GERRIT_CONFIG_PATH` env var
   - Cleans up stale config on each call
4. Export `fetch_gerrit_credentials` from `__init__.py`
5. Add gerrit entry to `.mcp.json`

### Phase 5: Documentation

1. Create `docs/gerrit-integration.md` covering setup, auth methods, multi-instance usage, API reference, security, troubleshooting

## Verification

```bash
cd components/backend && make test                    # Backend tests pass
cd components/frontend && npm run build               # Zero errors, zero warnings
cd components/runners/ambient-runner && python -m pytest tests/  # Runner tests pass
make lint                                             # Pre-commit hooks pass
```
