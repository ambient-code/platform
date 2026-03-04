---
title: "Spec-kit Workflow"
---

The Spec-kit workflow drives specification-driven development. It guides the agent through a structured process of creating detailed feature specifications, generating implementation plans, breaking plans into tasks, and implementing the result -- all grounded in a formal spec rather than ad-hoc prompting.

## When to use

- You have a feature idea and want to produce a detailed specification before writing any code.
- You want a structured breakdown from high-level requirements into implementable tasks.
- You want the agent to follow a spec-first methodology: specify, plan, break down, then implement.

## Commands

The workflow provides a set of slash commands that map to distinct phases:

| Command | Purpose |
|---------|---------|
| `/speckit.specify` | Create a detailed feature specification from a description. |
| `/speckit.analyze` | Analyze an existing specification or codebase for context. |
| `/speckit.clarify` | Ask clarifying questions to refine the specification. |
| `/speckit.plan` | Generate a technical implementation plan from the spec. |
| `/speckit.tasks` | Break the plan into discrete, actionable tasks. |
| `/speckit.implement` | Implement the tasks following the plan. |
| `/speckit.checklist` | Generate a validation checklist for the implementation. |
| `/speckit.constitution` | Review the guiding principles for the specification process. |

### Typical flow

1. **`/speckit.specify {feature description}`** -- Provide a feature description and the agent produces a formal specification document.
2. **`/speckit.clarify`** -- The agent asks targeted questions to fill gaps in the spec.
3. **`/speckit.plan`** -- The agent creates a technical implementation plan based on the finalized spec.
4. **`/speckit.tasks`** -- The plan is broken into discrete tasks with clear acceptance criteria.
5. **`/speckit.implement`** -- The agent works through the task list, implementing each one.
6. **`/speckit.checklist`** -- A validation checklist is generated to verify the implementation matches the spec.

## Sub-agent collaboration

The Spec-kit workflow has access to a roster of 21 specialized sub-agents that it engages automatically based on the phase and complexity. Key agents include:

**Engineering and architecture:**
- **Archie (Architect)** -- System design, technical vision, architectural patterns.
- **Stella (Staff Engineer)** -- Technical leadership, implementation excellence, code review.
- **Neil (Test Engineer)** -- Testing strategy, QA, test automation.
- **Lee (Team Lead)** -- Team coordination and delivery oversight.
- **Emma (Engineering Manager)** -- Engineering management and capacity planning.

**Product and strategy:**
- **Parker (Product Manager)** -- Market strategy, customer feedback, business value.
- **Olivia (Product Owner)** -- Backlog management, user stories, sprint planning.
- **Dan (Senior Director)** -- Strategic direction and executive alignment.
- **Diego (Program Manager)** -- Cross-team program management.

**UX and design:**
- **Aria (UX Architect)** -- UX strategy, journey mapping, design system architecture.
- **Felix (UX Feature Lead)** -- Feature-level UX design, interaction design.
- **Steve (UX Designer)** -- Visual design and interaction patterns.
- **Uma (UX Team Lead)** -- UX team coordination.

**Content and documentation:**
- **Terry (Technical Writer)** -- Technical documentation standards.
- **Tessa (Writing Manager)** -- Writing quality and editorial oversight.
- **Casey (Content Strategist)** -- Content strategy and information architecture.

The agent decides when to delegate to these specialists. You do not need to invoke them manually.

## Generated artifacts

All specification artifacts are written to the `artifacts/specs/` directory:

| Artifact | Path |
|----------|------|
| Feature specification | `artifacts/specs/**/spec.md` |
| Implementation plan | `artifacts/specs/**/plan.md` |
| Task breakdown | `artifacts/specs/**/tasks.md` |
| RFE document | `artifacts/rfe.md` |

## Workspace initialization

The first time the workflow runs, it executes an initialization script (`.specify/scripts/bash/init-workspace.sh`) that sets up the workspace structure and creates symlinks for shared artifacts. This happens automatically at session startup.

## Tips

- **Start with `/speckit.specify`.** The entire workflow builds on a good specification. Invest time in getting the spec right before moving to planning.
- **Use `/speckit.clarify` iteratively.** Run it multiple times if the agent identifies ambiguities. A well-clarified spec produces a much better plan.
- **Review the plan before tasks.** The implementation plan is your last chance to adjust the technical approach before the agent breaks it into work items.
- **Check the checklist.** After implementation, run `/speckit.checklist` to verify coverage against the original spec. This catches drift between what was specified and what was built.
