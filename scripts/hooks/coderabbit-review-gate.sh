#!/usr/bin/env bash
# coderabbit-review-gate.sh — CodeRabbit review gate.
#
# Two modes:
#   1. Claude Code hook: set as a PreToolUse hook on Bash. Intercepts
#      `gh pr create` and blocks until CodeRabbit passes. All other
#      commands pass through.
#   2. Standalone / CI: run directly. Reviews the current branch diff
#      against BASE_BRANCH and exits non-zero on findings.
#
# Exit codes:
#   0 = pass (or non-matching command in hook mode)
#   2 = blocked — CodeRabbit found issues
set -euo pipefail

# Hook mode: only gate `gh pr create`, pass everything else through
if [ -n "${CLAUDE_TOOL_INPUT:-}" ]; then
    COMMAND=$(echo "$CLAUDE_TOOL_INPUT" | jq -r '.command // ""')
    if ! echo "$COMMAND" | grep -qE '^\s*gh\s+pr\s+create\b'; then
        exit 0
    fi
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"
BASE_BRANCH="main"

# Skip if no changed files (gracefully handle shallow CI clones where HEAD~1 or base may not exist)
CHANGED_FILES=$(git diff "$BASE_BRANCH"...HEAD --name-only 2>/dev/null || git diff HEAD~1 --name-only 2>/dev/null || true)
if [ -z "$CHANGED_FILES" ]; then
    echo "Review Gate: no changed files, allowing PR creation" >&2
    exit 0
fi

# Require CodeRabbit CLI
if ! command -v coderabbit &>/dev/null; then
    echo "Review Gate: CodeRabbit CLI not found — cannot enforce review gate" >&2
    echo "Install: curl -fsSL https://cli.coderabbit.ai/install.sh | sh" >&2
    exit 2
fi

echo "Review Gate: running CodeRabbit review before PR creation..." >&2

CR_OUTPUT=$(cd "$REPO_ROOT" && coderabbit review --agent --base "$BASE_BRANCH" 2>&1 || true)

# Filter to valid JSON lines (--agent emits NDJSON plus non-JSON diagnostic lines)
CR_JSON=$(echo "$CR_OUTPUT" | grep -E '^\{' || true)

# Check for errors
CR_ERROR_TYPE=$(echo "$CR_JSON" | jq -r 'select(.type == "error") | .errorType' 2>/dev/null || true)

if [ "$CR_ERROR_TYPE" = "rate_limit" ]; then
    CR_WAIT=$(echo "$CR_JSON" | jq -r 'select(.type == "error") | .metadata.waitTime // "unknown"' 2>/dev/null || true)
    echo "Review Gate: CodeRabbit rate-limited (wait: $CR_WAIT) — allowing PR creation" >&2
    exit 0
fi

if [ "$CR_ERROR_TYPE" = "auth" ]; then
    echo "Review Gate: CodeRabbit auth failed — allowing PR creation" >&2
    echo "Configure API key in Integrations or run: coderabbit auth login" >&2
    exit 0
fi

# Check for blocking findings
if [ -n "$CR_JSON" ]; then
    BLOCKING=$(echo "$CR_JSON" | jq -r \
        'select(.type == "finding" or .findings != null) |
         (.findings[]? // .) | select(.severity == "error") |
         "  \(.file):\(.line) — \(.message)"' \
        2>/dev/null || true)

    if [ -n "$BLOCKING" ]; then
        echo "" >&2
        echo "=================================================" >&2
        echo "Review Gate: BLOCKED — CodeRabbit found issues" >&2
        echo "=================================================" >&2
        echo "$BLOCKING" >&2
        echo "" >&2
        echo "Fix these issues and retry gh pr create." >&2
        exit 2
    fi
fi

echo "Review Gate: CodeRabbit review passed" >&2
exit 0
