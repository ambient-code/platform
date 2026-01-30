# Claude Agent SDK Configuration UI

**Status:** Implementation Ready
**Created:** 2026-01-27

Complete web UI for configuring Claude Agent SDK options through the Ambient platform.

## What Was Built

### 1. Frontend Component (`SDKConfigurationPanel.tsx`)

Comprehensive React UI with:

**Features:**
- Model selection (Opus 4.5, Sonnet 4.5, Haiku 4.5)
- Generation parameters (max tokens, temperature)
- Permission modes (acceptEdits, prompt, reject)
- Tool toggles (Read, Write, Bash, etc.)
- MCP server management (add, edit, test, remove)
- System prompt customization
- JSON preview mode for developers
- Real-time validation
- Configuration testing

**Technology:**
- React + TypeScript
- Shadcn/ui components
- Tabs for organized sections
- Form validation
- REST API integration

### 2. Backend API (`sdk_config.go`)

Go HTTP handlers with:

**Endpoints:**
- `GET /api/projects/:project/sdk/configuration` - Get user config
- `PUT /api/projects/:project/sdk/configuration` - Save config
- `POST /api/projects/:project/sdk/mcp/test/:server` - Test MCP connectivity
- `GET /api/projects/:project/sdk/configuration/session/:session` - Get config for runner

**Features:**
- Per-user, per-project configuration storage
- Server-side validation
- Default configuration handling
- Database persistence (PostgreSQL via GORM)

### 3. Runner Integration (`sdk_config.py`)

Python module to load and apply configuration:

**Features:**
- Fetch configuration from backend API
- Apply to `ClaudeAgentOptions`
- Merge with existing settings
- Handle MCP servers
- Graceful fallback to defaults

### 4. Integration Guide (`INTEGRATION.md`)

Complete implementation guide:

**Covers:**
- Backend route setup
- Database migration
- Frontend routing
- Runner integration
- Testing procedures
- Troubleshooting

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│ Frontend (React)                                             │
│ ┌────────────────────────────────────────────────────────┐   │
│ │ SDKConfigurationPanel                                  │   │
│ │ - Model & Limits Tab                                   │   │
│ │ - Tools Tab                                            │   │
│ │ - MCP Servers Tab                                      │   │
│ │ - System Prompts Tab                                   │   │
│ │ - JSON Preview Mode                                    │   │
│ └────────────────────────────────────────────────────────┘   │
└───────────────────────────────┬──────────────────────────────┘
                                │ REST API
┌───────────────────────────────▼──────────────────────────────┐
│ Backend (Go)                                                 │
│ ┌────────────────────────────────────────────────────────┐   │
│ │ SDK Config Handlers                                    │   │
│ │ - GetSDKConfiguration                                  │   │
│ │ - UpdateSDKConfiguration                               │   │
│ │ - TestMCPServer                                        │   │
│ │ - GetSDKConfigForSession                               │   │
│ └────────────────────────────────────────────────────────┘   │
└───────────────────────────────┬──────────────────────────────┘
                                │ Database
┌───────────────────────────────▼──────────────────────────────┐
│ PostgreSQL                                                   │
│ ┌────────────────────────────────────────────────────────┐   │
│ │ sdk_config_models                                      │   │
│ │ - project_name                                         │   │
│ │ - user_id                                              │   │
│ │ - config_json (TEXT)                                   │   │
│ └────────────────────────────────────────────────────────┘   │
└───────────────────────────────┬──────────────────────────────┘
                                │ Fetch at runtime
┌───────────────────────────────▼──────────────────────────────┐
│ Python Runner                                                │
│ ┌────────────────────────────────────────────────────────┐   │
│ │ sdk_config.py                                          │   │
│ │ - SDKConfigLoader                                      │   │
│ │ - load_and_apply_sdk_config()                          │   │
│ │                                                        │   │
│ │ adapter.py (integration point)                         │   │
│ │ - Load config before client creation                   │   │
│ │ - Apply to ClaudeAgentOptions                          │   │
│ └────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

## Configuration Options

### Model & Limits
- **Model:** Opus 4.5, Sonnet 4.5, Haiku 4.5
- **Max Tokens:** 1-200,000
- **Temperature:** 0.0-1.0
- **Permission Mode:** acceptEdits, prompt, reject
- **Streaming:** Enable/disable partial messages
- **Continuation:** Auto-resume conversations

### Tools
Toggle any combination of:
- Read, Write, Edit, MultiEdit
- Bash
- Glob, Grep
- WebSearch, WebFetch
- NotebookEdit

### MCP Servers
For each server:
- **Command:** Executable path
- **Arguments:** JSON array
- **Environment:** JSON object
- **Enabled:** Toggle on/off
- **Test:** Connectivity check

### System Prompts
- Custom instructions injected into every session
- Merged with workspace context
- Markdown supported

## Usage

### For End Users

1. Navigate to **Workspace Settings** → **SDK Configuration**
2. Configure options in organized tabs
3. Test MCP servers before saving
4. Click **Save Configuration**
5. New sessions use saved configuration

### For Developers

1. Use **JSON Preview** mode to see raw config
2. Copy configuration for documentation
3. Test configurations before deployment
4. Share configurations across team

## Implementation Checklist

### Phase 1: Backend Setup

- [ ] Add route handlers to `main.go`
- [ ] Create database migration
- [ ] Run migration on dev/staging/prod
- [ ] Test API endpoints with curl/Postman

### Phase 2: Frontend Integration

- [ ] Add `SDKConfigurationPanel.tsx` to settings page
- [ ] Create route in App.tsx
- [ ] Test UI in development
- [ ] Verify API integration

### Phase 3: Runner Integration

- [ ] Add `sdk_config.py` to runner
- [ ] Modify `adapter.py` to load config
- [ ] Set `BACKEND_API_URL` and `BOT_TOKEN` env vars
- [ ] Test configuration application

### Phase 4: Testing

- [ ] Unit tests (backend validation)
- [ ] Integration tests (end-to-end)
- [ ] Load tests (concurrent configs)
- [ ] User acceptance testing

### Phase 5: Documentation

- [ ] Update platform docs
- [ ] Create user guide
- [ ] Record demo video
- [ ] Update CHANGELOG

## Testing

### Manual Testing

**Save Configuration:**
```bash
curl -X PUT http://localhost:8080/api/projects/test/sdk/configuration \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "claude-sonnet-4-5@20250929",
    "maxTokens": 8192,
    "temperature": 0.7,
    "permissionMode": "acceptEdits",
    "allowedTools": ["Read", "Write", "Bash"],
    "includePartialMessages": true,
    "continueConversation": true,
    "systemPrompt": "Be concise and direct.",
    "mcpServers": {}
  }'
```

**Get Configuration:**
```bash
curl http://localhost:8080/api/projects/test/sdk/configuration \
  -H "Authorization: Bearer $TOKEN"
```

**Test MCP Server:**
```bash
curl -X POST http://localhost:8080/api/projects/test/sdk/mcp/test/webfetch \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "command": "mcp-server-webfetch",
    "args": [],
    "enabled": true
  }'
```

### Verify Runner Integration

1. Save configuration via UI
2. Create new session
3. Check runner logs:
   ```
   Loading SDK configuration from: http://backend/api/...
   Loaded SDK configuration: model=claude-sonnet-4-5, tools=3
   Applied model: claude-sonnet-4-5@20250929
   Applied max_tokens: 8192
   Applied temperature: 0.7
   ```
4. Verify session uses saved model/tools

## Configuration Precedence

Settings are applied in order (later overrides earlier):

1. **Hardcoded defaults** in runner
2. **CLAUDE.md** (if `settingSources: ["project"]`)
3. **Workflow ambient.json** (system prompt)
4. **User SDK configuration** (from UI) ← This
5. **Environment variables** (LLM_MODEL, etc.)

## Security

### Authentication
- All endpoints require valid JWT token
- Configuration is per-user, per-project
- Users can only access their own configs

### Validation
- Server-side validation of all options
- Tool names validated against whitelist
- Model names validated against known models
- Numeric bounds enforced (tokens, temperature)

### Secrets
- MCP server environment variables stored in DB
- TODO: Encrypt sensitive env vars
- Credentials injected at runtime from K8s secrets

## Database Schema

```sql
CREATE TABLE sdk_config_models (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    project_name VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    config_json TEXT NOT NULL,
    UNIQUE(project_name, user_id)
);

CREATE INDEX idx_sdk_config_project_user
  ON sdk_config_models(project_name, user_id);
```

## File Locations

```
components/
├── frontend/src/components/settings/
│   └── SDKConfigurationPanel.tsx          # React UI component
│
├── backend/pkg/handlers/
│   └── sdk_config.go                      # Go API handlers
│
└── runners/claude-code-runner/
    ├── sdk_config.py                      # Python config loader
    └── INTEGRATION.md                     # Implementation guide

docs/
└── SDK-CONFIGURATION-UI.md                # This file
```

## Future Enhancements

### Phase 2 (Short-term)
- Configuration templates/presets
- Team-wide defaults (admin setting)
- Configuration history/versioning
- Import/export configurations

### Phase 3 (Medium-term)
- Real-time MCP health monitoring
- Cost estimation for token limits
- Usage analytics per configuration
- Recommendations based on patterns

### Phase 4 (Long-term)
- A/B testing configurations
- Auto-tune based on performance
- Configuration marketplace (share configs)
- Advanced permission controls

## Troubleshooting

### Configuration not applying

**Symptom:** Sessions use default config instead of saved

**Check:**
1. Runner can reach backend API (`BACKEND_API_URL`)
2. `BOT_TOKEN` environment variable set
3. Database has configuration row
4. Logs show "Loading SDK configuration"

**Debug:**
```bash
# Check database
kubectl exec -it postgres-pod -- psql -U user -d db \
  -c "SELECT * FROM sdk_config_models;"

# Check runner logs
kubectl logs -f deployment/claude-runner | grep "SDK configuration"
```

### MCP servers not connecting

**Symptom:** Test shows "Failed" or runtime errors

**Check:**
1. MCP server command is correct
2. MCP server installed in runner image
3. Arguments array valid JSON
4. Environment variables set correctly

**Debug:**
```bash
# Test MCP server manually in runner pod
kubectl exec -it claude-runner-pod -- mcp-server-webfetch
```

### Validation errors

**Symptom:** Cannot save configuration

**Solutions:**
- Max tokens: Must be 1-200,000
- Temperature: Must be 0.0-1.0
- At least one tool must be enabled
- Model must be valid (check MODELS list)
- MCP server commands cannot be empty

## Support

**For questions:**
- Review [SDK-REFERENCE.md](docs/claude-agent-sdk/SDK-REFERENCE.md)
- Check [INTEGRATION.md](components/runners/claude-code-runner/INTEGRATION.md)
- Consult Amber (SDK expert skill)

**For bugs:**
- Check logs (frontend console, backend, runner)
- Verify database state
- Test API endpoints directly

## License

Internal platform feature for Ambient.
