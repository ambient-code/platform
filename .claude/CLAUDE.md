
# --- WORKER PROTOCOL ---
# You are a worker session managed by an orchestrator.
# Your label: upstream-sdk
# Your mailbox: /Users/gkrumbac/Documents/vTeam/.claude/orchestrator/mailboxes/upstream-sdk

## On startup and at natural breakpoints (after commits, before PRs, when stuck):
1. Read `/Users/gkrumbac/Documents/vTeam/.claude/orchestrator/mailboxes/upstream-sdk/INBOX.md` for new instructions from the orchestrator
2. Act on any instructions found there

## After each significant milestone (task complete, PR created, blocker hit):
Update `/Users/gkrumbac/Documents/vTeam/.claude/orchestrator/mailboxes/upstream-sdk/STATUS.md` with this format:

# Status: upstream-sdk

Phase: <planning|implementing|testing|pr-open|blocked|done>
Updated: <ISO timestamp>
Blockers: <none | description>
PR: <none | URL>
Summary: <1-2 sentence description of current state>

## Workflow:
1. Investigate the issue thoroughly — read relevant code, understand the problem
2. Make a plan — write it out clearly
3. Ask clarifying questions if anything is ambiguous (use AskUserQuestion)
4. Implement the fix
5. Run tests: frontend (npx vitest run), backend (make test), runner (python -m pytest tests/)
6. Check e2e flows are not impacted
7. Lint: frontend (npm run build must pass with 0 errors 0 warnings), backend/operator (gofmt, go vet, golangci-lint), runner (ruff)
8. Commit with conventional commit format
9. Push to your feature branch
10. Create a PR with gh pr create
11. Open the PR in the browser with: open <PR_URL>

## Rules:
- Always update STATUS.md before stopping
- If you receive a "continue" message in INBOX, resume where you left off
- If you hit a blocker you cannot resolve, update STATUS.md and stop
- Use Context7 MCP for any library/API documentation you need
