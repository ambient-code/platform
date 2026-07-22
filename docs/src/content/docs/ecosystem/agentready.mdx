---
title: AgentReady
---

import { Badge } from '@astrojs/starlight/components';

<Badge text="Stable" variant="success" />

[AgentReady](https://github.com/ambient-code/agentready) (v2.29.4) is a CLI tool that evaluates repositories for AI-assisted development readiness.

## What it does

AgentReady scans a repository and scores it across **13 assessment categories**:

- Documentation Standards
- Repository Structure
- Testing & CI/CD
- Security
- Context Window Optimization (includes `CLAUDE.md` presence and quality)
- Dependency Management
- CI/CD Integration
- Error Handling
- API Documentation
- Code Quality (includes complexity, type annotations, code smells)
- Git & Version Control
- Modularity
- Build & Development

## Certification levels

Based on the overall score, repositories receive a certification:

| Level | Score range | Meaning |
|-------|-----------|---------|
| **Platinum** | 90--100 | Fully optimized for AI-assisted development |
| **Gold** | 75--89 | Well prepared with minor gaps |
| **Silver** | 60--74 | Functional but could benefit from improvements |
| **Bronze** | Below 60 | Significant preparation needed |

## Installation

```bash
# Using pip
pip install agentready

# Using uvx (no install needed)
uvx agentready

# Using a container
podman pull ghcr.io/ambient-code/agentready:latest
```

## Usage

```bash
# Assess the current directory
agentready assess .

# Assess a specific repository
agentready assess /path/to/repo

# Output JSON report
agentready assess /path/to/repo --format json

# Generate markdown report (git-friendly)
agentready assess /path/to/repo --format markdown --output report.md
```

## Output formats

- **Interactive HTML** -- Opens in your browser with detailed breakdowns per category.
- **Markdown** -- Git-friendly report suitable for committing to the repository.
- **JSON** -- Machine-readable output for CI pipelines.

## CI integration

Run AgentReady in CI to enforce a minimum readiness score:

```yaml
- name: Check AI readiness
  run: |
    pip install agentready
    agentready assess . --format json --output report.json
    score=$(python -c "import json; print(json.load(open('report.json'))['score'])")
    if [ "$score" -lt 75 ]; then
      echo "AI readiness score $score is below threshold (75)"
      exit 1
    fi
```
