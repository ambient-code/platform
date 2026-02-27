# acpctl

Command-line interface for the Ambient Code Platform API server. Follows the `oc`/`kubectl` verb-noun pattern (`acpctl get sessions`).

## Build

```bash
make build
```

This produces an `acpctl` binary in the current directory with embedded version info.

```bash
./acpctl version
# acpctl v0.0.25-16-g88393d5 (commit: 88393d5, built: 2026-02-25T03:22:58Z)
```

## Quick Start

### 1. Log in

```bash
# With a token and API server URL
./acpctl login --token <your-token> --url http://localhost:8000 --project myproject

# Verify
./acpctl whoami
```

### 2. Configure defaults

```bash
# Set or change the default project
./acpctl config set project myproject

# View current settings
./acpctl config get api_url
./acpctl config get project
```

### 3. List resources

```bash
# List sessions (table format)
./acpctl get sessions

# List projects
./acpctl get projects

# JSON output
./acpctl get sessions -o json

# Single resource by ID
./acpctl get session <session-id>
```

### 4. Create resources

```bash
# Create a project
./acpctl create project --name my-project --display-name "My Project"

# Create a session
./acpctl create session --name fix-bug-123 \
  --prompt "Fix the null pointer in handler.go" \
  --repo-url https://github.com/org/repo \
  --model sonnet

# Create with all options
./acpctl create session --name refactor-auth \
  --prompt "Refactor the auth middleware" \
  --model sonnet \
  --max-tokens 4000 \
  --temperature 0.7 \
  --timeout 3600
```

### 5. Session lifecycle

```bash
# Start a session
./acpctl start <session-id>

# Stop a session
./acpctl stop <session-id>
```

### 6. Inspect resources

```bash
# Full JSON detail of a session
./acpctl describe session <session-id>

# Full JSON detail of a project
./acpctl describe project <project-id>
```

### 7. Delete resources

```bash
./acpctl delete project <project-id>
./acpctl delete project-settings <id>
```

### 8. Log out

```bash
./acpctl logout
```

## Try It Now (No Server Required)

These commands work without a running API server:

```bash
make build

# Version and help
./acpctl version
./acpctl --help
./acpctl get --help
./acpctl create --help

# Login and config flow
./acpctl login --token test-token --url http://localhost:8000 --project demo
./acpctl whoami
./acpctl config get api_url
./acpctl config get project
./acpctl config set project other-project
./acpctl config get project

# Shell completion
./acpctl completion bash
./acpctl completion zsh

# Logout
./acpctl logout
./acpctl whoami  # errors: "not logged in"
```

## Configuration

Config is stored at `~/.config/ambient/config.json` (XDG default). Override with:

```bash
export AMBIENT_CONFIG=/path/to/config.json
```

Environment variables also work (override config file values):

| Variable | Description |
|---|---|
| `AMBIENT_TOKEN` | Bearer token |
| `AMBIENT_PROJECT` | Target project |
| `AMBIENT_API_URL` | API server URL |
| `AMBIENT_CONFIG` | Config file path |

## Makefile Targets

| Target | Description |
|---|---|
| `make build` | Build binary with version info |
| `make clean` | Remove binary |
| `make fmt` | Format code |
| `make vet` | Run go vet |
| `make lint` | Format + vet |
| `make test` | Run tests |

## Dependencies

- [Go SDK](../acpctl-sdk/go-sdk/) via `replace` directive — zero-dep HTTP client for the Ambient API
- [cobra](https://github.com/spf13/cobra) — command framework
- [golang-jwt](https://github.com/golang-jwt/jwt) — token introspection for `whoami`
- [x/term](https://pkg.go.dev/golang.org/x/term) — terminal detection for table output
