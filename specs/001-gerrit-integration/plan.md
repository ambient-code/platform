# Implementation Plan: Gerrit Integration Connector

**Branch**: `001-gerrit-integration` | **Date**: 2026-03-24 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-gerrit-integration/spec.md`

## Summary

Add Gerrit as a first-class integration connector in Ambient Code Platform, enabling users to connect their Gerrit code review instances and use AI-assisted code review workflows through the open-source Gerrit MCP server. The implementation follows existing integration patterns (Jira, GitLab) with a dedicated backend handler, frontend connection card, and runner-side credential injection that generates the Gerrit MCP server's native `gerrit_config.json` at runtime.

## Technical Context

**Language/Version**: Go 1.22+ (backend/operator), TypeScript/React (frontend), Python 3.12+ (runner, Gerrit MCP server)
**Primary Dependencies**: Gin (Go HTTP), React/Shadcn (frontend), gerrit-mcp-server (MCP tools), mcp SDK (Python)
**Storage**: Kubernetes Secrets (credential storage, same pattern as existing integrations)
**Testing**: `go test` (backend), Vitest (frontend), pytest (runner)
**Target Platform**: Kubernetes (Linux containers)
**Project Type**: Multi-component platform (backend + frontend + runner + operator)
**Performance Goals**: Credential validation < 30 seconds, MCP tool invocations complete within Gerrit API response times
**Constraints**: No new CRDs, no new K8s operators; fits within existing integration framework
**Scale/Scope**: Support multiple Gerrit instances per user, standard platform user base

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

No project-specific constitution defined. Default engineering principles apply:
- No `panic()` in production Go code
- No `any` types in frontend TypeScript
- User token auth for all user-facing API operations
- OwnerReferences on all child K8s resources
- Conventional commits

**Pre-design gate**: PASS (no violations)
**Post-design gate**: PASS (design follows all existing patterns, no new abstractions introduced)

## Project Structure

### Documentation (this feature)

```text
specs/001-gerrit-integration/
├── plan.md              # This file
├── research.md          # Phase 0: Technology research and decisions
├── data-model.md        # Phase 1: Entity definitions and storage design
├── quickstart.md        # Phase 1: Development setup guide
├── contracts/
│   ├── gerrit-api.yaml  # Phase 1: OpenAPI contract
│   └── frontend-types.ts # Phase 1: TypeScript type definitions
└── tasks.md             # Phase 2: Task breakdown (created by /speckit.tasks)
```

### Source Code (repository root)

```text
components/backend/
├── handlers/
│   ├── gerrit_auth.go              # NEW: Connect/disconnect/status/test handlers
│   ├── integrations_status.go      # MODIFY: Add Gerrit to unified status
│   ├── integration_validation.go   # MODIFY: Add ValidateGerritToken()
│   └── runtime_credentials.go      # MODIFY: Add session-scoped Gerrit fetch
└── routes.go                       # MODIFY: Register Gerrit routes

components/frontend/src/
├── components/
│   └── gerrit-connection-card.tsx   # NEW: Connection card with auth method toggle
├── services/
│   ├── api/
│   │   ├── gerrit-auth.ts          # NEW: API client functions
│   │   └── integrations.ts         # MODIFY: Add Gerrit to status type
│   └── queries/
│       └── use-gerrit.ts           # NEW: React Query hooks
└── app/integrations/
    └── IntegrationsClient.tsx       # MODIFY: Add GerritConnectionCard to grid

components/runners/ambient-runner/
├── .mcp.json                        # MODIFY: Add Gerrit MCP server entry
├── Dockerfile                       # MODIFY: Bundle Gerrit MCP server
└── ambient_runner/
    ├── platform/
    │   └── auth.py                  # MODIFY: Add Gerrit credential fetching
    └── bridges/claude/
        └── mcp.py                   # MODIFY: Add Gerrit config generation + auth check
```

**Structure Decision**: This feature adds files across three existing components (backend, frontend, runner) following the established integration pattern. No new components or services are introduced. The Gerrit MCP server is bundled into the runner image at build time.

## Design Decisions

### D1: Dedicated Handler vs Generic MCP Credentials

**Chosen**: Dedicated handler (`gerrit_auth.go`) following Jira/GitLab pattern.

**Why**: Gerrit requires auth method selection (HTTP vs gitcookies), dedicated validation against the Gerrit REST API, and a custom connection card UI. The generic MCP pattern (`map[string]string` fields) lacks typed validation, auth method switching, and multi-instance listing.

### D2: Runtime Config File Generation

**Chosen**: Runner generates `gerrit_config.json` at session startup from fetched credentials.

**Why**: The Gerrit MCP server reads configuration from a JSON file (configurable via `GERRIT_CONFIG_PATH`), not from environment variables. This is different from Jira/GitLab which use env vars directly. The runner must:
1. Fetch all Gerrit credentials for the user from the backend
2. Generate a `gerrit_config.json` with one `gerrit_hosts` entry per connected instance
3. For gitcookies auth, write cookie content to a temp file and reference its path
4. Set `GERRIT_CONFIG_PATH` before launching the MCP server

### D3: Multi-Instance Secret Key Format

**Chosen**: `{instanceName}:{userID}` keys within a single `gerrit-credentials` secret.

**Why**: Consistent with the MCP credentials pattern (`serverName:userID`). Allows listing all instances for a user by scanning keys ending with `:userID`. Single secret simplifies cleanup and backup.

### D4: Credential Validation Endpoint

**Chosen**: `GET /a/accounts/self` on the Gerrit REST API.

**Why**: Lightest authenticated endpoint. Returns the authenticated user's account info. Works with both HTTP basic and gitcookies auth. Returns 401 for invalid credentials. Must handle Gerrit's XSSI protection prefix (`)]}'`).

### D5: Gerrit MCP Server Bundling

**Chosen**: Clone at pinned commit and build in Dockerfile.

**Why**: Not published on PyPI, cannot use `uvx`. Build script creates a venv with hash-verified dependencies. Pinning to a specific commit ensures reproducibility.

## Component Interaction Flow

```
[User] → [Frontend: GerritConnectionCard]
         → POST /api/auth/gerrit/connect
         → [Backend: ConnectGerrit handler]
            → ValidateGerritToken (GET /a/accounts/self on Gerrit instance)
            → Store in K8s Secret (gerrit-credentials)
            → Return success

[User creates session] → [Operator creates Job]
         → [Runner startup]
            → fetch_gerrit_credentials() from backend API
            → Generate gerrit_config.json + gitcookies temp file
            → Set GERRIT_CONFIG_PATH env var
            → Launch Gerrit MCP server in STDIO mode
            → Agent has access to 21 Gerrit tools
```
