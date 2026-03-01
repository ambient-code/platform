---
name: session-config
description: Generates ACP session-config repositories interactively. Use when creating or refreshing a session-config repo with CLAUDE.md, .claude/rules, skills, agents, settings.json, .mcp.json, hooks, .ambient/workflows, or plugin marketplaces. Presents all configuration surfaces and scaffolds self-documented files.
---

# Session Config Generator

Generate a session-config repository for the Ambient Code Platform. The repo is cloned into `/workspace/` at session startup via `cp -rn` (workspace repo files take precedence).

**Reference template**: https://github.com/ambient-code/session-config-reference

## Process

1. Ask which configuration surfaces to include (use AskUserQuestion with multiSelect)
2. For each selected surface, ask about content and preferences
3. Generate files with inline documentation so the repo is self-explanatory
4. Create a README explaining overlay behavior and structure

## Configuration Surfaces

Present ALL surfaces. Use AskUserQuestion multiSelect to let the user pick.

| # | Surface | Location | Purpose |
|---|---------|----------|---------|
| 1 | **CLAUDE.md** | Root | Session instructions (code style, testing, security, tools) |
| 2 | **settings.json** | `.claude/` | Permissions, model defaults, sandbox, env vars |
| 3 | **Rules** | `.claude/rules/` | Path-scoped modular instructions (markdown + paths frontmatter) |
| 4 | **Skills** | `.claude/skills/<name>/SKILL.md` | Invocable commands via `/skill-name` |
| 5 | **Agents** | `.claude/agents/<name>.md` | Custom subagents with scoped tools |
| 6 | **MCP Servers** | `.mcp.json` | MCP server configuration (stdio, http, oauth) |
| 7 | **Hooks** | `.claude/settings.json` hooks key | Lifecycle automation (PreToolUse, PostToolUse, etc.) |
| 8 | **Workflows** | `.ambient/workflows/` | ACP workflow definitions with optional rubrics |
| 9 | **Plugin Marketplaces** | `.claude/settings.json` | Plugin sources, enablement, and marketplace declarations |

For full schemas, frontmatter fields, and examples for each surface, see [reference.md](reference.md).

## CLAUDE.md Sections to Offer

- Project context, tech stack, architecture
- Code style and naming conventions
- Commit standards (conventional commits)
- Testing practices and frameworks
- Code review priorities (security > correctness > perf > maintainability)
- Workspace conventions (`artifacts/`, `repos/`, `file-uploads/`)
- Tool preferences (package managers, container engines, linters)
- Security policies (secret handling, input validation)
- Custom sections

## Rules Categories to Offer

- Security (secrets, validation, auth, error exposure)
- Language-specific (Go, Python, TypeScript, Rust)
- Frontend (components, accessibility, state)
- Backend (API design, error handling, DB patterns)
- Testing (structure, mocking, coverage)
- Infrastructure (K8s, Docker, CI/CD)
- Documentation (docstrings, ADRs)
- Custom

## Skill Archetypes to Offer

- Code review (severity-grouped findings)
- Commit/PR generation
- Test generation / coverage analysis
- Documentation generation
- Deployment scripts / environment checks
- Security / dependency audit
- Custom

## Agent Archetypes to Offer

- Code reviewer (read-only: Read, Glob, Grep)
- Test writer (Read, Glob, Grep, Bash, Write)
- Documentation writer (Read, Glob, Grep, Write)
- Security auditor (read-only)
- Database agent (with MCP db server)
- Researcher (web search + codebase exploration)
- Custom

## Common MCP Servers to Offer

- **memory** — `@modelcontextprotocol/server-memory` (zero config knowledge graph)
- **filesystem** — `@modelcontextprotocol/server-filesystem`
- **github** — GitHub Copilot MCP or `@modelcontextprotocol/server-github`
- **postgres/sqlite** — `@bytebase/dbhub`
- **slack** — `@anthropic/mcp-server-slack`
- **sentry** — `https://mcp.sentry.dev/mcp` (http type)
- **linear** — `https://mcp.linear.app/sse`
- Custom (user provides details)

## Hook Patterns to Offer

- Linting on edit (PostToolUse on Edit)
- Command validation (PreToolUse on Bash)
- Session setup (on-session-start)
- Security gate (block dangerous ops)
- Custom

## Workflow Templates to Offer

- Code review — analyze repo, write findings
- Bug triage — investigate issues, propose fixes
- Documentation — generate project docs
- Onboarding — guided project walkthrough
- Custom

## Plugin Marketplace Options to Offer

- **odh-ai-helpers** — `opendatahub-io/ai-helpers` (FIPS compliance, Python packaging, Jira, GitLab, vLLM tools) — **enable by default**
- **Custom GitHub marketplace** — user provides `{owner}/{repo}` with marketplace.json
- **Custom git URL** — any git repo containing a marketplace
- **Custom npm package** — npm-distributed marketplace
- **Plugin enablement presets** — pre-enable/disable specific plugins from declared marketplaces

## File Generation Rules

- Every file MUST include inline comments explaining what it does
- Include frontmatter schema reference as comments in rules, skills, agents
- README.md must explain overlay behavior and link to reference repo
- Use the full schema from [reference.md](reference.md) when generating files

## Generated Repo Structure

```
session-config-repo/
├── README.md
├── CLAUDE.md
├── .mcp.json
├── .claude/
│   ├── settings.json
│   ├── rules/
│   │   └── *.md
│   ├── skills/
│   │   └── <name>/SKILL.md
│   └── agents/
│       └── <name>.md
└── .ambient/
    └── workflows/
        └── <name>.json
```

## Overlay Behavior

Include in generated README:

> Contents are copied into `/workspace/` using `cp -rn` (no-clobber).
> Config repo files are copied first; workspace repo files override at the same path.
> Use this for org-wide defaults that individual repos can override.
