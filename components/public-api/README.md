# Public API

The Public API is a lightweight gateway service that provides a simplified, versioned REST API for the Ambient Code Platform. It acts as the single entry point for all clients (Browser, SDK, MCP).

## Architecture

```
Browser ──┐
SDK ──────┼──▶ [Public API] ──▶ [Go Backend (internal)]
MCP ──────┘
```

## Endpoints

### Sessions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/sessions` | List sessions |
| POST | `/v1/sessions` | Create session |
| GET | `/v1/sessions/:id` | Get session details |
| DELETE | `/v1/sessions/:id` | Delete session |

### Health

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |

## Authentication

The API supports two authentication methods:

1. **Bearer Token**: Pass an OpenShift token or access key in the `Authorization` header
2. **OAuth Proxy**: Token passed via `X-Forwarded-Access-Token` header

### Project Selection

Projects can be specified via:

1. **Header**: `X-Ambient-Project: my-project`
2. **Token**: For ServiceAccount tokens, the project is extracted from the namespace

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8081` | Server port |
| `BACKEND_URL` | `http://backend-service:8080` | Internal backend URL |
| `BACKEND_TIMEOUT` | `30s` | Backend request timeout (Go duration format) |
| `GIN_MODE` | `release` | Gin mode (debug/release) |

## Development

```bash
# Run locally
export BACKEND_URL=http://localhost:8080
go run .

# Build
go build -o public-api .

# Build Docker image
docker build -t public-api .
```

## Example Usage

```bash
# List sessions
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Ambient-Project: my-project" \
     http://localhost:8081/v1/sessions

# Create session
curl -X POST \
     -H "Authorization: Bearer $TOKEN" \
     -H "X-Ambient-Project: my-project" \
     -H "Content-Type: application/json" \
     -d '{"task": "Refactor login.py"}' \
     http://localhost:8081/v1/sessions

# Get session
curl -H "Authorization: Bearer $TOKEN" \
     -H "X-Ambient-Project: my-project" \
     http://localhost:8081/v1/sessions/session-123

# Delete session
curl -X DELETE \
     -H "Authorization: Bearer $TOKEN" \
     -H "X-Ambient-Project: my-project" \
     http://localhost:8081/v1/sessions/session-123
```
