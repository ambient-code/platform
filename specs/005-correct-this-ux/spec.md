# Feature Specification: "Correct This" UX

**Feature Branch**: `005-correct-this-ux`
**Created**: 2026-04-15
**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Submit a Structured Correction via the Chat UI (Priority: P1)

A user reviewing an agent's response notices the agent used the wrong approach. Instead of hoping the agent detects the correction from a follow-up message, the user clicks "Correct this" on the specific agent message. A lightweight popover appears with a correction type dropdown (`incomplete`, `incorrect`, `out_of_scope`, `style`), a free-text field for "What should have happened instead?", and an optional checkbox to include the original message content as context. The user fills in the fields and submits. The correction is sent to the backend via `POST /api/projects/:projectName/corrections` and the UI confirms success.

**Why this priority**: This is the core interaction -- without it the feature does not exist. Users currently have no direct mechanism to submit structured corrections; they can only thumbs-down (which captures sentiment, not teaching signal) or hope the agent calls `log_correction` on its own.

**Independent Test**: Open any active or completed session, find an agent message, click "Correct this", fill in the form, submit, and verify a 200 response from the corrections endpoint.

**Acceptance Scenarios**:

1. **Given** an agent message in the chat view, **When** the user clicks the "Correct this" action, **Then** a popover/modal opens with: a correction type dropdown (incomplete, incorrect, out_of_scope, style), a free-text field labeled "What should have happened instead?", and a checkbox labeled "Include message content as context".
2. **Given** the correction popover is open, **When** the user selects a correction type, enters text, and clicks Submit, **Then** a POST request is sent to `/api/projects/:projectName/corrections` with `{ correction_type, user_correction, message_id, message_content? }` and the popover closes with a success indicator.
3. **Given** the correction popover is open, **When** the user clicks Cancel or presses Escape, **Then** the popover closes with no request sent.
4. **Given** the user has already submitted a correction for a message, **When** viewing that message, **Then** the "Correct this" button shows a visual indicator (e.g., a check mark or muted state) that a correction was submitted, but the user can still submit additional corrections.

---

### User Story 2 - Correction Reaches the Active Session's Correction Ledger (Priority: P1)

A user submits a correction on a message in an active (running) session. The backend receives the correction and forwards it to the runner's session via the existing AG-UI META event channel. The runner appends the correction to its session-scoped correction ledger (spec 004) so the agent can reference it for the remainder of the session.

**Why this priority**: The teaching value of corrections is highest when the agent can apply them immediately within the same session. Without forwarding to the active session, corrections are only useful in the offline feedback loop -- a much slower cycle.

**Independent Test**: Start a session, submit a correction via the UI while the session is running, then verify the runner's correction ledger contains the new entry (visible via runner logs or the corrections MCP tool output).

**Acceptance Scenarios**:

1. **Given** a session is in `Running` state, **When** a correction is submitted via the UI, **Then** the backend forwards a META event with `metaType: "user_correction"` to the runner's AG-UI feedback endpoint.
2. **Given** the runner receives a `user_correction` META event, **When** it processes the event, **Then** the correction is appended to the session-scoped correction ledger with the same schema as `log_correction` entries (`correction_type`, `agent_action` derived from the message content, `user_correction`).
3. **Given** a session is in `Completed` or `Failed` state, **When** a correction is submitted via the UI, **Then** the backend persists the correction to Langfuse but does NOT attempt to forward to the runner (no error is returned to the user).

---

### User Story 3 - Correction Is Visible in the Corrections Summary (Priority: P2)

After submitting corrections, a user or platform operator wants to see what corrections have been logged for a session. The corrections appear in the session's correction summary (surfaced by the Corrections Realtime Pipeline, spec 003) alongside agent-detected corrections from `log_correction`. User-submitted corrections are distinguished by `source: "user"` (vs. `source: "human"` for agent-detected corrections from the user's spoken words, or `source: "rubric"` for rubric-generated corrections).

**Why this priority**: Visibility is necessary for trust and debugging, but the system delivers value even without a dedicated corrections view -- corrections flow into Langfuse and the feedback loop regardless.

**Independent Test**: Submit multiple corrections on different messages in a session, then query the corrections endpoint or Langfuse to verify all are present with `source: "user"` and correct metadata.

**Acceptance Scenarios**:

1. **Given** a user has submitted corrections via the UI, **When** the corrections summary for that session is retrieved, **Then** user-submitted corrections appear with `source: "user"` and include the `message_id` linking them to the specific agent message.
2. **Given** a session has both agent-detected and user-submitted corrections, **When** viewing the corrections summary, **Then** the two sources are visually distinguishable (e.g., badge or icon indicating "User" vs. "Agent-detected").
3. **Given** no corrections have been submitted for a session, **When** viewing the corrections summary, **Then** an empty state is shown (not an error).

---

### Edge Cases

- **Session is completed/failed**: Correction is accepted and persisted to Langfuse. No attempt is made to forward to the runner. The UI does not show an error -- corrections on completed sessions still feed the offline feedback loop.
- **Session runner is unreachable**: Correction is persisted to Langfuse (source of truth). A warning toast is shown: "Correction saved. Could not forward to active session." The correction is not lost.
- **Network failure on submit**: The UI shows an inline error in the popover: "Failed to submit correction. Please try again." The popover remains open so the user does not lose their input.
- **Correction type not selected**: Submit button is disabled until a correction type is selected. The free-text field is required (minimum 10 characters).
- **Very long correction text**: Frontend truncates at 2000 characters with a visible character counter. Backend enforces the same limit and returns 400 if exceeded.
- **Multiple corrections on same message**: Allowed. Each correction is a separate entry. The "Correct this" button shows a count badge after the first correction (e.g., a small dot indicator).
- **Rapid duplicate submissions**: Frontend disables the submit button during the request and for 2 seconds after success to prevent accidental double-submits.
- **User lacks project access**: The existing `ValidateProjectContext()` middleware rejects the request with 403. No special handling needed.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST display a "Correct this" action on all agent text messages in the chat view, visually distinct from the existing thumbs-up/thumbs-down feedback buttons.
- **FR-002**: The "Correct this" action MUST open a popover or lightweight modal containing: a correction type dropdown with options `incomplete`, `incorrect`, `out_of_scope`, `style`; a required free-text field ("What should have happened instead?", minimum 10 characters, maximum 2000 characters); and an optional checkbox to include the original message content as context.
- **FR-003**: On submit, the frontend MUST POST to `POST /api/projects/:projectName/corrections` with a payload containing `correction_type`, `user_correction`, `message_id`, `session_name`, and optionally `message_content` (when the checkbox is checked).
- **FR-004**: The backend corrections endpoint MUST persist the correction to Langfuse as a categorical score with `source: "user"`, using the same schema as the runner's `log_correction` tool.
- **FR-005**: When the target session is in `Running` state, the backend MUST forward the correction to the runner via a META event with `metaType: "user_correction"` on the AG-UI feedback channel.
- **FR-006**: When the target session is NOT in `Running` state, the backend MUST persist the correction to Langfuse without attempting to forward to the runner, and MUST NOT return an error to the user.
- **FR-007**: The "Correct this" button MUST be visually distinct from the thumbs-down button. Thumbs-down means "bad response" (sentiment). "Correct this" means "teach the agent" (structured teaching signal). The button SHOULD use a differentiated icon (e.g., `Pencil`, `PencilLine`, or `MessageSquarePlus` from Lucide) and a distinct color (e.g., amber/yellow) to reinforce this distinction.
- **FR-008**: The frontend MUST show a success indicator on the message after a correction is submitted (e.g., a small dot or check on the "Correct this" button).
- **FR-009**: The frontend MUST handle network errors by displaying an inline error in the popover without closing it or losing user input.
- **FR-010**: The backend corrections endpoint MUST be accessible via the Next.js API proxy route pattern (`src/app/api/projects/[name]/corrections/route.ts`) consistent with existing frontend-to-backend routing.

### Non-Functional Requirements

- **NFR-001**: The popover MUST open in under 100ms (no lazy-loaded dependencies).
- **NFR-002**: The correction submission round-trip (frontend to backend to Langfuse) MUST complete in under 2 seconds under normal conditions.
- **NFR-003**: The "Correct this" button MUST NOT interfere with existing feedback button layout or accessibility (keyboard navigation, screen reader labels).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can submit a structured correction on any agent message in under 15 seconds (open popover, select type, type correction, submit).
- **SC-002**: 100% of user-submitted corrections are persisted to Langfuse with `source: "user"` and linked to the originating `message_id` and `session_name`.
- **SC-003**: For active sessions, corrections forwarded to the runner appear in the session-scoped correction ledger within 5 seconds of submission.
- **SC-004**: The "Correct this" action is visually distinguishable from thumbs-up/thumbs-down at a glance -- confirmed by at least one design review.
- **SC-005**: No regressions in existing feedback button functionality (thumbs up/down continue to work unchanged).

## Assumptions

- The Corrections Realtime Pipeline backend endpoint (`POST /api/projects/:projectName/corrections`) exists or will be created as part of spec 003. This spec defines the frontend UX and the contract; spec 003 defines the backend implementation.
- The session-scoped correction ledger (spec 004) exists in the runner and can accept corrections appended via META events.
- The existing AG-UI feedback channel (`/agui/feedback`) can transport arbitrary META event types including `user_correction` without protocol changes.
- Langfuse is deployed and accessible from the backend (existing infrastructure).
- The `correction_type` enum (`incomplete`, `incorrect`, `out_of_scope`, `style`) is stable and matches the runner's `CORRECTION_TYPES` constant in `corrections.py`.
- Shadcn/UI `Popover`, `Select`, `Textarea`, and `Checkbox` components are available in the frontend component library.

## Dependencies

- **Spec 003 - Corrections Realtime Pipeline**: Provides the backend `POST /api/projects/:projectName/corrections` endpoint that this feature posts to.
- **Spec 004 - Session-Scoped Correction Injection**: Provides the runner-side correction ledger that receives forwarded corrections from active sessions.
- **Existing infrastructure**: Langfuse (observability), AG-UI protocol (event transport), Shadcn/UI (frontend components), `FeedbackButtons` component (co-located rendering).
- **Key source files**:
  - `components/frontend/src/components/feedback/FeedbackButtons.tsx` -- extend with "Correct this" action
  - `components/frontend/src/components/feedback/FeedbackModal.tsx` -- reference pattern for the correction popover
  - `components/frontend/src/components/ui/stream-message.tsx` -- renders feedback buttons on agent messages
  - `components/frontend/src/contexts/FeedbackContext.tsx` -- session context for correction metadata
  - `components/backend/routes.go` -- register corrections endpoint
  - `components/runners/ambient-runner/ambient_runner/bridges/claude/corrections.py` -- correction type enum and Langfuse schema (source of truth for field names)
