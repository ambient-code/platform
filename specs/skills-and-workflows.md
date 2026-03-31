# Skills & Workflows: Discovery, Installation, and Usage

## Summary

A workflow is a `CLAUDE.md` prompt plus a list of sources in `ambient.json`. Skills are the atomic reusable unit. Everything installed is a source reference — the runner auto-detects what's inside. ACP automates the cloning and wiring. Locally, a `/load-workflow` skill or manual `--add-dir` does the same thing. The marketplace feature is behind a feature flag.

---

## Core Concepts

### Skill

The atomic unit of reusable capability. A directory containing a `SKILL.md` file with YAML frontmatter and markdown instructions. Claude Code discovers skills from `.claude/skills/{name}/SKILL.md` in the working directory, parent directories, `--add-dir` directories, and plugins.

Skills have live change detection in `--add-dir` directories — place a new skill file and Claude discovers it immediately without restarting. Skills are invoked via `/skill-name` or auto-triggered by Claude based on the description in frontmatter.

Commands (`.claude/commands/{name}.md`) and agents (`.claude/agents/{name}.md`) follow the same discovery pattern and are treated as peers to skills throughout this spec. "Skill" is used as shorthand for all three unless distinction matters.

### Workflow

A workflow is two things:

1. **A prompt** — the directive and methodology, written as `CLAUDE.md` in the workflow directory. This is the only prompt mechanism — no separate `systemPrompt` or `startupPrompt` fields.
2. **A list of sources** — references in `ambient.json` to skills, commands, agents, and plugins from various Git repos. Not embedded copies — references resolved at load time.

A workflow does not contain skills. It references them. The bug-fix workflow becomes:

**`CLAUDE.md`**:
```markdown
You are a systematic bug fixer. Follow these phases:
1. Use /assess to understand the issue
2. Use /reproduce to create a failing test
3. Use /diagnose to find the root cause
4. Use /fix to implement the minimal fix
5. Use /test to verify the fix
6. Use /review to self-review before PR
```

**`ambient.json`**:
```json
{
  "name": "Bug Fix",
  "description": "Systematic bug resolution with phased approach",
  "sources": [
    {"url": "https://github.com/ambient-code/skills.git", "branch": "main", "path": "bugfix/assess"},
    {"url": "https://github.com/ambient-code/skills.git", "path": "bugfix/reproduce"},
    {"url": "https://github.com/ambient-code/skills.git", "path": "bugfix/diagnose"},
    {"url": "https://github.com/ambient-code/skills.git", "path": "bugfix/fix"},
    {"url": "https://github.com/ambient-code/skills.git", "path": "bugfix/test"},
    {"url": "https://github.com/ambient-code/skills.git", "path": "bugfix/review"},
    {"url": "https://github.com/opendatahub-io/ai-helpers.git", "path": "helpers/skills/jira-activity"},
    "https://github.com/my-org/shared-skills/tree/main/code-review"
  ],
  "rubric": {
    "activationPrompt": "After completing the fix, evaluate your work",
    "criteria": [
      {"name": "Root cause identified", "weight": 0.3},
      {"name": "Tests added", "weight": 0.3},
      {"name": "Minimal change", "weight": 0.2},
      {"name": "No regressions", "weight": 0.2}
    ]
  }
}
```

Sources support two formats:
- **Structured object**: `{"url": "...", "branch": "...", "path": "..."}` — works with any Git host, branch is explicit. Supports optional `tag` or `sha` field for pinning.
- **Single URL string**: `"https://github.com/org/repo/tree/main/path"` — auto-parsed, convenient for sharing

Skills are the reusable atoms. Workflows are recipes. The same skill can appear in multiple workflows.

### Agent (future)

A persona — prose defining what an agent is responsible for. "Backend Agent", "Security Agent", "PM Agent". An Agent uses workflows and standalone skills to accomplish its goals. An Agent is a session template with a personality:

```
Agent = Persona (CLAUDE.md) + Workflows (skill bundles) + Standalone skills
```

A "Bug Fix Agent" = bug-fix persona + bug-fix workflow skills + any additional skills. Same skills reusable by different Agents with different motivations.

Multi-agent orchestration (research agent → writer agent → editor agent pipelines) is a separate design problem out of scope for this spec.

---

## Discovery

Discovery is behind a feature flag.

### What

A way to browse and find skills, workflows, and plugins from curated sources.

### Source Types

The scanner must support three types of sources:

1. **Claude Code plugins** — directories with `.claude-plugin/plugin.json` containing `skills/`, `commands/`, `agents/`, `hooks/`, `.mcp.json`. This is the primary format to follow and expect. Skills are namespaced as `plugin-name:skill-name`.

2. **Claude Code marketplace catalogs** — `marketplace.json` files listing plugins with their sources. Users could add the same marketplace from local Claude Code via `/plugin marketplace add`.

3. **Standalone repos with `.claude/`** — any Git repo containing `.claude/skills/`, `.claude/commands/`, `.claude/agents/`. Also supports root-level `skills/`, `commands/`, `agents/` (registry layout).

### How

A cluster-level ConfigMap (`marketplace-sources`) holds available registry sources. The Marketplace page in the ACP UI shows:

- Browsable catalogs from each source with search and type filters
- Compact card tiles with name, description, type badge
- Detail panel on click showing extracted skills, commands, and agents inside the source — rendered readably, not as a raw file viewer
- "Import Custom" to scan any Git URL and discover items
- Direct one-click install to workspace

### Scanning

When given a Git URL (from marketplace or custom), the backend:

1. Shallow clones the repo
2. Applies optional path filter (subdirectory)
3. Checks for `.claude-plugin/plugin.json` (Claude Code plugin format)
4. Scans for items in both patterns:
   - `.claude/skills/*/SKILL.md`, `.claude/commands/*.md`, `.claude/agents/*.md`
   - `skills/*/SKILL.md`, `commands/*.md`, `agents/*.md` (registry layout)
5. Checks for `.ambient/ambient.json` (indicates this is a workflow)
6. Checks for `CLAUDE.md` (indicates project instructions)
7. Returns discovered items with frontmatter metadata

### Format Alignment

We follow Claude Code's plugin and skill formats as the standard. The [Agent Skills](https://agentskills.io) open standard that Claude Code implements is the closest cross-tool specification.

---

## Installation & Configuration

### Everything is a Source Reference

All installed items are source references — Git URLs pointing to repos containing skills, plugins, or workflows. There is no type distinction in the data model. The runner auto-detects what each source contains when it clones and scans.

### Workspace Level

Source references installed at the workspace level are stored in the ProjectSettings CR. These represent the workspace's **registry** — what's available to sessions.

The registry is NOT auto-injected into every session. At session creation, users select which sources to load from the registry. The workflow they choose also pulls in its own dependencies from the `sources` array in `ambient.json`.

Items can optionally be marked as "always add" — these load into every session by default. This is useful for org-wide standards or team-shared skills. (Needs further design discussion.)

### Session Level

Sources can be added to a running session via the context panel:

- "Import Skills" in the Add Context dropdown
- Provide a Git URL + optional branch + path
- Backend clones, scans, loads skills into discoverable locations
- Claude discovers them via live change detection
- Persisted via S3 state-sync on session suspend/resume

### Skill Storage in the Runner

Skills, commands, and agents must end up where Claude Code expects them for automatic discovery. The workspace root already has a `.claude/` directory that is persisted by state-sync. Sources should be loaded so their contents are discoverable — either via `add_dirs` pointing to each source's `.claude/` structure, or by writing directly into the workspace `.claude/`. The exact mechanism needs further discussion, but the key constraint is: Claude must discover them without any non-standard configuration.

Plugins and workflows should also be co-located in this space for consistency.

### Versioning

Sources reference branches by default, which means sessions always get the latest version — providing auto-update behavior. For pinning, sources support optional `tag` or `sha` fields:

```json
{"url": "https://github.com/org/skills.git", "branch": "main", "path": "assess", "sha": "a1b2c3d4"}
```

When a SHA is specified, the runner checks out that exact commit. When only a branch is specified, the runner clones the latest. This gives users the choice: use `branch` for auto-update, use `sha` or `tag` for stability.

### How Selection Works

The workspace registry is not auto-injected. Selection happens at session creation:

1. User picks a workflow (or "General chat" for none)
2. The workflow's `ambient.json` `sources` array declares dependencies — those are auto-loaded
3. User can add any number of additional sources from the workspace registry — workflows, skills, commands, plugins, anything
4. The session stores the workflow reference + any additional source references

This means:
- Installing 50 sources to the workspace doesn't bloat every session
- The workflow controls its own dependencies
- Users can augment with extras per session
- "Always add" items provide workspace-level defaults (needs further design)

---

## Usage in Sessions

### Loading

When a session starts, sources are loaded in layers:

1. **Workflow sources** — skills from the workflow's `ambient.json` `sources` array, cloned and loaded
2. **Additional sources** — extra sources the user selected at session creation
3. **Live additions** — sources imported during the session via the context panel

All layers make skills discoverable by Claude Code through the standard `.claude/skills/`, `.claude/commands/`, `.claude/agents/` directory structure.

### Visualization

Users should be able to see what's loaded in a session — not as raw files, but as extracted, readable metadata:

**In the session context panel**: A dedicated Skills section shows all loaded skills, commands, and agents across all sources. Each item displays its name, type badge, and source. Expandable per source to see what came from where. Items can be removed individually.

**In the marketplace**: Clicking a source shows a detail panel with all the skills, commands, and agents it contains — rendered with descriptions and metadata, not as a file tree.

### Authentication for Sources

Private repos and authenticated services (MCP servers) use the existing workspace credential system. If the workspace has GitHub/GitLab integrations configured, private source repos are cloned using those credentials via the git credential helper. MCP sources that require auth and TLS are handled through workspace integration configuration. No new auth fields in the manifest.

### Workflow Metadata

The runner's `/content/workflow-metadata` endpoint returns all discovered skills, commands, and agents across all loaded sources and built-in Claude Code skills. The frontend uses this to populate the Skills toolbar button and `/` autocomplete in the chat input.

---

## Local Usage (outside ACP)

### 1. Manual

```bash
git clone --depth 1 https://github.com/ambient-code/skills.git /tmp/skills
git clone --depth 1 https://github.com/opendatahub-io/ai-helpers.git /tmp/ai-helpers

claude \
  --add-dir /tmp/skills/bugfix \
  --add-dir /tmp/ai-helpers/helpers
```

### 2. Load-workflow skill

A meta-skill that reads a workflow's `ambient.json`, clones each source, and sets them up for Claude:

```
~/.claude/skills/load-workflow/SKILL.md
```

Usage:
```
/load-workflow https://github.com/ambient-code/workflows/tree/main/workflows/bugfix
```

The skill instructs Claude to:
1. Fetch the workflow's `ambient.json`
2. Clone each source to temp directories
3. Set up `.claude/` structures so skills are discoverable
4. The workflow's `CLAUDE.md` is loaded automatically

This makes ACP workflows portable — anyone with Claude Code can use them without ACP.

---

## Open Questions

1. **Skill storage path**: Should sources be loaded into the workspace root `.claude/` (simple, persisted) or as separate `add_dirs` per source (clean separation)? Need to investigate `add_dirs` limits and understand the tradeoffs.

2. **"Always add" defaults**: Should some workspace-level sources be auto-loaded into every session? How is this configured? Needs further discussion.
