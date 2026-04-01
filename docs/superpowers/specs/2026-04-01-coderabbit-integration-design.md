# CodeRabbit Integration for ACP

## Overview

Add CodeRabbit as a native integration in the Ambient Code Platform, enabling AI-powered code review inside agentic sessions. Users store a CodeRabbit API key via the integrations page; the runner injects it as `CODERABBIT_API_KEY` so the `coderabbit` CLI can run local reviews. The repo's `.pre-commit-config.yaml` enforces reviews at commit time.

## Goals

- Store and manage CodeRabbit API keys per user (cluster-scoped K8s Secrets)
- Inject credentials into sessions so the CLI authenticates automatically
- Provide a frontend integration card consistent with existing integrations (Jira, GitLab)
- Enable a local review-resolve loop: agent makes changes, runs `coderabbit review --type uncommitted --prompt-only`, fixes findings, repeats until clean — all before any push

## Non-Goals

- GitHub App detection for CodeRabbit (API key only for now)
- In-session `pre-commit install` enforcement (separate PR)
- Adding the `coderabbit` binary to the runner Dockerfile (separate PR)
- MCP server wrapping around the CLI

## Architecture

### Session Flow

```
User configures API key → Backend stores in K8s Secret →
Session starts → Runner fetches /credentials/coderabbit →
Sets CODERABBIT_API_KEY env var → Agent uses coderabbit CLI →
Pre-commit hook enforces review before commits
```

### Backend

**New file:** `components/backend/handlers/coderabbit_auth.go`

Credential struct:

```go
type CodeRabbitCredentials struct {
    UserID    string    `json:"userId"`
    APIKey    string    `json:"apiKey"`
    UpdatedAt time.Time `json:"updatedAt"`
}
```

K8s Secret: `coderabbit-credentials` (cluster-scoped, keyed by `userID`).

Endpoints:

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| POST | `/api/auth/coderabbit/connect` | ConnectCodeRabbit | Validate + store API key |
| GET | `/api/auth/coderabbit/status` | GetCodeRabbitStatus | Connection status |
| DELETE | `/api/auth/coderabbit/disconnect` | DisconnectCodeRabbit | Remove credentials |
| POST | `/api/auth/coderabbit/test` | TestCodeRabbitConnection | Validate without storing |
| GET | `/api/projects/:project/agentic-sessions/:session/credentials/coderabbit` | GetCodeRabbitCredentialsForSession | Runtime credential fetch |

All handlers follow existing patterns: `GetK8sClientsForRequest(c)` for user auth, per-user secret isolation, conflict-retry on Secret updates, RBAC validation on runtime credential fetch.

API key validation: `GET https://api.coderabbit.ai/api/v1/health` with `Authorization: Bearer <key>`. A 200 response confirms the key is valid.

`GetIntegrationsStatus` updated to include `coderabbit` in its response.

### Runner

At session startup, the runner fetches `GET /credentials/coderabbit`. If credentials exist, it sets `CODERABBIT_API_KEY` in the session process environment. No config file needed — the CLI reads the env var directly.

Graceful degradation: if no credentials are configured, the runner skips injection. The pre-commit hook and CLI skip gracefully when no auth is available.

### Frontend

**New files:**

| File | Purpose |
|------|---------|
| `components/coderabbit-connection-card.tsx` | Integration card UI |
| `services/api/coderabbit-auth.ts` | API client functions |
| `services/queries/use-coderabbit.ts` | React Query hooks |
| `app/api/auth/coderabbit/connect/route.ts` | Next.js proxy route |
| `app/api/auth/coderabbit/disconnect/route.ts` | Next.js proxy route |
| `app/api/auth/coderabbit/status/route.ts` | Next.js proxy route |
| `app/api/auth/coderabbit/test/route.ts` | Next.js proxy route |

**Modified files:**

| File | Change |
|------|--------|
| `IntegrationsClient.tsx` | Add `<CodeRabbitConnectionCard>` to grid |
| `services/api/integrations.ts` | Add `coderabbit` to `IntegrationsStatus` type |
| `integrations-panel.tsx` | Add CodeRabbit to session settings |

Card UI: single-field connect form (API key with show/hide toggle), status indicator, edit/disconnect buttons. Follows Jira card pattern exactly.

### Pre-commit Hook (already committed)

- `scripts/pre-commit/coderabbit-review.sh` — resolves CLI binary, checks auth, runs `coderabbit review --type uncommitted --prompt-only` with 5-minute timeout
- `.coderabbit.yaml` — project-specific review config with path instructions, custom pre-merge checks (performance, security, K8s safety), and tool configuration
- Registered in `.pre-commit-config.yaml`

### Tests

**Backend:** `components/backend/handlers/coderabbit_auth_test.go` (Ginkgo)

- Connect with valid/invalid API key
- Status when connected/disconnected
- Disconnect
- Per-user secret isolation
- Runtime credential fetch with RBAC validation

**Frontend:** Update `integrations-panel.test.tsx` to include CodeRabbit card.

## Future Work

- Default `pre-commit install` to enabled when CodeRabbit integration is configured (separate PR)
- Add `coderabbit` binary to runner Dockerfile (separate PR)
