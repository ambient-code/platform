#!/usr/bin/env bash
# eslint.sh — run ESLint on staged frontend files.
# Skips gracefully if Node.js / npx is not available or node_modules missing.
set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
FRONTEND_DIR="$REPO_ROOT/components/frontend"

if ! command -v npx &>/dev/null; then
    echo "npx not found — skipping ESLint (install Node.js to enable)"
    exit 0
fi

if [ ! -d "$FRONTEND_DIR/node_modules" ]; then
    echo "node_modules not found in components/frontend — skipping ESLint"
    echo "Run: cd components/frontend && npm install"
    exit 0
fi

# Strip the components/frontend/ prefix so ESLint resolves paths relative to its config
relative_files=()
for file in "$@"; do
    # Handle both absolute and relative paths
    rel="${file#"$FRONTEND_DIR/"}"
    rel="${rel#components/frontend/}"
    relative_files+=("$rel")
done

if [ ${#relative_files[@]} -eq 0 ]; then
    exit 0
fi

cd "$FRONTEND_DIR"
exec npx eslint "${relative_files[@]}"
