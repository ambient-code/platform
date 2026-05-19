---
name: frontend-development
description: >
  Required context for all frontend work. Loads frontend standards
  (conventions, React Query patterns, adapter testing requirements)
  before any code changes. Use when modifying anything under
  components/frontend/. Triggers on: any frontend code change,
  UI component work, React Query hooks, API adapter work, frontend
  bug fix, frontend refactor.
---

# Frontend Development

Before writing or changing any frontend code, load the project's frontend standards. These are non-negotiable constraints — not suggestions.

## User Input

```text
$ARGUMENTS
```

## Required Reading

Read all files in `specs/standards/frontend/` before proceeding:

```bash
ls specs/standards/frontend/
```

At minimum, this includes:

1. `specs/standards/frontend/conventions.spec.md` — zero-tolerance rules (no `any`, Shadcn only, React Query for all data, `type` over `interface`, colocated components, feature flags)
2. `specs/standards/frontend/react-query.spec.md` — file structure (ports/adapters/queries), data fetching patterns, testing requirements, validation checklist

Also read the API adapter spec if working on the services layer:

3. `specs/frontend/api-adapter.spec.md` — port interface contract, canonical types, 30 API domains, testability requirements

## Key Constraints

These apply to every frontend change, no exceptions:

- **Zero `any` types.** Use proper types or `unknown`.
- **Shadcn UI components only.** No custom buttons, inputs, dialogs.
- **React Query for all data fetching.** No manual `fetch()` in components.
- **`type` over `interface`.** Always.
- **All adapters must be tested.** No adapter merged without unit tests using recorded responses.
- **All hooks must be tested.** Hook tests use mock adapters, not real backends.
- **Port interfaces, not raw API calls.** React Query hooks consume ports, not `services/api/` directly.

## After Reading

Confirm you have loaded the standards, then proceed with the task. If any of your changes would violate these constraints, stop and discuss with the user before writing code.
