---
name: gerrit-create
description: Use when creating or updating Gerrit changes — pushing code for review, amending patch sets, setting reviewers and topics. Triggers on "push to Gerrit", "create a Gerrit change", "upload for review", "update the patch set", "add reviewers to this change".
---

# Creating and Updating Gerrit Changes

Guide for pushing code changes to Gerrit for review.

## Prerequisites

The Gerrit MCP server must be available and git must be configured with the appropriate remote.

## Creating a new change

Gerrit changes are created by pushing commits to a special ref:

```
git push origin HEAD:refs/for/<target-branch>
```

### Before pushing

1. **Commit message format**: Gerrit requires a `Change-Id` footer in the commit message. If the Gerrit commit-msg hook is installed, it adds this automatically. If not:
   ```
   Subject line (50 chars max)

   Body explaining what and why (wrapped at 72 chars).

   Change-Id: I<40-char-hex>
   ```

2. **Single commit per change**: Each commit becomes one Gerrit change. Squash work-in-progress commits before pushing.

3. **Target branch**: Usually `main` or `master`, but verify the project's default branch.

### Push options

Add metadata via push options:
```
git push origin HEAD:refs/for/main \
  -o topic=my-feature \
  -o r=reviewer@example.com \
  -o cc=watcher@example.com \
  -o wip
```

- `topic=<name>` — group related changes
- `r=<email>` — add reviewer
- `cc=<email>` — add CC
- `wip` — mark as work-in-progress
- `ready` — mark as ready for review
- `hashtag=<tag>` — add hashtag

## Updating an existing change

To upload a new patch set for an existing change:

1. Amend the commit (keep the same Change-Id):
   ```
   git commit --amend
   ```

2. Push again:
   ```
   git push origin HEAD:refs/for/main
   ```

Gerrit matches the Change-Id to the existing change and creates a new patch set.

## Managing change metadata

After a change exists, use `mcp__gerrit__submit_gerrit_review` to:
- Add comments explaining what changed in the new patch set
- Vote on your own change (if allowed by project settings)
- Reply to reviewer comments

Use `mcp__gerrit__fetch_gerrit_change` to verify the change was created/updated correctly.

## Common issues

- **Missing Change-Id**: Install the commit-msg hook: `gitdir=$(git rev-parse --git-dir); scp -p -P 29418 <user>@<host>:hooks/commit-msg ${gitdir}/hooks/`
- **Push rejected (not permitted)**: Check project permissions and target branch
- **Merge conflict on push**: Rebase on the target branch first
- **Change already merged**: The Change-Id was reused — create a new commit with a fresh Change-Id
