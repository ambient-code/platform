---
title: "CLI Reference"
---

`acpctl` is the command-line interface for the Ambient Code Platform. You can use it to create and manage sessions, switch between projects, and automate workflows from your terminal.

## Install acpctl

Download the latest binary from your organization's ACP deployment or build it from source:

```bash
cd components/ambient-cli
go build -o acpctl ./cmd/acpctl
```

Move the binary to a directory in your `PATH`:

```bash
sudo mv acpctl /usr/local/bin/
```

Verify the installation:

```bash
acpctl version
```

## Authentication

### Log in

Authenticate with the ACP API server by providing an access token. The token is saved to your local configuration file.

```bash
acpctl login --token <your-token>
```

You can also specify the API server URL and a default project in the same command:

```bash
acpctl login --token <your-token> --url https://acp.example.com --project my-project
```

If the server uses a self-signed certificate, add the `--insecure-skip-tls-verify` flag:

```bash
acpctl login --token <your-token> --url https://acp.example.com --insecure-skip-tls-verify
```

You can pass the server URL as a positional argument instead of using `--url`:

```bash
acpctl login https://acp.example.com --token <your-token>
```

| Flag | Description |
|---|---|
| `--token` | Access token (required) |
| `--url` | API server URL (default: `http://localhost:8000`) |
| `--project` | Default project name |
| `--insecure-skip-tls-verify` | Skip TLS certificate verification |

### Log out

Remove saved credentials from the configuration file:

```bash
acpctl logout
```

### Check your identity

Display the current user, token expiration, API URL, and active project:

```bash
acpctl whoami
```

Example output for a JWT token:

```
User:       jane.doe
Email:      jane@example.com
Expires:    2026-04-01T12:00:00Z
API URL:    https://acp.example.com
Project:    my-project
```

## Configuration

`acpctl` stores configuration in a JSON file. The default location depends on your operating system: `~/.config/ambient/config.json` on Linux, `~/Library/Application Support/ambient/config.json` on macOS. Override the location with the `AMBIENT_CONFIG` environment variable.

### Get a configuration value

```bash
acpctl config get <key>
```

Valid keys: `api_url`, `project`, `pager`, `access_token` (redacted in output).

```bash
acpctl config get api_url
# https://acp.example.com
```

### Set a configuration value

```bash
acpctl config set <key> <value>
```

Valid keys: `api_url`, `project`, `pager`.

```bash
acpctl config set api_url https://acp.example.com
acpctl config set project my-project
```

### Configuration keys

| Key | Description | Default |
|---|---|---|
| `api_url` | ACP API server URL | `http://localhost:8000` |
| `project` | Active project name | (none) |
| `pager` | Pager program for output | (none) |
| `access_token` | Authentication token (set via `login`) | (none) |
| `request_timeout` | Request timeout in seconds | `30` |
| `polling_interval` | Watch polling interval in seconds | `2` |
| `insecure_tls_verify` | Skip TLS verification | `false` |

## Project context

Most commands operate within a project context. Set the active project before creating or managing sessions.

### Set the active project

```bash
acpctl project set my-project
```

Or use the shorthand:

```bash
acpctl project my-project
```

The CLI validates that the project exists on the server before saving it to your configuration.

### View the current project

```bash
acpctl project current
```

Or use the shorthand (no arguments):

```bash
acpctl project
```

### List all projects

```bash
acpctl project list
```

## Manage projects

### Create a project

```bash
acpctl create project --name my-project
```

Optionally set a display name and description:

```bash
acpctl create project --name my-project --display-name "My Project" --description "Team workspace"
```

| Flag | Description |
|---|---|
| `--name` | Project name (required). Must be lowercase alphanumeric with hyphens, 63 characters max. |
| `--display-name` | Human-readable display name |
| `--description` | Project description |
| `-o, --output` | Output format: `json` |

### List projects

```bash
acpctl get projects
```

Get a specific project:

```bash
acpctl get project my-project
```

### View project details

```bash
acpctl describe project my-project
```

Output is JSON with the full project resource.

### Delete a project

```bash
acpctl delete project my-project
```

Add `-y` to skip the confirmation prompt:

```bash
acpctl delete project my-project -y
```

## Manage sessions

### Create a session

A project context must be set before you create a session.

```bash
acpctl create session --name fix-login-bug --prompt "Fix the null pointer in the login handler" --repo-url https://github.com/org/repo
```

| Flag | Description |
|---|---|
| `--name` | Session name (required) |
| `--prompt` | Task prompt for the agent |
| `--repo-url` | Repository URL to clone |
| `--model` | LLM model to use |
| `--max-tokens` | Maximum output tokens |
| `--temperature` | LLM temperature (0.0-1.0) |
| `--timeout` | Session timeout in seconds |
| `-o, --output` | Output format: `json` |

### List sessions

```bash
acpctl get sessions
```

Get a specific session:

```bash
acpctl get session <session-id>
```

Output in JSON format:

```bash
acpctl get sessions -o json
```

Use wide output for additional columns:

```bash
acpctl get sessions -o wide
```

| Flag | Description |
|---|---|
| `-o, --output` | Output format: `json` or `wide` |
| `--limit` | Maximum number of items to return (default: `100`) |
| `-w, --watch` | Watch for real-time session changes |
| `--watch-timeout` | Timeout for watch mode (default: `30m`, e.g. `1h`, `10m`) |

### Watch sessions

Monitor session changes in real time:

```bash
acpctl get sessions -w
```

The watch uses gRPC streaming when available. If the server does not support streaming, the CLI falls back to polling. Press `Ctrl+C` to stop watching.

### View session details

```bash
acpctl describe session <session-id>
```

Output is JSON with the full session resource.

### Start a session

Start a session that is in a stopped or pending state:

```bash
acpctl start <session-id>
```

### Stop a session

Stop a running session:

```bash
acpctl stop <session-id>
```

### Delete a session

```bash
acpctl delete session <session-id>
```

Add `-y` to skip the confirmation prompt:

```bash
acpctl delete session <session-id> -y
```

## Other resources

You can also manage project settings and users through the `get` and `describe` commands.

### Project settings

```bash
acpctl get project-settings
acpctl describe project-settings <id>
acpctl delete project-settings <id>
```

Resource aliases: `projectsettings`, `ps`.

### Users

```bash
acpctl get users
acpctl describe user <username>
```

Resource aliases: `user`, `usr`.

## Global flags

| Flag | Description |
|---|---|
| `--insecure-skip-tls-verify` | Skip TLS certificate verification for all commands |
| `--version` | Print the version and exit |

## Environment variables

Environment variables override values in the configuration file.

| Variable | Description | Overrides config key |
|---|---|---|
| `AMBIENT_CONFIG` | Path to the configuration file | (file location) |
| `AMBIENT_API_URL` | API server URL | `api_url` |
| `AMBIENT_TOKEN` | Access token | `access_token` |
| `AMBIENT_PROJECT` | Active project name | `project` |
| `AMBIENT_REQUEST_TIMEOUT` | Request timeout in seconds | `request_timeout` |
| `AMBIENT_POLLING_INTERVAL` | Watch polling interval in seconds | `polling_interval` |

## Shell completion

Generate completion scripts for your shell:

```bash
# Bash
acpctl completion bash > /etc/bash_completion.d/acpctl

# Zsh
acpctl completion zsh > "${fpath[1]}/_acpctl"

# Fish
acpctl completion fish > ~/.config/fish/completions/acpctl.fish

# PowerShell
acpctl completion powershell > acpctl.ps1
```

Restart your shell or source the completion file to enable tab completion.
