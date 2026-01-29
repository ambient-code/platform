# Amber - GitHub Automation Tool

Amber is a GitHub Actions-based automation tool that automatically handles issues and creates pull requests.

## ⚠️ Important: Amber is NOT Part of the Core Platform

**Amber is a development tool for THIS repository** - it does NOT need to be deployed with the Ambient Code Platform. It runs via GitHub Actions and helps automate common development tasks.

## 📖 Documentation

### Getting Started
- **[5-Minute Quickstart](../../amber-quickstart.md)** - Quick setup guide
- **[Setup Instructions](../../../AMBER_SETUP.md)** - Initial configuration

### Complete Guide
- **[Amber Automation Guide](../../amber-automation.md)** - Full documentation
  - How it works
  - Available workflows
  - Configuration
  - Security
  - Best practices

### Configuration
- **[Amber Config](../../../.claude/amber-config.yml)** - Automation policies (if exists)
- **[GitHub Workflow](../../../.github/workflows/amber-issue-handler.yml)** - Workflow definition

## 🎯 What Amber Does

### Automated Workflows

| Workflow | Label | Use Case |
|----------|-------|----------|
| **Auto-Fix** | `amber:auto-fix` | Linting, formatting, trivial fixes |
| **Refactoring** | `amber:refactor` | Break large files, extract patterns |
| **Test Coverage** | `amber:test-coverage` | Add missing tests |

### Trigger Methods

**Method 1: Issue Label**
1. Create issue using Amber template
2. Label is automatically applied
3. Amber executes immediately

**Method 2: Manual Comment**
```
/amber execute
```
or
```
@amber
```

## 🚀 Quick Usage Examples

### Example 1: Fix Linting Errors
```yaml
Title: [Amber] Fix Go formatting
Label: amber:auto-fix
Files: components/backend/**/*.go
```

### Example 2: Refactor Large File
```yaml
Title: [Amber Refactor] Break sessions.go into modules
Label: amber:refactor
Current: handlers/sessions.go (3,495 lines)
Desired: Split into lifecycle.go, status.go, jobs.go
```

### Example 3: Add Tests
```yaml
Title: [Amber Tests] Add contract tests for Projects API
Label: amber:test-coverage
Target: handlers/projects.go
Coverage: 60%
```

## 🔧 Setup Requirements

**One-time setup for this repository:**

1. Add `ANTHROPIC_API_KEY` to GitHub secrets
2. Enable GitHub Actions workflow permissions
3. Install GitHub App (optional, for private repos)

See [AMBER_SETUP.md](../../../AMBER_SETUP.md) for detailed instructions.

## 📊 Monitoring Amber

```bash
# View workflow runs
gh run list --workflow=amber-issue-handler.yml

# View Amber-generated PRs
gh pr list --label amber-generated

# Check workflow status
gh workflow view amber-issue-handler.yml
```

## 🆘 Troubleshooting

**Workflow not triggering?**
- Check GitHub Actions are enabled
- Verify `ANTHROPIC_API_KEY` secret exists
- Ensure workflow permissions are set

**Amber created PR with errors?**
- Review workflow logs: `gh run view <run-id> --log`
- Check issue has clear instructions and file paths
- Verify project linters/tests passed locally

**Need help?**
- See [Amber Automation Guide](../../amber-automation.md)
- Create issue with label `amber:help`
- Check GitHub workflow logs

---

**Related Documentation:**
- [Contributing Guide](../../../CONTRIBUTING.md)
- [Code Standards](../../../CLAUDE.md)
- [GitHub Actions Workflows](../../../.github/workflows/)
