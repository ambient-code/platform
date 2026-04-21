---
title: CodeRabbit Integration
description: AI-powered code review with CodeRabbit — automatic for public repos, API key for private repos
---

CodeRabbit provides AI-powered code review for pull requests and local changes. Public repositories get free reviews automatically. Private repositories require an API key.

## Public Repositories (free, no configuration)

If an org admin has installed the [CodeRabbit GitHub App](https://github.com/apps/coderabbitai) on your organization, every PR to a public repo gets AI-powered review comments automatically. No API key, no integration configuration, no session setup.

This is the default path for most users. Nothing to configure.

## Private Repositories (API key required)

For private repos, you need a CodeRabbit [Pro plan](https://coderabbit.ai/pricing) or the usage-based add-on ($0.25/file reviewed).

### 1. Generate an API key

1. Go to [app.coderabbit.ai/settings/api-keys](https://app.coderabbit.ai/settings/api-keys)
2. Log in with **GitHub** (not email — this links your CodeRabbit account to your GitHub identity)
3. Generate an Agentic API key (starts with `cr-`)

### 2. Add to ACP

1. Navigate to **Integrations** in the ACP UI
2. On the CodeRabbit card, expand **Private repository access**
3. Paste your API key and click **Save Key**

### 3. Use in sessions

The next session you create will have `CODERABBIT_API_KEY` injected into the session environment automatically. The CodeRabbit CLI and pre-commit hook use this to authenticate.

:::caution[Billing]
Adding an API key routes all CodeRabbit CLI reviews through the usage-based plan. For public repos, PR reviews via the GitHub App are free — using an API key for the same reviews will incur charges. Only add an API key if you need CLI reviews on private repos.
:::

## Local Development

For reviewing changes on your own machine (outside of ACP sessions):

```bash
# Install the CLI
brew install coderabbit

# Authenticate (opens browser — free for public repos)
coderabbit auth login

# Review uncommitted changes
coderabbit review --agent
```

### Review Gate (PR creation)

A PreToolUse hook in `.claude/settings.json` intercepts `gh pr create` and runs CodeRabbit review on the full branch diff before allowing PR creation. If CodeRabbit finds blocking issues (severity=error), the PR creation is blocked and the agent fixes the findings before retrying.

This is the enforcement point for the inner-loop review described in [ADR-0008](../../../internal/adr/0008-automate-code-reviews.md). The same script works standalone for CI:

```bash
# Run the review gate directly (outside of Claude Code)
bash scripts/hooks/coderabbit-review-gate.sh
```

## How It Works in ACP Sessions

When a session starts, the runner fetches credentials from the backend:

1. **Backend** stores the API key in a Kubernetes Secret, scoped per user
2. **Runner** calls `GET /credentials/coderabbit` with RBAC enforcement
3. If an API key is configured, `CODERABBIT_API_KEY` is set in the session environment
4. If no API key is configured, the runner skips silently — no error, no delay
5. On turn completion, the key is cleared from the environment

For multi-user sessions, RBAC ensures the correct user's credentials are used based on who initiated the current run.

## Configuration File

The platform's `.coderabbit.yaml` configures CodeRabbit's review behavior for PR reviews. Key settings:

- **Review profile**: `chill` (less verbose, focuses on real issues)
- **Path instructions**: component-specific review guidance (Go backend, TypeScript frontend, Python runner, K8s manifests, GitHub Actions)
- **Pre-merge checks**: performance/algorithmic complexity, security/secret handling, Kubernetes resource safety
- **Auto-review**: enabled on `main` and `alpha` branches, skips WIP and dependency bot PRs

See the [CodeRabbit docs](https://docs.coderabbit.ai/cli#ai-agent-integration) for CLI integration best practices.

## Integration Test

Validate the full integration stack against a running cluster:

```bash
# Against the current kubectl context
./scripts/test-coderabbit-integration.sh

# Against a specific kind cluster
./scripts/test-coderabbit-integration.sh --context kind-ambient-001-coderabbit-integ

# With live API key validation
CODERABBIT_API_KEY=cr-... ./scripts/test-coderabbit-integration.sh
```
