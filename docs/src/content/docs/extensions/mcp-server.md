---
title: MCP Server
---

import { Badge } from '@astrojs/starlight/components';

<Badge text="Stable" variant="success" />

The [`mcp-acp`](https://github.com/ambient-code/mcp) server is a Model Context Protocol server that lets Claude manage Ambient Code Platform sessions programmatically.

## Capabilities

### Session management

- Create sessions with custom prompts, repos, model selection, and timeout
- Create sessions from predefined templates (triage, bugfix, feature, exploration)
- Restart, clone, delete, and update sessions
- Dry-run mode for previewing destructive operations

### Observability

- Retrieve container logs for a session
- Get conversation transcripts in JSON or Markdown format
- View usage statistics (tokens, duration, tool calls)

### Labels and bulk operations

- Add and remove labels for organizing and filtering sessions
- Filter sessions by label selectors
- Bulk operations on up to 3 sessions at a time (delete, stop, restart, label)

### Cluster management

- List configured clusters and check authentication status
- Switch between clusters
- Authenticate with Bearer tokens

## Safety

- **Dry-run mode** -- All mutating operations support `dry_run` for safe preview before executing.
- **Bulk operation limits** -- A maximum of 3 sessions can be affected per bulk operation.
- **Label validation** -- Labels must be 1-63 alphanumeric characters, dashes, dots, or underscores.

## Requirements

- Python 3.10+
- Bearer token for the ACP public-api gateway
- Access to an ACP cluster

## Installation

```bash
# From wheel
pip install dist/mcp_acp-*.whl

# From source
git clone https://github.com/ambient-code/mcp
cd mcp
uv pip install -e ".[dev]"
```

## Configuration

The primary configuration method uses a YAML config file at `~/.config/acp/clusters.yaml`.

### Cluster config file

Create `~/.config/acp/clusters.yaml`:

```yaml
clusters:
  my-staging:
    server: https://public-api-ambient.apps.my-staging.example.com
    token: your-bearer-token-here
    description: "Staging Environment"
    default_project: my-workspace

  my-prod:
    server: https://public-api-ambient.apps.my-prod.example.com
    token: your-bearer-token-here
    description: "Production"
    default_project: my-workspace

default_cluster: my-staging
```

Secure the file:

```bash
chmod 600 ~/.config/acp/clusters.yaml
```

Alternatively, set the `ACP_TOKEN` environment variable to provide a token without editing the config file.

### Claude Desktop

Add to your Claude Desktop configuration file (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "acp": {
      "command": "mcp-acp",
      "args": [],
      "env": {
        "ACP_CLUSTER_CONFIG": "${HOME}/.config/acp/clusters.yaml"
      }
    }
  }
}
```

Using uvx (zero-install):

```json
{
  "mcpServers": {
    "acp": {
      "command": "uvx",
      "args": ["mcp-acp"]
    }
  }
}
```

### Claude Code CLI

```bash
claude mcp add mcp-acp -t stdio mcp-acp
```

## Multi-cluster support

Define multiple clusters in `~/.config/acp/clusters.yaml` and switch between them using the `acp_switch_cluster` tool or by changing the `default_cluster` value. The server supports listing clusters, checking authentication status, and authenticating to new clusters at runtime.
