# Ambient Platform SDK

**Language-idiomatic HTTP client libraries for the Ambient Code Platform's public REST API.**

## Overview

The Ambient Platform SDK provides Go, Python, and TypeScript client libraries for interacting with the Ambient Platform API. It exists so that external developers and internal automation can create and manage AI agentic sessions **without Kubernetes dependencies or cluster access**. The SDK is the public-facing contract for the platform.

## Supported Languages

- **Go SDK** - `go-sdk/` - Go 1.21+ with standard library only
- **Python SDK** - `python-sdk/` - Python 3.8+ with minimal dependencies  
- **TypeScript SDK** - `ts-sdk/` - Modern TypeScript with proper type safety

## Quick Start

### Go

```bash
# Add to go.mod (local development)
require github.com/ambient-code/platform/components/ambient-sdk/go-sdk v0.0.0

# Usage
import "github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
```

### Python

```bash
pip install -e python-sdk/
```

```python
from ambient_platform.client import AmbientClient

client = AmbientClient.from_env()
session = client.sessions.create({
    "name": "My Analysis Session", 
    "prompt": "Analyze this codebase"
})
```

### TypeScript

```bash
cd ts-sdk && npm install
```

```typescript
import { AmbientClient } from './src/client'

const client = new AmbientClient({
  baseURL: process.env.AMBIENT_API_URL,
  token: process.env.AMBIENT_TOKEN,
  project: process.env.AMBIENT_PROJECT
})
```

## Environment Variables

All SDKs support these environment variables:

| Variable | Required | Description |
|---|---|---|
| `AMBIENT_TOKEN` | Yes | Bearer token (OpenShift `sha256~`, JWT, or GitHub `ghp_`) |
| `AMBIENT_PROJECT` | Yes | Target project / Kubernetes namespace |
| `AMBIENT_API_URL` | No | API base URL (default: `http://localhost:8080`) |

## API Resources

The SDK provides access to 4 core resources:

- **Sessions** - Create and manage AI agentic sessions
- **Users** - User management and authentication  
- **Projects** - Project configuration and settings
- **ProjectSettings** - Project-specific configuration

## Development

```bash
# Generate all SDKs from OpenAPI spec
make generate-sdk

# Verify all SDKs build correctly
make verify-sdk  

# Build generator binary
make build-generator
```

## Architecture

The SDK is generated from the OpenAPI specification at `../ambient-api-server/openapi/openapi.yaml` using a custom Go-based generator. This ensures type safety and consistency across all supported languages.

For detailed documentation, see:
- `docs/architecture.md` - Design decisions and platform integration
- `docs/authentication.md` - Auth flows and token formats
- `go-sdk/README.md` - Go-specific usage
- `python-sdk/README.md` - Python-specific usage