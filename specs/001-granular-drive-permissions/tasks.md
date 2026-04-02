# Tasks: Granular Permissions for Google Drive Integration

**Input**: Design documents from `/specs/001-granular-drive-permissions/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in the feature specification. Test tasks are excluded.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Web app monorepo**: `components/backend/` (Go/Gin), `components/frontend/` (Next.js/React)

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, dependencies, and feature flag configuration

- [x] T001 Add Google Picker API and Google Identity Services (GIS) script dependencies to `components/frontend/package.json` (or load via `<script>` in layout)
- [x] T002 [P] Create Unleash feature flag `granular-drive-permissions` for rollout control in `components/backend/handlers/feature_flags.go` (or existing flag config)
- [x] T003 [P] Register new Google Drive integration routes in `components/backend/handlers/routes.go` (or router configuration file) for all new endpoints under `/api/projects/:projectName/integrations/google-drive/`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Shared models and API client that ALL user stories depend on

**CRITICAL**: No user story work can begin until this phase is complete

- [x] T004 Create DriveIntegration and FileGrant Go structs with validation, status enums, and state transition methods in `components/backend/models/drive.go` (per data-model.md)
- [x] T005 [P] Create Kubernetes storage helpers for DriveIntegration (read/write to ConfigMap, following existing integration config patterns) and FileGrant (stored as a JSON list within the DriveIntegration ConfigMap) in `components/backend/handlers/drive_storage.go`
- [x] T006 [P] Create frontend API client with TanStack React Query hooks for all Drive integration endpoints (`setup`, `callback`, `picker-token`, `files`, `status`, `disconnect`) in `components/frontend/src/services/drive-api.ts`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - Set Up Google Drive with File-Level Permissions (Priority: P1) MVP

**Goal**: New Google Drive integrations default to `drive.file` scope instead of `drive`, and users see the granular consent screen

**Independent Test**: Initiate a new Google Drive integration and verify the OAuth consent screen shows "only the specific files" text instead of "all your Google Drive files"

### Implementation for User Story 1

- [x] T007 [US1] Update the Google OAuth scope configuration in `components/backend/handlers/oauth.go` to default to `drive.file` scope (instead of `drive`) when `permissionScope` is `granular` (the new default)
- [x] T008 [US1] Implement `POST /api/projects/:projectName/integrations/google-drive/setup` handler in `components/backend/handlers/drive_integration.go` that accepts `permissionScope` (default: `granular`) and `redirectUri`, generates OAuth URL with `drive.file` scope, and returns `authUrl` + HMAC-signed `state`
- [x] T009 [US1] Implement `GET /api/projects/:projectName/integrations/google-drive/callback` handler in `components/backend/handlers/drive_integration.go` that exchanges the authorization code for tokens, stores them in Kubernetes Secrets, creates a DriveIntegration record, and returns `integrationId`, `status`, and `pickerToken`
- [x] T010 [US1] Create the Google Drive integration setup page in `components/frontend/src/pages/integrations/google-drive/setup.tsx` (or equivalent app router path) that displays "only the specific files" permission text, a "Connect Google Drive" button that calls the setup endpoint, and handles the OAuth redirect flow
- [x] T011 [US1] Add the "Choose Files" button to the setup page in `components/frontend/src/pages/integrations/google-drive/setup.tsx` that appears after successful OAuth callback, indicating the user should now select files (Picker integration comes in US2)

**Checkpoint**: New integrations use `drive.file` scope. OAuth flow works end-to-end. Users see granular consent screen.

---

## Phase 4: User Story 2 - Select and Confirm Specific Files via Picker (Priority: P1)

**Goal**: Users can browse and select specific files from Google Drive using the Google Picker, and selected files are stored as grants

**Independent Test**: Click "Choose Files", verify Google Picker loads, select files, confirm selection, and verify the platform displays a summary of selected files and stores them server-side

### Implementation for User Story 2

- [x] T012 [US2] Implement `GET /api/projects/:projectName/integrations/google-drive/picker-token` handler in `components/backend/handlers/drive_integration.go` that returns a fresh (or refreshed) access token for the Google Picker with `expiresIn`
- [x] T013 [US2] Implement `PUT /api/projects/:projectName/integrations/google-drive/files` handler in `components/backend/handlers/drive_file_grants.go` that accepts a list of PickerFile objects, creates/updates FileGrant records, computes added/removed counts, and validates at least one file is provided
- [x] T014 [US2] Create the Google Picker React component in `components/frontend/src/components/google-picker/google-picker.tsx` that loads the Google Picker API via GIS, configures multi-select (`MULTISELECT_ENABLED`), search, and folder support, handles `picked` and `cancel` callbacks, and returns selected file metadata (id, name, mimeType, url, sizeBytes) to the parent
- [x] T015 [US2] Create a file selection summary component in `components/frontend/src/components/google-picker/file-selection-summary.tsx` (using Shadcn/ui) that displays selected files with name, type icon, and size after Picker closes
- [x] T016 [US2] Integrate the Google Picker component into the setup page in `components/frontend/src/pages/integrations/google-drive/setup.tsx`: wire "Choose Files" button to open the Picker (fetching picker-token first), handle Picker callback to display the file selection summary, and submit selected files to `PUT /files` endpoint on confirmation
- [x] T017 [US2] Add empty-selection validation: prevent confirmation when zero files are selected in the Picker callback handler in `components/frontend/src/components/google-picker/google-picker.tsx`, showing a prompt to select at least one file

**Checkpoint**: Full setup flow works end-to-end: OAuth → Picker → file selection → server-side storage. Users can select specific files.

---

## Phase 5: User Story 3 - Modify File Access After Initial Setup (Priority: P2)

**Goal**: Users can view, add, and remove files from their integration after the initial setup, and can disconnect the integration

**Independent Test**: Navigate to integration settings after setup, verify the list of granted files is displayed, open the Picker to add/remove files, confirm changes are persisted

### Implementation for User Story 3

- [x] T018 [US3] Implement `GET /api/projects/:projectName/integrations/google-drive/files` handler in `components/backend/handlers/drive_file_grants.go` that returns all FileGrant records for the integration with `totalCount`
- [x] T019 [P] [US3] Implement `GET /api/projects/:projectName/integrations/google-drive` handler in `components/backend/handlers/drive_integration.go` that returns integration status, permission scope, file count, and timestamps
- [x] T020 [P] [US3] Implement `DELETE /api/projects/:projectName/integrations/google-drive` handler in `components/backend/handlers/drive_integration.go` that disconnects the integration, revokes tokens (calling Google's revoke endpoint), and removes all FileGrant records
- [x] T021 [US3] Create the Google Drive integration settings page in `components/frontend/src/pages/integrations/google-drive/settings.tsx` that displays integration status, lists currently granted files using the file selection summary component, and shows a "Modify Files" button and a "Disconnect" button
- [x] T022 [US3] Wire the "Modify Files" button in `components/frontend/src/pages/integrations/google-drive/settings.tsx` to open the Google Picker pre-populated with currently selected file IDs (passed as `existingFileIds`), and on confirmation call `PUT /files` with the updated selection
- [x] T023 [US3] Wire the "Disconnect" button in `components/frontend/src/pages/integrations/google-drive/settings.tsx` to call `DELETE /integrations/google-drive` with a confirmation dialog, and redirect to the integrations list on success

**Checkpoint**: Users can view, modify, and disconnect their Google Drive integration. All user stories are independently functional.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Error handling, edge cases, and resilience improvements across all stories

- [x] T024 [P] Add Picker error handling in `components/frontend/src/components/google-picker/google-picker.tsx`: display an error message with a retry button when the Picker fails to load (network error, API outage), and handle token expiration during a Picker session by re-fetching the picker-token
- [x] T025 [P] Add unavailable file detection in `components/backend/handlers/drive_file_grants.go`: when listing files (`GET /files`), optionally verify file availability via Drive API and update FileGrant status to `unavailable` for deleted/inaccessible files, returning the updated status to the frontend
- [x] T026 [P] Add unavailable file notification in `components/frontend/src/pages/integrations/google-drive/settings.tsx`: display a warning badge/notification for files with `unavailable` status, prompting the user to remove or re-select them
- [x] T027 Add revoked access detection in `components/backend/handlers/drive_integration.go`: when any Drive API call returns a 401/403, update integration status to `disconnected` and return an error prompting the user to re-authenticate
- [x] T028 Wrap all new backend endpoints behind the `granular-drive-permissions` Unleash feature flag in `components/backend/handlers/routes.go` (route-group middleware)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational (Phase 2)
- **User Story 2 (Phase 4)**: Depends on Foundational (Phase 2) and User Story 1 (Phase 3) - needs OAuth flow working before Picker can be used
- **User Story 3 (Phase 5)**: Depends on Foundational (Phase 2) and User Story 2 (Phase 4) - needs file grants to exist before they can be managed
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2. Standalone - delivers OAuth with `drive.file` scope.
- **US2 (P1)**: Depends on US1 (needs OAuth tokens from the setup flow). Delivers file selection via Picker.
- **US3 (P2)**: Depends on US2 (needs file grants to exist). Delivers file management post-setup.

### Within Each User Story

- Backend handlers before frontend pages (frontend depends on API)
- Models → Storage → Handlers → Frontend pages → Integration

### Parallel Opportunities

- **Phase 1**: T002 and T003 can run in parallel
- **Phase 2**: T005 and T006 can run in parallel (after T004)
- **Phase 3**: T010 and T011 can be developed alongside T008/T009 (mock API during frontend dev)
- **Phase 5**: T019 and T020 can run in parallel
- **Phase 6**: T024, T025, and T026 can all run in parallel

---

## Parallel Example: Phase 2 (Foundational)

```bash
# First: Create shared models (blocks everything else)
Task T004: "Create DriveIntegration and FileGrant structs in components/backend/models/drive.go"

# Then launch storage and API client in parallel:
Task T005: "Create K8s storage helpers in components/backend/handlers/drive_storage.go"
Task T006: "Create frontend API client in components/frontend/src/services/drive-api.ts"
```

## Parallel Example: Phase 4 (User Story 2)

```bash
# Backend endpoints can be built in parallel:
Task T012: "Implement picker-token handler in components/backend/handlers/drive_integration.go"
Task T013: "Implement PUT /files handler in components/backend/handlers/drive_file_grants.go"

# Frontend Picker component can be built in parallel with backend (using mocks):
Task T014: "Create Google Picker component in components/frontend/src/components/google-picker/google-picker.tsx"
Task T015: "Create file selection summary component in components/frontend/src/components/google-picker/file-selection-summary.tsx"
```

---

## Implementation Strategy

### MVP First (User Stories 1 + 2)

1. Complete Phase 1: Setup (T001-T003)
2. Complete Phase 2: Foundational (T004-T006)
3. Complete Phase 3: User Story 1 (T007-T011)
4. Complete Phase 4: User Story 2 (T012-T017)
5. **STOP and VALIDATE**: Full setup flow works: OAuth → Picker → file selection → stored grants
6. Deploy/demo if ready - users can now set up granular Drive permissions

### Incremental Delivery

1. Setup + Foundational → Foundation ready
2. Add US1 → OAuth with `drive.file` scope works (partial MVP)
3. Add US2 → Full setup with Picker works (MVP!)
4. Add US3 → File management post-setup (full feature)
5. Add Polish → Error handling, resilience, feature flag gating

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- US1 and US2 are both P1 but have a sequential dependency (OAuth must work before Picker)
- US3 depends on US2 (can't manage file grants that don't exist yet)
- Backend handlers follow existing `oauth.go` and `mcp_credentials.go` patterns
- Frontend components use Shadcn/ui + TanStack React Query (existing patterns)
- All token storage uses Kubernetes Secrets (existing pattern)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
