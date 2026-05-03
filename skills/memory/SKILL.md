---
name: memory
description: >
  Manage the auto-memory system for this project. Search, audit, prune, and
  create memories with proper frontmatter. Use when you need to find a past
  decision, check if memories are stale, clean up duplicates, add a new
  memory, or understand what context is available. Triggers on: "check
  memory", "what do we remember about", "find the memory about", "clean up
  memories", "audit memories", "add to memory", "is there a memory for".
---

# Memory Management

Manage the auto-memory system for the Ambient Code Platform project.

## Usage

```text
/memory                    # Show summary of all memories
/memory search <query>     # Search for a topic
/memory audit              # Check for stale/duplicate memories
/memory prune              # Remove stale memories (with confirmation)
/memory add <topic>        # Create a new memory
```

## User Input

```text
$ARGUMENTS
```

Parse the subcommand from `$ARGUMENTS`. Default to summary if empty.

## Memory Location

All memory files live at:
```text
$HOME/.claude/projects/<project-slug>/memory/
```

Use the active project's slug (the repo path with `/` replaced by `-`).

The index file is `MEMORY.md` in that directory.

## Subcommands

### `/memory` — Summary

1. Read `MEMORY.md` to get the index
2. Count total memories by type (user, feedback, project, reference)
3. List the most recently modified files
4. Report: total count, breakdown by type, last modified dates

### `/memory search <query>`

1. Read `MEMORY.md` for the index
2. Grep through all memory files for the query term
3. Read matching files and show relevant excerpts
4. Report: matching files with frontmatter (name, type, description)

### `/memory audit`

Check for quality issues:

1. **Stale memories** — project/reference memories older than 3 months may be outdated
2. **Duplicate memories** — similar names or descriptions across files
3. **Missing frontmatter** — files without proper `name`, `description`, `type` fields
4. **Orphaned files** — memory files not referenced in `MEMORY.md`
5. **Broken links** — `MEMORY.md` entries pointing to nonexistent files
6. **Oversized index** — `MEMORY.md` approaching the 200-line truncation limit

Report each issue with the file path and suggested action.

### `/memory prune`

1. Run the audit checks
2. Present findings to the user
3. For each stale/duplicate/orphaned memory, ask: keep, update, or delete?
4. Execute confirmed deletions
5. Update `MEMORY.md` index accordingly

Never delete without explicit confirmation.

### `/memory add <topic>`

1. Ask the user what they want to remember (if not clear from context)
2. Determine the memory type (user, feedback, project, reference)
3. Create a new file with proper frontmatter:

```markdown
---
name: <descriptive name>
description: <one-line description for relevance matching>
type: <user|feedback|project|reference>
---

<memory content>
```

4. Add an entry to `MEMORY.md`
5. Verify the entry was added correctly

## Memory Types

| Type | What to store | When to save |
|------|--------------|-------------|
| **user** | Role, preferences, knowledge | Learning about the user |
| **feedback** | Corrections, validated approaches | User corrects or confirms approach |
| **project** | Decisions, initiatives, deadlines | Learning project context |
| **reference** | Pointers to external systems | Discovering external resources |

## What NOT to Store

- Code patterns derivable from reading current code
- Git history (use `git log`)
- Debugging solutions (the fix is in the code)
- Anything in CLAUDE.md
- Ephemeral task details
- Secrets or credentials (API keys, tokens, passwords, private keys, OAuth secrets)
- Sensitive personal data unless explicitly required

## Quality Guidelines

- Keep `MEMORY.md` under 200 lines (truncation risk)
- Each entry: one line, under 150 characters
- Update existing memories rather than creating duplicates
- Include absolute dates (not "next Thursday")
- For feedback/project types: include **Why:** and **How to apply:** lines
