# CLI Development Context

**When to load:** Working on `components/ambient-cli/` — `acpctl` commands, TUI dashboard, or session streaming

## Quick Reference

- **Binary:** `acpctl` (built to `components/ambient-cli/acpctl`)
- **Framework:** Cobra (commands) + Bubble Tea (TUI dashboard in `cmd/acpctl/ambient/tui/`)
- **SDK dependency:** Go SDK via `replace` directive → `../ambient-sdk/go-sdk`
- **Config:** `~/.config/ambient/config.json`
- **Env vars:** `AMBIENT_TOKEN`, `AMBIENT_PROJECT`, `AMBIENT_API_URL`, `AMBIENT_GRPC_URL`

---

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
│   ├── send            # Send a message to a running session
│   └── events          # Stream live AG-UI events from runner pod SSE
├── agent               # Agent subcommands
├── ambient             # TUI dashboard (Bubble Tea)
├── version
└── completion
```

---

## Adding a New Command

1. Create `cmd/acpctl/<command>/cmd.go`
2. Register in parent command's `AddCommand()` call
3. Use SDK client from `connection.NewClientFromConfig()` — never bypass the SDK
4. Follow existing patterns in `cmd/acpctl/session/messages.go`

**Standard command pattern:**
```go
var myCmd = &cobra.Command{
    Use:   "my-resource [name]",
    Short: "Short description",
    RunE: func(cmd *cobra.Command, args []string) error {
        client, err := connection.NewClientFromConfig()
        if err != nil {
            return err
        }
        result, err := client.MyResource().Get(cmd.Context(), args[0])
        if err != nil {
            return fmt.Errorf("getting resource: %w", err)
        }
        fmt.Println(result.Name)
        return nil
    },
}
```

**Never use raw `net/http` in CLI commands.** If a required SDK method doesn't exist, add it to the SDK first (see `sdk-development.md`), then write the CLI command against it. Auth header construction, `X-Ambient-Project` header injection, and base URL handling are all done by the SDK client — bypassing it breaks those invariants.

---

## SDK Extension Methods — Check First

Before writing any CLI command that calls a nested API endpoint (agents, inbox, ignite):

1. Check `go-sdk/client/agent_extensions.go` (or the relevant `*_extensions.go`) for the method you need
2. If it exists, use it
3. If it doesn't exist, add it to the extension file first, then write the CLI command

This is the most common source of "method not found" build failures on the CLI. See `sdk-development.md` for how to write extension methods.

---

## `go.mod` — Direct Import Rule

When adding a new file to the CLI that imports a package not previously used directly, run `go build ./...` immediately after adding the import. If it fails with `missing go.sum entry`, run:

```bash
cd components/ambient-cli
go get <package-path>
go mod tidy
```

Even if the package is transitively available (e.g. `gopkg.in/yaml.v3` via the SDK), Go modules require explicit declaration for direct imports. Fix `go.mod` before committing — do not commit with a broken build.

---

## Streaming Commands — SSE Pattern

Streaming commands (SSE / event streams) follow a specific pattern distinct from gRPC watch commands.

**gRPC streaming** (`session messages -f`):
- Uses `client.Sessions().WatchMessages(ctx, sessionID, afterSeq)`
- Returns typed events via channel or iterator
- File: `cmd/acpctl/session/messages.go`

**SSE streaming** (`session events`):
- SDK method returns `io.ReadCloser` — the raw HTTP response body
- CLI scans it line by line with `bufio.Scanner`
- Prints SSE data lines as they arrive
- Closes body on Ctrl+C or stream end

```go
// session/events.go — SSE streaming pattern
var eventsCmd = &cobra.Command{
    Use:   "events <session-id>",
    Short: "Stream live AG-UI events from a running session",
    Args:  cobra.ExactArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        client, err := connection.NewClientFromConfig()
        if err != nil {
            return err
        }

        body, err := client.Sessions().StreamEvents(cmd.Context(), args[0])
        if err != nil {
            return fmt.Errorf("streaming events: %w", err)
        }
        defer body.Close()

        scanner := bufio.NewScanner(body)
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "data: ") {
                fmt.Println(strings.TrimPrefix(line, "data: "))
            }
        }
        return scanner.Err()
    },
}
```

The SDK's `StreamEvents` method must return `io.ReadCloser` — see `sdk-development.md` for the implementation. Do not implement SSE in the CLI without that SDK method.

---

## Session Streaming (`session messages -f`)

The canonical follow-mode command. All new streaming commands should follow this pattern:

- `-f` / `--follow` flag — stream until Ctrl+C or `RUN_FINISHED`
- Renders incoming events as human-readable terminal output
- Exit cleanly on context cancellation (Ctrl+C)
- File: `cmd/acpctl/session/messages.go`

---

## `session send` — Interactive Follow Mode

`acpctl session send <id> "message" -f` sends a message then follows the response stream:

1. Post message via `client.Sessions().PushMessage(ctx, sessionID, payload)`
2. If `-f` flag set, immediately call `client.Sessions().WatchMessages(ctx, sessionID, afterSeq)` and stream until `RUN_FINISHED`

---

## TUI Dashboard (`acpctl ambient`)

- Entry: `cmd/acpctl/ambient/cmd.go`
- Model: `tui/model.go` (Bubble Tea model)
- Fetch: `tui/fetch.go` (API polling)
- View: `tui/view.go` (rendering)
- Port-forward entries: `tui/port_forward.go` — use local port `19000` for gRPC (port 9000 collides with minio)

---

## Build Commands

```bash
cd components/ambient-cli

make build      # builds ./acpctl binary
make test       # go test -race ./...
make lint       # gofmt + go vet + golangci-lint
make fmt        # gofmt -w
```

**Local testing against kind:**
```bash
export AMBIENT_API_URL=http://localhost:13595
export AMBIENT_TOKEN=$(kubectl get secret test-user-token -n ambient-code -o jsonpath='{.data.token}' | base64 -d)
./acpctl get sessions
./acpctl session messages -f <session-id>
./acpctl session events <session-id>
```

---

## Pre-Commit Checklist

- [ ] `make fmt` applied
- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] New commands registered in parent `AddCommand()`
- [ ] SDK client used via `connection.NewClientFromConfig()` — no raw `net/http`
- [ ] Extension method checked/added before implementing nested API calls
- [ ] `go build ./...` passes — `go.mod` updated if new direct imports added
- [ ] Env var overrides respected (`AMBIENT_TOKEN`, `AMBIENT_API_URL`)
- [ ] Error messages are user-friendly — `fmt.Errorf("doing X: %w", err)`, not raw errors
- [ ] `-f` / `--follow` pattern used for streaming commands
- [ ] SSE commands use `io.ReadCloser` + `bufio.Scanner`, not polling
