# E2E Testing Guide

## Overview

The E2E workflow only runs when a maintainer adds the `safe-to-test` label to a PR. This provides full test coverage with secrets while maintaining security.

## How It Works

**Trigger:** ONLY when `safe-to-test` label is added to a PR  
**Access:** Has full access to secrets (including `ANTHROPIC_API_KEY`)  
**Tests:** Runs complete suite including agent interaction

## For Maintainers

### Running E2E Tests on a PR

1. **Review the PR code** carefully:
   - Check for suspicious script modifications
   - Look for attempts to exfiltrate environment variables
   - Verify changes to workflows, dockerfiles, and scripts
   - Ensure code is trustworthy

2. **Add the label:**
   ```bash
   # Via GitHub CLI
   gh pr edit <PR_NUMBER> --add-label safe-to-test
   
   # Or via GitHub UI
   # Go to PR → Labels → Add "safe-to-test"
   ```

3. **Tests run automatically** with full access to secrets

4. **Results posted** to PR as a comment

### Creating the Label (First Time Setup)

```bash
# Via GitHub CLI
gh label create safe-to-test \
  --description "Maintainer-approved: Run E2E tests with secrets" \
  --color "0e8a16"

# Or via GitHub UI
# Settings → Labels → New label
# Name: safe-to-test
# Description: Maintainer-approved: Run E2E tests with secrets
# Color: Green (#0e8a16)
```

## For Contributors

### Testing Your PR Locally

You can run the full E2E test suite locally with your own API key:

```bash
# 1. Create e2e/.env with your API key
cat > e2e/.env << EOF
ANTHROPIC_API_KEY=sk-ant-api03-your-key-here
EOF

# 2. Run tests
make kind-up     # Setup kind cluster and deploy
make test-e2e    # Run Cypress tests

# 3. Clean up
make kind-down
```

### Waiting for Tests

- Your PR won't have E2E tests run automatically (security restriction)
- Ask a maintainer to review and add the `safe-to-test` label
- Tests will run with full secrets once approved
- Results will be posted as a PR comment

## Security Model

### Why Label-Only Trigger?

**GitHub Actions doesn't expose secrets to fork PRs** for security reasons. To safely test fork PRs:

1. ✅ **Maintainer reviews code** (manual security check)
2. ✅ **Maintainer adds label** (explicit approval)
3. ✅ **Workflow runs with secrets** (pull_request_target context)
4. ✅ **PR code tested** (builds images from PR branch)

This prevents malicious PRs from accessing secrets while still allowing full testing.

### What Gets Tested

- ✅ All UI interactions (workspace/session CRUD)
- ✅ Backend API endpoints
- ✅ Kubernetes operator functionality
- ✅ **Full agent interaction** with Claude API
- ✅ File operations and workflows
- ✅ Real-time chat interface

## Workflow Details

### Test Suite

**Location:** `e2e/cypress/e2e/*.cy.ts`

**Tests:**
- `vteam.cy.ts` (5 tests): Platform smoke tests
- `sessions.cy.ts` (7 tests): Session management and agent interaction

**Runtime:** ~15 seconds (all 12 tests)

### What Happens

1. Label added → Workflow triggered
2. Check if label is `safe-to-test` (skip if not)
3. Build component images from PR code
4. Deploy to kind cluster
5. Inject `ANTHROPIC_API_KEY` into runner secrets
6. Run full Cypress test suite
7. Post results to PR

### Image Building

- **Changed components:** Built from PR code
- **Unchanged components:** Pulled from `quay.io/ambient_code` (latest)
- **Optimization:** Only rebuilds what changed

## Troubleshooting

### Tests Don't Run

**Check:** Was `safe-to-test` label added?  
**Fix:** Add the label to trigger the workflow

### Agent Test Fails

**Check:** Is `ANTHROPIC_API_KEY` secret set in repo?  
**Fix:** Add secret in repo settings → Secrets and variables → Actions

### Build Fails

**Check:** Did PR code break the build?  
**Fix:** Review PR for build errors, ask contributor to fix

### Deployment Fails

**Check:** Are manifests valid?  
**Fix:** Test locally with `make kind-up`, validate K8s YAML

## Local Development

### Quick Start

```bash
# Full stack with e2e tests
make kind-up      # Creates cluster, deploys platform, saves token
make test-e2e     # Runs Cypress tests
make kind-down    # Cleanup

# Individual steps
cd e2e
./scripts/setup-kind.sh    # Create cluster
./scripts/deploy.sh        # Deploy platform
./scripts/run-tests.sh     # Run tests
./scripts/cleanup.sh       # Cleanup
```

### Manual Testing

```bash
# Deploy and keep running
make kind-up

# Access UI
open http://localhost        # Docker
open http://localhost:8080   # Podman

# Access backend
curl http://localhost/api/health

# Run tests multiple times
cd e2e && npm test

# Clean up when done
make kind-down
```

## CI Integration

### GitHub Actions

The E2E workflow runs in GitHub Actions using kind (Kubernetes in Docker).

**Environment:**
- Ubuntu 22.04
- Docker 20+
- Node.js 20
- kind v0.20.0

**Resources:**
- 7 GB RAM for GitHub Actions runner
- ~15 minute timeout
- Disk cleanup before tests

### Artifacts

**On failure, uploads:**
- Cypress screenshots
- Cypress videos
- Pod logs (frontend, backend, operator)

**Retention:** 7 days

## FAQ

**Q: Why can't I see test results on my fork PR?**  
A: Tests only run when a maintainer adds the `safe-to-test` label for security.

**Q: Can I run tests without a maintainer?**  
A: Yes, locally with your own API key. See "Testing Your PR Locally" above.

**Q: What if I don't have an API key?**  
A: You can run 6 of 7 tests without it. Only agent interaction test requires the key.

**Q: Why does it take so long?**  
A: Most time is building images (~8 min). Tests themselves run in ~15 seconds.

**Q: Can I speed up development?**  
A: Yes, use `DEV_MODE=true make dev-start` with hot-reloading for faster iteration.

## References

- [Kind Local Dev Guide](../developer/local-development/kind.md)
- [Cypress Best Practices](https://docs.cypress.io/guides/references/best-practices)
- [GitHub Actions Security](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
