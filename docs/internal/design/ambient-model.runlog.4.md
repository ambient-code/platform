# Ambient Model: Run Log 4 — 2026-03-21

**Status:** Agent+ProjectAgent collapse implemented (Waves 2–5). Image built and pushed to kind cluster.

---

## Spec Changes This Cycle

- `Agent` + `ProjectAgent` collapsed into a single project-scoped `Agent` (previously `ProjectAgent`)
- Global immutable `Agent` concept removed entirely
- New `Agent` fields: `project_id`, `name`, `prompt` (mutable), `current_session_id`, `labels`, `annotations`
- Removed: `agent_id`, `agent_version`, `owner_user_id`, `display_name` on Agent; `display_name` on Project

---

## Waves Completed

- Wave 2 (API): openapi rewritten — `projectAgents` → `agents` with new schema; global agents routes removed
- Wave 3 (SDK): `make generate-sdk` — 9 resources; `agent_extensions.go` hand-written for non-CRUD methods
- Wave 4 (BE): `plugins/agents/` rewritten; `plugins/projectAgents/` deleted; `plugins/inbox/` field names updated
- Wave 5 (CLI): `acpctl agent` subcommands rewritten; `acpctl inbox` flag names updated; tests fixed

---

## Notable Issues and Resolutions

**Kind image load failure with podman:** `kind load docker-image` calls `docker inspect` internally and cannot resolve podman-prefixed `localhost/` images. Fixed by using the `podman save | ctr import` approach, which writes directly to containerd's k8s.io namespace in the control-plane container.

**Build cache miss on source changes:** The Dockerfile copies source in layers. If `go.mod`/`go.sum` are unchanged, the `go build` step hits cache and emits the old binary. Fix: always pass `--no-cache` when pushing a code-only change.

**Active cluster mismatch:** The running cluster was `ambient-main`, not the branch-named cluster `ambient-feat-session-message`. Always verify with `podman ps | grep kind` before loading images.

**TS SDK generator nested path bug:** Both `project_agent_api.ts` and `inbox_message_api.ts` were generated with `/projects` hardcoded as the base path — they route to the Projects resource instead of the correct nested endpoints. The generator uses the first path segment of a resource's routes as the base path. For nested resources this is wrong. Fixed by hand-writing `ts-sdk/src/project_agent_api.ts` and `inbox_message_api.ts` (see Run 5).

**Missing TS SDK extension file:** Neither `ignite()` nor `send()` exist in the ts-sdk. The Go SDK has `agent_extensions.go`; the TypeScript SDK had no equivalent. Fixed in Run 5.

**No index.html dashboard:** The report about `client.projectAgents.ignite()` and `client.inboxMessages.send()` being called from a dashboard was incorrect; no such file exists in the project source — only inside `node_modules`.
