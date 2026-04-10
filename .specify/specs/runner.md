# Runner Component Spec

**Version**: 1.0.0
**Created**: 2026-03-28
**Constitution**: [Runner Constitution](../constitutions/runner.md)
**Component**: `components/runners/ambient-runner/`

---

## Overview

The ambient-runner is a Python application that executes AI agent sessions inside Kubernetes Job pods. It bridges AG-UI protocol events to multiple AI providers (Claude, Gemini, LangGraph) and exposes a FastAPI server on port 8001.

## Component Boundary

### Managed Paths

```text
components/runners/ambient-runner/
├── Dockerfile                    # Runner container image
├── main.py                      # FastAPI entry point
├── pyproject.toml                # Python dependencies
├── uv.lock                      # Resolved dependency lock
├── .mcp.json                    # MCP server configuration
├── ag_ui_claude_sdk/             # Claude AG-UI adapter
├── ag_ui_gemini_cli/             # Gemini AG-UI adapter
├── ambient_runner/               # Core runner package
│   ├── bridges/                  # Provider bridges
│   │   ├── claude/
│   │   ├── gemini_cli/
│   │   └── langgraph/
│   ├── endpoints/                # FastAPI routes
│   ├── middleware/                # Request middleware
│   └── platform/                 # Platform integration
├── tests/                        # Test suite
└── docs/
    └── UPDATE_PROCEDURE.md       # Maintenance procedure

.github/workflows/
└── runner-tool-versions.yml      # Automated freshness checks
```

### Supporting Frontend Paths

```text
components/frontend/src/components/claude-agent-options/
├── schema.ts                     # Zod schema (mirrors SDK types)
├── options-form.tsx              # Main form component
├── index.ts                      # Barrel exports
└── _components/                  # Per-section editors
```

## Current State (as of PR #1091)

### Base Image
- **UBI 10** (`registry.access.redhat.com/ubi10/ubi@sha256:...`)
- Python 3.12 (system default), Node.js (AppStream), Go (go-toolset)

### Pinned Tools

| Tool | Dockerfile ARG | Purpose |
|------|---------------|---------|
| gh | `GH_VERSION` | GitHub CLI for repo operations |
| glab | `GLAB_VERSION` | GitLab CLI for repo operations |
| uv | `UV_VERSION` | Python package management |
| pre-commit | `PRE_COMMIT_VERSION` | Git hook framework |
| gemini-cli | `GEMINI_CLI_VERSION` | Google Gemini CLI |

### Key Dependencies

| Package | Constraint | Role |
|---------|-----------|------|
| claude-agent-sdk | `>=0.1.50` | Claude Code agent SDK |
| anthropic | `>=0.86.0` | Anthropic API client |
| mcp | `>=1.9.2` | Model Context Protocol |
| ag-ui-protocol | `>=0.6.2` | AG-UI event protocol |

## Maintenance Workflows

### Weekly: Tool Freshness (`runner-tool-versions.yml`)
- Checks all pinned tools against upstream registries
- Opens a PR if any component has a newer version
- Does not auto-merge

### Monthly: Dependency Bump (`UPDATE_PROCEDURE.md`)
- Bumps all Python dependencies to latest stable
- Checks for SDK type changes → syncs Agent Options schema
- Regenerates lock file
- Runs housekeeping (type hints, dead code)

## Change Protocol

1. All changes to managed paths MUST go through the SDD workflow when the component is in `enforce` mode, and SHOULD when in `warn` mode (see `sdd-manifest.yaml`).
2. Changes MUST comply with the runner constitution.
3. SDK bumps MUST include a schema sync check.
4. Dockerfile changes MUST maintain version pinning and layer discipline.
5. Test coverage MUST not decrease.

## Verification Checklist

- [ ] Container image builds successfully
- [ ] All tests pass (`pytest`)
- [ ] Pre-commit hooks pass
- [ ] `gh version`, `glab version`, `uv --version`, `gemini --version` work in container
- [ ] Agent Options form renders correctly (if schema changed)
- [ ] No `Optional[X]` or `List[X]` style type hints (Python 3.12 uses `X | None`, `list[X]`)
