---
name: pr-fixer
description: Trigger the PR Fixer GitHub Actions workflow to automatically fix a pull request (rebase, address review comments, run lints/tests, push fixes). Use when user types /pr-fixer <number>.
---

# PR Fixer Skill

Triggers the `pr-fixer.yml` GitHub Actions workflow to automatically fix a pull request. The workflow creates an ACP session that rebases the PR, evaluates reviewer comments (fixes valid issues, responds to invalid ones), runs lints and tests, and pushes the fixes.

## Usage

`/pr-fixer <pr-number>`

The PR number is required. Example: `/pr-fixer 1234`

## What It Does

1. **Validate prerequisites**
   - Confirm `gh` CLI is authenticated (`gh auth status`)
   - Detect the repo from the local git remote (`gh repo view --json nameWithOwner -q .nameWithOwner`)
   - Confirm the repo has a `pr-fixer.yml` workflow

2. **Dispatch the workflow**
   ```bash
   gh workflow run pr-fixer.yml -f pr_number=<N> --repo <owner/repo>
   ```

3. **Locate the triggered run**
   - Wait a few seconds for the run to register
   - Find it via:
     ```bash
     gh run list --workflow=pr-fixer.yml --repo <owner/repo> --limit 5 --json databaseId,status,createdAt,headBranch
     ```
   - Match the most recent run created after dispatch

4. **Print the run URL** immediately so the user has it:
   ```
   PR Fixer dispatched for PR #<N>
   Run: https://github.com/<owner/repo>/actions/runs/<run-id>
   PR:  https://github.com/<owner/repo>/pull/<N>

   Monitoring in background — you'll be notified when it completes.
   ```

5. **Spawn a background agent** to monitor the run:
   - Poll `gh run view <run-id> --repo <owner/repo> --json status,conclusion` every 30 seconds
   - When the run reaches a terminal state, notify with:
     - Run conclusion (success/failure/cancelled)
     - Session name and phase (parse from `gh run view <run-id> --repo <owner/repo> --json jobs` — look for the "Session summary" step output)
     - Whether commits were pushed (check `gh pr view <N> --repo <owner/repo> --json commits` count before and after)
     - Links to the GHA run and the PR

## Error Handling

- **No PR number provided**: Print usage: `/pr-fixer <pr-number>`
- **`gh` not authenticated**: "Error: GitHub CLI is not authenticated. Run `gh auth login` first."
- **Workflow not found**: "Error: No pr-fixer.yml workflow found in <owner/repo>. This repo may not have the PR Fixer workflow configured."
- **Run not found after dispatch**: "Warning: Could not locate the triggered run. Check manually: https://github.com/<owner/repo>/actions/workflows/pr-fixer.yml"

## When to Invoke This Skill

Invoke when users say things like:
- "/pr-fixer 1234"
- "Fix PR 1234"
- "Run the PR fixer on 1234"
- "Trigger pr-fixer for PR #1234"
