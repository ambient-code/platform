# Runner Constitution

**Version**: 1.0.0
**Ratified**: 2026-03-28
**Parent**: [ACP Platform Constitution](../memory/constitution.md)

This constitution governs the `components/runners/ambient-runner/` component and its supporting CI workflows. It inherits all principles from the platform constitution and adds runner-specific constraints.

---

## Principle R-I: Version Pinning

All external tools installed in the runner image MUST be version-pinned.

- CLI tools (gh, glab) MUST use `ARG <TOOL>_VERSION=X.Y.Z` in the Dockerfile and be installed via pinned binary downloads — never from unpinned package repos.
- Python packages (uv, pre-commit) MUST use `==X.Y.Z` pins at install time.
- npm packages (gemini-cli) MUST use `@X.Y.Z` pins.
- The base image MUST be pinned by SHA digest.
- Versions MUST be declared as Dockerfile `ARG`s at the top of the file for automated bumping.

**Rationale**: Unpinned installs cause non-reproducible builds and silent regressions. Pinning enables automated freshness tracking and controlled upgrades.

## Principle R-II: Automated Freshness

Runner tool versions MUST be checked for staleness automatically.

- The `runner-tool-versions.yml` workflow runs weekly and on manual dispatch.
- It checks all pinned components against upstream registries.
- When updates are available, it opens a single PR with a version table.
- The workflow MUST NOT auto-merge; a human or authorized agent reviews.

**Rationale**: Pinned versions go stale. Automated freshness checks balance reproducibility with security and feature currency.

## Principle R-III: Dependency Update Procedure

Dependency updates MUST follow the documented procedure in `docs/UPDATE_PROCEDURE.md`.

- Python dependencies use `>=X.Y.Z` floor pins in pyproject.toml, resolved by `uv lock`.
- SDK bumps (claude-agent-sdk) MUST trigger a review of the frontend Agent Options schema for drift.
- Base image major version upgrades (e.g., UBI 9 → 10) require manual testing.
- Lock files MUST be regenerated after any pyproject.toml change.

**Rationale**: A structured procedure prevents partial updates, version conflicts, and schema drift between backend SDK types and frontend forms.

## Principle R-IV: Image Layer Discipline

Dockerfile layers MUST be optimized for size and cacheability.

- System packages (`dnf install`) SHOULD be consolidated into a single `RUN` layer.
- Build-only dependencies (e.g., `python3-devel`) MUST be removed in the same layer where they are last used, not in a separate layer.
- Binary CLI downloads (gh, glab) SHOULD share a single `RUN` layer to avoid redundant arch detection.
- `dnf clean all` and cache removal MUST happen in the same `RUN` as the install.

**Rationale**: Docker layers are additive. Removing packages in a later layer doesn't reclaim space — it only adds whiteout entries.

## Principle R-V: Agent Options Schema Sync

The frontend Agent Options form MUST stay in sync with the claude-agent-sdk types.

- `schema.ts` defines the Zod schema matching `ClaudeAgentOptions` from the SDK.
- `options-form.tsx` renders the form from the schema.
- Editor components in `_components/` MUST use stable React keys (ref-based IDs) for record/map editors to prevent focus loss on rename.
- Record editors MUST prevent key collisions on add operations.
- The form is gated behind the `advanced-agent-options` Unleash flag.

**Rationale**: Schema drift between SDK and frontend creates silent data loss or validation errors. Stable keys prevent UX bugs in dynamic form editors.

## Principle R-VI: Bridge Modularity

Agent bridges (Claude, Gemini, LangGraph) MUST be isolated modules.

- Each bridge lives in `ambient_runner/bridges/<name>/`.
- Bridges MUST NOT import from each other.
- Shared logic lives in `ambient_runner/` (bridge.py, platform/).
- New bridges follow the same directory structure and registration pattern.

**Rationale**: Bridge isolation enables independent testing, deployment, and addition of new AI providers without cross-contamination.

---

## Governance

- This constitution is versioned using semver.
- Amendments require a PR that updates this file and passes the SDD preflight check.
- The platform constitution takes precedence on any conflict.
- Compliance is reviewed as part of runner-related PR reviews.
