# Feature Specification: Corrections Realtime Pipeline

**Feature Branch**: `003-corrections-realtime-pipeline`
**Created**: 2026-04-15
**Status**: Draft

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Runner Posts Corrections to Backend (Priority: P1)

During an agentic session, the runner calls `log_correction` via the MCP tool. After logging the correction to Langfuse (existing behavior), it also POSTs the correction event to the backend's new `/api/projects/:projectName/corrections` endpoint. The backend accepts and buffers the event for downstream consumers (dashboards, threshold alerts, the planned "Correct This" UX).

**Why this priority**: This is the primary data path. Without runner-to-backend correction flow, no real-time consumer can exist.

**Independent Test**: Start a session, trigger a correction via the `log_correction` MCP tool, then query `GET /api/projects/:projectName/corrections` and confirm the event appears.

**Acceptance Scenarios**:

1. **Given** a running session in project `team-alpha`, **When** the runner logs a correction with type `incorrect` and target `my-workflow`, **Then** a POST to `/api/projects/team-alpha/corrections` succeeds with HTTP 201 and the event is retrievable via GET.
2. **Given** a correction event with all required fields (`sessionName`, `correctionType`, `agentAction`, `userCorrection`, `target`, `source`, `timestamp`), **When** POSTed to the endpoint, **Then** the backend validates the schema and stores it in the in-memory buffer.
3. **Given** a correction event missing `correctionType`, **When** POSTed, **Then** the backend returns HTTP 400 with a descriptive validation error.
4. **Given** more than 24 hours have passed since a correction was buffered, **When** the corrections list is queried, **Then** expired events are not returned.

---

### User Story 2 - Frontend Posts Corrections Directly (Priority: P1)

A user reviewing session output in the frontend clicks a "Correct This" action (future UX feature that depends on this pipeline). The frontend POSTs a correction event directly to the backend endpoint with `source: "ui"`. This path does not go through the runner or Langfuse.

**Why this priority**: The frontend is the second ingestion source. Both ingestion paths must work before any consumer feature can be built.

**Independent Test**: Use `curl` or the frontend to POST a correction event with `source: "ui"` to the endpoint. Confirm it appears in the GET response alongside runner-sourced corrections.

**Acceptance Scenarios**:

1. **Given** a user authenticated via their K8s token, **When** they POST a correction with `source: "ui"` for a session in their project, **Then** the backend accepts it with HTTP 201.
2. **Given** a user without access to the target project, **When** they POST a correction, **Then** the backend returns HTTP 403 (enforced by `ValidateProjectContext` middleware).
3. **Given** corrections from both the runner (`source: "human"`) and the frontend (`source: "ui"`), **When** querying `GET /corrections`, **Then** both sources appear in the response and are distinguishable by the `source` field.

---

### User Story 3 - Backend Aggregation and Summary Endpoint (Priority: P1)

A downstream consumer (dashboard, threshold-based automation, future "Correct This" UX) queries `GET /api/projects/:projectName/corrections/summary` to get aggregated correction counts grouped by `target`. This enables threshold detection (e.g., "workflow X has received 5 corrections in the last hour -- trigger an improvement session").

**Why this priority**: Aggregation is the primary value-add over raw Langfuse queries. Without it, consumers must fetch and group raw events themselves.

**Independent Test**: POST 3 corrections targeting `workflow-a` and 2 targeting `repo-b`, then query `/corrections/summary` and confirm counts `{"workflow-a": 3, "repo-b": 2}`.

**Acceptance Scenarios**:

1. **Given** 5 corrections for target `review-workflow` and 2 for target `platform`, **When** querying `/corrections/summary`, **Then** the response contains `{"review-workflow": 5, "platform": 2}` (or equivalent structured format).
2. **Given** a `target` query parameter set to `review-workflow`, **When** querying `/corrections/summary?target=review-workflow`, **Then** only counts for that target are returned.
3. **Given** no corrections in the last 24 hours, **When** querying `/corrections/summary`, **Then** the response is an empty object, not an error.
4. **Given** a `session` query parameter, **When** querying `/corrections?session=my-session`, **Then** only corrections for that session are returned.

---

### User Story 4 - Dual-Write Does Not Block Sessions (Priority: P1)

The runner's dual-write (Langfuse + backend POST) must not degrade session performance. The backend POST is fire-and-forget: if it fails (network error, backend down, timeout), the Langfuse write still succeeds and the session continues without interruption. The runner logs a warning but does not surface the failure to the agent or user.

**Why this priority**: Session reliability is non-negotiable. A new telemetry path must never cause session failures.

**Independent Test**: Start a session with the backend endpoint unreachable (e.g., wrong URL or backend scaled to zero). Trigger a correction. Confirm the Langfuse write succeeds and the session continues normally. Check runner logs for a warning about the failed backend POST.

**Acceptance Scenarios**:

1. **Given** the backend corrections endpoint is unreachable, **When** the runner logs a correction, **Then** the Langfuse write succeeds and the MCP tool returns a success response to the agent.
2. **Given** the backend corrections endpoint returns HTTP 500, **When** the runner logs a correction, **Then** the runner logs a warning and does not retry or raise an exception.
3. **Given** the backend corrections endpoint is slow (>2s), **When** the runner logs a correction, **Then** the POST times out without blocking the agent turn. The timeout must be no more than 3 seconds.
4. **Given** Langfuse is unreachable but the backend is available, **When** the runner logs a correction, **Then** the backend POST still fires (the two writes are independent).

---

### Edge Cases

- **Buffer overflow**: If a project accumulates thousands of corrections within 24 hours, the in-memory buffer must cap at a configurable maximum (e.g., 10,000 events per project) and evict oldest entries first (FIFO).
- **Concurrent writes**: Multiple sessions in the same project POST corrections simultaneously. The buffer must be goroutine-safe (mutex or channel-based).
- **Backend restart**: In-memory buffer is lost on restart. This is acceptable for v1 -- corrections are ephemeral signals, not durable state. Langfuse remains the system of record.
- **Invalid correctionType**: The backend rejects values not in the allowed enum (`incomplete`, `incorrect`, `out_of_scope`, `style`) with HTTP 400.
- **Empty target**: A correction with an empty `target` field is accepted. The summary endpoint groups these under a `(none)` or empty-string key.
- **Clock skew**: The `timestamp` field is optional. If omitted, the backend uses server-side `time.Now()` at receipt. If provided, it is stored as-is but the 24-hour expiry window is based on receipt time, not the provided timestamp.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST expose `POST /api/projects/:projectName/corrections` accepting a JSON body with fields: `sessionName` (string, required), `correctionType` (enum: `incomplete|incorrect|out_of_scope|style`, required), `agentAction` (string, required), `userCorrection` (string, required), `target` (string, optional), `source` (enum: `human|rubric|ui`, required), `timestamp` (ISO 8601 string, optional).
- **FR-002**: System MUST expose `GET /api/projects/:projectName/corrections` returning corrections from the last 24 hours, with optional query parameters `target` and `session` for filtering.
- **FR-003**: System MUST expose `GET /api/projects/:projectName/corrections/summary` returning correction counts grouped by `target`, with optional `target` query parameter for filtering.
- **FR-004**: All three endpoints MUST be registered under `projectGroup` in `routes.go` and protected by `ValidateProjectContext()` middleware, using `GetK8sClientsForRequest` for user token auth.
- **FR-005**: The runner MUST POST to the backend corrections endpoint after each successful `_log_correction_to_langfuse` call, using a fire-and-forget HTTP request with a maximum timeout of 3 seconds.
- **FR-006**: Failure of the backend POST in the runner MUST NOT affect the return value of the `log_correction` MCP tool or block the agent session.
- **FR-007**: The in-memory correction buffer MUST be goroutine-safe and enforce a per-project maximum of 10,000 events with FIFO eviction.
- **FR-008**: Events older than 24 hours MUST be excluded from GET responses. The backend MAY lazily evict expired events during reads or run a periodic cleanup goroutine.
- **FR-009**: The `source` field MUST accept `"ui"` in addition to the existing `"human"` and `"rubric"` values to distinguish frontend-originated corrections.

### Non-Functional Requirements

- **NFR-001**: The `POST /corrections` endpoint MUST respond within 50ms under normal load (in-memory write, no disk or network I/O).
- **NFR-002**: The in-memory buffer MUST NOT exceed 100MB of memory per project under maximum load (10,000 events with 1KB average payload = ~10MB, well within budget).
- **NFR-003**: The runner's fire-and-forget POST MUST use `asyncio.create_task` (or equivalent) so it does not block the calling coroutine.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Runner-originated corrections appear in `GET /corrections` within 1 second of the MCP tool call completing.
- **SC-002**: Frontend-originated corrections (POST with `source: "ui"`) appear in `GET /corrections` within 1 second.
- **SC-003**: `GET /corrections/summary` returns accurate counts matching the number of POSTed corrections, grouped by target.
- **SC-004**: A correction logged while the backend is unreachable still succeeds from the agent's perspective (Langfuse write completes, MCP tool returns success).
- **SC-005**: Backend restart clears the buffer without errors. GET endpoints return empty results until new corrections arrive.
- **SC-006**: All three backend endpoints pass authorization: unauthenticated requests get 401, requests to foreign projects get 403.

## Assumptions

- The in-memory buffer is sufficient for v1. If durable correction storage is needed later, it can be backed by a CRD or external store without API changes.
- The runner has network access to the backend API (same cluster, service DNS resolution via `BACKEND_URL` or equivalent env var).
- The existing `CORRECTION_TYPES` and `CORRECTION_SOURCES` constants in `corrections.py` are the source of truth for allowed enum values. The backend's validation mirrors them.
- The `source: "ui"` value is new and will be added to `CORRECTION_SOURCES` in the runner for parity, even though the runner itself never sends `source: "ui"`.
- SSE/polling for real-time streaming is deferred to a follow-up. The GET endpoints support polling at reasonable intervals (5-10s) for v1.

## Dependencies

- Backend handler patterns: `components/backend/handlers/` (Gin handler conventions, `GetK8sClientsForRequest`)
- Route registration: `components/backend/routes.go` (`projectGroup`)
- Runner correction logging: `components/runners/ambient-runner/ambient_runner/bridges/claude/corrections.py` (`_log_correction_to_langfuse`)
- Runner platform feedback API: `components/runners/ambient-runner/ambient_runner/platform/feedback.py` (`log_correction`)
- Runner HTTP client: runner must have an HTTP client available for POSTing to the backend (likely `aiohttp` or `httpx`, already in runner dependencies)
- Backend URL discovery: runner needs the backend service URL, either from `BACKEND_URL` env var or K8s service DNS
