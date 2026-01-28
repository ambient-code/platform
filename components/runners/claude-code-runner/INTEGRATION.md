# SDK Configuration UI Integration Guide

## Overview

The SDK Configuration UI allows users to customize Claude Agent SDK behavior through a web interface. This guide explains how to integrate the UI with the existing runner.

## Components

### Frontend (`SDKConfigurationPanel.tsx`)
- React component for workspace settings page
- Tabs for model config, tools, MCP servers, system prompts
- JSON preview mode for developers
- Validation and testing features

### Backend (`sdk_config.go`)
- Go API handlers for configuration CRUD
- Validation of SDK options
- MCP server testing endpoint
- Per-user, per-project configuration storage

### Runner (`sdk_config.py`)
- Python module to load configuration from API
- Apply configuration to `ClaudeAgentOptions`
- Merge with existing MCP servers

## Integration Steps

### 1. Add Backend Routes

In `components/backend/cmd/server/main.go`:

```go
import "your-project/pkg/handlers"

// Add routes
api.GET("/projects/:project/sdk/configuration", handlers.GetSDKConfiguration)
api.PUT("/projects/:project/sdk/configuration", handlers.UpdateSDKConfiguration)
api.POST("/projects/:project/sdk/mcp/test/:server", handlers.TestMCPServer)
api.GET("/projects/:project/sdk/configuration/session/:session", handlers.GetSDKConfigForSession)
```

### 2. Create Database Migration

Create migration file `migrations/XXX_add_sdk_configuration.up.sql`:

```sql
CREATE TABLE IF NOT EXISTS sdk_config_models (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    project_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    config_json TEXT NOT NULL,
    UNIQUE(project_name, user_id)
);

CREATE INDEX idx_sdk_config_project_user ON sdk_config_models(project_name, user_id);
```

### 3. Add Frontend Route

In `components/frontend/src/App.tsx` or routing file:

```typescript
import { SDKConfigurationPanel } from '@/components/settings/SDKConfigurationPanel';

// Add to workspace settings page
<Route path="/workspace/:project/settings/sdk" element={<SDKConfigurationPanel />} />
```

### 4. Integrate with Runner

In `components/runners/claude-code-runner/adapter.py`, modify `_run_claude_agent_sdk`:

```python
from sdk_config import load_and_apply_sdk_config

async def _run_claude_agent_sdk(self, prompt: str, thread_id: str, run_id: str):
    # ... existing code ...

    # Configure SDK options (existing code)
    options = ClaudeAgentOptions(
        cwd=cwd_path,
        permission_mode="acceptEdits",
        allowed_tools=allowed_tools,
        mcp_servers=mcp_servers,
        setting_sources=["project"],
        system_prompt=system_prompt_config,
        include_partial_messages=True,
    )

    # NEW: Load and apply user's SDK configuration
    await load_and_apply_sdk_config(
        options=options,
        session_id=self.context.session_id,
        project_name=self.context.get_env('PROJECT_NAME', '')
    )

    # ... continue with existing client creation ...
```

### 5. Update Requirements

Add to `components/runners/claude-code-runner/pyproject.toml`:

```toml
# No new dependencies required - uses stdlib urllib
```

## Usage

### User Workflow

1. Navigate to Workspace Settings → SDK Configuration
2. Configure model, tools, MCP servers, prompts
3. Click "Save Configuration"
4. Configuration applies to new sessions automatically

### Developer Workflow

1. Use JSON Preview mode to see raw ClaudeAgentOptions
2. Test MCP servers before saving
3. Validate configuration before save
4. Copy JSON for documentation/sharing

## Configuration Precedence

Configuration is applied in this order (later overrides earlier):

1. **Default configuration** (hardcoded in runner)
2. **CLAUDE.md** (if `settingSources: ["project"]`)
3. **Workflow ambient.json** (workspace context)
4. **User SDK configuration** (from UI)
5. **Environment variables** (LLM_MODEL, LLM_MAX_TOKENS, etc.)

## API Endpoints

### GET `/api/projects/:project/sdk/configuration`
Get current user's SDK configuration for project.

**Response:**
```json
{
  "model": "claude-sonnet-4-5@20250929",
  "maxTokens": 4096,
  "temperature": 1.0,
  "permissionMode": "acceptEdits",
  "allowedTools": ["Read", "Write", "Bash"],
  "includePartialMessages": true,
  "continueConversation": true,
  "systemPrompt": "",
  "mcpServers": {}
}
```

### PUT `/api/projects/:project/sdk/configuration`
Save SDK configuration.

**Request:** Same format as GET response

**Response:**
```json
{
  "message": "Configuration saved successfully"
}
```

### POST `/api/projects/:project/sdk/mcp/test/:server`
Test MCP server connectivity.

**Request:**
```json
{
  "command": "mcp-server-webfetch",
  "args": [],
  "env": {}
}
```

**Response:**
```json
{
  "server": "webfetch",
  "connected": true
}
```

### GET `/api/projects/:project/sdk/configuration/session/:session`
Get SDK configuration for specific session (used by runner).

**Response:** Same format as GET configuration

## Security Considerations

### Authentication
- All endpoints require authentication
- Configuration is per-user, per-project
- Session owner's configuration is used

### Validation
- All options validated server-side
- Invalid configurations rejected
- MCP server commands sanitized

### Secrets
- Environment variables not stored in configuration
- MCP server env vars stored encrypted (TODO)
- Credentials injected at runtime from secrets

## Testing

### Unit Tests

**Backend:**
```go
func TestValidateSDKConfig(t *testing.T) {
    config := SDKConfiguration{
        Model: "invalid-model",
        MaxTokens: 300000, // Too high
    }
    err := validateSDKConfig(&config)
    assert.Error(t, err)
}
```

**Runner:**
```python
async def test_load_configuration():
    loader = SDKConfigLoader("session-id", "project-name")
    config = await loader.load_configuration()
    assert config is not None
```

### Integration Tests

1. Save configuration via UI
2. Create new session
3. Verify SDK uses saved configuration
4. Check logs for "Applied model: ..."

## Troubleshooting

### Configuration not applying

**Check:**
1. Backend API accessible from runner pod
2. BOT_TOKEN set correctly
3. Logs show "Loading SDK configuration from: ..."
4. Database has configuration row

**Debug:**
```bash
# Check database
SELECT project_name, user_id, config_json FROM sdk_config_models;

# Check runner logs
kubectl logs -f deployment/claude-runner | grep "SDK configuration"
```

### MCP servers not connecting

**Check:**
1. MCP server enabled in configuration
2. Command and args correct
3. Test endpoint returns connected=true
4. Logs show "Added MCP server from config: ..."

## Future Enhancements

### Phase 2
- [ ] Configuration templates (presets)
- [ ] Team-wide default configurations
- [ ] Configuration versioning/history
- [ ] Import/export configurations
- [ ] Bulk MCP server management

### Phase 3
- [ ] Real-time MCP server health monitoring
- [ ] Configuration recommendations based on usage
- [ ] Cost estimation for model/token settings
- [ ] A/B testing different configurations

## References

- [SDK-REFERENCE.md](../../docs/claude-agent-sdk/SDK-REFERENCE.md) - Complete SDK options reference
- [adapter.py](adapter.py) - Runner implementation
- Backend API documentation
