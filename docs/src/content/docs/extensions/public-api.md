---
title: "Public API"
---

The Public API is a stateless HTTP gateway that provides a simplified REST interface for managing agentic sessions. It proxies authenticated requests to the backend service and never accesses Kubernetes directly. Use it to integrate external tools, scripts, or CI/CD pipelines with the Ambient Code Platform.

## Authentication

Every request to the `/v1` endpoints requires a bearer token in the `Authorization` header. The gateway forwards this token to the backend, which performs full validation including signature verification and RBAC enforcement.

```http
Authorization: Bearer <token>
```

The gateway also accepts tokens through the `X-Forwarded-Access-Token` header for compatibility with OAuth proxies.

## Project context

All session operations are scoped to a project. Set the project using the `X-Ambient-Project` header:

```http
X-Ambient-Project: my-project
```

If you use a Kubernetes ServiceAccount token, the gateway can extract the project from the token's namespace claim. When both the header and the token specify a project, they must match — a mismatch returns an authentication error to prevent routing attacks. If no project can be resolved from either the header or the token, the gateway returns `400 Bad Request`.

## Endpoints

### List sessions

```http
GET /v1/sessions
```

Returns all sessions in the specified project.

**Response:**

```json
{
  "items": [
    {
      "id": "session-abc123",
      "status": "completed",
      "task": "Refactor the authentication module",
      "model": "claude-sonnet-4-20250514",
      "createdAt": "2025-06-15T10:30:00Z",
      "completedAt": "2025-06-15T10:45:00Z",
      "result": "Refactored auth module to use middleware pattern..."
    }
  ],
  "total": 1
}
```

### Get session details

```http
GET /v1/sessions/:id
```

Returns a single session by ID.

**Response:**

```json
{
  "id": "session-abc123",
  "status": "running",
  "task": "Refactor the authentication module",
  "model": "claude-sonnet-4-20250514",
  "createdAt": "2025-06-15T10:30:00Z"
}
```

### Create a session

```http
POST /v1/sessions
```

Creates a new agentic session.

**Request body:**

```json
{
  "task": "Analyze this repository for security issues",
  "model": "claude-sonnet-4-20250514",
  "repos": [
    {
      "url": "https://github.com/org/repo.git",
      "branch": "main"
    }
  ]
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `task` | Yes | The prompt or task for the agent to perform |
| `model` | No | Model override (defaults to the project's configured model) |
| `repos` | No | Array of repositories to clone into the session |
| `repos[].url` | Yes | Git repository URL |
| `repos[].branch` | No | Branch to check out |

**Response** (`201 Created`):

```json
{
  "id": "session-abc123",
  "message": "Session created"
}
```

### Delete a session

```http
DELETE /v1/sessions/:id
```

Deletes a session by ID. Returns `204 No Content` on success.

## Response format

All session responses use a simplified DTO with the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Session name (Kubernetes resource name) |
| `status` | string | Normalized status: `pending`, `running`, `completed`, `failed`, or `stopped` |
| `task` | string | The task prompt that was submitted |
| `model` | string | Model used for the session (omitted if not set) |
| `createdAt` | string | ISO 8601 creation timestamp |
| `completedAt` | string | ISO 8601 completion timestamp (omitted if not finished) |
| `result` | string | Session result text (omitted if not completed) |
| `error` | string | Error message (omitted if no error) |

## Phase state mapping

The gateway normalizes Kubernetes phase values into simplified statuses:

| API status | Kubernetes phases |
|------------|-------------------|
| `pending` | `Pending`, `Creating`, `Initializing` |
| `running` | `Running`, `Active` |
| `completed` | `Completed`, `Succeeded` |
| `failed` | `Failed`, `Error` |

Phases not listed above (such as `Stopped` or `Stopping`) pass through as-is without normalization. If no status is set on the underlying resource, the gateway defaults to `pending`.

## Rate limiting

The gateway enforces per-IP rate limiting on all `/v1` endpoints. Health, readiness, and metrics endpoints are excluded from rate limiting.

| Setting | Default | Environment variable |
|---------|---------|---------------------|
| Requests per second | 100 | `RATE_LIMIT_RPS` |
| Burst size | 200 | `RATE_LIMIT_BURST` |

When the rate limit is exceeded, the gateway returns `429 Too Many Requests`:

```json
{
  "error": "Rate limit exceeded",
  "retry_after": "1s"
}
```

## Health endpoints

These endpoints do not require authentication.

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Returns `{"status": "ok"}` when the service is running |
| `GET /ready` | Returns `{"status": "ready"}` when the service can accept traffic |
| `GET /metrics` | Prometheus-format metrics |

## Configuration

Configure the gateway with environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKEND_URL` | `http://backend-service:8080` | Internal backend service URL |
| `BACKEND_TIMEOUT` | `30s` | Timeout for backend requests (Go duration format) |
| `PORT` | `8081` | Port the gateway listens on |
| `RATE_LIMIT_RPS` | `100` | Maximum requests per second per IP |
| `RATE_LIMIT_BURST` | `200` | Maximum burst size per IP |
| `CORS_ALLOWED_ORIGINS` | `""` (when unset, the gateway defaults to `localhost:3000`, `localhost:8080`, and `*.apps-crc.testing`) | Comma-separated list of allowed CORS origins |
| `GIN_MODE` | `release` | Gin framework mode (`release`, `debug`, `test`) |

## Error responses

All errors use a consistent JSON format:

```json
{
  "error": "Description of the error"
}
```

Common error codes:

| Status | Meaning |
|--------|---------|
| `400 Bad Request` | Invalid project name, session ID, or request body |
| `401 Unauthorized` | Missing or invalid bearer token |
| `429 Too Many Requests` | Rate limit exceeded |
| `502 Bad Gateway` | Backend service is unavailable |

## Input validation

The gateway validates all path parameters against Kubernetes naming rules before forwarding requests. Project names and session IDs must:

- Start with a lowercase letter or digit
- Contain only lowercase letters, digits, or hyphens
- End with a lowercase letter or digit
- Be at most 63 characters

Invalid names are rejected with `400 Bad Request` before reaching the backend.
