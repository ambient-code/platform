#!/usr/bin/env bash
# gofmt-check.sh — check that staged Go files are gofmt-formatted.
# Skips gracefully if Go is not installed.
set -euo pipefail

if ! command -v gofmt &>/dev/null; then
    echo "gofmt not found — skipping (install Go to enable)"
    exit 0
fi

unformatted=$(gofmt -l "$@" 2>/dev/null || true)
if [ -n "$unformatted" ]; then
    echo "The following files are not gofmt-formatted:"
    echo "$unformatted"
    echo ""
    echo "Run: gofmt -w <file>"
    exit 1
fi
