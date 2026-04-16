# Continuous Learning Integration Proposal: RFE Agents

**Date**: 2026-04-16
**Author**: Jeremy Eder
**Repos**: `jwforres/rfe-creator`, `n1hility/assess-rfe`

---

## TL;DR

Wire up the Continuous Learning (CL) pipeline in both RFE agent repos so they accumulate knowledge over time. Three files per repo, zero code changes. Agents get smarter every session.

## What changes

### Per repo: 3 new files

**1. `.ambient/config.json`** — Enables the CL write-side (`suggest_memory` tool, insight extraction)
```json
{
  "learning": {
    "enabled": true
  }
}
```

**2. `.wiki-compiler.json`** — Tells the wiki compiler what to compile and where to put it

For **rfe-creator**:
```json
{
  "version": 1,
  "name": "RFE Creator",
  "sources": [
    {"path": "docs/", "exclude": ["wiki/"]},
    {"path": "CLAUDE.md"}
  ],
  "output": "docs/wiki/"
}
```

For **assess-rfe**:
```json
{
  "version": 1,
  "name": "Assess-RFE",
  "sources": [
    {"path": "docs/", "exclude": ["wiki/"]},
    {"path": "agents/"}
  ],
  "output": "docs/wiki/"
}
```

**3. `docs/learned/`** — Directory scaffold
```
docs/learned/
  corrections/.gitkeep
  patterns/.gitkeep
```

That's it. No code changes, no workflow changes, no dependency additions.

## How it works

Once these files are in place and the repos are used as ACP workspaces:

1. **During sessions**: The `suggest_memory` tool becomes available. When the agent discovers something worth remembering (a scoring pattern, a submission convention, a rubric calibration insight), it writes a markdown file to `docs/learned/` and opens a **draft PR** with the `continuous-learning` label.

2. **Between sessions**: A team member reviews the draft PRs. Merge the useful ones, close the rest. This is the quality gate — no unreviewed knowledge reaches agents.

3. **After merge**: The wiki compiler runs (manually via `/wiki-compile` or automatically via GitHub Action) and compiles `docs/learned/` into topic articles in `docs/wiki/`.

4. **Next session**: The SessionStart hook injects only the **wiki INDEX** (a lightweight topic listing — a few hundred tokens, not the full articles). The agent reads specific topic articles on demand when they're relevant. This is progressive disclosure: the index tells the agent what knowledge exists, and the agent pulls in details only when it needs them. No context window bloat.

## What each repo learns

### rfe-creator

| Type | Example | Value |
|------|---------|-------|
| Correction | "RFEs covering >2 OpenShift components should be split before submission" | Avoids rejection for scope |
| Pattern | "Always include NFR section for GPU workloads — reviewers flag it" | Faster approval |
| Correction | "Don't use 'should' in acceptance criteria — use 'must'" | Rubric compliance |
| Pattern | "Strategy refinement requires architecture context from opendatahub-io" | Reduces iteration |

### assess-rfe

| Type | Example | Value |
|------|---------|-------|
| Pattern | "RFEs with score <3 on 'problem clarity' almost always fail review" | Better calibration |
| Correction | "Don't penalize RFEs for missing NFRs when the RFE type is 'investigation'" | Fairer scoring |
| Pattern | "Infrastructure RFEs need dependency analysis; feature RFEs need user story format" | Domain-specific rubric |
| Correction | "Scoring 5/5 on any criterion should require explicit justification" | Prevents score inflation |

### Cross-repo compound learning

The two repos operate on the same artifacts (RHAIRFE Jira issues). Over time:
- **rfe-creator** learns what assess-rfe predicts will fail → writes better RFEs upfront
- **assess-rfe** learns from rfe-creator's submission outcomes → refines its rubric weights

This creates a feedback loop where both agents improve each other.

## Optional: GitHub Action for auto-compilation

If you want wiki compilation to happen automatically on merge (instead of manually):

```yaml
# .github/workflows/compile-wiki-ambient.yml
name: Compile Wiki
on:
  push:
    branches: [main]
    paths: ['docs/**', 'CLAUDE.md', 'agents/**']
jobs:
  compile:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: '22' }
      - run: npm install -g @llm-wiki-compiler/cli
      - run: wiki-compile --config .wiki-compiler.json
      - uses: stefanzweifel/git-auto-commit-action@v5
        with: { commit_message: 'chore: compile wiki' }
```

This is optional — the wiki can always be compiled manually via `/wiki-compile` in any Claude Code session.

## Prerequisites

- ACP platform PR #1327 (learning agent loop) must be merged — it ships the `suggest_memory` tool and insight extraction pipeline
- `llm-wiki-compiler` plugin installed in Claude Code
- Repos used as ACP workspaces with `learning-agent-loop` feature flag enabled

## Next steps

1. Review this proposal
2. If approved, open PRs adding the 3 files to each repo
3. Enable `learning-agent-loop` flag on the workspace
4. Run a few sessions — knowledge accumulates from the first correction
