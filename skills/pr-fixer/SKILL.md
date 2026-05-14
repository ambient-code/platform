---
name: pr-fixer
description: >
  Trigger the Amber Handler to automatically fix a pull request (rebase, address
  review comments, run lints/tests, push fixes). Posts @ambient-code on the PR
  to trigger the fix prompt. Use when user types /pr-fixer <number>, says "fix PR",
  "run pr-fixer", "address PR comments", or "auto-fix pull request".
---

# PR Fixer Skill

Triggers the Amber Handler (`amber-issue-handler.yml`) to automatically fix a pull request. Posting `@ambient-code` on a PR triggers the fix prompt, which creates an ACP session that rebases the PR, evaluates reviewer comments (fixes valid issues, responds to invalid ones), runs lints and tests, and pushes the fixes.

## Usage

`/pr-fixer <pr-number>`

The PR number is required. Example: `/pr-fixer 1234`

## What It Does

1. **Validate prerequisites**
   - Confirm `gh` CLI is authenticated (`gh auth status`)
   - Detect the repo from the local git remote (`gh repo view --json nameWithOwner --jq .nameWithOwner`)

2. **Pre-flight PR check**
   - Check the PR exists and its state:
     ```bash
     gh pr view <N> --repo <owner/repo> --json state,isCrossRepository --jq '{state, isCrossRepository}'
     ```
   - `gh pr view` returns state in **uppercase**: `OPEN`, `CLOSED`, or `MERGED`. A non-zero exit code (with stderr error) means the PR does not exist.
   - If the exit code is non-zero: abort with "PR #N not found in <owner/repo>."
   - If state is `CLOSED` or `MERGED`: abort with "PR #N is already <state>. Nothing to fix." (use the actual uppercase value returned)
   - Only proceed if state is exactly `OPEN`.
   - If PR is from a fork (`isCrossRepository: true`), warn: "PR #N is from a fork. The Amber Handler skips fork PRs. Push the branch to the org repo and re-open the PR, or fix locally."

3. **Trigger the fix**
   ```bash
   gh pr comment <N> --repo <owner/repo> --body "@ambient-code"
   ```

4. **Locate the triggered run**
   - Wait a few seconds for the run to register
   - Find it via:
     ```bash
     gh run list --workflow=amber-issue-handler.yml --repo <owner/repo> --limit 5 --json databaseId,status,createdAt,event
     ```
   - Match the most recent `issue_comment`-triggered run created after the comment

5. **Print the run URL** immediately so the user has it:
   ```
   PR Fixer dispatched for PR #<N>
   Run: https://github.com/<owner/repo>/actions/runs/<run-id>
   PR:  https://github.com/<owner/repo>/pull/<N>

   Monitoring in background — you'll be notified when it completes.
   ```

6. **Spawn a background agent** to monitor the run (30-minute timeout):
   - Poll `gh run view <run-id> --repo <owner/repo> --json status,conclusion` every 30 seconds
   - If 30 minutes elapse without completion, notify: "PR Fixer timed out after 30 minutes. Check the run manually."
   - When the run reaches a terminal state, notify with:
     - Run conclusion (success/failure/cancelled)
     - Session name and phase (parse from `gh run view <run-id> --repo <owner/repo> --json jobs` — look for the "Session summary" step output)
     - Whether commits were pushed (check `gh pr view <N> --repo <owner/repo> --json commits` count before and after)
     - Links to the GHA run and the PR

## Error Handling

- **No PR number provided**: Print usage: `/pr-fixer <pr-number>`
- **`gh` not authenticated**: "Error: GitHub CLI is not authenticated. Run `gh auth login` first."
- **Fork PR detected**: "Warning: PR #N is from a fork. The Amber Handler skips fork PRs. Push the branch to the org repo and re-open the PR, or fix locally."
- **Run not found after dispatch**: "Warning: Could not locate the triggered run. Check manually: https://github.com/<owner/repo>/actions/workflows/amber-issue-handler.yml"
- **All session steps skipped**: The Amber Handler's fix steps have conditions (non-fork, correct prompt type). If steps are skipped, check the run logs for why.

## When to Invoke This Skill

Invoke when users say things like:
- "/pr-fixer 1234"
- "Fix PR 1234"
- "Run the PR fixer on 1234"
- "Trigger pr-fixer for PR #1234"
