---
name: spec
description: >
  Create or modify a spec following the project's spec format and conventions.
  Use when the user wants to write a new spec, add requirements or scenarios
  to an existing spec, or restructure spec content. Triggers on: "write a spec",
  "create a spec", "add a requirement", "spec this out", "define the behavior",
  "what should the spec look like", "new spec for", "update the spec".
---

# Write or Modify a Spec

Help the user create or change a spec that describes desired system behavior.

## User Input

```text
$ARGUMENTS
```

## Before Anything Else

Read `specs/index.spec.md` in full. It defines what a spec is, the required format, naming conventions, and what does and does not belong. Do not proceed until you have read it.

## Steps

### 1. Understand the Desired Behavior

Ask the user what behavior they want to describe. Focus on:
- What should the system do? (not how it should be built)
- Who or what observes this behavior? (user, API consumer, downstream system)
- What are the constraints? (security, performance, compatibility)

If the user describes implementation details (class names, library choices, execution steps), redirect: those belong in `.agents/workflows/`, not in a spec.

### 2. Identify the Domain

Determine which capability domain this spec belongs to:

```bash
ls -d specs/*/
```

If no existing domain fits, propose a new one — but only if existing domains are genuinely too broad.

### 3. Discover Domain Context

`cd` into the target domain and check for existing specs and skills:

```bash
cd specs/{domain}
ls *.spec.md 2>/dev/null
ls .claude/skills/ 2>/dev/null
```

Read any existing specs in the domain to understand what's already covered and avoid duplication.

### 4. Write the Spec

Follow the format from `specs/index.spec.md`:

- **Purpose section** — one paragraph describing the domain or feature
- **Requirements** — each states an observable behavior using RFC 2119 keywords (SHALL, MUST, SHOULD, MAY)
- **Scenarios** — concrete Given/When/Then examples for each requirement that could be turned into tests

Quick checks before writing:
- Every requirement describes externally observable behavior
- Every scenario is testable
- No implementation details leaked in (class names, frameworks, step-by-step plans)
- RFC 2119 keywords are used deliberately, not decoratively

### 5. Name and Place the File

- Filename: `<descriptive-title>.spec.md`
- If the spec exceeds ~300 words or covers multiple distinct topics, split into a directory with multiple files
- Place in `specs/{domain}/`

### 6. Review with the User

Present the spec and ask:
- Does this capture the desired behavior?
- Are any scenarios missing (edge cases, error conditions)?
- Is the requirement strength right (MUST vs SHOULD vs MAY)?
