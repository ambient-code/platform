#!/usr/bin/env bash
# go-vet.sh — run `go vet` in each affected Go module.
# Determines which module(s) contain staged files and runs vet per-module.
# Skips gracefully if Go is not installed.
set -euo pipefail

if ! command -v go &>/dev/null; then
    echo "go not found — skipping (install Go to enable)"
    exit 0
fi

REPO_ROOT="$(git rev-parse --show-toplevel)"

# Known Go module directories (relative to repo root)
GO_MODULES=(
    "components/backend"
    "components/operator"
    "components/public-api"
)

# Determine which modules are affected by the staged files
declare -A affected_modules
for file in "$@"; do
    for mod in "${GO_MODULES[@]}"; do
        if [[ "$file" == "$mod/"* || "$file" == "$REPO_ROOT/$mod/"* ]]; then
            affected_modules["$mod"]=1
        fi
    done
done

if [ ${#affected_modules[@]} -eq 0 ]; then
    exit 0
fi

exit_code=0
for mod in "${!affected_modules[@]}"; do
    mod_dir="$REPO_ROOT/$mod"
    if [ -f "$mod_dir/go.mod" ]; then
        echo "go vet: $mod"
        if ! (cd "$mod_dir" && go vet ./...); then
            exit_code=1
        fi
    fi
done

exit $exit_code
