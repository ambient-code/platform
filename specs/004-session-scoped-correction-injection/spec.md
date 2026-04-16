# Feature Specification: Session-Scoped Correction Injection

**Feature Branch**: `004-session-scoped-correction-injection`
**Created**: 2026-04-15
**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Correction Injected Into Next Turn (Priority: P1)

A user corrects the agent mid-session (e.g., "No, use snake_case for those variables"). The agent calls `log_correction` which logs to Langfuse as it does today. On the next turn, the runner injects a `## Corrections from this session` block into the agent's context containing the correction. The agent reads that block and avoids repeating the same mistake.

**Why this priority**: This is the core value proposition. Without re-injection, corrections are fire-and-forget -- the agent can repeat the exact same mistake two turns later because its context has no memory of the correction.

**Independent Test**: Start a session, trigger a correction via `log_correction`, then inspect the system prompt or context file on the next turn to confirm the correction text appears in a `## Corrections from this session` block.

**Acceptance Scenarios**:

1. **Given** the agent calls `log_correction` with type=`style`, agent_action="Used camelCase", user_correction="Use snake_case", **When** the next turn begins, **Then** the agent's context includes a `## Corrections from this session` block containing those details.
2. **Given** no corrections have been logged in the session, **When** a turn begins, **Then** no `## Corrections from this session` block is present in the context.
3. **Given** a correction was logged, **When** the injected block is rendered, **Then** it includes the correction type, what the agent did, and what the user expected.

---

### User Story 2 - Corrections Accumulate and Cap at 20 (Priority: P1)

Over a long session, the user makes many corrections. Each one is appended to the in-memory correction ledger. The injected block lists corrections in reverse chronological order (most recent first). When the ledger exceeds 20 entries, the oldest entries beyond position 20 are summarized into a single count line (e.g., "...and 7 earlier corrections omitted") to avoid prompt bloat.

**Why this priority**: Without a cap, long sessions accumulate unbounded correction text that consumes context window budget and degrades agent performance. Without ordering, the agent may focus on stale corrections instead of recent ones.

**Independent Test**: Programmatically append 25 corrections to the ledger, render the block, and verify: exactly 20 individual entries appear (most recent first), plus a summary line for the 5 omitted entries. Total rendered text stays under 4,000 characters.

**Acceptance Scenarios**:

1. **Given** 5 corrections have been logged across 5 turns, **When** turn 6 starts, **Then** all 5 corrections appear in the block, most recent first.
2. **Given** 21 corrections have been logged, **When** the next turn starts, **Then** the block contains 20 individual entries plus a summary line "...and 1 earlier correction omitted".
3. **Given** 25 corrections have been logged, **When** the block is rendered, **Then** only 20 entries appear individually, and the summary reads "...and 5 earlier corrections omitted".
4. **Given** corrections are accumulating, **When** a new correction is logged, **Then** it does not displace existing entries -- it is prepended (most recent first) and the cap truncates from the tail.

---

### User Story 3 - Corrections Work Across Both Claude and Gemini Bridges (Priority: P1)

The correction ledger and injection logic live in the platform layer (`ambient_runner.platform`), not in the Claude-specific bridge. Both the Claude SDK bridge and the Gemini CLI bridge use the same ledger and rendering code. The Claude bridge injects corrections via the system prompt `append` field. The Gemini bridge injects corrections via a context file written to the workspace before each turn.

**Why this priority**: The platform supports multiple LLM backends. A Claude-only solution would leave Gemini sessions without correction memory, creating an inconsistent user experience and duplicating logic when eventually implemented for Gemini.

**Independent Test**: Run the correction ledger's `render()` method directly and confirm it produces identical output regardless of which bridge consumes it. Then verify each bridge's injection path: Claude via `build_sdk_system_prompt` appending the block, Gemini via a file write to `.gemini/corrections.md`.

**Acceptance Scenarios**:

1. **Given** a correction is logged through the Claude bridge's `log_correction` tool, **When** the next Claude turn starts, **Then** the correction appears in the system prompt's `append` field.
2. **Given** a correction is logged through the Gemini feedback server's `log_correction` handler, **When** the next Gemini turn starts, **Then** the correction appears in a workspace context file readable by the Gemini CLI.
3. **Given** the same 3 corrections are in the ledger, **When** `render()` is called, **Then** the output is identical regardless of which bridge will consume it.

---

### Edge Cases

- **Session restart (pod eviction)**: Correction ledger is lost. This is intentional -- corrections are session-local context, not persistent state. The agent starts fresh after restart.
- **Correction logged with empty fields**: `agent_action` or `user_correction` is empty string. The entry is still appended to the ledger but renders with a placeholder like "(not specified)" to avoid blank lines.
- **Concurrent turns on the same session**: The ledger is per-SessionWorker (Claude) or per-process (Gemini). The existing per-thread lock in `SessionManager` serializes turns, so concurrent mutation is not possible within a single session.
- **Langfuse disabled but corrections still logged**: The ledger operates independently of Langfuse. Even if `_log_correction_to_langfuse` fails, the correction is appended to the in-memory ledger for context injection.
- **Correction type not in CORRECTION_TYPES enum**: The ledger accepts any string for `correction_type` since it is display-only in the injected block. Validation remains in the tool schema.
- **Very long agent_action or user_correction text**: Truncate each field to 500 characters in the ledger entry (matching the existing Langfuse truncation in `_log_correction_to_langfuse`).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST maintain a per-session in-memory correction ledger as a list of correction entries, where each entry contains `correction_type`, `agent_action`, `user_correction`, and a timestamp.
- **FR-002**: When `log_correction` is called (on either bridge), the system MUST append the correction to the session's ledger in addition to logging to Langfuse.
- **FR-003**: Before each turn after the first correction, the system MUST inject a `## Corrections from this session` block into the agent's context containing all ledger entries.
- **FR-004**: The injected block MUST list corrections in reverse chronological order (most recent first).
- **FR-005**: The injected block MUST cap at 20 individual entries. When the ledger contains more than 20 entries, the block MUST include a summary count of omitted entries.
- **FR-006**: The correction ledger and rendering logic MUST reside in the platform layer (`ambient_runner.platform`) so both bridges can use it without cross-bridge imports.
- **FR-007**: The Claude bridge MUST inject corrections by appending the rendered block to the system prompt's `append` field in `build_sdk_system_prompt`.
- **FR-008**: The Gemini bridge MUST inject corrections by writing the rendered block to a context file in the workspace before each turn.
- **FR-009**: The ledger MUST NOT persist beyond the session lifetime. Pod restart or session termination clears it.
- **FR-010**: The ledger MUST truncate `agent_action` and `user_correction` fields to 500 characters per entry.
- **FR-011**: The system MUST still append to the ledger even if Langfuse logging fails, ensuring context injection is independent of observability availability.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After a correction is logged, the next turn's context contains a `## Corrections from this session` block with that correction's details.
- **SC-002**: A session with 25 corrections produces an injected block with exactly 20 individual entries and one summary line.
- **SC-003**: The rendered correction block for 20 entries stays under 4,000 characters (under 1% of a 200k context window).
- **SC-004**: Both Claude and Gemini bridges inject the same rendered correction text, verified by unit tests calling `render()` directly.
- **SC-005**: Existing `log_correction` behavior (Langfuse scoring, tool response) is unchanged -- ledger append is additive only.
- **SC-006**: Runner unit tests cover: empty ledger renders nothing, single correction renders correctly, 20+ corrections trigger cap and summary, field truncation at 500 chars.

## Assumptions

- The Claude Agent SDK's system prompt `append` field supports dynamic content that can change between turns (the prompt is rebuilt per-adapter construction, and `_ensure_adapter` is called per turn).
- The Gemini CLI reads context files from the workspace on each turn (consistent with existing `.gemini/settings.json` behavior).
- Session workers are single-threaded per session -- the existing `SessionManager` lock prevents concurrent turns on the same worker, so the ledger does not need its own synchronization.
- The `log_correction` tool on the Claude bridge (`create_correction_mcp_tool`) can be extended to accept a callback or ledger reference without changing its public MCP schema.

## Dependencies

- `ambient_runner.bridges.claude.corrections` -- existing `log_correction` tool and Langfuse logging (modified to also append to ledger).
- `ambient_runner.platform.feedback` -- existing platform-level `log_correction` wrapper (modified to also append to ledger).
- `ambient_runner.platform.prompts` -- existing prompt builder (modified to accept and render correction block).
- `ambient_runner.bridges.claude.bridge` -- `ClaudeBridge` (modified to pass ledger to prompt builder and correction tool).
- `ambient_runner.bridges.gemini_cli.feedback_server` -- Gemini MCP server (modified to append to a shared ledger and write context file).
