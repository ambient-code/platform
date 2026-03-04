# Proposal: ACP Public REST API

**Issue:** [#557 - Feature: ACP Rest API](https://github.com/ambient-code/platform/issues/557)
**Date:** 2026-01-29
**Status:** Approved

---

## Executive Summary

Create a **Public API** service that acts as the single entry point for all clients (Browser, SDK, MCP). This thin shim layer proxies requests to the existing Go backend while providing simplified DTOs and a versioned API.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│ Clients                                                         │
│  - Browser (via OAuth proxy)                                    │
│  - SDK (direct with access key)                                 │
│  - MCP (direct with access key)                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Public API (NEW)                                                │
│  - Versioned endpoints (/v1/sessions, etc.)                     │
│  - Simplified JSON responses                                    │
│  - Token validation & project extraction                        │
│  - Proxies to Go Backend                                        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ Go Backend (internal only)                                      │
│  - Full K8s/CRD operations                                      │
│  - ClusterIP service (no external route)                        │
│  - Existing /api/projects/... endpoints                         │
└─────────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Single entry point** - All clients use the Public API
2. **Frontend simplified** - Remove Next.js API routes, just static files + token forwarding middleware
3. **Backend internalized** - Go backend becomes ClusterIP only, not exposed externally
4. **Thin shim** - Public API is a lightweight proxy (~500 lines), not a full rewrite

---

## Authentication

| Client | Auth Method | Flow |
|--------|-------------|------|
| Browser | OAuth Proxy | OAuth Proxy → Public API (token in header) |
| SDK | Access Key | Direct to Public API with Bearer token |
| MCP | Access Key | Direct to Public API with Bearer token |

### Token Handling

- **Browser:** OAuth proxy sets `X-Forwarded-Access-Token`, forwarded to Public API
- **SDK/MCP:** Pass OpenShift token or access key as `Authorization: Bearer <token>`
- **Project extraction:** For SDK/MCP, extract project from ServiceAccount namespace in JWT

---

## API Endpoints (v1)

### Sessions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/sessions` | List sessions (requires `X-Ambient-Project` header or token-based project) |
| POST | `/v1/sessions` | Create session |
| GET | `/v1/sessions/{id}` | Get session details |
| DELETE | `/v1/sessions/{id}` | Delete session |

### Simplified Response DTOs

```go
// GET /v1/sessions/{id}
type SessionResponse struct {
    ID          string `json:"id"`
    Status      string `json:"status"`      // "pending", "running", "completed", "failed"
    Task        string `json:"task"`
    Model       string `json:"model,omitempty"`
    CreatedAt   string `json:"createdAt"`
    CompletedAt string `json:"completedAt,omitempty"`
    Result      string `json:"result,omitempty"`
    Error       string `json:"error,omitempty"`
}

// GET /v1/sessions
type SessionListResponse struct {
    Items []SessionResponse `json:"items"`
    Total int               `json:"total"`
}

// POST /v1/sessions
type CreateSessionRequest struct {
    Task    string `json:"task" binding:"required"`
    Model   string `json:"model,omitempty"`
    Repos   []Repo `json:"repos,omitempty"`
}

type Repo struct {
    URL    string `json:"url" binding:"required"`
    Branch string `json:"branch,omitempty"`
}
```

---

## Component Changes

### New: `components/public-api/`

```
components/public-api/
├── main.go              # Entry point, server setup
├── handlers/
│   ├── sessions.go      # /v1/sessions handlers
│   ├── middleware.go    # Auth, project extraction
│   └── proxy.go         # Backend proxy utilities
├── types/
│   └── dto.go           # Simplified DTOs
├── Dockerfile
└── README.md
```

### Modified: `components/frontend/`

- Remove `src/app/api/**` routes (proxy handlers)
- Keep simple middleware for OAuth token forwarding
- Update API calls to use Public API URL

### Modified: `components/manifests/`

- Add `public-api-deployment.yaml`
- Add `public-api-service.yaml`
- Add `public-api-route.yaml` (external)
- Modify `backend-route.yaml` → remove from production (backend becomes internal)
- Update frontend to point to Public API

---

## Implementation Plan

### Phase 1: Public API Service

1. Create `components/public-api/` structure
2. Implement `/v1/sessions` endpoints with backend proxy
3. Add auth middleware (token validation, project extraction)
4. Add Dockerfile and deployment manifests

### Phase 2: Frontend Migration

1. Remove Next.js API route handlers
2. Update frontend to call Public API directly
3. Keep OAuth token forwarding middleware

### Phase 3: Backend Internalization

1. Remove `backend-route.yaml` from production
2. Verify backend is only accessible via ClusterIP
3. Update documentation

---

## Success Criteria

- [x] Public API service is deployable (Dockerfile, manifests, CI/CD in place)
- [x] Can list/create/get/delete sessions via `curl` to Public API (verified via e2e tests)
- [ ] Browser continues to work through Public API (Phase 2: Frontend migration)
- [x] SDK/MCP can authenticate with access keys (Bearer token and X-Forwarded-Access-Token supported)
- [ ] Latency < 200ms for API requests (requires Prometheus metrics - Phase 3)

---

## Example Usage

### SDK/CLI

```bash
# Create access key (one-time, via UI or existing API)
# Then use it for all API calls:

export AMBIENT_API="https://api.ambient-code.example.com"
export AMBIENT_TOKEN="<access-key>"
export AMBIENT_PROJECT="my-project"

# List sessions
curl -H "Authorization: Bearer $AMBIENT_TOKEN" \
     -H "X-Ambient-Project: $AMBIENT_PROJECT" \
     "$AMBIENT_API/v1/sessions"

# Create session
curl -X POST \
     -H "Authorization: Bearer $AMBIENT_TOKEN" \
     -H "X-Ambient-Project: $AMBIENT_PROJECT" \
     -H "Content-Type: application/json" \
     -d '{"task": "Refactor login.py", "model": "claude-sonnet-4"}' \
     "$AMBIENT_API/v1/sessions"
```

---

## References

- [Issue #557 - Feature: ACP Rest API](https://github.com/ambient-code/platform/issues/557)
- [MCP Server Issue #4](https://github.com/ambient-code/mcp/issues/4)
- [Backend Routes](../../components/backend/routes.go)
- [Access Keys Handler](../../components/backend/handlers/permissions.go)
