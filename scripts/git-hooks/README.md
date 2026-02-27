# Git Hooks (pre-commit Framework)

This project uses the [pre-commit](https://pre-commit.com/) framework to manage git hooks. All hook configuration lives in `.pre-commit-config.yaml` at the repository root.

## What's Included

### Pre-commit Stage (runs on `git commit`)

| Hook | Source | Scope |
|------|--------|-------|
| trailing-whitespace | pre-commit-hooks | All files |
| end-of-file-fixer | pre-commit-hooks | All files |
| check-yaml | pre-commit-hooks | YAML files |
| check-added-large-files | pre-commit-hooks | All files (>1MB) |
| check-merge-conflict | pre-commit-hooks | All files |
| detect-private-key | pre-commit-hooks | All files |
| ruff-format | ruff-pre-commit | Python (runners, scripts) |
| ruff | ruff-pre-commit | Python (runners, scripts) |
| gofmt | local wrapper | Go files |
| go vet | local wrapper | Go files (per-module) |
| golangci-lint | local wrapper | Go files (per-module) |
| eslint | local wrapper | Frontend TS/JS files |
| branch-protection | local (this directory) | All commits |

### Pre-push Stage (runs on `git push`)

| Hook | Source | Scope |
|------|--------|-------|
| push-protection | local (this directory) | All pushes |

### Local Wrapper Scripts

Go and ESLint hooks use wrapper scripts in `scripts/pre-commit/` because:

- **Go** has 3 separate modules (`backend`, `operator`, `public-api`) — tools must `cd` into each module directory
- **ESLint** config and `node_modules` live in `components/frontend/`
- All wrappers skip gracefully if the toolchain is not installed

## Installation

```bash
make setup-hooks
```

Or directly:

```bash
./scripts/install-git-hooks.sh
```

This installs pre-commit (if needed) and registers hooks for both `pre-commit` and `pre-push` stages.

## Usage

### Automatic (default)

Hooks run automatically on every `git commit` and `git push`. Only files staged for commit are checked.

### Manual

Run all hooks against the entire repo:

```bash
make lint
# or: pre-commit run --all-files
```

Run a specific hook:

```bash
pre-commit run gofmt-check --all-files
pre-commit run eslint --all-files
pre-commit run golangci-lint --all-files
```

### Skip Hooks

```bash
git commit --no-verify -m "hotfix: critical fix"
git push --no-verify
```

## Branch Protection Scripts

The Python scripts in this directory (`pre-commit` and `pre-push`) implement branch protection logic. They are invoked by the pre-commit framework — not as raw git hooks.

### `pre-commit` (Python)

Blocks commits to protected branches: `main`, `master`, `production`.

### `pre-push` (Python)

Blocks pushes to protected branches by checking both the current branch and push targets.

## Uninstallation

```bash
make remove-hooks
```

## Customization

### Add More Protected Branches

Edit `PROTECTED_BRANCHES` in both `pre-commit` and `pre-push` Python scripts.

### Modify Linter Configuration

Edit `.pre-commit-config.yaml` at the repo root.

## Troubleshooting

### Hooks Not Running

```bash
# Verify installation
pre-commit --version
ls -la .git/hooks/pre-commit

# Reinstall
make remove-hooks
make setup-hooks
```

### Specific Linter Failing

Run the failing hook in isolation to see detailed output:

```bash
pre-commit run <hook-id> --all-files --verbose
```

### Old Symlink Hooks

The installer automatically removes old symlink-based hooks pointing to `scripts/git-hooks/`. If you still have issues, manually remove them:

```bash
rm -f .git/hooks/pre-commit .git/hooks/pre-push
make setup-hooks
```
