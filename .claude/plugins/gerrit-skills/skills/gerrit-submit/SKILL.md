---
name: gerrit-submit
description: Use when submitting or preparing a Gerrit change for merge — checking submit requirements, rebasing, resolving comments, and submitting. Triggers on "submit this Gerrit change", "merge this CL", "is this change ready to submit", "check submit requirements".
---

# Gerrit Change Submission

Guide for preparing and submitting Gerrit changes for merge.

## Prerequisites

The Gerrit MCP server must be available and the user must have submit permissions on the target project.

## Pre-submission checklist

Before submitting, verify all requirements are met using `mcp__gerrit__fetch_gerrit_change`:

1. **Labels satisfied** — All required labels have sufficient votes:
   - Code-Review: typically needs +2
   - Verified: typically needs +1
   - Any project-specific labels (check the change's `submit_requirements`)

2. **No unresolved comments** — All inline comments must be resolved or addressed

3. **Mergeable** — The change can be merged without conflicts (check the `mergeable` field)

4. **Up to date** — The change is based on the current branch head (rebase if needed)

## Workflow

### 1. Check readiness

Fetch the change and examine:
- `submittable` field — true if all requirements are met
- `submit_requirements` — list of conditions and their status
- `labels` — current votes
- Unresolved comment count

If not submittable, report what's missing and stop.

### 2. Rebase if needed

If the change is not based on the current branch head:
- The Gerrit MCP server's submit operation may auto-rebase depending on project settings
- If conflicts exist, the change owner must resolve them manually and upload a new patch set

### 3. Submit

Use `mcp__gerrit__submit_gerrit_review` with the appropriate approval labels if you need to add final votes before submitting.

Submission itself happens through the Gerrit REST API. If the MCP server provides a submit tool, use it. Otherwise, inform the user to submit via the Gerrit web UI.

### 4. Post-submit

After successful submission:
- Verify the change shows status MERGED
- Check if there are dependent changes that can now be submitted (submitted_together)
- Notify the user of the merge commit hash if available

## Common issues

- **Missing votes**: Check which labels are blocking and who can provide them
- **Merge conflicts**: The owner needs to rebase and resolve conflicts locally
- **Submit rules**: Some projects have custom submit rules (e.g., require Verified from CI)
- **Permission denied**: The user may not have submit permissions — check with project admin
