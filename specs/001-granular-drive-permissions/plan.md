# Implementation Plan: Granular Permissions for Google Drive Integration

**Branch**: `001-granular-drive-permissions` | **Date**: 2026-03-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-granular-drive-permissions/spec.md`

## Summary

Change the Google Drive integration to default to file-level (picker-based) permissions using the `drive.file` OAuth scope and Google Picker API, instead of requesting access to the user's entire Drive via the `drive` scope. This involves updating the backend OAuth handler to default to `drive.file`, adding a Google Picker component to the frontend, and creating file grant management endpoints.

## Technical Context

**Language/Version**: Go 1.24+ (backend), TypeScript/React 19.1.0 with Next.js 16.1.5 (frontend), Python 3.11+ (runner)
**Primary Dependencies**: Gin (Go web framework), Shadcn/ui + Radix UI (frontend components), TanStack React Query (data fetching), Google Identity Services (auth), Google Picker API (file selection)
**Storage**: Kubernetes Custom Resources + Secrets (no traditional database)
**Testing**: Vitest + @testing-library/react (frontend), Ginkgo/Gomega + Testify (backend), Cypress (E2E)
**Target Platform**: Web application (browser-based, Kubernetes deployment)
**Project Type**: Web (monorepo with microservices)
**Performance Goals**: Picker loads within 2 seconds, file grant operations complete within 500ms
**Constraints**: Must work within existing OAuth infrastructure, Kubernetes-native storage only
**Scale/Scope**: Standard user-facing feature; no high-concurrency concerns beyond existing platform capacity

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The project constitution is not yet configured (template placeholders only). No gates to evaluate. Proceeding without constitution violations.

**Post-Phase 1 Re-check**: Design follows existing platform patterns (Go handlers, K8s storage, React components with Shadcn/ui). No new architectural patterns introduced. No complexity violations.

## Project Structure

### Documentation (this feature)

```text
specs/001-granular-drive-permissions/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: Research findings
├── data-model.md        # Phase 1: Entity definitions
├── quickstart.md        # Phase 1: Architecture overview
├── contracts/
│   └── drive-integration-api.yaml  # Phase 1: OpenAPI contract
├── checklists/
│   └── requirements.md  # Spec quality checklist
└── tasks.md             # Phase 2 output (created by /speckit.tasks)
```

### Source Code (repository root)

```text
components/
├── backend/
│   ├── handlers/
│   │   ├── oauth.go                    # UPDATE: default to drive.file scope
│   │   ├── drive_integration.go        # NEW: integration status endpoints
│   │   └── drive_file_grants.go        # NEW: file grant CRUD endpoints
│   ├── models/
│   │   └── drive.go                    # NEW: DriveIntegration, FileGrant structs
│   └── tests/
│       └── drive_integration_test.go   # NEW: handler tests
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   └── google-picker/          # NEW: Picker component
│   │   │       ├── google-picker.tsx
│   │   │       └── google-picker.test.tsx
│   │   ├── pages/ (or app/)
│   │   │   └── integrations/
│   │   │       └── google-drive/       # NEW/UPDATE: setup & settings pages
│   │   └── services/
│   │       └── drive-api.ts            # NEW: API client for drive endpoints
│   └── tests/
│       └── e2e/                        # Cypress tests
└── manifests/                          # K8s resource definitions if needed
```

**Structure Decision**: Web application structure following existing monorepo layout. Backend changes in `components/backend/handlers/`, frontend changes in `components/frontend/src/`. No new top-level directories needed.

## Complexity Tracking

No constitution violations to justify. Design follows existing patterns:
- Backend handlers follow the same pattern as `oauth.go` and `mcp_credentials.go`
- Frontend components use Shadcn/ui + TanStack Query (existing patterns)
- Storage uses Kubernetes Secrets + ConfigMaps (existing patterns)
