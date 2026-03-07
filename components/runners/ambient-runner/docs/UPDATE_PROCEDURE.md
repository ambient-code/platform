# Ambient Runner Update Procedure

Procedure for updating dependencies, base images, the Agent Options form, and performing housekeeping. Designed for execution by an AI agent (via Claude Code skill) or a developer.

## Prerequisites

- Access to the `platform` repository
- `uv` installed (for lock file regeneration)
- `gh` CLI authenticated (for PR creation)
- `pre-commit` installed (`pip install pre-commit && pre-commit install`)

## Procedure

### 1. Create a branch

```bash
git checkout -b chore/bump-runner-deps
```

### 2. Bump Python dependencies

For **every** dependency in `pyproject.toml`, look up the latest stable version on PyPI and set the minimum pin to match (`>=X.Y.Z`).

**File:** `components/runners/ambient-runner/pyproject.toml`

**Sections:**
- `[project] dependencies` — core runtime
- `[project.optional-dependencies]` — claude, observability, mcp-atlassian extras
- `[dependency-groups] dev` — dev/test

**How to find latest versions:**
```bash
pip index versions <package-name>
# or check https://pypi.org/project/<package-name>/
```

**Rules:**
- Pin to the exact latest stable release, not a conservative intermediate.
- `ag-ui-protocol` is internal — only bump if PyPI has a newer version than the current pin.

### 3. Bump MCP server versions

Check `.mcp.json` for version-pinned servers (using `@X.Y.Z` syntax).

| Server | How to check |
|--------|-------------|
| `workspace-mcp` | `pip index versions workspace-mcp` |
| `mcp-server-fetch` | Unpinned (auto-resolves) — no action needed |
| `mcp-atlassian` | Version controlled in pyproject.toml — no action here |

### 4. Update base images

**`Dockerfile`:**
- Check for a newer UBI major version at [Red Hat Catalog](https://catalog.redhat.com/en/software/base-images)
- Pin by SHA digest: `registry.access.redhat.com/ubi<ver>/ubi@sha256:...`
- If upgrading Python (e.g. 3.11 → 3.12), also update `requires-python` in `pyproject.toml`
- Node.js: use current LTS via `dnf module enable -y nodejs:<version>`
- Go: `dnf install go-toolset` (version managed by base image)

**`state-sync/Dockerfile`:**
- Update `FROM alpine:X.YY` to latest stable from [alpinelinux.org](https://www.alpinelinux.org/releases/)

### 5. Regenerate the lock file

```bash
cd components/runners/ambient-runner
uv lock
```

Verify it resolves cleanly. Packages may resolve newer than your `>=` pins — that's expected.

### 6. Sync the Agent Options form

When `claude-agent-sdk` is bumped, update the frontend Zod schema and form to match.

**Files:**
- `components/frontend/src/components/claude-agent-options/schema.ts`
- `components/frontend/src/components/claude-agent-options/options-form.tsx`

**Steps:**
1. Inspect the new SDK types:
   ```bash
   pip install --target /tmp/sdk claude-agent-sdk==<new-version>
   cat /tmp/sdk/claude_agent_sdk/types.py
   ```

2. Diff `ClaudeAgentOptions` against `schema.ts`:
   - **New fields** → add to schema + form
   - **Removed fields** → delete from schema + form
   - **Changed types** → update Zod validators (new enum values, TypedDict shapes)
   - **New nested types** → add sub-schemas

3. Types that change most often:

   | Type | What to check |
   |------|--------------|
   | `ClaudeAgentOptions` | New/removed fields on the main dataclass |
   | `HookEvent` | New lifecycle events (add to `hookEventSchema` enum) |
   | `McpServerConfig` | New transport types (add discriminated union variant) |
   | `ThinkingConfig` | New thinking modes |
   | `SandboxSettings` | New sandbox options |
   | `SdkBeta` | New beta feature literals |
   | `AgentDefinition` | New model options |
   | `SdkPluginConfig` | New plugin types |

4. Update `claudeAgentOptionsDefaults` if defaults changed.

5. **Omit non-serializable fields** (callbacks, runtime objects):
   `can_use_tool`, `hooks[].hooks` (HookCallback list), `mcp_servers` with `type: "sdk"`, `debug_stderr`, `stderr`

### 7. Run housekeeping

#### a. pytest-asyncio config
If bumped across a major version, verify `pyproject.toml` has `asyncio_mode = "auto"`.

#### b. Type hints
If Python version was bumped, modernize: `Optional[X]` → `X | None`, `List[X]` → `list[X]`, etc.

```bash
grep -r "from typing import.*\(Optional\|List\|Dict\|Union\|Tuple\)" --include="*.py"
```

#### c. Dead code
- Remove large commented-out Dockerfile blocks
- Address or remove stale `TODO`/`FIXME`/`HACK` comments
- Convert `pytest.skip()` bodies to `@pytest.mark.skip(reason=...)`

#### d. Deprecated patterns
Check for deprecation warnings from upgraded dependencies (Pydantic v1→v2, old SDK patterns, etc.).

### 8. Lint and commit

Run pre-commit on all changed files:
```bash
pre-commit run --files <changed-files>
```

Create a **draft PR** with a version-change table:
```bash
gh pr create --draft \
  --title "chore(runner): bump all dependencies to latest versions" \
  --body "$(cat <<'EOF'
## Summary
- Bumps all ambient-runner dependencies to latest PyPI releases
- Updates base images and Agent Options form
- Housekeeping: type hints, dead code, config

### Version changes
| Package | Old | New |
|---------|-----|-----|
| ... | ... | ... |

## Test plan
- [ ] CI passes
- [ ] Verify major version bumps don't break APIs
- [ ] Smoke test MCP integrations
- [ ] Frontend builds with agent options schema changes
EOF
)"
```

### 9. Post-PR verification

After CI runs, check for:
- Test failures from breaking API changes
- Container build failures from base image changes
- Lint failures from type hint changes
- Frontend build failures from schema changes

## Frequency

Run **monthly** or when a critical security patch is released.

## Automation Strategy

This procedure is designed to be fully automatable. Below is the recommended path from manual to autonomous.

### Phase 1: Claude Code Skill (current target)

Create a Claude Code user-invocable skill at `.claude/commands/bump-runner-deps.md` that executes this procedure end-to-end:

```markdown
---
description: Bump all ambient-runner dependencies, base images, and Agent Options form.
---

Execute the update procedure in components/runners/ambient-runner/docs/UPDATE_PROCEDURE.md.

For each step:
1. Create a branch `chore/bump-runner-deps-YYYY-MM-DD`
2. For each package in pyproject.toml, run `pip index versions <pkg>` and update the pin
3. Check .mcp.json for pinned server versions and update
4. Check Dockerfile base images (use registry API for SHA digests)
5. Run `uv lock`
6. If claude-agent-sdk was bumped, read the new types.py and diff against schema.ts
7. Run housekeeping checks (type hints, dead code, deprecated patterns)
8. Run `pre-commit run --files <changed>` and fix any failures
9. Commit, push, and create a draft PR with a version-change table
```

The skill replaces the need to read and follow this document manually. Invoke with `/bump-runner-deps`.

### Phase 2: Scheduled Agent Sessions

Use the platform's own session scheduling to run the skill on a cron:

1. Create a Project Settings CRD with a scheduled session:
   ```yaml
   schedule:
     cron: "0 9 1 * *"  # First of each month at 09:00
     prompt: "/bump-runner-deps"
     autoMerge: false  # Always create draft PRs
   ```

2. The session runs the skill, creates a draft PR, and notifies via Slack/email.
3. A human reviews and merges.

### Phase 3: Subagents for Parallelism

Break the procedure into independent subagents that run in parallel:

| Agent | Task | Dependencies |
|-------|------|-------------|
| `dep-checker` | Query PyPI for all packages, return version map | None |
| `image-checker` | Query container registries for latest base images | None |
| `sdk-syncer` | Diff claude-agent-sdk types against schema.ts | `dep-checker` (needs new SDK version) |
| `housekeeping` | Type hints, dead code, deprecated patterns | `dep-checker` (needs Python version) |
| `committer` | Lint, commit, push, create PR | All above |

The orchestrator agent spawns `dep-checker` and `image-checker` in parallel, then `sdk-syncer` and `housekeeping`, then `committer`.

### Phase 4: Full CI Integration

Move version checking into CI as a scheduled GitHub Action:

```yaml
# .github/workflows/dependency-freshness.yml
on:
  schedule:
    - cron: '0 9 * * 1'  # Weekly Monday 09:00
jobs:
  check:
    steps:
      - run: |
          # Compare current pins against PyPI latest
          # Post Slack alert if any package is >30 days behind
```

The CI job detects staleness and triggers the Claude Code skill via the platform API, creating a fully hands-off pipeline:

```
CI detects stale deps → API creates session → Skill runs procedure → Draft PR created → Human reviews
```

### What Cannot Be Automated

- **Base image major version upgrades** (UBI 9 → UBI 10) — require manual testing for compatibility
- **Breaking API changes** in dependencies — require human judgment on migration
- **Unleash flag creation** — the `advanced-agent-options` flag must be created in Unleash with tag `scope: workspace`

## Reference

Last executed: 2026-03-07
PR: https://github.com/ambient-code/platform/pull/845
