# Ambient Platform Go SDK Examples

This directory contains examples demonstrating how to use the Ambient Platform Go SDK to interact with the platform via HTTP API.

## Overview

The Ambient Platform SDK provides a simple HTTP client for creating and managing AI agent sessions through the platform's public REST API. No Kubernetes dependencies are required - just HTTP and Bearer token authentication.

## Quick Start

### Prerequisites

1. **Ambient Platform**: Deployed and accessible via HTTP
2. **Bearer Token**: Valid authentication token for API access  
3. **Go 1.21+**: For running the examples

### Environment Setup

Set the required environment variables:

```bash
# Required: Your API authentication token
export AMBIENT_TOKEN="your-bearer-token-here"

# Required: Project name (Kubernetes namespace)
export AMBIENT_PROJECT="your-project-name"

# Optional: API endpoint (defaults to working public API URL)
export AMBIENT_API_URL="https://your-platform.example.com"

# Optional: Monitor session completion (defaults to false)
export MONITOR_SESSION="true"
```

### Running the Examples

```bash
# Run the HTTP client example (recommended)
go run main.go

# Legacy Kubernetes example (deprecated - requires k8s access)
go run kubernetes_main.go
```

The HTTP example demonstrates:
- ✅ Creating a new agentic session
- ✅ Retrieving session details by ID  
- ✅ Listing all accessible sessions
- ✅ Optional session completion monitoring

## Example Output

```
🌐 Ambient Platform SDK - HTTP Client Example
============================================
✓ Created client for API: https://public-api-route-mturansk.apps.rosa.xezue-pjejw-oy9.ag90.p3.openshiftapps.com
✓ Using project: mturansk

📝 Creating new session...
✓ Created session: session-1771013589

🔍 Getting session details...
   ID: session-1771013589
   Status: pending
   Task: Analyze the repository structure and provide a brief summary...
   Model: claude-3.5-sonnet
   Created: 2026-02-13T20:13:09Z

📋 Listing all sessions...
✓ Found 2 sessions (total: 2)
  1. session-1771013589 (pending) - Analyze the repository structure and...
  2. example-session (pending) - 

✅ HTTP Client demonstration complete!
```

## Key Features

### Simple HTTP Client
- Pure REST API integration with no Kubernetes dependencies
- Bearer token authentication with project routing
- JSON request/response handling
- Proper error handling with detailed messages

### Session Management
- **Create**: Submit new tasks to AI agents
- **Retrieve**: Get session details and status
- **List**: Browse all accessible sessions  
- **Monitor**: Poll for session completion

### Type Safety
- Strongly-typed request and response structures
- Compile-time validation of API payloads
- Clear error messages for debugging

## API Reference

### Client Creation

```go
import "github.com/ambient/platform-sdk/client"

// Basic client
client := client.NewClient(apiURL, token, project)

// Client with custom timeout
client := client.NewClientWithTimeout(apiURL, token, project, 60*time.Second)
```

### Session Operations

```go
// Create session
createReq := &types.CreateSessionRequest{
    Task:  "Analyze this repository",
    Model: "claude-3.5-sonnet",
    Repos: []types.RepoHTTP{{URL: "https://github.com/user/repo"}},
}
resp, err := client.CreateSession(ctx, createReq)

// Get session
session, err := client.GetSession(ctx, sessionID)

// List sessions  
list, err := client.ListSessions(ctx)

// Wait for completion
completed, err := client.WaitForCompletion(ctx, sessionID, 5*time.Second)
```

## Working Configuration (Tested)

The following configuration has been tested and verified working:

```bash
# Tested working configuration
export AMBIENT_API_URL="https://public-api-route-mturansk.apps.rosa.xezue-pjejw-oy9.ag90.p3.openshiftapps.com"
export AMBIENT_TOKEN="$(oc whoami -t)"  # OpenShift token
export AMBIENT_PROJECT="mturansk"       # Valid namespace
export MONITOR_SESSION="true"           # Enable completion monitoring

go run main.go
```

## Error Handling

The SDK provides detailed error information:

```go
session, err := client.GetSession(ctx, "invalid-id")
if err != nil {
    // Errors include HTTP status codes and API error messages
    log.Printf("Failed to get session: %v", err)
    // Example: "API error (404): session not found: invalid-id"
}
```

## Troubleshooting

### Authentication Issues
```
❌ AMBIENT_TOKEN environment variable is required
```
**Solution**: Set your Bearer token in the `AMBIENT_TOKEN` environment variable.

### Project Header Missing
```
API error (400): Project required. Set X-Ambient-Project header...
```
**Solution**: Set the `AMBIENT_PROJECT` environment variable to a valid namespace.

### Connection Errors
```
Failed to execute request: dial tcp: connection refused
```
**Solution**: Verify the API endpoint is correct and accessible. Check `AMBIENT_API_URL`.

### API Errors
```
API error (401): Unauthorized - Invalid or expired token
```
**Solution**: Verify your Bearer token is valid and has appropriate permissions.

### Session Not Found
```
API error (404): session not found: session-xyz
```
**Solution**: Verify the session ID exists and you have access to it.

## Migration from Legacy Example

If you're migrating from the previous Kubernetes-based example:

### Old (Kubernetes)
```go
import "k8s.io/client-go/kubernetes"

client, err := client.NewClientFromKubeconfig("")
session, err := client.Sessions.Create(ctx, agenticSession)
```

### New (HTTP)
```go
import "github.com/ambient/platform-sdk/client"

client := client.NewClient(apiURL, token, project)
resp, err := client.CreateSession(ctx, sessionRequest)
```

## Architecture Notes

### Design Principles
- **HTTP-First**: Pure REST API client with no Kubernetes dependencies
- **Minimal Dependencies**: Uses only Go standard library
- **Simple Integration**: Easy to embed in any Go application
- **Clear Separation**: Public SDK vs internal platform implementation

### Internal vs Public
- **Backend**: Can continue using strongly-typed Kubernetes structs for internal operations
- **SDK**: Exposes only HTTP API functionality for external integrators
- **Types**: Shared type definitions support both internal and public usage

## Session Monitoring

The SDK supports three approaches to session completion monitoring:

1. **Simple Polling** (Implemented):
   ```go
   session, err := client.WaitForCompletion(ctx, sessionID, 5*time.Second)
   ```

2. **Manual Status Checking**:
   ```go
   for {
       session, err := client.GetSession(ctx, sessionID)
       if session.Status == types.StatusCompleted || session.Status == types.StatusFailed {
           break
       }
       time.Sleep(5 * time.Second)
   }
   ```

3. **Future: WebSocket/SSE** (Planned):
   - Real-time status updates
   - Progress streaming
   - Event notifications

## Next Steps

1. **Explore Advanced Features**: Session monitoring, batch operations
2. **Integration Testing**: Test with your actual platform deployment  
3. **Error Handling**: Implement retry logic and circuit breakers
4. **Observability**: Add logging and metrics for production usage

For complete API documentation, see the [public API reference](../../public-api/README.md).