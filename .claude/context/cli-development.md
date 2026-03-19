# CLI Development Context

**When to load:** Working on `components/ambient-cli/` — `acpctl` commands, TUI dashboard, or session streaming

## Quick Reference

- **Binary:** `acpctl` (built to `components/ambient-cli/acpctl`)
- **Framework:** Cobra (commands) + Bubble Tea (TUI dashboard in `cmd/acpctl/ambient/tui/`)
- **SDK dependency:** Go SDK via `replace` directive → `../ambient-sdk/go-sdk`
- **Config:** `~/.config/ambient/config.json`
- **Env vars:** `AMBIENT_TOKEN`, `AMBIENT_PROJECT`, `AMBIENT_API_URL`, `AMBIENT_GRPC_URL`

## Command Tree

```
acpctl
├── login               # Store token + URL in config
├── logout
├── whoami              # Print current user from /api/ambient/v1/users/~
├── config get/set      # Config file management
├── get                 # List resources (sessions, projects, agents, roles)
├── create              # Create resources
├── describe            # Get single resource details
├── delete              # Delete resources
├── start               # Start a session
├── stop                # Stop a session
├── project             # Project-scoped subcommands
├── session
│   ├── messages        # List or follow session messages (gRPC stream with -f)
│   └── send            # Send a message to a running session
├── agent               # Agent subcommands
├── ambient             # TUI dashboard (Bubble Tea)
├── version
└── completion
```

## Adding a New Command

1. Create `cmd/acpctl/<command>/cmd.go`
2. Register in parent command's `AddCommand()` call
3. Use SDK client from `pkg/` for API calls
4. Follow existing patterns in `cmd/acpctl/get/cmd.go` or `cmd/acpctl/session/cmd.go`

**Standard command pattern:**
```go
var myCmd = &cobra.Command{
    Use:   "my-resource [name]",
    Short: "Short description",
    RunE: func(cmd *cobra.Command, args []string) error {
        cfg, err := config.Load()
        if err != nil {
            return err
        }
        client := sdk.NewClient(cfg.APIURL, cfg.Token)
        // ... use client
        return nil
    },
}
```

## Session Streaming (`session messages -f`)

The `session messages` command with `-f` flag streams gRPC `WatchSessionMessages` RPC:
- File: `cmd/acpctl/session/messages.go`
- Uses `client.Sessions.WatchMessages(ctx, project, sessionName)`
- Renders AG-UI events as human-readable terminal output
- Exit on `RUN_FINISHED` or Ctrl+C

**This is the key command for agent-deck integration** (`acpctl session messages -f <session>` as a terminal follow mode).

## TUI Dashboard (`acpctl ambient`)

- Entry: `cmd/acpctl/ambient/cmd.go`
- Model: `tui/model.go` (Bubble Tea model)
- Fetch: `tui/fetch.go` (API polling)
- View: `tui/view.go` (rendering)
- Dashboard: `tui/dashboard.go`

Poll interval, key bindings, and layout are in `tui/model.go`.

## Build & Test

```bash
cd components/ambient-cli
make build      # builds ./acpctl binary
make test       # go test ./...
make lint       # gofmt + go vet + golangci-lint
make fmt        # gofmt -w
```

**Local testing against kind:**
```bash
export AMBIENT_API_URL=http://localhost:13595
export AMBIENT_TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d)
./acpctl get sessions
./acpctl session messages -f --project my-project my-session
```

## SDK Dependency

The CLI uses the Go SDK via a `replace` directive in `go.mod`:
```
replace github.com/ambient/platform/ambient-sdk => ../ambient-sdk/go-sdk
```

When the SDK changes, rebuild the CLI: `make build`.

## Pre-Commit Checklist

- [ ] `make fmt` applied
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] New commands registered in parent `AddCommand()`
- [ ] Config loaded via `config.Load()` (not hardcoded)
- [ ] Env var overrides respected (`AMBIENT_TOKEN`, `AMBIENT_API_URL`)
- [ ] Error messages are user-friendly (not raw Go errors)
- [ ] `-f` / `--follow` pattern used for streaming commands
