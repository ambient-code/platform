# Research: Gerrit Integration Connector

**Branch**: `001-gerrit-integration` | **Date**: 2026-03-24

## R1: Gerrit MCP Server Capabilities & Configuration

**Decision**: Use the open-source Gerrit MCP server from `gerrit.googlesource.com/gerrit-mcp-server` as-is, bundled into the runner image.

**Rationale**: The project is maintained by the Gerrit community, exposes 21 MCP tools covering all code review operations needed (query, review, comment, manage), and supports the two authentication methods required (HTTP basic and gitcookies). Building from source is straightforward with a single shell script.

**Alternatives considered**:
- Community npm package (`@hicho-lge/gerrit-mcp`): Less mature, fewer tools, not maintained by Gerrit project.
- Building a custom MCP server: Unnecessary duplication of effort; the official server already covers all required operations.

### Key Technical Details

**21 MCP Tools Available**: `query_changes`, `query_changes_by_date_and_filters`, `get_change_details`, `get_commit_message`, `list_change_files`, `get_file_diff`, `list_change_comments`, `get_bugs_from_cl`, `get_most_recent_cl`, `suggest_reviewers`, `add_reviewer`, `set_ready_for_review`, `set_work_in_progress`, `post_review_comment`, `abandon_change`, `set_topic`, `create_change`, `revert_change`, `revert_submission`, `changes_submitted_together`.

**Configuration**: Uses `gerrit_config.json` file (path overrideable via `GERRIT_CONFIG_PATH` env var):
```json
{
  "default_gerrit_base_url": "https://review.opendev.org/",
  "gerrit_hosts": [
    {
      "name": "OpenStack",
      "external_url": "https://review.opendev.org/",
      "authentication": {
        "type": "http_basic",
        "username": "user",
        "auth_token": "token"
      }
    }
  ]
}
```

**Authentication types**: `http_basic` (username + auth_token) and `git_cookies` (gitcookies_path). A third type `gob_curl` exists but is Google-internal only — out of scope.

**STDIO launch**: `.venv/bin/python gerrit_mcp_server/main.py stdio` with `PYTHONPATH` set to project root.

**Dependencies**: Python 3.12+, `mcp`, `uvicorn`, `websockets`, system `curl`.

**Not on PyPI**: Must be built from source via `./build-gerrit.sh` (creates venv, installs deps with hash verification).

---

## R2: Platform Integration Architecture Patterns

**Decision**: Use a dedicated backend handler pattern (like Jira/GitLab) rather than the generic MCP credentials pattern.

**Rationale**: Gerrit requires a specialized connection card UI (auth method selector between HTTP credentials and gitcookies), dedicated credential validation against the Gerrit REST API, and custom runtime config file generation (`gerrit_config.json`). The generic MCP pattern (`map[string]string` fields) is too simple for this.

**Alternatives considered**:
- Generic MCP credentials (`POST /api/auth/mcp/:serverName/connect`): Simpler but no auth method selection UI, no dedicated validation, no typed fields.
- Hybrid approach: Store via generic MCP but add custom validation. Adds complexity without benefit.

### Platform Patterns to Follow

**Backend (Go)**:
- Credential struct: `GerritCredentials { UserID, URL, AuthMethod, Username, HTTPToken, GitcookiesContent, UpdatedAt }`
- Secret name: `gerrit-credentials` with label `ambient-code.io/provider: gerrit`
- Three core endpoints: `ConnectGerrit`, `GetGerritStatus`, `DisconnectGerrit`, plus `TestGerritConnection`
- Session-scoped credential fetch: `GET /api/projects/:project/agentic-sessions/:session/credentials/gerrit`
- Validation: Test request to Gerrit REST API `GET /a/accounts/self` with provided credentials
- Conflict retry pattern: 3 retries on K8s optimistic locking conflicts

**Frontend (React/TypeScript)**:
- `GerritConnectionCard` component with auth method toggle (HTTP credentials vs gitcookies paste)
- `use-gerrit.ts` React Query hooks (connect/disconnect mutations with query invalidation)
- `gerrit-auth.ts` API client functions
- Add to `IntegrationsClient.tsx` grid layout

**Runner (Python)**:
- Cannot use simple env var expansion like Jira — Gerrit MCP server requires a `gerrit_config.json` file
- Runtime flow: fetch credentials from backend → generate `gerrit_config.json` → set `GERRIT_CONFIG_PATH` → launch MCP server
- Add `fetch_gerrit_credentials()` to `auth.py`
- Add Gerrit config generation to `mcp.py` or a new `gerrit.py` helper
- Add `check_mcp_authentication()` case for Gerrit server
- Register in `.mcp.json` with custom command pointing to bundled Gerrit MCP server

**Integrations Status**:
- Add `gerrit` field to unified status response in `integrations_status.go`
- Add `gerrit` type to frontend `IntegrationsStatus` TypeScript type

---

## R3: Gerrit Credential Validation

**Decision**: Validate by calling `GET /a/accounts/self` on the Gerrit REST API.

**Rationale**: This is the lightest authenticated endpoint on Gerrit. It returns the authenticated user's account info. Works with both HTTP basic auth and gitcookies. Returns 401 on invalid credentials.

**Alternatives considered**:
- `GET /a/changes/?q=limit:1`: Heavier, may return results requiring parsing.
- No validation: User experience suffers if invalid credentials are silently accepted.

### Validation Details

**HTTP Basic**: Standard HTTP Basic Auth header with `username:http_password`.

**Gitcookies**: Parse the pasted content to extract the cookie for the target host, send as `Cookie` header. The gitcookies format is: `host\tFALSE\t/\tTRUE\t2147483647\to\tgit-user.cookies=value`.

**Gerrit REST API note**: Gerrit prepends `)]}'` to JSON responses as an XSSI protection prefix. The validation must strip this before parsing.

---

## R4: Runner Bundling Strategy

**Decision**: Clone and build the Gerrit MCP server into the runner Docker image at build time.

**Rationale**: The Gerrit MCP server is not on PyPI, so it cannot be installed via `uvx` or `pip install`. Building at image build time ensures reproducible, hash-verified dependencies and avoids runtime network dependencies.

**Alternatives considered**:
- Runtime clone + build: Slow, unreliable, requires network access from runner pods.
- Fork and publish to PyPI: Maintenance burden, diverges from upstream.
- Vendoring source directly: Works but harder to update; Git clone with pinned tag is cleaner.

### Bundling Approach

In the runner Dockerfile:
1. Clone `gerrit-mcp-server` at a pinned tag/commit
2. Run `./build-gerrit.sh` to create venv with verified deps
3. Copy built venv + source to final image
4. Configure `.mcp.json` entry pointing to the bundled installation

---

## R5: Multi-Instance Support

**Decision**: Store each Gerrit instance as a separate entry in the `gerrit-credentials` K8s secret, keyed by `instanceName:userID`.

**Rationale**: Consistent with how the platform handles multi-value secrets. At runtime, the runner fetches all Gerrit credentials for the user and generates a single `gerrit_config.json` with multiple `gerrit_hosts` entries — leveraging the Gerrit MCP server's native multi-host support.

**Alternatives considered**:
- One secret per instance: More K8s objects to manage, harder to list all instances for a user.
- Single credential only (no multi-instance): Limits users contributing to multiple Gerrit communities.
