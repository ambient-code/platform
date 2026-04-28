# Tasks: Advanced SDK Options

**Input**: Design documents from `/specs/010-advanced-sdk-options/`
**Prerequisites**: plan.md (required), spec.md (required)

**Execution skill**: `superpowers:subagent-driven-development` (one subagent per phase, review between phases)

## Phase 1: Setup

- [ ] T001 Add `advanced-sdk-options` feature flag to `components/manifests/base/core/flags.json` with `scope:workspace` tag

### Commit: `feat(flags): add advanced-sdk-options workspace feature flag`

---

## Phase 2: Backend — SDK Options Filtering (TDD)

**Goal**: Backend accepts `sdkOptions` on session create, filters through allowlist, validates types, serializes to `SDK_OPTIONS` env var on the CR.

- [ ] T010 [US1] Add `SdkOptions map[string]interface{}` field with `json:"sdkOptions,omitempty"` to `CreateAgenticSessionRequest` in `components/backend/types/session.go`
- [ ] T011 [US1] Create `components/backend/handlers/sessions_sdk_options_test.go` (TDD — tests first, then implement). Tests: valid keys pass through, unknown keys silently dropped, empty map returns nil, string/numeric/bool/slice type checks, invalid type returns error
- [ ] T012 [US1] Implement `allowedSdkOptionKeys` map and `filterSdkOptions` in `components/backend/handlers/sessions.go`. Allowlist keys: all fields from `claudeAgentOptionsSchema` minus denylisted keys (`cwd`, `resume`, `mcp_servers`, `setting_sources`, `continue_conversation`, `add_dirs`, `cli_path`, `settings`, `permission_prompt_tool_name`, `fork_session`). Include `validateSdkOptionValue` for basic type checks on primitives (string, float, int, bool, slice). Complex objects (hooks, agents, sandbox, thinking, mcp_servers) pass through as-is — JSON marshal handles them.
- [ ] T013 [US1] Wire into `CreateAgenticSession` handler: if `req.SdkOptions` is non-empty, call `filterSdkOptions`, return 400 on error, JSON-serialize into `envVars["SDK_OPTIONS"]`
- [ ] T014 [US1] Run backend tests: `cd components/backend && go test -tags test -run TestSdkOptions ./handlers/`

### Commit: `feat(backend): add sdkOptions allowlist and type validation`

---

## Phase 3: Runner — SDK_OPTIONS Parsing (TDD)

**Goal**: Runner parses `SDK_OPTIONS` env var, applies denylist, merges system_prompt append-only, passes remaining options to adapter.

- [ ] T020 [US1] Create `components/runners/ambient-runner/tests/test_sdk_options.py` (TDD). Tests: valid JSON parsed, malformed JSON returns empty dict, JSON array returns empty dict, denylisted keys blocked with warning, non-denylisted keys pass, system_prompt appended under `## Custom Instructions` heading
- [ ] T021 [US1] Implement `_SDK_OPTIONS_DENYLIST` frozenset and SDK_OPTIONS parsing in `components/runners/ambient-runner/ambient_runner/bridges/claude/bridge.py`. In `_ensure_adapter`: parse env var, apply denylist with per-key warning logs, handle system_prompt append, merge remaining keys into options dict
- [ ] T022 [US1] Run runner tests: `cd components/runners/ambient-runner && python -m pytest tests/test_sdk_options.py -v`

### Commit: `feat(runner): parse SDK_OPTIONS env var with denylist and system prompt merge`

---

## Phase 4: Frontend — Wire Form into Session Creation (TDD)

**Goal**: Wrap existing `claude-agent-options/` form in a collapsible container, gate behind feature flag, wire into session create flow.

**Existing on main**: `components/frontend/src/components/claude-agent-options/` has `AgentOptionsFields`, `claudeAgentOptionsSchema`, `claudeAgentOptionsDefaults`, and 11 field editors. Reuse these.

- [ ] T030 [US1] Rename `agentOptions` to `sdkOptions` in `components/frontend/src/types/api/sessions.ts` and `components/frontend/src/types/agentic-session.ts`
- [ ] T031 [US1] Create `components/frontend/src/components/__tests__/advanced-sdk-options.test.tsx` (TDD). Tests: not rendered when flag is disabled, renders collapsed by default when flag enabled, expands on click, form fields visible when expanded
- [ ] T032 [US1] Create `components/frontend/src/components/advanced-sdk-options.tsx` — collapsible wrapper using Shadcn `Collapsible`. Imports `AgentOptionsFields` from `claude-agent-options`. Props: `projectName`, `form: UseFormReturn<ClaudeAgentOptionsForm>`, `disabled?`. Uses `useWorkspaceFlag(projectName, "advanced-sdk-options")` to gate visibility
- [ ] T033 [US1] Wire into `components/frontend/src/app/projects/[name]/sessions/[sessionName]/components/new-session-view.tsx`: add `useForm<ClaudeAgentOptionsForm>` with defaults, render `<AdvancedSdkOptions>`, pass non-empty form values as `sdkOptions` in `onCreateSession` callback
- [ ] T034 [US1] Wire into `components/frontend/src/app/projects/[name]/new/page.tsx`: accept `sdkOptions` in config, spread into create mutation payload
- [ ] T035 [US1] Run frontend tests and build: `cd components/frontend && npx vitest run && npm run build`

### Commit: `feat(frontend): add collapsible AdvancedSdkOptions gated by workspace flag`

---

## Phase 5: Drift Detection (US2)

**Goal**: Weekly GHA workflow introspects `ClaudeAgentOptions` from PyPI, compares against manifest, opens PR on drift.

- [ ] T040 [US2] Generate `components/runners/ambient-runner/sdk-options-manifest.json` by introspecting the current `claude-agent-sdk` package: install via `uv pip install claude-agent-sdk`, extract fields from `ClaudeAgentOptions.model_fields` (Pydantic), write `{"generatedFrom": "claude-agent-sdk", "generatedAt": "<ISO>", "sdkVersion": "<version>", "options": {"field_name": {"type": "<annotation>", "required": <bool>}}}`
- [ ] T041 [US2] Create `scripts/sdk-options-drift-check.py`: import `ClaudeAgentOptions`, introspect via `model_fields`, compare against manifest, exit 0 (no drift), exit 1 (drift found — write updated manifest), exit 2 (error). Must handle: `ImportError` (hard fail), Pydantic v1 vs v2 (check for `model_fields` vs `__fields__`)
- [ ] T042 [US2] Add drift check step to `.github/workflows/sdk-version-bump.yml`: after "Apply updates" step, `pip install claude-agent-sdk`, run `scripts/sdk-options-drift-check.py`, include updated manifest in the commit if drift found. No standalone workflow — drift detection runs as part of the daily SDK version bump.
- [ ] T043 [US2] Test drift detection end-to-end: run `python scripts/sdk-options-drift-check.py` locally, verify clean exit with current manifest

### Commit: `refactor(ci): consolidate drift detection into sdk-version-bump workflow`

---

## Phase 6: Verify

- [ ] T050 Run all component test suites: backend (`make test`), frontend (`npx vitest run --coverage`), runner (`python -m pytest tests/ -v`)
- [ ] T051 Run `npm run build` in frontend (must pass with 0 errors, 0 warnings)
- [ ] T052 Run `make lint` (pre-commit hooks on all changed files)
- [ ] T053 Grep changed `.tsx`/`.ts` files for `: any` or `as any` — must be zero
- [ ] T054 Cross-reference spec acceptance scenarios SC-001 through SC-005 against test coverage

### Commit (if fixes needed): `chore: lint and polish for advanced SDK options`

---

## Dependencies

- **Phase 1** → Phases 2, 3, 4 (flag must exist before frontend gate works)
- **Phase 2** → Phase 4 (backend must accept `sdkOptions` before frontend sends it)
- **Phase 3** → independent (runner reads env var, no compile-time dependency on backend)
- **Phase 4** → depends on Phase 2 (API contract)
- **Phase 5** → independent (drift workflow has no code dependency on other phases)
- **Phase 6** → all phases complete

### Parallel opportunities

- **Phases 2 + 3 + 5** can run in parallel (backend, runner, drift are independent)
- Within Phase 4: T030 (types) must precede T031-T035
