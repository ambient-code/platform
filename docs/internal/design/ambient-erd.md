# Ambient Platform — Data Model ERD

This is the canonical entity-relationship diagram for the `ambient-api-server`. It reflects the current OpenAPI spec (`platform-api-server/components/ambient-api-server/openapi/openapi.yaml`) and the backing PostgreSQL models.

All Kinds inherit `id` (KSUID), `kind`, `href`, `created_at`, `updated_at`, and `deleted_at` from `api.Meta` (TRex base). These are omitted from individual entity fields below for clarity but are present on every record.

> **This document is the Desired State spec.** To propose a new Kind or field, open a proposal in `docs/internal/proposals/` that references a diff against this ERD. When the proposal is accepted, this document is updated and the Software Factory reconciles the codebase to match.

---

```mermaid
erDiagram

    User {
        string id PK
        string username
        string name
        string email
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    Project {
        string id PK
        string name
        string display_name
        string description
        string labels
        string annotations
        string status
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    ProjectSettings {
        string id PK
        string project_id FK
        string group_access "JSON blob: [{group, role}] — interim RBAC"
        string repositories "JSON blob: repo config"
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    Session {
        string id PK
        string name
        string project_id FK
        string created_by_user_id FK "readOnly: set from auth token"
        string assigned_user_id FK
        string parent_session_id FK
        string workflow_id
        string repo_url
        string repos "JSON blob: [{url, branch}]"
        string prompt
        integer timeout
        string llm_model
        float llm_temperature
        integer llm_max_tokens
        string bot_account_name
        string resource_overrides "JSON blob"
        string environment_variables "JSON blob"
        string labels "JSON blob"
        string annotations "JSON blob"
        string phase "readOnly: Pending|Creating|Running|Stopping|Stopped|Completed|Failed"
        timestamp start_time "readOnly"
        timestamp completion_time "readOnly"
        string sdk_session_id "readOnly"
        integer sdk_restart_count "readOnly"
        string conditions "readOnly: JSON blob"
        string reconciled_repos "readOnly: JSON blob"
        string reconciled_workflow "readOnly: JSON blob"
        string kube_cr_name "readOnly"
        string kube_cr_uid "readOnly"
        string kube_namespace "readOnly"
        timestamp created_at
        timestamp updated_at
        timestamp deleted_at
    }

    User ||--o{ Session : "creates (created_by_user_id)"
    User ||--o{ Session : "assigned (assigned_user_id)"
    Project ||--o{ Session : "contains"
    Project ||--|| ProjectSettings : "has"
    Session ||--o{ Session : "parent (parent_session_id)"
```

---

## Notes

### `ProjectSettings.group_access`
A raw JSON string blob today: `[{"group": "my-team", "role": "ambient-project-admin"}]`. This is the current mechanism for granting Kubernetes RBAC access to a project namespace. It is not enforced at the API layer — it is only read by the Control Plane to create K8s `RoleBinding` objects.

**This is a known limitation.** See `docs/internal/proposals/rbac-rolebinding.md` for the proposed replacement.

### Session status fields
Fields marked `readOnly` are populated exclusively by the Control Plane's write-back mechanism (`PATCH /api/ambient/v1/sessions/{id}/status`). They reflect runtime state from the Kubernetes operator and should never be set directly by API clients.

### Session phases
Valid phase transitions:

```
nil/empty ──► Pending ──► Creating ──► Running ──► Stopping ──► Stopped
                                    └──────────────────────────► Completed
                                    └──────────────────────────► Failed
Stopped/Completed/Failed ──► Pending  (restart via /start)
```

### Implicit TRex meta fields
Every entity includes these fields via `api.Meta` (not shown in ERD to reduce noise):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | KSUID — globally unique, sortable |
| `kind` | string | Resource type name, e.g. `Session` |
| `href` | string | Self-link, e.g. `/api/ambient/v1/sessions/{id}` |
| `created_at` | timestamp | Set on insert, immutable |
| `updated_at` | timestamp | Updated on every write |
| `deleted_at` | timestamp | Soft-delete marker (null = active) |
