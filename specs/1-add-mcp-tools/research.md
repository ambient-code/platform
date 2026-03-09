# Research: Add Context7 and DeepWiki MCP Tools + Generic MCP Credential Framework

## Part 1: Context7 and DeepWiki Decisions

### Decision 1: Where to configure MCP servers

**Decision**: Add entries to `components/runners/ambient-runner/.mcp.json`

**Rationale**: This is the existing central MCP configuration file loaded by `load_mcp_config()` and passed to the Claude Agent SDK via `build_mcp_servers()`. All existing MCP servers are defined here.

**Alternatives rejected**: Per-workflow config (duplicates), CRD-level config (requires operator changes), env var injection (`.mcp.json` is the established pattern).

### Decision 2: Tool authorization

**Decision**: No changes needed. `build_allowed_tools()` auto-generates `mcp__{server_name}__*` wildcards for all servers in `.mcp.json`.

### Decision 3: MCP transport type

**Decision**: Use `"type": "http"` with `"url"` (streamable HTTP). The SDK supports it natively. `load_mcp_config()` is transport-agnostic.

**Verification needed**: Confirm deployed SDK version supports HTTP MCP transport.

### Decision 4: Context7/DeepWiki authentication

**Decision**: No authentication needed today. Both are public endpoints.

### Decision 5: E2E test config

**Decision**: No changes to `.mcp.e2e.json`. It's intentionally empty for isolated testing.

---

## Part 2: Generic MCP Credential Framework Design

### Current Architecture (Bespoke Pattern)

```
User (Integrations Page)
  → Backend stores in per-integration K8s Secret (jira-credentials, google-oauth-credentials, etc.)
    → Operator creates Pod with identity env vars (NO credential tokens)
      → Runner calls per-integration backend API at runtime
        → Runner runs per-integration env var population logic
          → .mcp.json env block consumes via ${VAR} expansion
```

Each integration has: backend handler, runner fetch function, runner injection logic, auth check, frontend card. ~5 files touched per new integration.

### Generic Architecture (Target)

```
User (Integrations Page → Generic MCP Credential Card)
  → Backend stores in single K8s Secret "mcp-server-credentials"
    → Key format: {serverName}:{sanitizedUserID}
    → Value: JSON with arbitrary credential fields
      → Runner calls generic endpoint: GET /credentials/mcp/{serverName}
        → Runner maps credential fields to env vars using naming convention
          → .mcp.json env block consumes via ${VAR} expansion
```

One set of handlers serves ALL MCP servers. Adding a new server = `.mcp.json` edit only.

### Decision 6: K8s Secret structure

**Decision**: Single Secret named `mcp-server-credentials`. Keys: `{serverName}:{sanitizedUserID}`. Values: JSON-marshaled credential fields.

**Rationale**: Follows the existing pattern (Jira uses `jira-credentials` with `userID` keys). Single Secret simplifies RBAC and reduces Secret count. K8s Secrets have a 1MB limit, but credential payloads are small (a few hundred bytes per user per server) — this supports thousands of user/server combinations.

**Alternative considered**: One Secret per server name (e.g., `mcp-context7-credentials`). Simpler per-server access patterns but creates Secret proliferation. Rejected in favor of single Secret for simplicity; can be revisited if size becomes an issue.

### Decision 7: Backend API design

**Decision**: Parameterized endpoints using `:serverName` path parameter.

**Endpoints**:

| Method | Path | Purpose | Auth |
|--------|------|---------|------|
| POST | `/auth/mcp/:serverName/connect` | Store credentials for user | User token |
| GET | `/auth/mcp/:serverName/status` | Check if user has credentials | User token |
| DELETE | `/auth/mcp/:serverName/disconnect` | Remove user's credentials | User token |
| GET | `/projects/:project/agentic-sessions/:session/credentials/mcp/:serverName` | Runtime fetch | Session ownership |

**Rationale**: Mirrors the existing per-integration pattern but parameterized. Same auth model: user token for CRUD, session ownership check for runtime fetch.

**Credential schema**: Arbitrary JSON fields stored as-is. The backend doesn't validate field names — it stores whatever the frontend sends. The runner and `.mcp.json` env block determine which fields are consumed.

```json
// Example stored credential for "some-private-server"
{
  "userId": "user-123",
  "serverName": "some-private-server",
  "fields": {
    "apiKey": "sk-...",
    "baseUrl": "https://custom.endpoint.com"
  },
  "updatedAt": "2026-03-06T12:00:00Z"
}
```

### Decision 8: Env var naming convention for credential injection

**Decision**: Convention-based mapping: `MCP_{UPPERCASED_SERVER_NAME}_{UPPERCASED_FIELD_NAME}`.

Example: Server `context7` with field `apiKey` → env var `MCP_CONTEXT7_APIKEY`.

**Rationale**: Predictable naming allows `.mcp.json` authors to reference credentials without special configuration:

```json
{
  "context7": {
    "type": "http",
    "url": "https://mcp.context7.com/mcp",
    "env": {
      "CONTEXT7_API_KEY": "${MCP_CONTEXT7_APIKEY}"
    }
  }
}
```

**Alternative considered**: Explicit mapping in a separate config file. Rejected — adds complexity with no clear benefit. Convention-based is simpler and self-documenting.

### Decision 9: Runner credential fetching strategy

**Decision**: In `populate_runtime_credentials()`, after existing integration fetches, iterate over all MCP servers from config that have an `env` block with `${MCP_*}` references. For each, call the generic backend endpoint. Map response fields to env vars using the naming convention.

**Key design choice**: The runner determines which servers need credentials by inspecting the `.mcp.json` env blocks for `${MCP_*}` patterns. Servers without env blocks (like context7/deepwiki today) are skipped. This is a pull model — the runner asks for what it needs rather than the backend pushing everything.

**Error handling**: Missing credentials are logged as warnings, not errors. The session continues — the MCP server may fail to connect, but other tools remain unaffected.

### Decision 10: Generic MCP auth status check

**Decision**: Extend `check_mcp_authentication()` with a generic fallback. For servers not matching any bespoke case, check if the env vars matching the `MCP_{SERVER}_{FIELD}` convention are populated.

**Rationale**: Reuses existing auth check infrastructure. The `/mcp/status` endpoint already queries all servers — this makes it report correctly for generic credential-backed servers.

### Decision 11: Frontend approach

**Decision**: Add a generic "MCP Servers" section to the Integrations page. It renders a card for each MCP server that has credential requirements (detected from a backend endpoint that returns the list of servers accepting credentials). Each card has a simple key-value form for credential fields.

**Rationale**: Reuses the existing IntegrationsClient pattern. Cards are dynamic — no new component needed per server.

**What the backend provides**: A new `GET /auth/mcp/servers` endpoint that returns the list of MCP servers configured in `.mcp.json` that have `env` blocks (indicating they accept credentials). This list is derived from the runner config.

**Alternative considered**: Hardcoding server names in the frontend. Rejected — defeats the purpose of a generic framework.

### Decision 12: Not migrating existing integrations

**Decision**: Existing bespoke integrations (Jira, Google, GitHub, GitLab) remain as-is. The generic framework is for NEW MCP servers going forward.

**Rationale**: Migrating existing integrations is risky and high-effort with no user-visible benefit. The bespoke handlers work well. The generic framework proves itself with new servers first.

---

## Existing Credential CRUD Pattern Reference

### Backend Pattern (from jira_auth.go)

- **Secret**: `corev1.SecretTypeOpaque`, cluster-wide namespace
- **Labels**: `app: ambient-code`, `ambient-code.io/provider: {name}`
- **CRUD**: 3-retry optimistic locking for writes, nil-on-not-found for reads, idempotent deletes
- **Auth**: `GetK8sClientsForRequest(c)` for user token validation, `isValidUserID()` for input sanitization
- **Runtime fetch**: Session ownership via CR read + `spec.userContext.userId` check

### Runner Pattern (from auth.py)

- **Fetch**: `_fetch_credential(context, credential_type)` → HTTP GET to backend API
- **Auth**: `BOT_TOKEN` header on requests
- **Injection**: Set `os.environ[VAR_NAME]` before MCP config loading
- **Error handling**: `try/except` with warning log, no session abort

### Frontend Pattern (from IntegrationsClient.tsx)

- **Status query**: `useIntegrationsStatus()` hook → `GET /auth/integrations/status`
- **Cards**: Per-integration React components with connect/edit/disconnect actions
- **State**: React Query for cache invalidation on connect/disconnect
