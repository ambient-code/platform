# Research: Granular Permissions for Google Drive Integration

**Branch**: `001-granular-drive-permissions` | **Date**: 2026-03-14

## R-001: OAuth Scope Strategy

**Decision**: Use `drive.file` scope instead of `drive` (full access) scope as the default for new integrations.

**Rationale**:
- `drive.file` grants access only to files the user explicitly selects via the Picker or that the app creates
- `drive` is a restricted scope requiring Google's security assessment; `drive.file` is a recommended scope with no additional verification burden
- Consent screen text changes from "See, edit, create, and delete all your Google Drive files" to "See, edit, create, and delete only the specific Google Drive files you use with this app"
- Aligns with principle of least privilege and GDPR/SOC2 compliance posture

**Alternatives Considered**:
- `drive` (full access): Overly broad, increasing verification burden and security exposure
- `drive.readonly`: Still grants access to ALL files (just read-only), doesn't solve the problem
- `drive.file` without Picker: Too restrictive, users can't grant access to existing files

## R-002: Google Picker API Integration Pattern

**Decision**: Use Google Picker API (client-side JavaScript dialog) with Google Identity Services (GIS) for token acquisition.

**Rationale**:
- Picker runs in a Google-hosted sandboxed iframe; the platform never sees the user's full Drive contents
- Returns file IDs, names, MIME types, sizes, and URLs for selected files
- Supports multi-select via `Feature.MULTISELECT_ENABLED`
- GIS (`google.accounts.oauth2.initTokenClient()`) is the current auth method; `gapi.auth2` is deprecated

**Alternatives Considered**:
- Custom file browser using Drive API: Requires `drive` scope to list files, defeating the purpose
- Google Drive "Open With" UI integration: Only works when user initiates from Drive, not from the platform
- Server-side file listing: No server-side equivalent of the Picker exists

## R-003: Existing Platform Integration Points

**Decision**: Extend the existing OAuth handler in `components/backend/handlers/oauth.go` which already supports Google OAuth with `drive`, `drive.readonly`, and `drive.file` scopes.

**Rationale**:
- The backend already has OAuth2 infrastructure with HMAC-signed state parameters, token storage in Kubernetes Secrets, and refresh token support
- The `drive.file` scope is already listed in the backend OAuth handler configuration
- Change is primarily about defaulting to `drive.file` instead of `drive` and adding the frontend Picker component
- Credential storage pattern via Kubernetes Secrets already handles Google tokens

**Alternatives Considered**:
- Building a new OAuth flow from scratch: Unnecessary, existing infrastructure is sufficient
- Using MCP credential handler pattern: The OAuth handler is more appropriate for Google's OAuth2 flow

## R-004: File Selection Persistence

**Decision**: Store selected file IDs server-side after Picker selection. The Picker is a one-time grant mechanism, not a repeated access pattern.

**Rationale**:
- Once a file ID is granted via Picker, the `drive.file` token retains access until user revokes it
- File IDs should be stored as persistent authorization records (not just temporary session data)
- Backend needs file IDs for server-side operations (read file content, check availability)
- File metadata (name, MIME type, size) should be cached for display in integration settings

**Alternatives Considered**:
- Re-showing Picker every time: Poor UX, unnecessary since access persists
- Storing only in browser localStorage: Insecure, not available to backend for server-side operations

## R-005: Migration Strategy for Existing Users

**Decision**: New integrations default to `drive.file` + Picker. Existing users with `drive` scope are not auto-migrated but can optionally switch.

**Rationale**:
- Auto-migration would break existing integrations since switching scopes invalidates refresh tokens
- Existing users would need to re-select files they were using, which is disruptive
- A prompt-based migration allows users to transition at their convenience
- Feature flag (via Unleash, already in the platform) can control the rollout

**Alternatives Considered**:
- Forced migration: Too disruptive, breaks existing workflows
- No migration path: Leaves existing users on overly broad permissions indefinitely
- Automatic re-authentication: Users would lose access to files they don't remember to re-select

## R-006: Picker API Limitations

**Decision**: Accept known limitations and design around them.

**Key Limitations**:
| Limitation | Mitigation |
| ---------- | ---------- |
| Client-side only (JavaScript iframe) | Use Drive API with stored file IDs for server-side operations after initial selection |
| No programmatic file selection | Store file IDs after first selection; only show Picker for new grants |
| Folder selection is shallow (no recursive access with `drive.file`) | Users must select individual files within folders, or select the folder and document this limitation |
| Mobile UX is responsive iframe, not native | Acceptable for initial release; consider deep-linking to Drive app later |
| Deprecated auth flow (gapi.auth2) | Build on GIS from the start |

## R-007: Platform Technical Context

**Decision**: Implement as changes to the existing monorepo components.

**Platform Stack**:
- **Backend**: Go 1.24+ with Gin framework (`components/backend/`)
- **Frontend**: Next.js 16.1.5 with React 19.1.0, Shadcn/ui, Tailwind CSS (`components/frontend/`)
- **Storage**: Kubernetes Custom Resources + Secrets (no traditional database)
- **Testing**: Vitest (frontend), Ginkgo/Gomega + Testify (backend), Cypress (E2E)
- **Feature Flags**: Unleash
- **Auth**: OpenShift OAuth + JWT, existing Google OAuth handler
- **Project Type**: Web application (monorepo with microservices)
