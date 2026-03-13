# Proposal: Public API Session Control Endpoints

**Branch:** `feat/public-api-session-control`
**Date:** 2026-03-06
**Status:** Draft
**Spec:** `components/public-api/openapi.yaml`

---

## Summary

Extend the public API from 4 session CRUD endpoints to a full session control surface. These endpoints let SDK/MCP clients manage the complete session lifecycle — send messages, retrieve output, start/stop sessions, and interrupt runs — without needing to understand the AG-UI protocol or construct complex request envelopes.

All endpoints proxy to the existing Go backend. No new K8s operations are introduced in the public API.

---

## Current State (Phase 1)

The public API currently exposes only basic CRUD:

| Method | Endpoint | Status |
|--------|----------|--------|
| GET | `/v1/sessions` | Implemented |
| POST | `/v1/sessions` | Implemented |
| GET | `/v1/sessions/{id}` | Implemented |
| DELETE | `/v1/sessions/{id}` | Implemented |

---

## Proposed Endpoints

### Messaging

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/sessions/{id}/runs` | **Raw AG-UI run.** Accepts full `RunAgentInput` with caller-provided run/thread IDs and messages array. Direct proxy to backend `/agui/run`. |
| POST | `/v1/sessions/{id}/message` | **Simplified send.** Accepts `{"content": "..."}`, constructs the AG-UI `RunAgentInput` envelope server-side (generates run ID, thread ID, message ID). Proxies to backend `/agui/run`. |

### Output Retrieval

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/v1/sessions/{id}/output` | Returns session output in one of three formats via `?format=` query param. |
| GET | `/v1/sessions/{id}/runs` | Lists all runs with status, timestamps, event counts, and originating user message. |

**Output formats:**

- **`transcript`** (default) — Assembled conversation messages from `MESSAGES_SNAPSHOT` events. Smallest, most useful for consumers.
- **`compact`** — AG-UI events with streaming deltas merged per message/tool call. Preserves event structure but significantly smaller than raw.
- **`events`** — Raw AG-UI events exactly as persisted. Can be very large.

All formats support optional `?run_id=` filtering.

### Lifecycle Control

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/sessions/{id}/start` | Resume a stopped/completed session. Returns 422 if already running. |
| POST | `/v1/sessions/{id}/stop` | Stop a running session (kills pod). Returns 422 if already stopped. |
| POST | `/v1/sessions/{id}/interrupt` | Cancel the current run without killing the session. Equivalent to the red stop button in the UI. |

---

## What This Enables

Today, SDK/MCP clients can create sessions via the public API but must either:
- Hit the backend API directly for messaging and lifecycle control
- Construct AG-UI protocol envelopes manually

With these additions, the public API becomes a complete session management interface. The `/message` endpoint is the key simplification — clients send plain text and the public API handles all AG-UI plumbing.

---

## Open Question

The backend API is externally routed on all current deployments (lab OKD, ROSA UAT). All proposed endpoints have direct backend equivalents (except `/message` envelope construction, transcript assembly, and run listing). If the backend remains externally accessible, clients can use it directly — making these public API additions a convenience layer rather than a necessity.

The value depends on whether Phase 3 (backend internalization) from the [original proposal](acp-public-rest-api.md) proceeds. If the backend route is removed, these endpoints become required for external clients.
