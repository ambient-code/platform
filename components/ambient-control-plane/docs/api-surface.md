# API Surface Reference

The ambient-control-plane consumes the REST API produced by `ambient-api-server`. This document covers the full API surface relevant to the control plane.

## Base URL

```
/api/ambient-api-server/v1
```

Default local dev: `http://localhost:8000`

## Authentication

All endpoints require HTTP Bearer token (JWT):

```
Authorization: Bearer <token>
```

In the Go client, set via `AMBIENT_API_TOKEN` env var or by adding the default header:

```go
cfg.AddDefaultHeader("Authorization", "Bearer "+token)
```

## Resource Endpoints

All resources follow a uniform CRUD pattern. No DELETE operations exist except on Users.

### Sessions

| Method | Path | Description |
|---|---|---|
| GET | `/sessions` | List sessions |
| POST | `/sessions` | Create session |
| GET | `/sessions/{id}` | Get session by ID |
| PATCH | `/sessions/{id}` | Update session |

**Fields**: `name` (required), `repo_url`, `prompt`, `created_by_user_id`, `assigned_user_id`, `workflow_id`

### Workflows

| Method | Path | Description |
|---|---|---|
| GET | `/workflows` | List workflows |
| POST | `/workflows` | Create workflow |
| GET | `/workflows/{id}` | Get workflow by ID |
| PATCH | `/workflows/{id}` | Update workflow |

**Fields**: `name` (required), `repo_url`, `prompt`, `agent_id`

### Tasks

| Method | Path | Description |
|---|---|---|
| GET | `/tasks` | List tasks |
| POST | `/tasks` | Create task |
| GET | `/tasks/{id}` | Get task by ID |
| PATCH | `/tasks/{id}` | Update task |

**Fields**: `name` (required), `repo_url`, `prompt`

### Agents

| Method | Path | Description |
|---|---|---|
| GET | `/agents` | List agents |
| POST | `/agents` | Create agent |
| GET | `/agents/{id}` | Get agent by ID |
| PATCH | `/agents/{id}` | Update agent |

**Fields**: `name` (required), `repo_url`, `prompt`

### Skills

| Method | Path | Description |
|---|---|---|
| GET | `/skills` | List skills |
| POST | `/skills` | Create skill |
| GET | `/skills/{id}` | Get skill by ID |
| PATCH | `/skills/{id}` | Update skill |

**Fields**: `name` (required), `repo_url`, `prompt`

### Users

| Method | Path | Description |
|---|---|---|
| GET | `/users` | List users |
| POST | `/users` | Create user |
| GET | `/users/{id}` | Get user by ID |
| PATCH | `/users/{id}` | Update user |
| DELETE | `/users/{id}` | Delete user |

**Fields**: `username` (required), `name` (required)

### WorkflowSkills (join table)

| Method | Path | Description |
|---|---|---|
| GET | `/workflow_skills` | List workflow-skill associations |
| POST | `/workflow_skills` | Create workflow-skill association |
| GET | `/workflow_skills/{id}` | Get by ID |
| PATCH | `/workflow_skills/{id}` | Update |

**Fields**: `workflow_id` (required), `skill_id` (required), `position` (required, int32)

### WorkflowTasks (join table)

| Method | Path | Description |
|---|---|---|
| GET | `/workflow_tasks` | List workflow-task associations |
| POST | `/workflow_tasks` | Create workflow-task association |
| GET | `/workflow_tasks/{id}` | Get by ID |
| PATCH | `/workflow_tasks/{id}` | Update |

**Fields**: `workflow_id` (required), `task_id` (required), `position` (required, int32)

## Common Model Fields (ObjectReference)

All resources inherit:

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique identifier |
| `kind` | string | Resource type name |
| `href` | string | Self-link |
| `created_at` | datetime | Creation timestamp |
| `updated_at` | datetime | Last modification timestamp |

The `updated_at` field is used by the informer to detect modifications.

## Pagination

All list endpoints accept these query parameters:

| Parameter | Type | Default | Description |
|---|---|---|---|
| `page` | integer | 1 | 1-indexed page number |
| `size` | integer | 100 | Max records per page |
| `search` | string | — | SQL WHERE clause (e.g. `name like 'foo%'`) |
| `orderBy` | string | — | SQL ORDER BY (e.g. `name asc, created_at desc`) |
| `fields` | string | — | Field projection (e.g. `id,name,href`) |

List response envelope:

```json
{
  "kind": "SessionList",
  "page": 1,
  "size": 100,
  "total": 42,
  "items": [...]
}
```

Total pages: `ceil(total / size)`

## Error Responses

| Code | Meaning |
|---|---|
| 400 | Validation error (POST/PATCH) |
| 401 | Invalid auth token |
| 403 | Unauthorized |
| 404 | Resource not found |
| 409 | Conflict (already exists) |
| 500 | Internal server error |

Error body: `{ "id", "kind", "href", "code", "reason", "operation_id" }`

## Entity Relationships

```
User ──creates──▶ Session (created_by_user_id)
User ──assigned──▶ Session (assigned_user_id)
Agent ──owns──▶ Workflow (agent_id)
Workflow ──has──▶ Session (workflow_id)
Workflow ◀──join──▶ Skill (via WorkflowSkill)
Workflow ◀──join──▶ Task (via WorkflowTask)
```

## Generated Go Client

The control plane imports the OpenAPI-generated client:

```go
import openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
```

Key types:
- `openapi.APIClient` — HTTP client with `DefaultAPI` service
- `openapi.Session`, `openapi.Workflow`, `openapi.Task` — model structs
- `openapi.SessionList`, `openapi.WorkflowList`, `openapi.TaskList` — list envelopes
- `openapi.Configuration` — server URLs, HTTP client, default headers

List call pattern (builder style):

```go
list, httpResp, err := client.DefaultAPI.
    ApiAmbientApiServerV1SessionsGet(ctx).
    Page(1).
    Size(100).
    Execute()
```
