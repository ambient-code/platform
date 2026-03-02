# GitHub Actions Workflows

This directory contains automated workflows for the Ambient Code Platform.

For a complete inventory of workflows, read the `name:` field in each
`.yml`/`.yaml` file in this directory. Do not rely on a static list here —
see [#769](https://github.com/ambient-code/platform/issues/769).

## Security

### Permissions

All workflows follow **principle of least privilege**:

```yaml
permissions:
  contents: read      # Default for reading code
  issues: write       # Only for issue-handling workflows
  pull-requests: write # Only for PR-creating workflows
  packages: write     # Only for image publishing
```

### Command Injection Prevention

Pass user input through environment variables, never through string
interpolation in `run:` blocks.

Safe:
```yaml
env:
  ISSUE_TITLE: ${{ github.event.issue.title }}
run: echo "$ISSUE_TITLE"
```

Unsafe:
```yaml
run: echo "${{ github.event.issue.title }}"  # Vulnerable to injection
```

Reference: [GitHub Actions Security Guide](https://github.blog/security/vulnerability-research/how-to-catch-github-actions-workflow-injections-before-attackers-do/)

## Monitoring and Troubleshooting

```bash
gh run list                              # All recent runs
gh run list --workflow=e2e.yml           # Runs for a specific workflow
gh run watch                             # Watch a running workflow
gh run view <run-id> --log               # View logs
gh run view <run-id> --log-failed        # View only failure logs
gh run rerun <run-id> --failed           # Re-run failed jobs
```

## Best Practices

### ✅ Do

- Use latest action versions (`actions/checkout@v4`)
- Set explicit permissions per workflow
- Pass user input via environment variables
- Cache dependencies (npm, pip, Go modules)
- Fail fast for critical errors

### ❌ Don't

- Use `permissions: write-all` (too broad)
- Interpolate user input directly in `run:` commands
- Hardcode secrets (use GitHub Secrets)
- Run workflows on every push (use path filters)
- Ignore security warnings from GitHub

## Related Documentation

- [Amber Automation Guide](../../docs/amber-automation.md)
- [E2E Testing Guide](../../docs/testing/e2e-guide.md)
- [GitHub Actions Docs](https://docs.github.com/en/actions)
- [Security Best Practices](https://docs.github.com/en/actions/security-for-github-actions)
