# Skills & Workflows: Discovery, Installation, and Usage

## Summary

A workflow is a `CLAUDE.md` prompt plus a list of sources in `ambient.json`. Skills are the atomic reusable unit. ACP automates the cloning and wiring. Locally, a `/load-workflow` skill or manual `--add-dir` does the same thing.

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
- **Structured object**: `{"url": "...", "branch": "...", "path": "..."}` — works with any Git host, branch is explicit
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

### What

A way to browse and find skills, workflows, and plugins from curated sources.

### Source Types

The scanner must support three types of sources:

1. **Claude Code plugins** — directories with `.claude-plugin/plugin.json` containing `skills/`, `commands/`, `agents/`, `hooks/`, `.mcp.json`. This is the primary format to follow and expect. Skills are namespaced as `plugin-name:skill-name`.

2. **Claude Code marketplace catalogs** — `marketplace.json` files listing plugins with their sources. Users could add the same marketplace from local Claude Code via `/plugin marketplace add`.

3. **Standalone repos with `.claude/`** — any Git repo containing `.claude/skills/`, `.claude/commands/`, `.claude/agents/`. Also supports root-level `skills/`, `commands/`, `agents/` (registry layout like ai-helpers).

### How

A cluster-level ConfigMap (`marketplace-sources`) holds available registry sources. The Marketplace page in the ACP UI shows:

- Browsable catalogs from each source with search and type filters
- Compact card tiles with name, description, type badge
- Detail panel on click with full description, source repo link
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

We follow Claude Code's plugin and skill formats as the standard. The [Agent Skills](https://agentskills.io) open standard that Claude Code implements is the closest cross-tool specification. Our catalog format normalizes to the same shape regardless of source type.

---

## Installation & Configuration

### Workspace Level

Items installed at the workspace level are stored in the ProjectSettings CR (`spec.installedItems`). These represent the workspace's **library** — what's available, not what's auto-loaded into every session.

When creating a session, users select which installed items to include. The workflow they choose pulls in its own skill dependencies from the `sources` array in `ambient.json`.

### Session Level

Skills can be added to a running session via the context panel:

- "Import Skills" in the Add Context dropdown
- Provide a Git URL + optional branch + path
- Backend clones, scans, writes skill files to `/workspace/file-uploads/.claude/`
- Claude discovers them via live change detection (already in `add_dirs`)
- Persisted via S3 state-sync on session suspend/resume

### Workflow Builder

A UI for composing workflows from standalone skills:

- Select skills from the workspace library or browse marketplace
- Each skill is a reference (source URL + path), not a copy
- Write the workflow prompt as `CLAUDE.md`
- Configure metadata in `ambient.json` (name, description, rubric)
- The `sources` array is built from selected skills
- Save as a workflow that can be:
  - Stored in the workspace
  - Exported as a Git repo
  - Exported as a Claude Code plugin

The key constraint: skills are never copied into the workflow. The `ambient.json` holds source references. At load time, the runner resolves dependencies and clones each source.

### How Selection Works

The workspace library is not auto-injected. Selection happens at session creation:

1. User picks a workflow (or "General chat" for none)
2. The workflow's `ambient.json` `sources` array declares dependencies — those are auto-loaded
3. User can optionally add standalone skills from the workspace library
4. The session CRD stores the workflow reference + any additional skill sources

This means:
- Installing 50 skills to the workspace doesn't bloat every session
- The workflow controls its own dependencies
- Users can augment with extras per session
- Workspace-level "always-on" skills could be supported via a flag but are not the default

---

## Usage in Sessions

### Loading

When a session starts, sources are loaded in layers:

1. **Workflow sources** — skills from the workflow's `ambient.json` `sources` array, cloned and added to `add_dirs`
2. **Additional standalone sources** — extra skills the user selected at session creation
3. **Live additions** — skills imported during the session via the context panel

All layers produce directories with `.claude/skills/`, `.claude/commands/`, `.claude/agents/` structure. Each directory is passed to the Claude Agent SDK as an `--add-dir`. Claude Code handles discovery from there.

The `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` env var is set so that `CLAUDE.md` files from add-dirs are also loaded.

### Authentication for Sources

Private repos and authenticated services (MCP servers) use the existing workspace credential system. If the workspace has GitHub/GitLab integrations configured, private source repos are cloned using those credentials via the git credential helper. MCP sources that require auth and TLS are handled through workspace integration configuration. No new auth fields in the manifest.

### Runtime Management

The session context panel shows:

- **Repositories** — Git repos cloned as working directories (existing)
- **Skills** — imported skills/commands/agents with type badges and source links
- **Uploads** — uploaded files (existing)

Users can add skill sources live (Import Skills button). The backend clones the source, writes files to `/workspace/file-uploads/.claude/`, and Claude picks them up immediately. Users can remove individual skills — the file is deleted and Claude stops seeing it.

### Workflow Metadata

The runner's `/content/workflow-metadata` endpoint returns all discovered skills, commands, and agents from:
- The active workflow's `.claude/` directory
- Any additional source directories
- `/workspace/file-uploads/.claude/` (live imports)
- Built-in Claude Code skills (batch, simplify, debug, claude-api, loop)

The frontend uses this to populate the Skills toolbar button and `/` autocomplete in the chat input.

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

A meta-skill that reads a workflow's `ambient.json`, clones each source, and passes them as `--add-dir`:

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
3. Symlink `.claude/` structures into the project
4. The workflow's `CLAUDE.md` is loaded automatically

This makes ACP workflows portable — anyone with Claude Code can use them without ACP.

---

## Open Questions

1. **Skill versioning**: Sources reference branches today. Should we support tags or SHAs for pinning? What happens when a skill source updates — do sessions get the latest on next start?

2. **Plugin format**: Should workflow export produce a Claude Code plugin (`plugin.json`)? Pros: portable, namespaced, versioned. Cons: plugins cache/copy files which breaks the dynamic reference model.

3. **RHAI alignment**: How does this map to RHAIRFE-1370 (Skills Registry)? Our `sources` format and marketplace could inform the product's in-cluster registry design.

4. **Security**: How do we verify skill sources haven't been tampered with? Git commit SHAs provide content-addressable verification. Enterprise customers may need signed manifests.

5. **Workspace defaults**: Should some workspace-level items be "always-on" (loaded in every session regardless of workflow)? Or should this be handled via org-level Claude Code managed settings?
