#!/usr/bin/env bash
# coderabbit-review.sh — run CodeRabbit CLI review on staged changes.
# Skips gracefully if the CLI or auth is not available.
# Treats transient failures (rate limits, network errors) as warnings.
set -euo pipefail

# Resolve binary name
CR=""
if command -v cr &>/dev/null; then
    CR="cr"
elif command -v coderabbit &>/dev/null; then
    CR="coderabbit"
else
    echo "CodeRabbit CLI not found — skipping review (install from https://cli.coderabbit.ai)"
    exit 0
fi

# Skip if nothing is staged
if git diff --cached --quiet; then
    exit 0
fi

# Require auth: API key or OAuth login
if [ -z "${CODERABBIT_API_KEY:-}" ]; then
    if ! "$CR" auth status &>/dev/null; then
        echo "CODERABBIT_API_KEY not set and not logged in — skipping CodeRabbit review"
        exit 0
    fi
fi

# Run review; capture output to distinguish findings from transient errors
OUTPUT=$(timeout 300 "$CR" review --type uncommitted --prompt-only 2>&1) || EXIT_CODE=$?
EXIT_CODE=${EXIT_CODE:-0}

echo "$OUTPUT"

if [ "$EXIT_CODE" -eq 0 ]; then
    exit 0
fi

# Rate limits and network errors should warn, not block
if echo "$OUTPUT" | grep -qi "rate limit\|network\|timeout\|connection"; then
    echo "CodeRabbit: transient error (see above) — not blocking commit"
    exit 0
fi

# Actual review findings — block the commit
exit "$EXIT_CODE"
