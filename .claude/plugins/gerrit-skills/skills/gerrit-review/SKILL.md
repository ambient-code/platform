---
name: gerrit-review
description: Use when reviewing a Gerrit change — fetching change details, reading diffs, posting inline comments, and voting with labels. Triggers on "review this Gerrit change", "review change 12345", "look at this Gerrit CL", "code review on Gerrit".
---

# Gerrit Code Review

Guide for reviewing changes on a Gerrit instance using the Gerrit MCP server tools.

## Prerequisites

The Gerrit MCP server must be available (credentials configured via Integrations page). If `mcp__gerrit__*` tools are not listed, the user needs to connect their Gerrit instance first.

## Workflow

### 1. Fetch the change

Use `mcp__gerrit__fetch_gerrit_change` with the change ID (numeric or change-Id).

Extract from the response:
- **Subject and commit message** — understand what the change does
- **Owner** — who authored it
- **Status** — NEW (open), MERGED, ABANDONED
- **Current patch set number** — the latest revision
- **Files changed** — list of modified files with insertions/deletions
- **Existing comments** — what reviewers have already said
- **Labels** — current votes (Code-Review, Verified, etc.)

### 2. Read the diffs

Use `mcp__gerrit__fetch_patchset_diff` to compare patch sets or view the current diff.

- To see the full change: compare patch set 1 vs current (or base vs current)
- To see what changed between review rounds: compare two specific patch sets
- Filter by file path if the change is large — focus on the most impactful files first

### 3. Analyze and form review feedback

Apply standard code review principles:
- Correctness — does the logic do what the commit message claims?
- Edge cases — missing null checks, boundary conditions, error handling
- Style — follows project conventions (check CLAUDE.md if available)
- Security — no credentials in code, proper input validation, no injection vectors
- Performance — unnecessary allocations, N+1 queries, missing caching

### 4. Post the review

Use `mcp__gerrit__submit_gerrit_review` to post feedback:

- **message**: A summary of your review findings (top-level comment)
- **labels**: Vote appropriately:
  - `Code-Review +1` — looks good but someone else should also review
  - `Code-Review +2` — approved, ready to submit (only if you have authority)
  - `Code-Review -1` — needs changes before merging
  - `Code-Review -2` — blocks submission (serious issues)
  - `Verified +1` — tests pass, builds clean
- **comments**: Inline comments on specific files and lines

### Gerrit conventions

- **Attention set**: After reviewing, the change owner gets added to the attention set automatically. Don't manually manage attention set unless needed.
- **Patchset iteration**: When the owner uploads a new patch set addressing your comments, re-review by comparing the previous and new patch sets (`fetch_patchset_diff`), not re-reading the entire change.
- **Resolved comments**: In Gerrit, comments can be marked as "resolved" or "unresolved". Focus on unresolved comments when re-reviewing.
- **Vote semantics**: Code-Review -2 is a hard block — use it only for architectural issues or security problems, not style nits.

## Error handling

- If the change ID is not found, verify the instance name and change number
- If authentication fails, suggest the user reconnect via the Integrations page
- If the MCP server is not available, check that Gerrit credentials are configured
