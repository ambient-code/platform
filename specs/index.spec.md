# Specs

Specs describe the **desired state** of the system. Code is the actual state. Development work reconciles the two.

## Principles

1. **Desired state, not issue tracking.** A spec says "the system SHALL do X" — not "here's a bug, here's my proposed fix." If the system doesn't match the spec, the code is wrong.

2. **Living documents.** Specs are never archived or superseded. They are amended, replaced, or deleted. A spec that no longer reflects desired behavior is removed, not moved to a graveyard.

3. **Behavior contracts, not implementation plans.** Specs describe observable behavior — inputs, outputs, error conditions, constraints. Implementation invariants, such as library choices, etc. (NOTE: NOT IMPLEMENTATION PLANS. THIS IS A DEVELOPMENT CONCERN.) belong in `workflows/`.

4. **Organized by capability domain.** Add domains when existing ones become too broad, not preemptively.

5. **Named descriptively.** Filenames follow `<descriptive-title>.spec.md`. Specs exceeding ~300 words or covering multiple distinct topics should be split into files within a containing directory.

## Format

Specs contain requirements, and each requirement has scenarios.

```markdown
# <Domain> Specification

## Purpose
High-level description of this spec's domain.

## Requirements

### Requirement: <Name>
The system SHALL <observable behavior>.

#### Scenario: <Name>
- GIVEN <precondition>
- WHEN <action>
- THEN <expected outcome>
- AND <additional outcome>
```

### Elements

| Element | Purpose |
|---------|---------|
| `## Purpose` | High-level description of the spec's domain |
| `### Requirement:` | A specific behavior the system must have |
| `#### Scenario:` | A concrete example of the requirement in action |
| `SHALL` / `MUST` | Absolute requirement (RFC 2119) |
| `SHOULD` | Recommended, but exceptions exist |
| `MAY` | Optional |

### What belongs in a spec

- Observable behavior users or downstream systems rely on
- Inputs, outputs, and error conditions
- External constraints (security, privacy, reliability, compatibility)
- Scenarios that can be tested or explicitly validated

### What does not belong in a spec

- Internal class/function names
- Library or framework choices
- Step-by-step implementation details

**Quick test:** if the implementation can change without changing externally visible behavior, it does not belong in the spec.

## Directory Structure

```
specs/
  index.spec.md                    # This file
  {capability-domain}/             # One directory per domain
    *.spec.md                      # Domain specs
    .agents/skills -> symlink      # Domain-specific skills
    .claude/skills -> symlink      # Claude Code discovery
  standards/                       # Cross-cutting engineering constraints
    {domain}/*.spec.md
```

### Current Domains

| Domain | Covers |
|--------|--------|
| `sessions/` | Lifecycle, initialization, status, messages, events, data model |
| `agents/` | Agent model, runtime registry, prompts, runners |
| `control-plane/` | Reconciliation, operator, scheduling |
| `integrations/` | MCP, Gerrit, external services |
| `standards/` | Cross-cutting engineering constraints by component |
