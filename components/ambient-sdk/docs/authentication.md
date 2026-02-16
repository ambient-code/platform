# Authentication and Authorization

## Overview

The Ambient Platform SDK authenticates via **Bearer token** and scopes requests to a **project** (Kubernetes namespace). Both are required for every API call.

### Headers

| Header | Required | Purpose |
|---|---|---|
| `Authorization: Bearer <token>` | Yes | Authenticates the user |
| `X-Ambient-Project: <project>` | Yes | Scopes request to a namespace |
| `Content-Type: application/json` | Yes (POST) | Request body format |

## Token Formats

The SDK validates tokens on client construction. Accepted formats:

### OpenShift SHA256 Tokens

```
sha256~_3FClshuberfakepO_BGI_tZg_not_real_token_Jv72pRN-r5o
```

- Prefix: `sha256~`
- Minimum length: 20 characters
- Obtained via: `oc whoami -t`

### JWT Tokens

```
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.signature
```

- Three dot-separated base64url-encoded parts
- Each part must be non-empty and contain only `[a-zA-Z0-9_-]`
- Minimum total length: 50 characters

### GitHub Tokens

```
ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

- Prefixes: `ghp_`, `gho_`, `ghu_`, `ghs_`
- Minimum length: 40 characters

### Generic Tokens

- Minimum length: 20 characters
- Must contain both alphabetic and numeric characters

## Token Validation

Both SDKs validate tokens before making any API calls:

1. **Empty check** — Token cannot be empty
2. **Placeholder detection** — Rejects common placeholders (`YOUR_TOKEN_HERE`, `token`, `password`, `secret`, `example`, `test`, `demo`, `placeholder`, `TODO`)
3. **Minimum length** — At least 10 characters
4. **Format validation** — Checks against known token format rules above

### Go

```go
client, err := client.NewClient(baseURL, token, project)
if err != nil {
    // Token validation failed
    log.Fatal(err)
}
```

### Python

```python
try:
    client = AmbientClient(base_url, token, project)
except ValueError as e:
    # Token validation failed
    print(f"Invalid token: {e}")
```

## Log Safety

Tokens are never written to logs in plaintext.

### Go — Dual-Layer Redaction

1. **`SecureToken.LogValue()`** — Implements `slog.LogValuer`. Any `SecureToken` logged via `slog` automatically renders as `sha256***(<N>_chars)` instead of the raw value.

2. **`sanitizeLogAttrs()`** — A `slog.ReplaceAttr` function applied to the logger that catches sensitive values by:
   - Key name: `token`, `password`, `secret`, `apikey`, `authorization` (case variants)
   - Key suffix: `_token`, `_password`, `_secret`, `_key`
   - Value pattern: `Bearer ` prefix, `sha256~` prefix, JWT structure (`ey...` with 2+ dots)

### Python

The Python SDK does not log tokens. The `httpx.Client` is configured with auth headers at construction time, and no debug logging exposes them.

## Project Validation

The project name maps to a Kubernetes namespace:

- Cannot be empty
- Alphanumeric characters, hyphens, and underscores only
- Maximum 63 characters (Kubernetes namespace limit)

## RBAC Requirements

The authenticated user must have these Kubernetes RBAC permissions in the target namespace:

```yaml
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions"]
  verbs: ["get", "list", "create"]

- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get"]
```

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `AMBIENT_TOKEN` | Yes | — | Bearer authentication token |
| `AMBIENT_PROJECT` | Yes | — | Target project (Kubernetes namespace) |
| `AMBIENT_API_URL` | No | `http://localhost:8080` | API server base URL |

### Quick Setup

```bash
# OpenShift users
export AMBIENT_TOKEN="$(oc whoami -t)"
export AMBIENT_PROJECT="$(oc project -q)"
export AMBIENT_API_URL="https://public-api-route-yournamespace.apps.your-cluster.com"

# Manual setup
export AMBIENT_TOKEN="your-bearer-token"
export AMBIENT_PROJECT="your-namespace"
export AMBIENT_API_URL="https://api.ambient-code.io"
```

## Common Errors

| HTTP Status | Error | Cause | Fix |
|---|---|---|---|
| 400 | `Project required` | Missing `X-Ambient-Project` header | Set `AMBIENT_PROJECT` env var |
| 401 | `Unauthorized` | Invalid or expired token | Refresh token via `oc login` or regenerate |
| 403 | `Forbidden` | Insufficient RBAC permissions | Request `agenticsessions` access in the namespace |
| 404 | `Session not found` | Wrong session ID or no access to namespace | Verify ID and project match |

### Diagnosing Permission Issues

```bash
# Verify identity
oc whoami

# Check token validity
oc whoami -t

# Test RBAC permissions
oc auth can-i create agenticsessions.vteam.ambient-code -n <project>
oc auth can-i list agenticsessions.vteam.ambient-code -n <project>
oc auth can-i get agenticsessions.vteam.ambient-code -n <project>
```
