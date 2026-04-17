#!/usr/bin/env bash
# PostToolUse hook: auto-format edited files. Skips if formatter not installed.

set -uo pipefail

FILE_PATH=$(echo "$TOOL_INPUT" | jq -r '.file_path // empty' 2>/dev/null) || true
[[ -z "$FILE_PATH" || ! -f "$FILE_PATH" ]] && exit 0

case "$FILE_PATH" in
  *.go)
    command -v gofmt &>/dev/null && gofmt -w "$FILE_PATH"
    ;;
  *.py)
    command -v ruff &>/dev/null && ruff format --quiet "$FILE_PATH"
    ;;
  *.ts|*.tsx|*.js|*.jsx)
    if [[ -x node_modules/.bin/prettier ]]; then
      node_modules/.bin/prettier --write --log-level silent "$FILE_PATH" 2>/dev/null
    elif command -v prettier &>/dev/null; then
      prettier --write --log-level silent "$FILE_PATH" 2>/dev/null
    fi
    ;;
esac

exit 0
