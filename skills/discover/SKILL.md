---
name: discover
description: >
  Discover domain-specific skills and context before starting any coding or
  spec work. This is the mandatory first step for every task — no matter how
  trivial. Triggers on: any code change, spec change, implementation task,
  bug fix, refactor, feature work, or documentation update that touches
  component code or specs.
---

# Discover Domain Skills

Before writing or changing anything, discover what skills and context are available for the domains you're about to work in.

## When to Use

Every time. This skill is the entry point for all coding and spec work. Do not skip it, even for one-line fixes.

## User Input

```text
$ARGUMENTS
```

If provided, use the input to narrow which domains to check. Otherwise, infer from the task context.

## Steps

### 1. Identify Relevant Domains

Map the work to one or more capability domains:

| Domain | When working on |
|--------|----------------|
| `sessions` | Session lifecycle, data model, API resources, messages, events |
| `control-plane` | Operator, reconciliation, scheduling, runner provisioning |
| `agents` | Agent model, runner, runtime registry, prompts |
| `integrations` | MCP server, Gerrit, external services, feature flags |
| `frontend` | UI components, pages, React Query hooks |
| `auth` | OAuth, credentials, RBAC, tokens |
| `projects` | Project management, settings, workspaces |
| `cli` | TUI, acpctl commands |
| `specs` | specs |

If unsure, check which `specs/` subdirectories exist:

```bash
ls -d specs/*/
```

### 2. Visit Each Domain

For every relevant domain, `cd` into its spec directory and list available skills:

```bash
cd specs/{domain}
ls .claude/skills/ 2>/dev/null
ls skills/ 2>/dev/null
```

Read the SKILL.md of any discovered skill that looks relevant to the task at hand.

### 3. Load Domain Specs

While in the domain directory, check for specs that describe the desired state of what you're about to change:

```bash
ls *.spec.md 2>/dev/null
```

Read the relevant spec before implementing — code reconciles against specs, not the other way around.

### 4. Check Standards

Load the standards for the component you're modifying:

```bash
ls specs/standards/*/
```

Pick the relevant domain (backend, frontend, control-plane, security, platform).

### 5. Report and Proceed

Summarize what you found:
- Which domains are involved
- Which domain skills were discovered (if any)
- Which specs describe the area being changed
- Which standards apply

Do not output irrelevant context. Be extremely brief, like:
> I will use skills `x` and `y` for this task.

Then proceed with the task, using the discovered skills and context.
