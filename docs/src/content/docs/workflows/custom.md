---
title: "Custom Workflows"
---

Create your own workflows from any Git repository. Custom workflows use the same structure as the built-in ones -- they are loaded dynamically when a session starts.

## Getting started

To load a custom workflow:

1. Open the **New Session** dialog.
2. In the **Workflow** dropdown, select **Custom Workflow...**.
3. Enter the **Git URL**, **branch**, and **path** to the workflow directory.

The platform clones the repository and loads the configuration into the session. You can also use this to test workflow changes on a feature branch before merging.

To start from scratch:

```bash
mkdir -p my-workflow/.ambient
```

Create `.ambient/ambient.json` (the only required file):

```json
{
  "name": "My Workflow",
  "description": "A workflow that helps with X",
  "systemPrompt": "You are an assistant for X.",
  "startupPrompt": "Welcome! Use /start to begin."
}
```

Push to a Git repository and load it as a custom workflow. Or use one of the existing workflows in the [workflows repository](https://github.com/ambient-code/workflows) as a starting point and modify it.

---

## Anatomy of a workflow

```
my-workflow/
  .ambient/
    ambient.json              # Workflow configuration (ACP-specific)
    rubric.md                 # Quality evaluation criteria (optional)
  .claude/                    # Standard Claude Code configuration
  scripts/                    # Helper scripts the agent can execute
  templates/                  # Reference files for generating outputs
  CLAUDE.md                   # Main workflow instructions
```

The only ACP-specific files are under `.ambient/`. Everything else (`.claude/`, `CLAUDE.md`) is standard [Claude Code configuration](https://docs.anthropic.com/en/docs/claude-code).

### ambient.json

The configuration file that makes a directory an ACP workflow. All fields are optional -- the runner handles missing values gracefully.

| Field | Purpose |
|-------|---------|
| `name` | Display name in the workflow selector. |
| `description` | Short explanation shown in the selector. |
| `systemPrompt` | Agent persona and hard rules. Best for short, stable instructions like role definitions ("You are a security auditor") and guardrails ("NEVER modify production configs"). Works the same as `CLAUDE.md` but lives in JSON -- use `CLAUDE.md` for anything longer than a few paragraphs. |
| `startupPrompt` | Hidden prompt sent to the agent at session start. The user doesn't see it, but the agent responds to it. Best for interactive sessions: "Introduce yourself and list your available commands." Less useful for autonomous workflows. |
| `rubric` | Quality evaluation config. See [Rubrics](#rubrics). |

### CLAUDE.md

Most mature workflows use `CLAUDE.md` as the primary instruction source rather than packing everything into `systemPrompt`. A JSON string gets awkward fast for multi-phase methodologies.

The pattern:

- **`systemPrompt`** -- Short persona + hard rules.
- **`CLAUDE.md`** -- The full methodology: phases, decision trees, output formats, examples, edge cases.

Both are loaded into the agent's context, so the effect is the same. But `CLAUDE.md` is easier to read, edit, and review in PRs.

### Scripts

Scripts give you **reproducibility** and **efficiency**. A task the agent would handle through dozens of tool calls can be collapsed into a single `bash scripts/do-the-thing.sh` -- same result every time, fewer tokens.

Use input parameters to make scripts dynamic and reusable:

```bash
#!/bin/bash
# scripts/analyze-component.sh <component-path> <output-dir>
COMPONENT=$1
OUTPUT_DIR=$2

find "$COMPONENT" -name '*.go' | xargs grep -l 'TODO' > "$OUTPUT_DIR/todos.txt"
go vet "$COMPONENT/..." 2> "$OUTPUT_DIR/vet-results.txt"
```

The agent calls `bash scripts/analyze-component.sh src/backend artifacts/analysis` instead of running each step individually. The same script works for any component passed in.

### Templates

For workflows that produce structured outputs, include templates the agent uses as a starting point:

```
templates/
  report-template.md
  rfe-template.md
  config-template.yaml
```

In your `CLAUDE.md`:

> Follow the format in `templates/report-template.md`. Generate a customized version at `artifacts/my-workflow/report.md`.

Templates ensure consistent structure across sessions and make it easy to update the format without rewriting prompt instructions.

---

## Rubrics

Rubrics define quality criteria the agent uses to self-evaluate its output. Scores are logged to Langfuse via the `evaluate_rubric` tool.

### Configuration in ambient.json

The `rubric` field tells the platform when to evaluate and defines an optional schema for structured scores:

```json
{
  "rubric": {
    "activationPrompt": "After generating the report, evaluate its quality.",
    "schema": {
      "type": "object",
      "properties": {
        "completeness": {
          "type": "number",
          "description": "Structural completeness (1-5)"
        },
        "clarity": {
          "type": "number",
          "description": "Language clarity (1-5)"
        }
      },
      "required": ["completeness", "clarity"]
    }
  }
}
```

| Field | Purpose |
|-------|---------|
| `activationPrompt` | When to run the evaluation (e.g., "after creating the RFE document"). |
| `schema` | JSON Schema for structured metadata. These become input parameters on the `evaluate_rubric` tool, so the agent submits per-criterion scores -- not just a single number. |

### The rubric.md file

A markdown file at `.ambient/rubric.md` with the detailed scoring criteria. The agent reads this before evaluating:

```markdown
**Structural Completeness** (1-5)
Score 1: No discernible structure.
Score 3: Contains core sections. Optional sections ignored.
Score 5: Perfectly organized with all relevant sections.

**Language Quality** (1-5)
Score 1: Unprofessional or overly verbose.
Score 3: Standard technical writing.
Score 5: Concise, objective, suitable for external stakeholders.
```

### How it works

When both the `rubric` config and `rubric.md` are present:

1. The platform registers an `evaluate_rubric` MCP tool on the session.
2. The agent reads `.ambient/rubric.md` and evaluates its output.
3. The agent calls `evaluate_rubric` with a score, reasoning, and structured metadata.
4. The score is logged to Langfuse.

If the `rubric` key is missing from `ambient.json`, no evaluation tool is registered.

---

## Autonomous vs. interactive design

Workflows generally fall into two categories, and the design approach is different for each.

### Autonomous workflows

Run without human intervention -- triggered by GitHub Actions, scheduled jobs, or fire-and-forget sessions.

- **Be prescriptive.** Spell out every decision point. The agent can't ask for clarification.
- **Define exit criteria.** "Generate the report, write it to `artifacts/`, and complete the session."
- **Use rubrics as quality gates.** No one is reviewing in real-time.
- **Handle errors explicitly.** Retry, skip, or report -- don't assume a human will intervene.
- **Minimize reliance on slash commands.** The system prompt should drive the agent through all phases automatically. Workflows can still define commands for optional interactive use, but the autonomous path should not depend on a human typing them.
- **Pass structured inputs.** Repo URLs, JQL queries, issue numbers -- not vague prompts.

Most workflows support both autonomous and interactive use. For example, [Bugfix](/workflows/bugfix/) and [Triage](/workflows/triage/) work well autonomously but also provide slash commands for interactive sessions.

### Interactive workflows

A human collaborates with the agent, guiding the process and providing judgment.

- **Use slash commands for phases.** `/analyze`, `/implement`, `/review` -- the user controls pacing and reviews between steps.
- **Write a `startupPrompt`.** Greet the user, list commands, suggest where to begin.
- **Leave room for iteration.** Let the user say "redo that" or "try a different approach."
- **Pause at decision points.** "Before submitting the PR, present the changes and wait for approval."
- **Keep commands focused.** One command, one job. Don't bundle diagnose + fix + test into a single command.

Examples: [Spec-kit](/workflows/spec-kit/), [PRD/RFE](/workflows/prd-rfe/).

---

## Refining your workflow

The best workflows are built through iteration.

### Debrief after each run

Ask the agent:

- "Did you have to work around anything not covered in the instructions?"
- "Were any commands or instructions unclear or contradictory?"
- "Did you deviate from the workflow at any point? Why?"
- "What information were you missing?"

### Use a fresh session

Do the debrief in a **separate session** -- not the one that just ran the workflow. Export the conversation and paste it into a new session or locally in Claude Code. The agent evaluates more honestly with a fresh context rather than defending decisions it just made.

### Iterate on the prompt

Most improvements come from refining `CLAUDE.md`, not writing more tooling:

- Add handling for edge cases the agent encountered.
- Clarify criteria that were ambiguous.
- Remove instructions the agent consistently ignored -- they may contradict something else.
- Add examples of expected output when the format wasn't right.

---

## General tips

- **Start simple.** Begin with `ambient.json` and `CLAUDE.md`. Add scripts, templates, and rubrics as the workflow matures.
- **Test end-to-end.** Run through the entire workflow before sharing it.
- **Keep prompts focused.** Clear and concise beats exhaustive.
- **Make rubric criteria measurable.** "Code should be good" is vague. "All new functions have unit tests" is actionable.
- **See the [Claude Code docs](https://docs.anthropic.com/en/docs/claude-code)** for commands, skills, CLAUDE.md, and settings.json.
