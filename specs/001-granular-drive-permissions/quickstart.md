# Quickstart: Granular Permissions for Google Drive Integration

**Branch**: `001-granular-drive-permissions` | **Date**: 2026-03-14

## Overview

This feature changes the Google Drive integration to default to file-level permissions using the Google Picker API, instead of requesting access to the user's entire Drive.

## Architecture Summary

```
┌─────────────────────┐     ┌──────────────────────┐     ┌────────────────┐
│   Frontend          │     │   Backend (Go/Gin)   │     │  Google APIs   │
│   (Next.js/React)   │     │                      │     │                │
│                     │     │                      │     │                │
│  Integration Page   │────▶│  POST /setup         │────▶│  OAuth2 Auth   │
│  (setup flow)       │     │  (init OAuth flow)   │     │  (drive.file)  │
│                     │     │                      │     │                │
│  Google Picker      │◀───│  GET /callback        │◀───│  Auth callback │
│  (file selection)   │     │  (exchange tokens)   │     │                │
│                     │     │                      │     │                │
│  File List View     │────▶│  PUT /files          │     │                │
│  (manage grants)    │     │  (store file grants) │     │                │
│                     │     │                      │     │                │
│  Settings Page      │────▶│  GET /files          │────▶│  Drive API v3  │
│  (view/modify)      │     │  (list grants)       │     │  (file access) │
└─────────────────────┘     └──────────────────────┘     └────────────────┘
```

## Key Components

### Backend (Go)

1. **OAuth Handler Update** (`components/backend/handlers/oauth.go`)
   - Change default scope from `drive` to `drive.file`
   - Add `permissionScope` field to integration configuration
   - Ensure `drive.file` scope is used for new integrations

2. **Picker Token Endpoint** (new handler)
   - `GET /api/projects/:projectName/integrations/google-drive/picker-token`
   - Returns a fresh access token for the frontend Picker component
   - Refreshes expired tokens automatically

3. **File Grant Management** (new handler)
   - `GET /api/projects/:projectName/integrations/google-drive/files` - list granted files
   - `PUT /api/projects/:projectName/integrations/google-drive/files` - update file grants
   - Stores file grants in Kubernetes ConfigMap or CR

4. **Integration Status** (new/updated handler)
   - `GET /api/projects/:projectName/integrations/google-drive` - get integration status
   - `DELETE /api/projects/:projectName/integrations/google-drive` - disconnect

### Frontend (React/Next.js)

1. **Google Picker Component** (new)
   - Loads Google Picker API via `google.api.load('picker', ...)`
   - Configures multi-select, search, and folder support
   - Handles `picked` and `cancel` callbacks
   - Returns selected file metadata to parent component

2. **Integration Setup Page** (updated)
   - Shows "only the specific" permission text
   - Displays "Choose Files" button that launches the Picker
   - Shows file selection summary after Picker closes

3. **Integration Settings Page** (new/updated)
   - Lists currently granted files with status indicators
   - "Modify Files" button to re-open Picker with pre-selected files
   - Handles unavailable/deleted file notifications

### Storage (Kubernetes)

- **OAuth Tokens**: Kubernetes Secrets (existing pattern)
- **File Grants**: Kubernetes ConfigMap or Custom Resource per project
- **Integration Config**: Part of project settings CR

## User Flow

1. User navigates to Integrations page
2. User clicks "Connect Google Drive"
3. Backend initiates OAuth flow with `drive.file` scope
4. Google consent screen shows "only the specific files" text
5. User authorizes → callback stores tokens
6. Frontend shows "Choose Files" button
7. User clicks → Google Picker opens
8. User selects files → Picker callback fires
9. Frontend sends file IDs to backend → stored as FileGrants
10. Integration is active with granular access

## Key Decisions

| Decision | Choice | Reference |
| -------- | ------ | --------- |
| OAuth scope | `drive.file` (not `drive`) | research.md R-001 |
| File picker | Google Picker API | research.md R-002 |
| Auth library | Google Identity Services (GIS) | research.md R-002 |
| Token storage | Kubernetes Secrets (existing) | research.md R-003 |
| File grant storage | Kubernetes ConfigMap/CR | research.md R-004 |
| Migration | Opt-in for existing users | research.md R-005 |

## Prerequisites

- Google Cloud project with Picker API enabled
- Google API Key configured (for Picker)
- Google OAuth2 Client ID and Secret (existing)
- Unleash feature flag for rollout control (existing infrastructure)
