#!/usr/bin/env bash
# pr-review-gate.sh — PreToolUse hook for Bash tool calls.
# Gates `gh pr create` behind mechanical code quality checks and
# CodeRabbit AI review. Stateless: runs checks inline on every attempt.
#
# Exit codes (Claude Code PreToolUse convention):
#   0 = allow the tool call
#   2 = block the tool call (stderr shown to agent as reason)
set -euo pipefail

HOOK_INPUT=$(cat)
COMMAND=$(echo "$HOOK_INPUT" | jq -r '.tool_input.command // ""')

if ! echo "$COMMAND" | grep -qE '^\s*gh\s+pr\s+create\b'; then
    exit 0
fi

echo "PR Review Gate: checking code quality before opening PR..." >&2

REPO_ROOT="$(git rev-parse --show-toplevel)"
BASE_BRANCH="main"
ERRORS=()

block_with_errors() {
    echo "" >&2
    echo "=================================================" >&2
    echo "PR Review Gate: BLOCKED — fix these issues first" >&2
    echo "=================================================" >&2
    printf '%s\n' "${ERRORS[@]}" >&2
    echo "" >&2
    echo "Fix these issues and retry gh pr create." >&2
    exit 2
}

# Compute diff once — reuse for file list and secret scan
FULL_DIFF=$(git diff "$BASE_BRANCH"...HEAD 2>/dev/null || git diff HEAD~1)
CHANGED_FILES=$(echo "$FULL_DIFF" | grep -E '^diff --git' | sed 's|^diff --git a/.* b/||' || true)

if [ -z "$CHANGED_FILES" ]; then
    echo "PR Review Gate: no changed files detected, allowing" >&2
    exit 0
fi

# ── Go checks ─────────────────────────────────────────────────────────
GO_FILES=$(echo "$CHANGED_FILES" | grep '\.go$' || true)
if [ -n "$GO_FILES" ]; then
    GO_MODULES=$(echo "$GO_FILES" | while read -r f; do
        dir=$(dirname "$f")
        while [ "$dir" != "." ]; do
            if [ -f "$REPO_ROOT/$dir/go.mod" ]; then
                echo "$dir"
                break
            fi
            dir=$(dirname "$dir")
        done
    done | sort -u)

    for mod in $GO_MODULES; do
        if command -v gofmt &>/dev/null; then
            MOD_GO_FILES=$(echo "$GO_FILES" | grep "^$mod/" || true)
            if [ -n "$MOD_GO_FILES" ]; then
                UNFORMATTED=$(cd "$REPO_ROOT" && echo "$MOD_GO_FILES" | xargs gofmt -l 2>/dev/null || true)
                if [ -n "$UNFORMATTED" ]; then
                    ERRORS+=("gofmt: unformatted files:" "$UNFORMATTED")
                fi
            fi
        fi

        if command -v go &>/dev/null; then
            VET_OUTPUT=$(cd "$REPO_ROOT/$mod" && go vet ./... 2>&1) || \
                ERRORS+=("go vet failed in ${mod}:" "$VET_OUTPUT")
        fi
    done

    # Batch panic() check across all non-test Go files
    PROD_GO_FILES=$(echo "$GO_FILES" | grep -v '_test\.go$' || true)
    if [ -n "$PROD_GO_FILES" ]; then
        PANIC_HITS=$(cd "$REPO_ROOT" && echo "$PROD_GO_FILES" | xargs grep -Hn 'panic(' 2>/dev/null \
            | grep -v '//.*panic' | grep -v 'nolint' || true)
        if [ -n "$PANIC_HITS" ]; then
            ERRORS+=("panic() in production code (use fmt.Errorf):" "$PANIC_HITS")
        fi
    fi
fi

# ── Frontend checks ──────────────────────────────────────────────────
TS_FILES=$(echo "$CHANGED_FILES" | grep -E '^components/frontend/.*\.(ts|tsx|js|jsx)$' || true)
if [ -n "$TS_FILES" ]; then
    FRONTEND_DIR="$REPO_ROOT/components/frontend"

    if command -v npx &>/dev/null && [ -d "$FRONTEND_DIR/node_modules" ]; then
        RELATIVE_FILES=$(echo "$TS_FILES" | sed 's|^components/frontend/||')
        ESLINT_OUTPUT=$(cd "$FRONTEND_DIR" && echo "$RELATIVE_FILES" | xargs npx eslint 2>&1) || \
            ERRORS+=("eslint failed on frontend files:" "$ESLINT_OUTPUT")
    fi

    ANY_HITS=$(cd "$REPO_ROOT" && echo "$TS_FILES" | xargs grep -Hn ': any\b\|<any>\|as any\b' 2>/dev/null \
        | grep -v '//.*any\|nolint\|eslint-disable' || true)
    if [ -n "$ANY_HITS" ]; then
        ERRORS+=("'any' type usage in frontend (use proper types, unknown, or generics):" "$ANY_HITS")
    fi
fi

# ── Python checks ────────────────────────────────────────────────────
PY_FILES=$(echo "$CHANGED_FILES" | grep -E '^(components/runners/|scripts/).*\.py$' || true)
if [ -n "$PY_FILES" ]; then
    if command -v ruff &>/dev/null; then
        RUFF_CHECK=$(cd "$REPO_ROOT" && echo "$PY_FILES" | xargs ruff check 2>&1) || \
            ERRORS+=("ruff check failed:" "$RUFF_CHECK")
        RUFF_FMT=$(cd "$REPO_ROOT" && echo "$PY_FILES" | xargs ruff format --check 2>&1) || \
            ERRORS+=("ruff format failed:" "$RUFF_FMT")
    fi
fi

# ── Secret detection ─────────────────────────────────────────────────
SECRET_PATTERNS='(PRIVATE[_-]KEY|SECRET[_-]KEY|API[_-]KEY|PASSWORD|TOKEN)\s*[=:]\s*["'"'"'][^\s]+'
SECRET_HITS=$(echo "$FULL_DIFF" | grep '^\+' | grep -iE "$SECRET_PATTERNS" | head -5 || true)
if [ -n "$SECRET_HITS" ]; then
    ERRORS+=("Possible secrets in diff:" "$SECRET_HITS")
fi

# Bail early if mechanical checks failed — don't waste CodeRabbit's time
if [ ${#ERRORS[@]} -gt 0 ]; then
    block_with_errors
fi

# ── CodeRabbit AI review ────────────────────────────────────────────
# Uses .coderabbit.yaml config (security, performance, K8s safety checks).
# Skips gracefully if CLI or auth is unavailable.
if command -v coderabbit &>/dev/null; then
    echo "PR Review Gate: running CodeRabbit review..." >&2

    CR_OUTPUT=$(coderabbit review --agent --base "$BASE_BRANCH" 2>&1 || true)

    # --agent emits NDJSON (one JSON object per line) plus non-JSON
    # lines like "[error] stopping cli". Filter to valid JSON first.
    CR_JSON=$(echo "$CR_OUTPUT" | grep -E '^\{' || true)

    CR_ERROR_TYPE=$(echo "$CR_JSON" | jq -r 'select(.type == "error") | .errorType' 2>/dev/null || true)

    if [ "$CR_ERROR_TYPE" = "rate_limit" ]; then
        CR_WAIT=$(echo "$CR_JSON" | jq -r 'select(.type == "error") | .metadata.waitTime // "unknown"' 2>/dev/null || true)
        echo "PR Review Gate: CodeRabbit rate-limited (wait: $CR_WAIT)" >&2
    elif [ "$CR_ERROR_TYPE" = "auth" ] || echo "$CR_JSON" | jq -e 'select(.type == "error")' &>/dev/null; then
        CR_MSG=$(echo "$CR_JSON" | jq -r 'select(.type == "error") | .message // "unknown error"' 2>/dev/null || true)
        echo "PR Review Gate: CodeRabbit skipped ($CR_MSG)" >&2
    elif [ -n "$CR_JSON" ]; then
        BLOCKING=$(echo "$CR_JSON" | jq -r \
            'select(.type == "finding" or .findings != null) |
             (.findings[]? // .) | select(.severity == "error") |
             "  \(.file):\(.line) — \(.message)"' \
            2>/dev/null || true)

        if [ -n "$BLOCKING" ]; then
            ERRORS+=("CodeRabbit found blocking issues:" "$BLOCKING")
        else
            echo "PR Review Gate: CodeRabbit review passed" >&2
        fi
    fi
else
    echo "PR Review Gate: CodeRabbit CLI not found — skipping AI review" >&2
fi

if [ ${#ERRORS[@]} -gt 0 ]; then
    block_with_errors
fi

echo "PR Review Gate: all checks passed" >&2
exit 0
