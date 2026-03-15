# Data Model: Granular Permissions for Google Drive Integration

**Branch**: `001-granular-drive-permissions` | **Date**: 2026-03-14

## Entities

### 1. DriveIntegration

Represents a user's Google Drive connection to the platform.

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | string (UUID) | Unique identifier for this integration |
| userId | string | The platform user who owns this integration |
| projectName | string | The project this integration belongs to |
| provider | string | Always "google" for this feature |
| permissionScope | enum | `granular` (default) or `full` (legacy) |
| status | enum | `active`, `disconnected`, `expired`, `error` |
| accessToken | string (encrypted) | OAuth2 access token (stored in K8s Secret) |
| refreshToken | string (encrypted) | OAuth2 refresh token (stored in K8s Secret) |
| tokenExpiresAt | timestamp | When the access token expires |
| createdAt | timestamp | When the integration was created |
| updatedAt | timestamp | Last modification time |

**State Transitions**:
```
[not connected] → active (on successful OAuth + file selection)
active → expired (on token expiration without refresh)
active → disconnected (on user disconnect or external revocation)
active → error (on persistent API failure)
expired → active (on successful token refresh)
disconnected → active (on re-authentication)
error → active (on successful retry)
```

### 2. FileGrant

Represents an individual file/folder that a user has granted the platform access to via the Picker.

| Field | Type | Description |
| ----- | ---- | ----------- |
| id | string (UUID) | Unique identifier for this grant |
| integrationId | string | Reference to the parent DriveIntegration |
| googleFileId | string | Google Drive file ID (from Picker response) |
| fileName | string | Display name of the file |
| mimeType | string | MIME type (e.g., `application/vnd.google-apps.document`) |
| fileUrl | string | Web URL to the file in Google Drive |
| sizeBytes | integer | File size in bytes (nullable for folders) |
| isFolder | boolean | Whether this grant is for a folder |
| status | enum | `active`, `unavailable`, `revoked` |
| grantedAt | timestamp | When the user selected this file via Picker |
| lastAccessedAt | timestamp | When the platform last accessed this file |
| lastVerifiedAt | timestamp | When file availability was last confirmed |

**Validation Rules**:
- `googleFileId` must be non-empty and unique within an integration
- `fileName` must be non-empty
- `mimeType` must be a valid MIME type string
- An integration must have at least one active FileGrant

**State Transitions**:
```
[selected in Picker] → active
active → unavailable (file deleted/moved on Google Drive)
active → revoked (user removes from selection)
unavailable → active (file restored on Google Drive)
revoked → active (user re-selects via Picker)
```

### 3. PickerSession (transient)

Represents an in-progress file selection via the Google Picker. Not persisted to storage.

| Field | Type | Description |
| ----- | ---- | ----------- |
| sessionId | string | Temporary session identifier |
| integrationId | string | Reference to the DriveIntegration being configured |
| accessToken | string | Short-lived OAuth2 token for the Picker |
| existingFileIds | string[] | Pre-selected file IDs (for modify flow) |
| selectedFiles | PickerFile[] | Files selected in the current Picker session |
| action | enum | `picked`, `cancel` |

## Relationships

```
DriveIntegration (1) ──── has many ───→ FileGrant (*)
     │
     └── belongs to ──→ User / Project
```

## Storage Strategy

All entities are stored as **Kubernetes resources**:

- **DriveIntegration**: Stored as a ConfigMap in the project namespace (following existing integration config patterns)
- **FileGrant**: Stored as a JSON-encoded list within the DriveIntegration ConfigMap
- **OAuth Tokens**: Stored in Kubernetes Secrets (following existing `oauth.go` credential storage pattern)
- **PickerSession**: Frontend-only state, not persisted to backend

## Indexes / Lookups

- Lookup FileGrants by `integrationId` (list all files for an integration)
- Lookup DriveIntegration by `userId` + `projectName` (find a user's integration)
- Lookup FileGrant by `googleFileId` + `integrationId` (check if a specific file is already granted)
