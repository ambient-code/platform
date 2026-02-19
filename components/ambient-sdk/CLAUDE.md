# CLAUDE.md — Ambient Platform SDK

## Project Intent

The Ambient Platform SDK provides language-idiomatic HTTP client libraries for the Ambient Code Platform's public REST API. It exists so that external developers and internal automation can create and manage AI agentic sessions **without Kubernetes dependencies or cluster access**. The SDK is the public-facing contract for the platform — everything behind it (CRDs, operators, runners) is an implementation detail.

## Role in the Platform

This SDK is one piece of a multi-component system coordinated via `../working.md`:

| Component | Purpose | Relationship to SDK |
|---|---|---|
| **ambient-api-server** | REST API gateway (Go + Gin) | The server this SDK talks to — implements `/v1/sessions` |
| **ambient-control-plane** | Reconciler / controller | Watches API server for session changes; SDK users never interact with it |
| **ambient-sdk** (this) | Client libraries (Go, Python) | Consumes the API server's public endpoints |
| **Frontend** | NextJS web UI | Will eventually share generated types from `openapi.yaml` |
| **Operator** | Kubernetes controller | Internal only — spawns Jobs from CRs |
| **Runner** | Claude Code CLI executor | Internal only — runs inside Job pods |

Cross-session coordination with `ambient-api-server` and `ambient-control-plane` happens through `../working.md`. Read it before making contract-breaking changes.

**ALWAYS** report changes, suggestions, and next steps in `../working.md` because we are in active development mode. I *will* remove this directive when we're done, but we're not done yet. **ALWAYS** report status to working.md via the coordinator server.

### Coordinator Server Protocol

The coordinator server at `http://localhost:4345` manages `../working.md`. Each agent owns a section between `<!-- BEGIN:SDK -->` and `<!-- END:SDK -->` markers.

- **Post your update:** Write content to a temp file, then `curl -s -X POST http://localhost:4345/agent/sdk -H 'Content-Type: text/plain' --data-binary @/tmp/sdk_update.md`
- **Read your section only:** `curl -s http://localhost:4345/agent/sdk`
- **Read full document:** `curl -s http://localhost:4345/raw`
- **Do NOT use** `POST /update` with JSON — that endpoint does not exist. Use `/agent/sdk` with `text/plain`.
- Always write to a temp file first and use `--data-binary @file` to avoid shell mangling of markdown.

## Quick Reference

```bash
# Go SDK
cd go-sdk && go test ./...
cd go-sdk/examples && go run main.go

# Python SDK
cd python-sdk && ./test.sh
cd python-sdk && pip install -e ".[dev]" && pytest
cd python-sdk && python examples/main.py
```

### Environment Variables (all SDKs)

| Variable | Required | Description |
|---|---|---|
| `AMBIENT_TOKEN` | Yes | Bearer token (OpenShift `sha256~`, JWT, or GitHub `ghp_`) |
| `AMBIENT_PROJECT` | Yes | Target project / Kubernetes namespace |
| `AMBIENT_API_URL` | No | API base URL (default: `http://localhost:8080`) |

## Directory Structure

```
ambient-sdk/
├── CLAUDE.md              # This file
├── README.md              # Public-facing overview and roadmap
├── docs/                  # Detailed documentation
│   ├── architecture.md    # Design decisions, platform integration
│   └── authentication.md  # Auth flows, token formats, RBAC requirements
├── go-sdk/                # Go client library
│   ├── client/client.go   # HTTP client with structured logging and token sanitization
│   ├── types/types.go     # Request/response types, SecureToken, input validation
│   ├── examples/main.go   # Complete session lifecycle example
│   ├── go.mod             # Module: github.com/ambient-code/platform/components/ambient-sdk/go-sdk
│   └── README.md          # Go-specific usage and API reference
└── python-sdk/            # Python client library
    ├── ambient_platform/  # Package root
    │   ├── __init__.py    # Public exports, version
    │   ├── client.py      # AmbientClient with httpx, env-based factory
    │   ├── types.py       # Dataclasses matching OpenAPI schemas
    │   └── exceptions.py  # Typed exception hierarchy
    ├── examples/main.py   # Complete session lifecycle example
    ├── test.sh            # Integration test runner with env validation
    ├── pyproject.toml     # Package config (black, isort, mypy, pytest)
    └── README.md          # Python-specific usage and API reference
```

## Code Conventions

### Go SDK

- **Go 1.21+**, standard library only (no third-party deps)
- `go fmt ./...` and `golangci-lint run` enforced
- `SecureToken` type wraps tokens and implements `slog.LogValuer` for automatic redaction
- `sanitizeLogAttrs` in `client.go` provides defense-in-depth log sanitization
- All client constructors return `(*Client, error)` — token validation is mandatory
- Input validation via `Validate()` methods on request types

### Python SDK

- **Python 3.8+**, single dependency: `httpx>=0.25.0`
- `black` formatting, `isort` with black profile, `mypy` strict mode
- Dataclasses for all types (no Pydantic — intentionally lightweight)
- `AmbientClient.from_env()` factory for environment-based configuration
- Context manager support (`with AmbientClient(...) as client:`)
- Typed exception hierarchy rooted at `AmbientAPIError`

### Both SDKs

- Never log tokens — use `len(token)` or `SecureToken.LogValue()` / `[REDACTED]`
- All request types have `Validate()` / `validate()` methods called before HTTP calls
- API errors return structured `ErrorResponse` without leaking raw response bodies
- Token format validation: OpenShift `sha256~`, JWT (3 dot-separated base64 parts), GitHub `ghp_/gho_/ghu_/ghs_`

## OpenAPI Specification

The API server owns the canonical OpenAPI spec at `../ambient-api-server/openapi/openapi.yaml`. The SDK does **not** maintain its own copy — it derives types and client behavior from the API server's spec.

- **Spec location**: `../ambient-api-server/openapi/` (split by resource: sessions, agents, tasks, workflows, etc.)
- **Session endpoints**: `GET /api/ambient-api-server/v1/sessions`, `POST ...`, `GET .../sessions/{id}`
- **Auth**: `Authorization: Bearer <token>` header (project scoping via `X-Ambient-Project`)
- **Statuses**: `pending` → `running` → `completed` | `failed`
- Update the API server's spec before changing SDK types or client behavior

## Security Considerations

- Tokens are validated on client construction (format, length, placeholder detection)
- Go SDK uses `slog.LogValuer` + `ReplaceAttr` for dual-layer log redaction
- Bearer tokens, SHA256 tokens, and JWTs are pattern-matched and redacted in logs
- API error responses are sanitized before returning to callers
- URL validation rejects placeholder domains (`example.com`) and dangerous schemes

## Smoke Test

Run `cd go-sdk && go run examples/main.go` until it passes. This is the SDK's end-to-end smoke test against the live API server. It currently returns 404 because the API server has not been migrated to serve `/api/ambient-api-server/v1/sessions` yet. Once the full migration (api-server + control-plane + deployment) is complete, this test will pass. Keep running it — when it stops returning 404, the platform is wired up.

## Loadable Context

| Topic | File |
|---|---|
| Architecture and platform integration | `docs/architecture.md` |
| Authentication, tokens, and RBAC | `docs/authentication.md` |
| Go SDK details | `go-sdk/README.md` |
| Python SDK details | `python-sdk/README.md` |
| API contract (source of truth) | `../ambient-api-server/openapi/openapi.yaml` |
| Cross-session coordination | `../working.md` |
