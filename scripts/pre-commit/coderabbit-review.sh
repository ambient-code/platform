#!/usr/bin/env bash
# coderabbit-review.sh — run CodeRabbit CLI review on staged changes.
# Skips gracefully if the cr binary or CODERABBIT_API_KEY is not available.
set -euo pipefail

if ! command -v cr &>/dev/null && ! command -v coderabbit &>/dev/null; then
    echo "CodeRabbit CLI not found — skipping review (install from https://cli.coderabbit.ai)"
    exit 0
fi

if [ -z "${CODERABBIT_API_KEY:-}" ]; then
    # Check if authenticated via OAuth
    if ! cr auth status &>/dev/null 2>&1; then
        echo "CODERABBIT_API_KEY not set and not logged in — skipping CodeRabbit review"
        exit 0
    fi
fi

exec cr review --type uncommitted --prompt-only
