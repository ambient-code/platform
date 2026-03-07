# Ambient Runner Update Procedure

Step-by-step procedure for updating the ambient-runner's dependencies, base images, and performing housekeeping. Designed to be executed by an AI agent or developer.

## Prerequisites

- Access to the `platform` repository
- `uv` installed (for lock file regeneration)
- `gh` CLI authenticated (for PR creation)

## Procedure

### 1. Create a branch

```bash
cd components/runners/ambient-runner
git checkout -b chore/bump-runner-deps
```

### 2. Bump Python dependencies in pyproject.toml

For **every** dependency in `pyproject.toml` (core, optional, and dev), look up the latest version on [PyPI](https://pypi.org) and update the minimum version pin to match.

**Files to update:** `pyproject.toml`

**Sections to check:**
- `[project] dependencies` — core runtime deps
- `[project.optional-dependencies]` — claude, observability, mcp-atlassian extras
- `[dependency-groups] dev` — dev/test dependencies

**How to find latest versions:**
- Search PyPI for each package (e.g., `https://pypi.org/project/<package-name>/`)
- Set the minimum to the exact latest stable release (not pre-releases)
- Use `>=X.Y.Z` format

**Packages to check (as of last update):**

| Section | Package |
|---------|---------|
| core | fastapi, uvicorn, ag-ui-protocol, pydantic, aiohttp, requests, pyjwt |
| claude | anthropic, claude-agent-sdk |
| observability | langfuse |
| mcp-atlassian | mcp-atlassian |
| dev | pytest, pytest-asyncio, pytest-cov, ruff, black, httpx |

**Important:**
- `ag-ui-protocol` is an internal package — only bump if a new version has been published to PyPI. If the current pin is higher than PyPI, keep it as-is.
- Use the exact latest version, not a conservative intermediate bump.

### 3. Bump MCP server versions in .mcp.json

Check `.mcp.json` for any version-pinned MCP servers (using `@X.Y.Z` syntax).

**Known pinned servers:**
- `workspace-mcp` — check latest on [PyPI](https://pypi.org/project/workspace-mcp/)

**Unpinned servers (no action needed):**
- `mcp-server-fetch` — invoked via `uvx` without a pin, auto-resolves to latest
- `mcp-atlassian` — installed as a Python dependency, version controlled in pyproject.toml

### 4. Update base images in Dockerfiles

**`Dockerfile` (ambient-runner):**
- Check for newer UBI base image (e.g., UBI 9 → UBI 10)
  - Image: `registry.access.redhat.com/ubi<version>/ubi:latest` (standard) or `ubi<version>/python-<pyversion>` (python-specific)
  - Check [Red Hat Ecosystem Catalog](https://catalog.redhat.com/en/software/base-images) for available images
- Check Python version: RHEL 10 ships Python 3.12, RHEL 9 ships 3.11
  - If upgrading Python, also update `requires-python` in `pyproject.toml`
- Check Node.js version: prefer the current LTS (even-numbered releases)
  - On UBI, install via `dnf install nodejs npm`
- Go toolset: installed via `dnf install go-toolset`, version managed by the base image

**`state-sync/Dockerfile`:**
- Check latest Alpine version at [alpinelinux.org](https://www.alpinelinux.org/releases/)
- Update `FROM alpine:X.YY` to latest stable

### 5. Regenerate the lock file

```bash
cd components/runners/ambient-runner
uv lock
```

- Verify it resolves cleanly with no errors
- Check for deprecation warnings (e.g., `[tool.uv] dev-dependencies` → `[dependency-groups] dev`)
- Note any packages that resolved to versions newer than your pins (the `>=` constraint allows this — that's expected)

### 6. Run housekeeping checks

After dependency and image updates, check for these common housekeeping items:

#### a. pytest-asyncio configuration
If pytest-asyncio was bumped across a major version, verify `pyproject.toml` has:
```toml
[tool.pytest.ini_options]
asyncio_mode = "auto"
```

#### b. Type hint modernization
If the minimum Python version was bumped, modernize type hints:
- `Optional[X]` → `X | None`
- `List[X]` → `list[X]`
- `Dict[K, V]` → `dict[K, V]`
- `Tuple[X, Y]` → `tuple[X, Y]`
- `Union[X, Y]` → `X | Y`
- Remove unused imports from `typing` (`Optional`, `List`, `Dict`, `Tuple`, `Union`)

Search with: `grep -r "from typing import.*\(Optional\|List\|Dict\|Union\|Tuple\)" --include="*.py"`

#### c. Dead code removal
- Check Dockerfile for large commented-out blocks (>10 lines) — remove or move to docs
- Search for `TODO`, `FIXME`, `HACK` comments — address or remove stale ones
- Check for `pytest.skip()` inside test bodies — convert to `@pytest.mark.skip(reason=...)`

#### d. Deprecated API patterns
Check if any dependency upgrades introduced deprecation warnings. Common ones:
- `[tool.uv] dev-dependencies` → `[dependency-groups] dev` (uv)
- Pydantic v1 patterns in v2 code
- Old anthropic SDK patterns

### 7. Commit, push, and create PR

Use separate commits for logical groups:

```
Commit 1: chore(runner): bump all dependencies to latest versions
  - pyproject.toml changes + uv.lock regeneration

Commit 2: chore(runner): bump MCP server versions
  - .mcp.json changes

Commit 3: chore(runner): upgrade base images
  - Dockerfile + state-sync/Dockerfile + requires-python

Commit 4: chore(runner): housekeeping
  - Type hints, dead code, config fixes
```

Create a **draft PR** with a table of version changes and a test plan:

```bash
gh pr create --draft \
  --title "chore(runner): bump all dependencies to latest versions" \
  --body "## Summary
- Bumps all ambient-runner dependencies to latest PyPI releases
- Updates base images (UBI, Alpine, Node.js)
- Housekeeping: type hints, dead code, config

### Version changes
| Package | Old | New |
|---------|-----|-----|
| ... | ... | ... |

## Test plan
- [ ] CI pipeline passes
- [ ] Verify major version jumps don't break APIs
- [ ] Smoke test MCP integrations
"
```

### 8. Post-PR verification

After CI runs, check for:
- Test failures from breaking API changes in bumped dependencies
- Container build failures from base image changes
- Lint failures from type hint changes

## Frequency

Run this procedure **monthly** or when a critical security patch is released for any dependency.

## Reference

Last executed: 2026-03-07
PR: https://github.com/ambient-code/platform/pull/845
