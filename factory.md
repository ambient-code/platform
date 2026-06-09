# Factory Configuration
<!-- This file configures the Remote Factory for your project. -->
<!-- The factory reads this during Init mode and generates .factory/config.json from it. -->
<!-- Fill in each section below. -->

## Goal
<!-- A single sentence describing what this project should achieve. -->

Kubernetes-native AI automation platform that orchestrates agentic sessions through containerized microservices (Go backend/operator, NextJS frontend, Python runner).

## Scope

### Modifiable
<!-- Files and directories the factory is allowed to create or edit. -->
<!-- One path per line. Glob patterns are supported. -->

- components/backend/**/*.go
- components/operator/**/*.go
- components/frontend/src/**/*.ts
- components/frontend/src/**/*.tsx
- components/runners/ambient-runner/**/*.py
- components/ambient-cli/**/*.go
- components/public-api/**/*.go
- components/ambient-api-server/**/*.go
- components/ambient-sdk/**/*.go
- components/ambient-sdk/**/*.py

### Read-only
<!-- Files the factory may read but must never modify. -->

- CLAUDE.md
- README.md
- components/manifests/**/*
- eval/**/*

## Guards
<!-- Rules the factory must never violate. Checked before every commit. -->

- Do not delete or overwrite existing tests
- Do not modify files outside the declared scope
- Do not introduce secrets or credentials into the repository
- All user-facing API ops must use GetK8sClientsForRequest(c), never the backend service account
- No tokens in logs/errors/responses — use len(token) for logging
- No panic() in production Go code — return fmt.Errorf with context
- No any types in frontend TypeScript

## Eval

### Command
<!-- The shell command the factory runs to score a change. -->
<!-- It must output JSON to stdout matching the EvalResult format. -->

```bash
python eval/score.py
```

### Threshold
<!-- Minimum composite score (0.0-1.0) required to keep a change. -->

0.8

## Target Branch
<!-- Branch that experiment PRs target. Default: main -->
<!-- Set to a different branch (e.g. factory/dev) to stage factory changes before merging to main -->

main

## Project Eval
<!-- No project-specific eval dimensions -->

## Eval Weights
- hygiene: 0.50
- growth: 0.50

## Smoke Test
<!-- Optional shell command that must pass before any change is kept. -->
<!-- If configured, this runs as part of `factory precheck` — failure = mandatory revert. -->
<!-- Use for e2e verification: hit an endpoint, run a CLI command, check a process starts. -->
<!-- Example:
```bash
curl -sf http://localhost:8000/health
```
-->

## Constraints
<!-- Soft rules that guide behavior but don't block commits. -->

- Prefer small, incremental changes over large rewrites
- Each change should be accompanied by at least one test
- Follow the existing code style and conventions
- Use conventional commits (squashed on merge to main)
- OwnerReferences on all K8s child resources

## Research Target
<!-- Not a research project -->

## Mutable Surfaces
<!-- Files the Builder is allowed to modify during research experiments. -->
<!-- One glob pattern per line. Only used in research mode. -->
<!-- Example:
- src/**/*.py
- config/*.yaml
-->

## Fixed Surfaces
<!-- Ground truth files, test data, eval infrastructure. -->
<!-- These files are fingerprinted for leakage detection and MUST NOT be modified. -->
<!-- One glob pattern per line. Only used in research mode. -->
<!-- Example:
- tests/gold/*.json
- eval/**/*.py
- data/benchmark/*.jsonl
-->

## Research Constraints
<!-- Additional rules for the research loop. Only used in research mode. -->
<!-- Example:
- Do not use GPT-4 (cost constraint)
- Each experiment must complete within 30 minutes
-->

## Cost Budget
<!-- Per-cycle or total budget constraints for research experiments. -->
<!-- Example: $5/cycle, $50 total -->
