# Public API

The Public API is a lightweight gateway service that provides a simplified, versioned REST API for the Ambient Code Platform. It acts as the single entry point for all clients (Browser, SDK, MCP).

## Architecture

```
Browser ──┐
SDK ──────┼──▶ [Public API] ──▶ [Go Backend (internal)]
MCP ──────┘
```

## Features

- **CORS Support**: Configured for browser clients
- **Rate Limiting**: Per-IP rate limiting to prevent abuse
- **Structured Logging**: JSON-formatted logs for production
- **OpenTelemetry Tracing**: Distributed tracing support (optional)
- **Prometheus Metrics**: `/metrics` endpoint for monitoring
- **Input Validation**: Kubernetes name validation to prevent injection attacks
- **Security**: Token redaction in logs, project mismatch detection

## Endpoints

### Sessions

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/sessions` | List sessions |
| POST | `/v1/sessions` | Create session |
| GET | `/v1/sessions/:id` | Get session details |
| DELETE | `/v1/sessions/:id` | Delete session |

### Health & Monitoring

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/ready` | Readiness check |
| GET | `/metrics` | Prometheus metrics |

## Authentication

The API supports two authentication methods:

1. **Bearer Token**: Pass an OpenShift token or access key in the `Authorization` header
2. **OAuth Proxy**: Token passed via `X-Forwarded-Access-Token` header

### Project Selection

Projects can be specified via:

1. **Header**: `X-Ambient-Project: my-project`
2. **Token**: For ServiceAccount tokens, the project is extracted from the namespace

**Security Note**: If both header and token specify a project, they must match. This prevents routing attacks.

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8081` | Server port |
| `BACKEND_URL` | `http://backend-service:8080` | Internal backend URL |
| `BACKEND_TIMEOUT` | `30s` | Backend request timeout (Go duration format) |
| `GIN_MODE` | `release` | Gin mode (debug/release) |
| `RATE_LIMIT_RPS` | `100` | Requests per second per IP |
| `RATE_LIMIT_BURST` | `200` | Maximum burst size |
| `CORS_ALLOWED_ORIGINS` | (see below) | Comma-separated list of allowed origins |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | (disabled) | OpenTelemetry collector endpoint |
| `OTEL_ENABLED` | `false` | Enable OpenTelemetry tracing |

### Default CORS Origins

If `CORS_ALLOWED_ORIGINS` is not set, the following origins are allowed:
- `http://localhost:3000` (Next.js dev server)
- `http://localhost:8080` (Frontend in kind)
- `https://*.apps-crc.testing` (CRC routes)

## Development

```bash
# Run locally
export BACKEND_URL=http://localhost:8080
go run .

# Run with debug logging
GIN_MODE=debug go run .

# Build
go build -o public-api .

# Build Docker image
docker build -t public-api .

# Run tests
go test ./... -v
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

# Check metrics
curl http://localhost:8081/metrics
```

## OpenTelemetry Integration

To enable distributed tracing:

```bash
# Set the OTLP endpoint
export OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
export OTEL_ENABLED=true

# Or in Kubernetes deployment:
env:
  - name: OTEL_EXPORTER_OTLP_ENDPOINT
    value: "http://otel-collector:4318"
  - name: OTEL_ENABLED
    value: "true"
```

Traces will include:
- HTTP request spans with method, path, status
- Backend proxy call spans
- Error details and latency metrics
