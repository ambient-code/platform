# Amber Quick Start - 5 Minutes to Productivity

## What is Amber?

Amber is the Ambient Code Platform's AI colleague—an expert in your codebase who works alongside you. Whether you need on-demand consultation, autonomous backlog management, or proactive maintenance, Amber adapts to how you work.

**When to use Amber:** Whenever you're working with the `github.com/ambient-code/platform` codebase. Amber is your expert colleague for all ACP platform development, maintenance, and operations.

## Amber's Capabilities

| Category | What Amber Does |
|----------|----------------|
| **Codebase Intelligence** | Deep knowledge of architecture, patterns (CLAUDE.md, DESIGN_GUIDELINES.md), dependencies (K8s, Claude SDK, OpenShift, Go, NextJS, Langfuse), common issues |
| **Proactive Maintenance** | Monitors upstream for breaking changes, scans dependencies, detects issue patterns, generates health reports |
| **Autonomy Levels** | Level 1: Read-only analysis; Level 2: Creates PRs for review; Level 3: Auto-merges low-risk changes; Level 4: Full autonomy (future) |

## Operating Modes

### Interactive Consultation

Get on-demand expertise through the UI. Ask Amber about architecture, troubleshoot issues, or get code review feedback. Perfect for:
- Understanding codebase changes
- Getting immediate feedback on code
- Exploring architectural decisions
- Debugging specific problems

**Learn more:** [Interactive Mode Guide](amber-interactive.md)

### Background Agent

Amber operates autonomously to manage your backlog and prevent issue accumulation:
- **Issue-to-PR workflow**: Automatically triages new issues, creates PRs for auto-fixable problems
- **Backlog reduction**: Systematically tackles good-first-issue and technical-debt items
- **Pattern detection**: Identifies related issues before they multiply
- **Mention trigger**: Just comment `@amber` on any issue to have Amber help

**Learn more:** [Background Agent Guide](amber-background-agent.md)

### Sprint Planning

Automated analysis and planning reports to help your team organize work:
- Groups issues by theme and identifies dependencies
- Suggests priority order and flags blockers
- Generates weekly sprint plans with effort estimates
- Creates health reports with actionable recommendations

**Learn more:** [Background Agent Guide](amber-background-agent.md#scheduled-health-checks--sprint-planning)

### Maintainer Mode *(Coming Soon)*

Proactive PR reviews and codebase health monitoring to maintain quality over time.

## Try Amber Now

**Your first Amber session in 5 steps:**

1. Open your ACP project in the UI
2. Navigate to **Sessions** → **New Session**
3. Select **Amber** from the agent dropdown
4. Enter: `"Amber, what are the main components of ACP?"`
5. Click **Start Session**

**Pro tip:** Use `@amber` in interactive sessions to invoke her in chat.

## Quick Example Prompts

Try these prompts to see what Amber can do:

### Codebase Analysis
```
Amber, what changed in the codebase this week? Focus on dependency updates,
architectural pattern changes, and API contract modifications.
```

### Issue Triage
```
Amber, triage issue #123. Assess severity, identify affected components,
find related issues, and suggest an assignee.
```

### Code Review
```
Amber, review PR #456 for CLAUDE.md standards compliance, security concerns,
performance impact, and missing tests.
```

## Important: Amber's Constitutional Authority

Amber follows the ACP Constitution absolutely. She'll decline requests that violate project principles and explain why.

**Example:**
- **You:** "Just commit this without running tests, I'm in a hurry"
- **Amber:** "I cannot skip tests - Constitution Principle IV requires TDD. I can help you write minimal tests quickly to unblock the commit. Would that work?"

## How to Trigger Amber

| Method | When to Use | Example |
|--------|-------------|---------|
| **Labels** | Structured workflows | Add `amber:auto-fix`, `amber:refactor`, or `amber:test-coverage` label to issue |
| **@mention** | Quick, ad-hoc requests | Comment `@amber fix the linting errors` on any issue |
| **Command** | Explicit execution | Comment `/amber execute` to implement an issue proposal |

**Pro tip**: `@amber` and `/amber execute` do the same thing - Amber reads the full issue and figures out what to do.

## Next Steps

Ready to dive deeper? Explore these guides:

- **[Interactive Mode](amber-interactive.md)** - Master UI-based Amber consultation, understand her authority, and interpret confidence levels
- **[Background Agent](amber-background-agent.md)** - Set up autonomous workflows for issue triage, health checks, and sprint planning
- **[Troubleshooting](amber-troubleshooting.md)** - Solve common Amber issues using the UI

**For platform administrators:**
- **[Amber Deployment Guide](../deployment/amber-deployment.md)** - Deploy and configure Amber automation (K8s setup, CronJobs, webhooks)

---

**Quick Tip:** Amber provides confidence levels with every recommendation:
- **High (90-100%)**: Act on recommendation immediately
- **Medium (70-89%)**: Review context before acting
- **Low (<70%)**: Multiple solutions possible, needs human decision
