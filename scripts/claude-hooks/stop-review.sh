#!/usr/bin/env bash
# Stop hook: suggest running /amber-review if files were modified during the session.
# Called by Claude Code's Stop hook — reads the stop_hook_input from stdin.

set -euo pipefail

# Check for any working tree changes (staged, unstaged, or untracked)
if [[ -z "$(git status --porcelain 2>/dev/null)" ]]; then
  exit 0
fi

cat <<'MSG'
Files modified. Consider running /amber-review before completing.
MSG
