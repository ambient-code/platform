# Feature Specification: Advanced SDK Options

**Feature Branch**: `feat/advanced-sdk-options-v2`
**Created**: 2026-04-15
**Status**: Draft
**Input**: Re-implementation of PR #1146 — expose Claude Agent SDK options in session creation UI

## Overview

Allow platform users to configure Claude Agent SDK parameters when creating sessions. Options flow from a frontend form through backend validation to the runner, where they merge into `ClaudeAgentOptions`. Gated behind workspace feature flag `advanced-sdk-options` (disabled by default). No CRD changes — options travel as a JSON string in the existing `environmentVariables` map.

### Data Flow

```
Frontend form → sdkOptions on POST request
  → Backend: allowlist filter + type validation → JSON string
    → CR environmentVariables["SDK_OPTIONS"]
      → Runner: parse, denylist filter, merge into adapter options
        → ClaudeAgentAdapter(options)
```

### Security

- **Backend allowlist**: Only permitted keys with valid types pass through. Everything else silently dropped.
- **Runner denylist**: Blocks platform-internal keys (`cwd`, `api_key`, etc.) even if backend is bypassed.
- **System prompt**: Append-only. User text goes under `## Custom Instructions`, never replaces platform prompt.
- **Feature flag**: UI-only gate. The API always accepts `sdkOptions` for programmatic callers.

## User Scenarios & Testing

### User Story 1 - Configure SDK Options on Session Creation (Priority: P1)

A user creating a session wants to tune Claude — lower temperature, increase token budget, set a custom system prompt, or restrict tools.

**Why this priority**: The entire feature. Everything else is a subset of this.

**Independent Test**: Create a session with `sdkOptions` via API, verify the runner receives and applies them.

**Acceptance Scenarios**:

1. **Given** `advanced-sdk-options` is enabled for a workspace, **When** a user opens the new session page, **Then** a collapsible "Advanced SDK Options" section appears (collapsed by default).

2. **Given** the user sets temperature to 0.3 and max_turns to 5 and submits, **When** the backend processes the request, **Then** the CR has `SDK_OPTIONS={"temperature":0.3,"max_turns":5}` in its env vars.

3. **Given** the runner pod starts with `SDK_OPTIONS`, **When** the adapter initializes, **Then** the parsed options are merged into `ClaudeAgentOptions` (minus denylisted keys).

4. **Given** `advanced-sdk-options` is disabled, **When** a user opens the new session page, **Then** the advanced options section is not visible.

5. **Given** the user provides a system_prompt, **When** the runner merges options, **Then** the platform prompt is preserved and the user text is appended under `## Custom Instructions`.

6. **Given** `SDK_OPTIONS` contains invalid JSON, **When** the runner parses it, **Then** it logs a warning and proceeds with platform defaults.

---

### User Story 2 - SDK Options Drift Detection (Priority: P2)

The Claude Agent SDK evolves. The platform must detect when `ClaudeAgentOptions` fields change and alert maintainers so the allowlist/UI stay current.

**Why this priority**: Without this, the platform silently drifts from the SDK. Users can't access new options and removed options cause silent failures.

**Independent Test**: Run the drift workflow via `workflow_dispatch`, verify it detects a simulated field change.

**Acceptance Scenarios**:

1. **Given** `claude-agent-sdk` on PyPI has added a new field, **When** the weekly workflow runs, **Then** it updates `sdk-options-manifest.json` and opens a PR labeled `amber:auto-fix`.

2. **Given** no drift exists, **When** the workflow runs, **Then** no PR is created and the job succeeds cleanly.

3. **Given** the workflow encounters a PyPI install failure, **When** it runs, **Then** it fails loudly (non-zero exit) rather than silently skipping.

---

### Edge Cases

- `SDK_OPTIONS` is a JSON array instead of object → runner logs warning, uses platform defaults.
- User sends `sdkOptions` with unknown keys → backend silently drops them, no error.
- User sends `temperature: "hot"` → backend returns 400 with type validation error.
- `SDK_OPTIONS` contains `api_key` → runner denylist blocks it.
- User sends empty `sdkOptions: {}` → no `SDK_OPTIONS` env var set (no-op).

## Requirements

### Functional Requirements

**Backend:**

- **FR-001**: `CreateAgenticSessionRequest` accepts optional `sdkOptions map[string]interface{}`.
- **FR-002**: Backend filters `sdkOptions` through an allowlist and validates types per key. Returns 400 on type mismatch. Silently drops unknown keys.
- **FR-003**: Filtered options are JSON-serialized into `environmentVariables["SDK_OPTIONS"]` on the CR.

**Runner:**

- **FR-004**: Runner parses `SDK_OPTIONS` env var as JSON on adapter init. Malformed input → warn + use defaults.
- **FR-005**: Runner applies a denylist for platform-internal keys (`cwd`, `api_key`, `mcp_servers`, `setting_sources`, `stderr`, `resume`, `continue_conversation`, `add_dirs`, `cli_path`, `env`). Logs a warning per blocked key.
- **FR-006**: `system_prompt` is appended under `## Custom Instructions`, not replaced.

**Frontend:**

- **FR-007**: `AdvancedSdkOptions` component renders behind `advanced-sdk-options` workspace flag. Collapsed by default.
- **FR-008**: Field names use snake_case matching the Python SDK wire format.
- **FR-009**: `sdkOptions` is only included in the create request when at least one value is set.

**Drift Detection:**

- **FR-010**: Weekly GHA workflow introspects `ClaudeAgentOptions` from `claude-agent-sdk` PyPI package and compares against `sdk-options-manifest.json`.
- **FR-011**: On drift: updates manifest, opens PR with `amber:auto-fix` label. On no drift: clean exit. On error: hard fail.

**Feature Flag:**

- **FR-012**: `advanced-sdk-options` defined in `flags.json` with `scope:workspace` tag. Gates UI only.

### Key Entities

- **SdkOptions**: Map of SDK parameter names (snake_case) to values. Travels as JSON string through CR env vars.
- **SDK Options Manifest**: JSON file recording `ClaudeAgentOptions` fields/types from PyPI. Source of truth for drift detection.

## Success Criteria

- **SC-001**: Sessions created with custom SDK options produce observably different agent behavior.
- **SC-002**: Backend rejects invalid types with 400.
- **SC-003**: Runner never passes denylisted keys to the SDK.
- **SC-004**: System prompt append-only behavior verified by test.
- **SC-005**: Drift workflow detects field changes on manual trigger.
