# Quickstart: Gerrit Integration Connector

**Branch**: `001-gerrit-integration` | **Date**: 2026-03-24

## Prerequisites

- Local Kind cluster running (`make kind-up`)
- Platform deployed (`make deploy`)
- Access to a Gerrit instance for testing (e.g., `https://review.opendev.org`)
- Gerrit HTTP credentials or gitcookies for your account

## Development Setup

### 1. Backend (Go)

```bash
cd components/backend
# After adding new handler files:
go vet ./...
gofmt -l .
go build ./...
```

### 2. Frontend (TypeScript)

```bash
cd components/frontend
npm install
npm run dev  # http://localhost:3000
```

### 3. Runner (Python)

```bash
cd components/runners/ambient-runner
uv venv && uv pip install -e .

# Clone and build Gerrit MCP server locally for testing:
git clone https://gerrit.googlesource.com/gerrit-mcp-server /tmp/gerrit-mcp-server
cd /tmp/gerrit-mcp-server && ./build-gerrit.sh
```

## Key Files to Modify

### Backend
| File | Action |
|------|--------|
| `handlers/gerrit_auth.go` | **Create** - Connect/disconnect/status/test handlers |
| `handlers/integrations_status.go` | **Modify** - Add Gerrit to unified status |
| `handlers/integration_validation.go` | **Modify** - Add `ValidateGerritToken()` |
| `handlers/runtime_credentials.go` | **Modify** - Add session-scoped Gerrit credential fetch |
| `routes.go` | **Modify** - Register Gerrit routes |

### Frontend
| File | Action |
|------|--------|
| `components/gerrit-connection-card.tsx` | **Create** - Connection UI with auth method toggle |
| `services/api/gerrit-auth.ts` | **Create** - API client functions |
| `services/queries/use-gerrit.ts` | **Create** - React Query hooks |
| `services/api/integrations.ts` | **Modify** - Add Gerrit to IntegrationsStatus type |
| `app/integrations/IntegrationsClient.tsx` | **Modify** - Add GerritConnectionCard to grid |

### Runner
| File | Action |
|------|--------|
| `.mcp.json` | **Modify** - Add Gerrit MCP server entry |
| `ambient_runner/platform/auth.py` | **Modify** - Add Gerrit credential fetching |
| `ambient_runner/bridges/claude/mcp.py` | **Modify** - Add Gerrit config generation and auth check |
| `Dockerfile` | **Modify** - Bundle Gerrit MCP server |

## Testing the Integration

### Manual Test Flow

1. Start the platform locally
2. Navigate to Integrations page
3. Click "Connect Gerrit"
4. Enter instance name, URL, and credentials
5. Verify "Connected" status appears
6. Create a new session
7. Ask the agent: "List my open changes on Gerrit"
8. Verify the agent returns results from the Gerrit instance

### Credential Validation Test

```bash
# Test HTTP basic auth against Gerrit REST API:
curl -u "username:http_password" "https://review.opendev.org/a/accounts/self" | tail -c +5 | jq .

# Test with gitcookies:
curl -b ~/.gitcookies "https://review.opendev.org/a/accounts/self" | tail -c +5 | jq .
```

Note: Gerrit REST API responses are prefixed with `)]}'` (XSSI protection). The `tail -c +5` strips this prefix.
