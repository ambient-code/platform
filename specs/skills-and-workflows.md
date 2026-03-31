# Skills & Workflows: Discovery, Installation, and Usage

## Summary

A workflow is a prompt plus a list of skill sources. Skills are the atomic reusable unit. ACP automates the cloning and wiring; locally, the same manifest format works with a simple skill or script. The manifest format is the contract — ACP and local tooling are just consumers.

---

## Core Concepts

### Skill

The atomic unit of reusable capability. A directory containing a `SKILL.md` file with YAML frontmatter and markdown instructions. Claude Code discovers skills from `.claude/skills/{name}/SKILL.md` in the working directory, parent directories, `--add-dir` directories, and plugins.

Skills have live change detection in `--add-dir` directories — place a new skill file and Claude discovers it immediately without restarting. Skills are invoked via `/skill-name` or auto-triggered by Claude based on the description in frontmatter.

Commands (`.claude/commands/{name}.md`) and agents (`.claude/agents/{name}.md`) follow the same discovery pattern and are treated as peers to skills throughout this spec. "Skill" is used as shorthand for all three unless distinction matters.

### Workflow

A workflow is two things:

1. **A prompt** — the directive, methodology, persona instructions. What today lives in `CLAUDE.md` + `.ambient/ambient.json` (system prompt, startup prompt, phase descriptions).
2. **A list of skill sources** — references to skills, commands, and agents from various Git repos or registries. Not embedded copies — references resolved at load time.

A workflow does not contain skills. It references them. The bug-fix workflow becomes:

```yaml
name: Bug Fix
description: Systematic bug resolution with phased approach
prompt: |
  You are a systematic bug fixer. Follow these phases:
  1. Use /assess to understand the issue
  2. Use /reproduce to create a failing test
  3. Use /diagnose to find the root cause
  4. Use /fix to implement the minimal fix
  5. Use /test to verify the fix
  6. Use /review to self-review before PR
sources:
  - url: https://github.com/ambient-code/skills.git
    path: bugfix/assess
  - url: https://github.com/ambient-code/skills.git
    path: bugfix/reproduce
  - url: https://github.com/ambient-code/skills.git
    path: bugfix/diagnose
  - url: https://github.com/ambient-code/skills.git
    path: bugfix/fix
  - url: https://github.com/ambient-code/skills.git
    path: bugfix/test
  - url: https://github.com/ambient-code/skills.git
    path: bugfix/review
  - url: https://github.com/opendatahub-io/ai-helpers.git
    path: helpers/skills/jira-activity
```

Skills are the reusable atoms. Workflows are recipes. The same skill can appear in multiple workflows.

### Agent (future)

A persona — prose defining what an agent is responsible for. "Backend Agent", "Security Agent", "PM Agent". An Agent uses workflows and standalone skills to accomplish its goals. An Agent is a session template with a personality:

```
Agent = Persona (prompt/directive) + Workflows (skill bundles) + Standalone skills
```

A "Bug Fix Agent" = bug-fix persona + bug-fix workflow skills + any additional skills. Same skills reusable by different Agents with different motivations.

### Manifest Format

The workflow manifest (`workflow.yaml` or `ambient.json`) is the core deliverable. It defines prompt + skill sources. Multiple consumers read the same format:

- **ACP runner**: clones sources, adds to `add_dirs`, injects prompt
- **Local script/skill**: clones sources to temp dirs, passes as `--add-dir`
- **Claude Code plugin**: `SessionStart` hook reads manifest, sets up environment

The format must support:
- Prompt text (inline or file reference)
- Skill sources (Git URL + branch + path)
- Optional: system prompt, startup prompt, rubric
- Optional: environment variables, MCP server configs

---

## Discovery

### What

A way to browse and find skills, workflows, and plugins from curated sources.

### How

A cluster-level ConfigMap (`marketplace-sources`) holds available registry sources. Each source is one of:

- A Git repo with a `data.json` catalog (like ai-helpers at `https://opendatahub-io.github.io/ai-helpers/data.json`)
- A Claude Code `marketplace.json` (native plugin marketplace format)
- A direct Git repo with skills/commands/agents in `.claude/` or at root level

The Marketplace page in the ACP UI shows:

- Browsable catalogs from each source with search and type filters (skill / command / agent / workflow)
- Compact card tiles with name, description, type badge
- Detail panel on click with full description, source repo link, allowed tools
- "Import Custom" to scan any Git URL and discover items
- Direct one-click install to workspace

### Scanning

When given a Git URL (from marketplace or custom), the backend:

1. Shallow clones the repo
2. Applies optional path filter (subdirectory)
3. Scans for items in both patterns:
   - `.claude/skills/*/SKILL.md`, `.claude/commands/*.md`, `.claude/agents/*.md`
   - `skills/*/SKILL.md`, `commands/*.md`, `agents/*.md` (registry layout like ai-helpers)
4. Checks for `.ambient/ambient.json` (indicates this is a workflow)
5. Checks for `CLAUDE.md` (indicates project instructions)
6. Returns discovered items with frontmatter metadata

### Alignment with Claude Code

Adopting Claude Code's `marketplace.json` format where possible makes ACP skills portable. Users could add the same marketplace from local Claude Code via `/plugin marketplace add`. The catalog format should normalize to the same shape regardless of source type.

---

## Installation & Configuration

### Workspace Level

Items installed at the workspace level are stored in the ProjectSettings CR (`spec.installedItems`). These represent the workspace's **library** — what's available, not what's auto-loaded into every session.

When creating a session, users select which installed items to include. The workflow they choose may also pull in its own skill dependencies from the manifest.

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
- Define the workflow prompt (methodology, phases, instructions)
- Optionally configure: system prompt, startup prompt, rubric
- Save as a workflow manifest that can be:
  - Stored in the workspace
  - Exported as a Git repo
  - Exported as a Claude Code plugin (`plugin.json`)

The key constraint: skills are never copied into the workflow. The manifest holds references. At load time, the runner resolves dependencies and clones each source.

### How Selection Works

The workspace library is not auto-injected. Selection happens at session creation:

1. User picks a workflow (or "General chat" for none)
2. The workflow manifest declares its skill dependencies — those are auto-loaded
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

1. **Workflow sources** — skills referenced in the workflow manifest, cloned and added to `add_dirs`
2. **Additional standalone sources** — extra skills the user selected at session creation
3. **Live additions** — skills imported during the session via the context panel

All layers produce directories with `.claude/skills/`, `.claude/commands/`, `.claude/agents/` structure. Each directory is passed to the Claude Agent SDK as an `--add-dir`. Claude Code handles discovery from there.

The `CLAUDE_CODE_ADDITIONAL_DIRECTORIES_CLAUDE_MD=1` env var is set so that `CLAUDE.md` files from add-dirs are also loaded (workflow instructions, skill documentation).

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

The same workflow manifest works locally. Two approaches:

### 1. Manual (no tooling)

```bash
git clone --depth 1 https://github.com/ambient-code/skills.git /tmp/skills
git clone --depth 1 https://github.com/opendatahub-io/ai-helpers.git /tmp/ai-helpers

claude \
  --add-dir /tmp/skills/bugfix \
  --add-dir /tmp/ai-helpers/helpers
```

### 2. Load-workflow skill

A meta-skill that reads a workflow manifest and sets up the environment:

```
~/.claude/skills/load-workflow/SKILL.md
```

Usage:
```
/load-workflow https://github.com/ambient-code/workflows/tree/main/workflows/bugfix
```

The skill instructs Claude to:
1. Fetch the workflow manifest
2. Clone each skill source to temp directories
3. Symlink `.claude/skills/` structures into the project
4. Apply the workflow prompt

This makes ACP workflows portable — anyone with Claude Code can use them without ACP. The skill ships as part of ai-helpers or as its own standalone skill.

### 3. Claude Code plugin (future)

A plugin with a `SessionStart` hook that reads workflow manifests:

```json
{
  "name": "ambient-workflows",
  "hooks": {
    "SessionStart": [{
      "type": "command",
      "command": "${CLAUDE_PLUGIN_ROOT}/scripts/load-workflow.sh"
    }]
  }
}
```

The hook script reads a `workflow.yaml` from the project root (if present), clones skill sources, and symlinks them into place. Fully automated, no user action needed.

---

## Manifest Format (proposed)

```yaml
# workflow.yaml or ambient.json
name: Bug Fix
description: Systematic bug resolution with phased approach

# The prompt/instructions — can be inline or reference a file
prompt: |
  You are a systematic bug fixer...

# Optional: injected as system prompt (short, persona-level)
systemPrompt: You are an expert software engineer focused on fixing bugs systematically.

# Optional: sent as first message when session starts
startupPrompt: |
  Welcome! I'm ready to help you fix a bug.
  Use /assess to get started, or describe the issue directly.

# Skill sources — cloned and added to add_dirs
sources:
  - url: https://github.com/ambient-code/skills.git
    branch: main
    path: bugfix/assess
  - url: https://github.com/ambient-code/skills.git
    path: bugfix/reproduce
  - url: https://github.com/opendatahub-io/ai-helpers.git
    path: helpers/skills/jira-activity
  - url: https://github.com/my-org/internal-skills.git
    branch: v2
    path: code-review

# Optional: quality evaluation criteria
rubric:
  activationPrompt: After completing the fix, evaluate your work
  criteria:
    - name: Root cause identified
      weight: 0.3
    - name: Minimal change
      weight: 0.2
    - name: Tests added
      weight: 0.3
    - name: No regressions
      weight: 0.2
```

This format is consumed by:
- ACP runner (production)
- `/load-workflow` skill (local)
- Claude Code plugin hook (automated local)
- Workflow builder UI (creation/editing)

---

## Open Questions

1. **Manifest format**: YAML vs JSON? Should we extend `ambient.json` or create a new `workflow.yaml`? The existing `ambient.json` has `name`, `description`, `systemPrompt`, `startupPrompt`, `rubric` — we'd add `sources` and `prompt`.

2. **Skill versioning**: Sources reference branches today. Should we support tags or SHAs for pinning? What happens when a skill source updates — do sessions get the latest on next start?

3. **Plugin format**: Should workflow export produce a Claude Code plugin (`plugin.json`)? Pros: portable, namespaced, versioned. Cons: plugins cache/copy files which breaks the dynamic reference model.

4. **RHAI alignment**: How does this map to RHAIRFE-1370 (Skills Registry)? Our manifest format and marketplace could inform the product's in-cluster registry design.

5. **Security**: How do we verify skill sources haven't been tampered with? Git commit SHAs provide content-addressable verification. Enterprise customers may need signed manifests.

6. **Workspace defaults**: Should some workspace-level items be "always-on" (loaded in every session regardless of workflow)? Example: company-wide coding standards skill. Or should this be handled via org-level Claude Code managed settings?
