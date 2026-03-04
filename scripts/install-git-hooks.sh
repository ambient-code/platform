#!/usr/bin/env bash
#
# Install git hooks via the pre-commit framework.
#
# This replaces the old symlink-based installation. All hooks (branch
# protection, linters, formatters) are now managed by pre-commit and
# configured in .pre-commit-config.yaml at the repo root.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Determine repository root
REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null)"
if [ -z "$REPO_ROOT" ]; then
    echo -e "${RED}Not in a git repository${NC}"
    exit 1
fi

if [ ! -f "$REPO_ROOT/.pre-commit-config.yaml" ]; then
    echo -e "${RED}.pre-commit-config.yaml not found at repo root${NC}"
    exit 1
fi

# ── Install pre-commit if missing ──────────────────────────────────────
if ! command -v pre-commit &>/dev/null; then
    echo -e "${YELLOW}pre-commit not found — installing...${NC}"
    if command -v uv &>/dev/null; then
        uv pip install pre-commit
    elif command -v pip &>/dev/null; then
        pip install --user pre-commit
    elif command -v pip3 &>/dev/null; then
        pip3 install --user pre-commit
    else
        echo -e "${RED}Cannot install pre-commit: no pip/uv found${NC}"
        echo "  Install manually: https://pre-commit.com/#install"
        exit 1
    fi
fi

# ── Remove old symlink-based hooks if present ──────────────────────────
for hook in pre-commit pre-push; do
    hook_path="$REPO_ROOT/.git/hooks/$hook"
    if [ -L "$hook_path" ]; then
        target="$(readlink "$hook_path")"
        if [[ "$target" == *"scripts/git-hooks/"* ]]; then
            echo -e "${YELLOW}Removing old symlink: $hook -> $target${NC}"
            rm -f "$hook_path"
        fi
    fi
done

# ── Install pre-commit hooks ──────────────────────────────────────────
cd "$REPO_ROOT"

echo "Installing pre-commit hooks..."
pre-commit install
pre-commit install --hook-type pre-push

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}pre-commit hooks installed${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Hooks run automatically on commit/push. What runs:"
echo "   pre-commit   trailing-whitespace, end-of-file-fixer, check-yaml,"
echo "                check-added-large-files, check-merge-conflict,"
echo "                detect-private-key, ruff-format, ruff, gofmt,"
echo "                go vet, golangci-lint, eslint, branch-protection"
echo "   pre-push     push-protection"
echo ""
echo "To run all hooks manually:"
echo "   pre-commit run --all-files"
echo ""
echo "To skip hooks when needed:"
echo "   git commit --no-verify"
echo "   git push --no-verify"
echo ""
echo "To uninstall:"
echo "   make remove-hooks"
echo ""
