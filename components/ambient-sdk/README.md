# Ambient Platform SDK

**SDK for external developers integrating with the Ambient Platform.**

## Overview

This SDK provides a simple, HTTP-only client for interacting with the Ambient Code Platform via its public REST API. The SDK is designed for external developers who want to integrate AI agent capabilities into their applications without Kubernetes dependencies.

## Design Philosophy

The Ambient Platform SDK follows these core principles:
- **REST API**: Pure REST API client with no Kubernetes dependencies
- **Minimal Dependencies**: Uses only Go standard library
- **Simple Integration**: Easy to embed in any Go application  
- **Type Safety**: Strongly-typed request and response structures
- **Clear Separation**: Public SDK vs internal platform implementation

## Quick Start

### Installation

```bash
go get github.com/ambient/platform-sdk
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/ambient/platform-sdk/client"
    "github.com/ambient/platform-sdk/types"
)

func main() {
    // Create HTTP client
    client := client.NewClient("https://your-platform.example.com", "your-bearer-token", "your-project")

    // Create a new session
    req := &types.CreateSessionRequest{
        Task:  "Analyze this repository structure",
        Model: "claude-3.5-sonnet",
        Repos: []types.RepoHTTP{{
            URL:    "https://github.com/user/repo",
            Branch: "main",
        }},
    }

    resp, err := client.CreateSession(context.Background(), req)
    if err != nil {
        log.Fatal(err)
    }

    // Monitor session progress
    session, err := client.WaitForCompletion(
        context.Background(), 
        resp.ID, 
        5*time.Second,
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Session completed: %s", session.Status)
}
```

## Architecture

### Public SDK (This Repository)
- **HTTP Client**: Simple REST API client for session management
- **Type Safety**: Request/response types matching public API
- **Zero K8s Dependencies**: Pure Go standard library implementation

### Internal Platform Usage
- **Backend**: Can import `types.internal_types` for Kubernetes struct compatibility
- **Operator**: Continues using existing Kubernetes client patterns
- **Shared Types**: Common type definitions support both public and internal usage

## Implementation Status

### Phase 1: HTTP-Only SDK ✅
- [x] HTTP client with AgenticSession management
- [x] Type-safe request/response handling  
- [x] Bearer token authentication
- [x] Session status polling and monitoring
- [x] Comprehensive examples and documentation
- [x] Zero Kubernetes dependencies

### Phase 2: Frontend Integration  
- [ ] Generate TypeScript types from OpenAPI
- [ ] Create TypeScript SDK with React Query integration
- [ ] Migrate frontend to use generated types
- [ ] Replace manual `fetch()` calls with SDK client

### Phase 3: Python SDK for External Users
- [ ] Generate Pydantic models from OpenAPI
- [ ] Create Python client SDK with async support
- [ ] Add authentication (API key + kubeconfig)
- [ ] Implement real-time session monitoring

### Phase 4: Advanced Features
- [ ] SDK-based testing utilities
- [ ] Cross-language validation rules
- [ ] Automatic type migration tools
- [ ] OpenTelemetry instrumentation

## Directory Structure

```
ambient-sdk/
├── README.md                 # This file
├── docs/                    # Architecture and auth documentation
├── go-sdk/                  # Go client library
│   ├── types/              # Generated Go types
│   ├── client/             # K8s client utilities  
│   └── examples/           # Usage examples
├── python-sdk/             # Python client library
│   ├── ambient_platform/   # Generated Pydantic models
│   ├── client/             # HTTP client implementation
│   └── examples/           # Usage examples
└── typescript-sdk/         # TypeScript client library (future)
    ├── types/              # Generated TypeScript types
    ├── client/             # React Query integration
    └── examples/           # Usage examples
```

## Benefits by Component

### Backend (`components/backend/`)
**Before**: Manual JSON parsing with type assertions
```go
if timeout, ok := spec["timeout"].(float64); ok {
    result.Timeout = int(timeout)
}
```

**After**: Type-safe operations  
```go
import "github.com/ambient/platform-sdk/types"

session := types.AgenticSession{}
// Compile-time type safety, automatic validation
```

### Operator (`components/operator/`)
**Before**: Fragile unstructured access
```go
spec, found, err := unstructured.NestedMap(obj.Object, "spec")
displayName := spec["displayName"].(string) // Can panic!
```

**After**: Type-safe field access
```go
session, err := sdk.FromUnstructured(obj)
displayName := session.Spec.DisplayName // Type-safe
```

### Frontend (`components/frontend/`)
**Before**: Manual type synchronization
```typescript
// Types drift from backend changes
export type AgenticSession = { /* manually maintained */ }
```

**After**: Generated types
```typescript
import { AgenticSession } from '@ambient/platform-types'
// Always in sync with API
```

### Python SDK (New)
**Target**: External automation users
```python
from ambient_platform_sdk import AmbientClient

client = AmbientClient.from_env()
session = await client.sessions.create(
    task="Review PR #123 for security vulnerabilities",
    model="claude-4-5-sonnet",
    repos=["github.com/myorg/myrepo"]
)

# Real-time monitoring
async for update in session.watch():
    if update.status.phase == "Completed":
        print(f"Session completed: {update.status}")
        break
```

## Migration Strategy

1. **Backward Compatibility**: Existing APIs remain unchanged
2. **Gradual Adoption**: Components migrate incrementally  
3. **Type Safety**: Compile-time guarantees prevent regressions
4. **Automated Testing**: SDK includes comprehensive test suites

## OpenAPI Specification

The canonical API spec lives in the API server at `../ambient-api-server/openapi/openapi.yaml`. The SDK does not maintain its own copy — it derives types and client behavior from the API server's spec. When the API server adds or changes endpoints, the SDK wrappers are updated to match.

## Getting Started

### For Backend/Operator Development
```bash
cd components/ambient-sdk/go-sdk
go mod init github.com/ambient/platform-sdk
```

### For Python Automation  
```bash
pip install ambient-platform-sdk
export AMBIENT_API_KEY="your-key"
```

### For Frontend Development
```bash
npm install @ambient/platform-types @ambient/platform-sdk
```

This SDK establishes the Ambient Platform as a cohesive system with shared types, eliminating the manual synchronization burden while providing rich, language-idiomatic client libraries.