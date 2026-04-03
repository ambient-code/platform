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
acpctl login <api-url> --token <your-token>

# Skip TLS verification (e.g. local Kind cluster)
acpctl login <api-url> --token <your-token> --insecure-skip-tls-verify

# Use RH SSO
acpctl login --use-auth-code --url https://ambient-api-server-ambient-code--ambient-s0.apps.int.spoke.dev.us-east-1.aws.paas.redhat.com

# Verify
acpctl whoami
# User: service-account-bob
# Project: myproject
```

### 2. Configure defaults

```bash
# Set or change the default project
acpctl config set project myproject

# Switch active project context (shorthand)
acpctl project myproject

# Show the currently active project
acpctl project current

# View current settings
acpctl config get api_url
acpctl config get project
```

### 3. List resources

```bash
# Sessions
acpctl get sessions
acpctl get sessions -o json

# Single session by ID
acpctl get session <session-id>
acpctl get session <session-id> -o json

# Projects
acpctl get projects

# Agents
acpctl get agents
acpctl get agents -o json

# Credentials
acpctl get credentials
acpctl get credentials -o json

# Roles
acpctl get roles
acpctl get roles -o json
```

### 4. Create resources

```bash
# Create a project
acpctl create project --name my-project --display-name "My Project" --description "Demo project"

# Create a session
acpctl create session --name fix-bug-123 \
  --prompt "Fix the null pointer in handler.go" \
  --repo-url https://github.com/org/repo \
  --model sonnet

# Create with all options
acpctl create session --name refactor-auth \
  --prompt "Refactor the auth middleware" \
  --model sonnet \
  --max-tokens 4000 \
  --temperature 0.7 \
  --timeout 3600

# Create an agent
acpctl agent create \
  --project-id my-project \
  --name my-agent \
  --prompt "You are a GitHub automation agent."

# Create a role binding (bind a credential role to an agent)
acpctl create role-binding \
  --user-id <user-id> \
  --role-id <role-id> \
  --scope agent \
  --scope-id <agent-id>
```

### 5. Apply declarative manifests

`acpctl apply` creates or updates resources from YAML files. Token values can be
injected via environment variables referenced in the manifest.

```bash
# Apply a credential manifest
cat > credential.yaml <<'EOF'
kind: Credential
name: my-github-pat
provider: github
token: $GITHUB_TOKEN
description: GitHub PAT for CI
EOF

GITHUB_TOKEN="ghp_..." acpctl apply -f credential.yaml
```

Supported `kind` values: `Credential` (additional kinds vary by deployment).

### 6. Agent sessions

```bash
# Start a session for a named agent with an initial prompt
acpctl agent start <agent-name> \
  --project-id <project-name> \
  --prompt "Open a test issue in org/repo"

# Start and capture the session ID
SESSION_JSON=$(acpctl agent start my-agent --project-id my-project --prompt "..." -o json)
SESSION_ID=$(echo "$SESSION_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
```

### 7. Session lifecycle

```bash
# Start a session
acpctl start <session-id>

# Stop a session
acpctl stop <session-id>
```

### 8. Session messages

```bash
# List all messages for a session (table format)
acpctl session messages <session-id>

# JSON output
acpctl session messages <session-id> -o json

# Stream new messages live (follow mode)
acpctl session messages <session-id> -f

# Stream only messages after a known sequence number
acpctl session messages <session-id> -f --after 42

# Send a user message to a running session (multi-turn)
acpctl session send <session-id> "Please also update the test file."
```

### 9. Inspect resources

```bash
# Full detail of a session
acpctl describe session <session-id>

# Full detail of a project
acpctl describe project <project-id>
```

### 10. Delete resources

```bash
acpctl delete session <session-id> -y
acpctl delete project <project-id> -y
acpctl delete project-settings <id>
acpctl credential delete <credential-id> --confirm
```

### 11. Log out

```bash
acpctl logout
```

## Credentials

Credentials store secrets (e.g. GitHub PATs, API keys) that are injected into
agent sessions at runtime. The runner retrieves the raw token via the
credentials API, so the secret is never embedded in session configuration.

```bash
# List credentials
acpctl get credentials

# Create via apply (token injected from env var — never passed as a flag)
GITHUB_TOKEN="ghp_..." acpctl apply -f credential.yaml

# Delete
acpctl credential delete <credential-id> --confirm
```

### Role bindings

Access to credentials is controlled by role bindings. The relevant roles are:

| Role | Permission |
|---|---|
| `credential:token-reader` | Retrieve the raw credential token via `GET /credentials/{id}/token` |
| `credential:reader` | Read credential metadata (name, provider, description) |

```bash
# Look up a role ID
ROLE_ID=$(acpctl get roles -o json | python3 -c "
import sys, json
data = json.load(sys.stdin)
items = data.get('items', []) if isinstance(data, dict) else data
for r in items:
    if r.get('name') == 'credential:token-reader':
        print(r['id']); break
")

# Get your user ID
MY_USER_ID=$(acpctl whoami | awk '/^User:/{print $2}')

# Bind the role to an agent (agent can now retrieve the token)
acpctl create role-binding \
  --user-id "${MY_USER_ID}" \
  --role-id "${ROLE_ID}" \
  --scope agent \
  --scope-id <agent-id>
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
./acpctl login http://localhost:8000 --token test-token
./acpctl whoami
./acpctl config get api_url
./acpctl config get project
./acpctl config set project other-project
./acpctl project current

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
| `make lint` | Vet + golangci-lint |
| `make test` | Run tests |

## Dependencies

- [Go SDK](../ambient-sdk/go-sdk/) via `replace` directive — zero-dep HTTP client for the Ambient API
- [cobra](https://github.com/spf13/cobra) — command framework
- [golang-jwt](https://github.com/golang-jwt/jwt) — token introspection for `whoami`
- [x/term](https://pkg.go.dev/golang.org/x/term) — terminal detection for table output
