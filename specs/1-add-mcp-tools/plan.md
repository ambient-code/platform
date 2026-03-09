# Implementation Plan: Add Context7/DeepWiki MCP Tools + Generic MCP Credential Framework

## Status

| Field       | Value                                  |
|-------------|----------------------------------------|
| Status      | Ready                                  |
| Created     | 2026-03-06                             |
| Jira        | RHOAIENG-52212                         |
| Branch      | 1-add-mcp-tools                        |
| Spec        | [spec.md](spec.md)                     |
| Research    | [research.md](research.md)             |

## Technical Context

### Current State

- Runner `.mcp.json` has 3 subprocess-based MCP servers (webfetch, mcp-atlassian, google-workspace)
- Tool permissions auto-granted via `build_allowed_tools()` wildcards
- Each integration (Jira, Google, GitHub, GitLab) has bespoke credential handlers across backend, runner, and frontend (~5 files per integration)

### Target State

- `.mcp.json` has 5 servers (+ context7 and deepwiki using HTTP transport)
- A generic MCP credential framework handles credential storage, delivery, and status for any server
- Adding a new authenticated MCP server requires only a `.mcp.json` edit

### Key Files — Existing (Read-Only Reference)

| File | Role |
|------|------|
| `components/backend/handlers/jira_auth.go` | Bespoke credential CRUD pattern to replicate generically |
| `components/backend/handlers/runtime_credentials.go` | Runtime fetch pattern with session ownership check |
| `components/backend/routes.go` | Route registration pattern |
| `components/runners/ambient-runner/ambient_runner/platform/auth.py` | Runner credential fetch/injection pattern |
| `components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py` | MCP auth check + tool authorization |
| `components/frontend/src/app/integrations/IntegrationsClient.tsx` | Integration cards pattern |

### Key Files — New or Modified

| File | Change |
|------|--------|
| `components/runners/ambient-runner/.mcp.json` | Add context7 + deepwiki entries |
| `components/backend/handlers/mcp_credentials.go` | **NEW** — Generic credential CRUD + runtime fetch |
| `components/backend/routes.go` | Register generic MCP credential routes |
| `components/runners/ambient-runner/ambient_runner/platform/auth.py` | Add generic MCP credential fetching |
| `components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py` | Add generic auth check fallback |
| `components/frontend/src/services/api/mcp-credentials.ts` | **NEW** — API client for generic MCP credentials |
| `components/frontend/src/services/queries/use-mcp-credentials.ts` | **NEW** — React Query hooks |
| `components/frontend/src/components/mcp-credential-card.tsx` | **NEW** — Generic credential card component |
| `components/frontend/src/app/integrations/IntegrationsClient.tsx` | Add MCP Servers section |

---

## Implementation Steps

### Step 1: Add Context7 and DeepWiki to `.mcp.json`

**File**: `components/runners/ambient-runner/.mcp.json`

Add `context7` and `deepwiki` entries. Existing entries preserved as-is.

```json
{
  "mcpServers": {
    "context7": {
      "type": "http",
      "url": "https://mcp.context7.com/mcp"
    },
    "deepwiki": {
      "type": "http",
      "url": "https://mcp.deepwiki.com/mcp"
    },
    "webfetch": { ... },
    "mcp-atlassian": { ... },
    "google-workspace": { ... }
  }
}
```

This alone satisfies FR-1 through FR-5. Tool permissions auto-granted via existing `build_allowed_tools()`.

---

### Step 2: Backend — Generic MCP Credential Handler

**New file**: `components/backend/handlers/mcp_credentials.go`

#### Data Model

```go
type MCPServerCredentials struct {
    UserID     string            `json:"userId"`
    ServerName string            `json:"serverName"`
    Fields     map[string]string `json:"fields"`     // arbitrary key-value credential fields
    UpdatedAt  time.Time         `json:"updatedAt"`
}
```

- **Secret name**: `mcp-server-credentials`
- **Secret key**: `{serverName}:{sanitizedUserID}`
- **Labels**: `app: ambient-code`, `ambient-code.io/provider: mcp`, `ambient-code.io/mcp-server: {serverName}`

#### CRUD Functions

Follow the same patterns as `jira_auth.go`:

- `storeMCPCredentials(ctx, creds)` — 3-retry optimistic locking upsert
- `getMCPCredentials(ctx, serverName, userID)` — nil-on-not-found read
- `deleteMCPCredentials(ctx, serverName, userID)` — idempotent retry delete

#### HTTP Handlers

| Handler | Method | Path | Auth |
|---------|--------|------|------|
| `ConnectMCPServer` | POST | `/auth/mcp/:serverName/connect` | User token via `GetK8sClientsForRequest` |
| `GetMCPServerStatus` | GET | `/auth/mcp/:serverName/status` | User token |
| `DisconnectMCPServer` | DELETE | `/auth/mcp/:serverName/disconnect` | User token |
| `GetMCPCredentialsForSession` | GET | `/projects/:project/agentic-sessions/:session/credentials/mcp/:serverName` | Session ownership check |

**Connect request body** (flexible schema):
```json
{
  "fields": {
    "apiKey": "sk-...",
    "baseUrl": "https://..."
  }
}
```

**Status response**:
```json
{
  "connected": true,
  "serverName": "some-server",
  "fieldNames": ["apiKey", "baseUrl"],
  "updatedAt": "2026-03-06T12:00:00Z"
}
```

**Runtime fetch response**:
```json
{
  "serverName": "some-server",
  "fields": {
    "apiKey": "sk-...",
    "baseUrl": "https://..."
  }
}
```

#### Server Name Validation

Validate `:serverName` against a safe pattern (lowercase alphanumeric + hyphens, max 63 chars) to prevent injection via path parameter.

---

### Step 3: Backend — Route Registration

**File**: `components/backend/routes.go`

Add routes in the cluster-level auth group:
```go
// Generic MCP server credentials
api.POST("/auth/mcp/:serverName/connect", handlers.ConnectMCPServer)
api.GET("/auth/mcp/:serverName/status", handlers.GetMCPServerStatus)
api.DELETE("/auth/mcp/:serverName/disconnect", handlers.DisconnectMCPServer)

// Runtime fetch (session-scoped)
projectGroup.GET("/agentic-sessions/:sessionName/credentials/mcp/:serverName", handlers.GetMCPCredentialsForSession)
```

Also add to integrations status aggregator:
```go
// In GetIntegrationsStatus handler, add MCP server credentials to response
```

---

### Step 4: Runner — Generic Credential Fetching

**File**: `components/runners/ambient-runner/ambient_runner/platform/auth.py`

Add to `populate_runtime_credentials()`:

```python
async def _fetch_mcp_credentials(context: RunnerContext, server_name: str) -> dict:
    """Fetch credentials for an MCP server from the generic backend endpoint."""
    return await _fetch_credential(context, f"mcp/{server_name}")

async def populate_mcp_server_credentials(context: RunnerContext, mcp_servers: dict):
    """For each MCP server with env vars referencing MCP_* pattern, fetch and inject credentials."""
    for server_name, config in mcp_servers.items():
        env_block = config.get("env", {})
        # Find env vars that use ${MCP_*} pattern
        mcp_vars = [v for v in env_block.values() if "${MCP_" in str(v)]
        if not mcp_vars:
            continue

        try:
            creds = await _fetch_mcp_credentials(context, server_name)
            if creds and creds.get("fields"):
                for field_name, field_value in creds["fields"].items():
                    env_var = f"MCP_{server_name.upper().replace('-', '_')}_{field_name.upper()}"
                    os.environ[env_var] = field_value
                    logger.info(f"Injected MCP credential: {env_var} for server {server_name}")
        except Exception as e:
            logger.warning(f"No credentials available for MCP server {server_name}: {e}")
```

**Integration point**: Call `populate_mcp_server_credentials()` in the bridge's `_setup_platform()` after loading MCP config but before `expand_env_vars()` runs.

---

### Step 5: Runner — Generic Auth Check

**File**: `components/runners/ambient-runner/ambient_runner/bridges/claude/mcp.py`

Extend `check_mcp_authentication()` with a generic fallback:

```python
def check_mcp_authentication(server_name: str) -> tuple[bool | None, str | None]:
    # ... existing bespoke checks for google-workspace and mcp-atlassian ...

    # Generic fallback: check if MCP_* env vars are populated for this server
    prefix = f"MCP_{server_name.upper().replace('-', '_')}_"
    mcp_env_vars = {k: v for k, v in os.environ.items() if k.startswith(prefix)}
    if mcp_env_vars:
        return True, f"MCP credentials configured ({len(mcp_env_vars)} fields)"

    # No bespoke or generic credentials found — may not need any
    return None, None
```

---

### Step 6: Frontend — API Client

**New file**: `components/frontend/src/services/api/mcp-credentials.ts`

```typescript
export interface MCPCredentialFields {
  [key: string]: string;
}

export async function connectMCPServer(serverName: string, fields: MCPCredentialFields): Promise<void>
export async function getMCPServerStatus(serverName: string): Promise<MCPServerStatus>
export async function disconnectMCPServer(serverName: string): Promise<void>
```

---

### Step 7: Frontend — React Query Hooks

**New file**: `components/frontend/src/services/queries/use-mcp-credentials.ts`

```typescript
export function useMCPServerStatus(serverName: string)
export function useConnectMCPServer()
export function useDisconnectMCPServer()
```

---

### Step 8: Frontend — Generic Credential Card

**New file**: `components/frontend/src/components/mcp-credential-card.tsx`

A reusable card component that:
- Takes `serverName` and `fieldDefinitions` (field name + label + type) as props
- Shows connection status
- Renders a dynamic form for credential input
- Handles connect/disconnect actions

---

### Step 9: Frontend — Integrations Page Update

**File**: `components/frontend/src/app/integrations/IntegrationsClient.tsx`

Add a new "MCP Servers" section below existing integrations. Renders `MCPCredentialCard` for each server that accepts credentials. Initially empty (Context7/DeepWiki don't need credentials), but the infrastructure is ready.

---

## Testing

| Test | Method | Expected Result |
|------|--------|-----------------|
| Context7/DeepWiki in `.mcp.json` | Unit: `load_mcp_config()` returns 5 servers | Dict includes all 5 servers |
| Tool permissions | Unit: `build_allowed_tools()` output | Includes `mcp__context7__*` and `mcp__deepwiki__*` |
| Generic credential CRUD | Integration: POST/GET/DELETE on `/auth/mcp/test-server/connect` | Credentials stored/read/deleted in K8s Secret |
| Runtime credential fetch | Integration: GET `/credentials/mcp/test-server` with session ownership | Returns stored credentials |
| Runner credential injection | Unit: `populate_mcp_server_credentials()` sets env vars | `MCP_TESTSERVER_APIKEY` populated |
| Generic auth check | Unit: `check_mcp_authentication("test-server")` with env vars set | Returns `(True, ...)` |
| Context7/DeepWiki end-to-end | Manual: Agent uses documentation tools in session | Tools work without approval prompts |
| Existing tools regression | E2E: `make test-e2e-local` | No regressions |

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| SDK doesn't support HTTP transport | Verify before merging; fallback to SSE |
| External services unreachable | Test in staging; document network requirements |
| K8s Secret size limit | Small payloads (~100 bytes/credential); supports thousands of entries |
| Generic framework misses edge cases | Keep bespoke path for existing integrations; expand framework iteratively |

## Rollback

- **Context7/DeepWiki**: Revert `.mcp.json` (remove 2 entries)
- **Credential framework**: Revert new files + route changes. No data migration — K8s Secret can be deleted cleanly
