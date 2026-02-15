# Ambient Platform Go SDK

Simple HTTP client library for the Ambient Code Platform - Create and manage AI agent sessions without Kubernetes complexity.

## Installation

```bash
go get github.com/ambient/platform-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/ambient/platform-sdk/client"
    "github.com/ambient/platform-sdk/types"
)

func main() {
    // Create HTTP client
    apiURL := "https://your-platform.example.com"
    token := os.Getenv("AMBIENT_TOKEN")      // Bearer token
    project := os.Getenv("AMBIENT_PROJECT")  // Project namespace

    client := client.NewClient(apiURL, token, project)

    // Create a session
    createReq := &types.CreateSessionRequest{
        Task:  "Analyze the repository structure and provide a summary",
        Model: "claude-3.5-sonnet",
        Repos: []types.RepoHTTP{
            {
                URL:    "https://github.com/ambient-code/platform",
                Branch: "main",
            },
        },
    }

    resp, err := client.CreateSession(context.Background(), createReq)
    if err != nil {
        log.Fatalf("Failed to create session: %v", err)
    }

    fmt.Printf("Created session: %s\n", resp.ID)

    // Get session details
    session, err := client.GetSession(context.Background(), resp.ID)
    if err != nil {
        log.Fatalf("Failed to get session: %v", err)
    }

    fmt.Printf("Status: %s\n", session.Status)

    // List all sessions
    listResp, err := client.ListSessions(context.Background())
    if err != nil {
        log.Fatalf("Failed to list sessions: %v", err)
    }

    fmt.Printf("Found %d sessions\n", len(listResp.Items))
}
```

## Authentication & Authorization

The SDK uses Bearer token authentication with project-scoped authorization:

### Token Requirements

- **Bearer Token**: Must be a valid authentication token (OpenShift, JWT, or GitHub format)
- **Project Header**: `X-Ambient-Project` specifies the target Kubernetes namespace
- **RBAC**: User must have appropriate permissions in the target namespace

### Supported Token Formats

- **OpenShift**: `sha256~...` format tokens from `oc whoami -t`
- **JWT**: Standard JSON Web Tokens with 3 base64 parts
- **GitHub**: Tokens starting with `ghp_`, `gho_`, `ghu_`, or `ghs_`

### Required Permissions

Your user account must have these Kubernetes RBAC permissions in the target project/namespace:

```yaml
# Minimum required permissions
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions"]
  verbs: ["get", "list", "create"]

- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get"]
```

### Common Permission Errors

**403 Forbidden**:
```bash
# Check your permissions
oc auth can-i create agenticsessions.vteam.ambient-code -n your-project
oc auth can-i list agenticsessions.vteam.ambient-code -n your-project
```

**401 Unauthorized**:
```bash
# Check token validity  
oc whoami  # Should return your username
oc whoami -t  # Should return a token starting with sha256~
```

**400 Bad Request - Project required**:
- Ensure `AMBIENT_PROJECT` environment variable is set
- Project must be a valid Kubernetes namespace name
- User must have access to the specified project

```bash
# Set environment variables
export AMBIENT_TOKEN="your-bearer-token"      # Required
export AMBIENT_PROJECT="your-project-name"    # Required  
export AMBIENT_API_URL="https://your-api.com" # Optional
```

**OpenShift Users:**
```bash
# Use your OpenShift token
export AMBIENT_TOKEN="$(oc whoami -t)"
export AMBIENT_PROJECT="$(oc project -q)"
```

## Core Operations

### Create Session

```go
createReq := &types.CreateSessionRequest{
    Task:  "Review this code for security issues",
    Model: "claude-3.5-sonnet", // Optional, uses platform default if omitted
    Repos: []types.RepoHTTP{
        {URL: "https://github.com/user/repo", Branch: "main"},
    },
}

resp, err := client.CreateSession(ctx, createReq)
```

### Get Session Details

```go
session, err := client.GetSession(ctx, "session-1234567")
if err != nil {
    log.Printf("Session error: %v", err)
}

fmt.Printf("Status: %s\n", session.Status)
if session.Status == types.StatusCompleted {
    fmt.Printf("Result: %s\n", session.Result)
}
```

### List Sessions

```go
listResp, err := client.ListSessions(ctx)
if err != nil {
    return err
}

for _, session := range listResp.Items {
    fmt.Printf("- %s (%s): %s\n", session.ID, session.Status, session.Task)
}
```

### Monitor Session Completion

```go
// Wait for session to complete
completed, err := client.WaitForCompletion(ctx, sessionID, 5*time.Second)
if err != nil {
    return fmt.Errorf("monitoring failed: %w", err)
}

if completed.Status == types.StatusCompleted {
    fmt.Printf("Success: %s\n", completed.Result)
} else {
    fmt.Printf("Failed: %s\n", completed.Error)
}
```

## Session Status Values

```go
const (
    StatusPending   = "pending"   // Session created, waiting to start
    StatusRunning   = "running"   // AI agent actively working
    StatusCompleted = "completed" // Task finished successfully
    StatusFailed    = "failed"    // Task failed with error
)
```

## Configuration Options

### Custom Timeout

```go
client := client.NewClientWithTimeout(apiURL, token, project, 60*time.Second)
```

### Error Handling

```go
session, err := client.GetSession(ctx, sessionID)
if err != nil {
    // Detailed error messages include HTTP status and API responses
    log.Printf("Failed: %v", err)
    // Example: "API error (404): session not found: session-xyz"
}
```

## Examples

See the `examples/` directory for complete working examples:

- **`main.go`** - Complete session lifecycle demonstration
- **`README.md`** - Detailed usage guide with troubleshooting

## API Reference

### Client Methods

```go
// Client creation
func NewClient(baseURL, token, project string) *Client
func NewClientWithTimeout(baseURL, token, project string, timeout time.Duration) *Client

// Session operations  
func (c *Client) CreateSession(ctx context.Context, req *CreateSessionRequest) (*CreateSessionResponse, error)
func (c *Client) GetSession(ctx context.Context, sessionID string) (*SessionResponse, error)
func (c *Client) ListSessions(ctx context.Context) (*SessionListResponse, error)
func (c *Client) WaitForCompletion(ctx context.Context, sessionID string, pollInterval time.Duration) (*SessionResponse, error)
```

### Types

```go
// Request types
type CreateSessionRequest struct {
    Task  string     `json:"task"`
    Model string     `json:"model,omitempty"`
    Repos []RepoHTTP `json:"repos,omitempty"`
}

type RepoHTTP struct {
    URL    string `json:"url"`
    Branch string `json:"branch,omitempty"`
}

// Response types
type SessionResponse struct {
    ID          string `json:"id"`
    Status      string `json:"status"`
    Task        string `json:"task"`
    Model       string `json:"model,omitempty"`
    CreatedAt   string `json:"createdAt"`
    CompletedAt string `json:"completedAt,omitempty"`
    Result      string `json:"result,omitempty"`
    Error       string `json:"error,omitempty"`
}

type SessionListResponse struct {
    Items []SessionResponse `json:"items"`
    Total int               `json:"total"`
}

type CreateSessionResponse struct {
    ID      string `json:"id"`
    Message string `json:"message"`
}

type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
}
```

## Architecture

### Design Principles

- **HTTP-First**: Pure REST API client with no Kubernetes dependencies
- **Minimal Dependencies**: Uses only Go standard library
- **Simple Integration**: Easy to embed in any Go application  
- **Type Safety**: Strongly-typed requests and responses with compile-time validation
- **Clear Separation**: Public SDK vs internal platform implementation

### HTTP vs Kubernetes

This SDK provides a **simplified HTTP interface** to the Ambient Platform:

| Aspect | HTTP SDK (This Package) | Internal Platform |
|--------|------------------------|-------------------|
| **API** | Simple REST endpoints (`/v1/sessions`) | Complex Kubernetes CRDs |
| **Auth** | Bearer token + project header | RBAC + service accounts |
| **Types** | Flat JSON structs | Full K8s metadata/spec/status |
| **Usage** | Any HTTP client, any environment | Kubernetes cluster access required |
| **Target** | External integrators, simple automation | Internal platform components |

### Internal vs Public

- **Backend Components**: Can use internal Kubernetes types for cluster operations
- **SDK Users**: Get simplified HTTP API without Kubernetes complexity  
- **Type Definitions**: Shared between internal and public usage where appropriate

## Migration from Kubernetes SDK

If migrating from a previous Kubernetes-based version:

### Before (Kubernetes)
```go
import "k8s.io/client-go/kubernetes"

client, err := sdk.NewClientFromKubeconfig("")
session := &types.AgenticSession{/* complex K8s structure */}
created, err := client.Sessions.Create(ctx, session)
```

### After (HTTP)
```go
import "github.com/ambient/platform-sdk/client"

client := client.NewClient(apiURL, token, project)
req := &types.CreateSessionRequest{Task: "...", Model: "..."}
resp, err := client.CreateSession(ctx, req)
```

## Troubleshooting

### Authentication Issues
```
❌ AMBIENT_TOKEN environment variable is required
```
**Solution**: Set your Bearer token: `export AMBIENT_TOKEN="your-token"`

### Project Header Missing  
```
API error (400): Project required. Set X-Ambient-Project header
```
**Solution**: Set project name: `export AMBIENT_PROJECT="your-project"`

### Connection Errors
```
Failed to execute request: dial tcp: connection refused
```
**Solution**: Verify API endpoint and network connectivity

### Session Not Found
```
API error (404): session not found: session-xyz
```
**Solution**: Verify session ID and check if you have access to the project

## Testing

```bash
go test ./...
```

Run the complete example:
```bash
cd examples/
export AMBIENT_TOKEN="your-token"
export AMBIENT_PROJECT="your-project"
go run main.go
```

## OpenAPI Specification

This SDK is built to match the OpenAPI specification in `../openapi.yaml`. The specification defines:

- **Endpoints**: `/v1/sessions` (create, list, get)
- **Authentication**: Bearer token + X-Ambient-Project header
- **Request/Response**: JSON types matching this SDK
- **Error Handling**: Structured error responses with HTTP status codes

## Contributing

1. **SDK Changes**: Modify code in `client/` or `types/` directories
2. **API Changes**: Update `../openapi.yaml` specification first
3. **Examples**: Add working examples to `examples/` directory
4. **Testing**: Ensure all changes work with real API endpoints

For complete platform documentation, see the main [platform repository](https://github.com/ambient-code/platform).