# Data Model: Gerrit Integration Connector

**Branch**: `001-gerrit-integration` | **Date**: 2026-03-24

## Entities

### GerritCredentials

Represents a user's connection to a single Gerrit instance.

| Field              | Type      | Required | Description                                                  |
|--------------------|-----------|----------|--------------------------------------------------------------|
| userID             | string    | Yes      | Platform user identifier (from auth context)                 |
| instanceName       | string    | Yes      | User-assigned name for this Gerrit instance (e.g., "openstack") |
| url                | string    | Yes      | Gerrit instance base URL (e.g., `https://review.opendev.org`) |
| authMethod         | enum      | Yes      | One of: `http_basic`, `git_cookies`                          |
| username           | string    | Conditional | Required when authMethod is `http_basic`                  |
| httpToken          | string    | Conditional | HTTP password/access token. Required when authMethod is `http_basic` |
| gitcookiesContent  | string    | Conditional | Raw gitcookies file content. Required when authMethod is `git_cookies` |
| updatedAt          | timestamp | Yes      | Last modification time (RFC3339)                             |

**Identity**: Unique by `(instanceName, userID)`.

**Validation rules**:
- `url` must be a valid HTTPS URL (HTTP allowed for local development only)
- `instanceName` must match `^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$` (lowercase, alphanumeric with hyphens, 1-63 chars)
- When `authMethod` is `http_basic`: both `username` and `httpToken` must be non-empty
- When `authMethod` is `git_cookies`: `gitcookiesContent` must be non-empty and contain at least one cookie line for the target host

### GerritIntegrationStatus

Read-only projection used for status reporting. Not stored separately.

| Field         | Type    | Description                                      |
|---------------|---------|--------------------------------------------------|
| connected     | boolean | Whether credentials exist for this instance       |
| instanceName  | string  | User-assigned name                                |
| url           | string  | Gerrit instance URL                               |
| authMethod    | string  | Authentication method in use                      |
| updatedAt     | string  | When credentials were last updated                |

## Storage

### Kubernetes Secret: `gerrit-credentials`

| Aspect          | Value                                        |
|-----------------|----------------------------------------------|
| Secret name     | `gerrit-credentials`                         |
| Namespace       | Platform namespace (same as other creds)     |
| Labels          | `app: ambient-code`, `ambient-code.io/provider: gerrit` |
| Data key format | `{instanceName}:{userID}`                    |
| Data value      | JSON-marshaled `GerritCredentials`           |

**Multi-instance**: A user with two Gerrit instances ("openstack" and "android") has two entries in the same secret:
- Key: `openstack:user123` → Value: `{"userID":"user123","instanceName":"openstack","url":"https://review.opendev.org",...}`
- Key: `android:user123` → Value: `{"userID":"user123","instanceName":"android","url":"https://android-review.googlesource.com",...}`

## Runtime Configuration

### Generated `gerrit_config.json`

At session startup, the runner generates this file from fetched credentials:

```json
{
  "default_gerrit_base_url": "https://review.opendev.org/",
  "gerrit_hosts": [
    {
      "name": "openstack",
      "external_url": "https://review.opendev.org/",
      "authentication": {
        "type": "http_basic",
        "username": "user",
        "auth_token": "token"
      }
    },
    {
      "name": "android",
      "external_url": "https://android-review.googlesource.com/",
      "authentication": {
        "type": "git_cookies",
        "gitcookies_path": "/tmp/gerrit-mcp/.gitcookies"
      }
    }
  ]
}
```

For `git_cookies` auth, the runner also writes the gitcookies content to a temporary file and references its path in the config.

## State Transitions

```
[Not Connected] --connect()--> [Connected]
[Connected]     --disconnect()--> [Not Connected]
[Connected]     --connect() with same instanceName--> [Connected] (credentials updated)
```

No intermediate states. Connection is synchronous (validate + store).

## Relationships

```
User (1) ---has many---> GerritCredentials (N)
GerritCredentials (N) ---generates at runtime---> gerrit_config.json (1 per session)
gerrit_config.json (1) ---consumed by---> Gerrit MCP Server (1 per session)
```
