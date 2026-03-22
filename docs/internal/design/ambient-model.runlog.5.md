# Ambient Model: Run Log 5 — 2026-03-22

**Status:** API consistency audit + bug fixes. Image built and pushed to kind cluster. Commit: `6da3794d`.

---

## Fixes Applied

| File | Fix |
|---|---|
| `sessions.proto` | Added `agent_id` field (32); regenerated `sessions.pb.go` |
| `plugins/sessions/grpc_presenter.go` | Wired `AgentId` in `sessionToProto()` — root cause of CP never receiving `agent_id` in watch events |
| `openapi.yaml` `ProjectHomeAgent` | `project_agent_id` → `agent_id`, removed stale `agent_version` field |
| `go-sdk/types/ignite.go` `ProjectHomeAgent` | Same field rename |
| `openapi.yaml` `IgniteResponse` | `ignition_context` → `ignition_prompt` (matches backend struct and Go SDK type) |
| `inbox/handler.go` Create | Enforce `agent_id = mux.Vars(r)["pa_id"]` — body can no longer spoof a different agent |
| `inbox/handler.go` List | Scope results via TSL filter `agent_id = 'X'` injected from URL `pa_id` param; `listArgs.Search` is `string` not `*string` — nil-check must be empty-string check |
| `inbox/presenter.go` | Nil-guard `UpdatedAt` independently from `CreatedAt` (panic fix) |
| `inbox/model.go` `InboxMessagePatchRequest` | Trimmed to only `Read *bool` — removed over-permissive fields |
| `go-sdk/client/client.go` | Added `doMultiStatus()` accepting variadic expected status codes |
| `go-sdk/client/agent_extensions.go` `Ignite` | Now accepts HTTP 200 or 201 (re-ignite returns 200, new ignite returns 201) |
| `ts-sdk/src/project_agent_api.ts` | Complete rewrite with correct nested paths and proper method signatures |
| `ts-sdk/src/inbox_message_api.ts` | Complete rewrite with correct nested paths |

---

## gRPC Streaming Issue (partially resolved)

`acpctl session messages <id> -f` was failing with:
```
dial tcp 127.0.0.1:9000: connect: connection refused
```

Root cause: The Go SDK's `deriveGRPCAddress()` strips the port from the REST base URL and appends `:9000`. When the CLI targets `http://127.0.0.1:8000`, it derives `127.0.0.1:9000` for gRPC. But local port 9000 is occupied by minio.

Fix: `kubectl port-forward svc/ambient-api-server 19000:9000 -n ambient-code` and set `AMBIENT_GRPC_URL=127.0.0.1:19000`.

The TUI's `PortForwardEntry` for gRPC already maps to local port 19000 — this is the canonical local gRPC port.

Longer-term: add `grpc_url` to CLI config struct (`pkg/config/config.go`) so users can set it once with `acpctl config set grpc_url 127.0.0.1:19000` rather than needing the env var on every command.

---

## Stopped At

Commit `6da3794d` pushed. gRPC streaming local workaround documented. `GET /sessions/{id}/events` proxy spec added.
