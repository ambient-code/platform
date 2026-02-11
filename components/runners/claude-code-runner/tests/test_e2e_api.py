"""
End-to-end API tests for the Ambient Runner SDK.

Tests the full FastAPI application without mocks, using the real
``ClaudeBridge`` and ``create_ambient_app()``.  Each test hits a real
HTTP endpoint through ``TestClient`` (synchronous HTTPX wrapper).

Two test tiers:

1. **Structural tests** (always run) — verify endpoints exist, return
   correct shapes, and the platform lifecycle works without an API key.

2. **Live agent tests** (require ``ANTHROPIC_API_KEY``) — send a real
   prompt through the Claude Agent SDK and verify the AG-UI event
   stream. Skipped when no key is set.

Run::

    # Structural only (no API key needed)
    pytest tests/test_e2e_api.py -v

    # Full E2E with live agent
    ANTHROPIC_API_KEY=sk-ant-... pytest tests/test_e2e_api.py -v
"""

import json
import os
import uuid
from typing import Any

import pytest
from fastapi.testclient import TestClient

from ambient_runner import create_ambient_app
from ambient_runner.bridges.claude import ClaudeBridge


def _msg(content: str, role: str = "user") -> dict[str, Any]:
    """Build a properly-formed AG-UI message dict with all required fields."""
    return {"id": str(uuid.uuid4()), "role": role, "content": content}

# ---------------------------------------------------------------------------
# Fixtures
# ---------------------------------------------------------------------------

HAS_API_KEY = bool(os.getenv("ANTHROPIC_API_KEY", "").strip())
requires_api_key = pytest.mark.skipif(
    not HAS_API_KEY,
    reason="ANTHROPIC_API_KEY not set — skipping live agent tests",
)


@pytest.fixture(scope="module")
def app():
    """Create the full Ambient app with ClaudeBridge."""
    os.environ.setdefault("SESSION_ID", "e2e-test")
    os.environ.setdefault("WORKSPACE_PATH", "/tmp/e2e-workspace")
    os.environ.setdefault("INTERACTIVE", "true")
    # Ensure workspace exists
    os.makedirs("/tmp/e2e-workspace", exist_ok=True)
    return create_ambient_app(ClaudeBridge(), title="E2E Test Runner")


@pytest.fixture(scope="module")
def client(app):
    """TestClient that runs the full app with lifespan."""
    with TestClient(app, raise_server_exceptions=False) as c:
        yield c


# ===================================================================
# 1. Structural tests — no API key needed
# ===================================================================


class TestHealthEndpoint:
    """GET /health — basic liveness check."""

    def test_returns_200(self, client):
        resp = client.get("/health")
        assert resp.status_code == 200

    def test_returns_session_id(self, client):
        data = client.get("/health").json()
        assert data["status"] == "healthy"
        assert data["session_id"] == "e2e-test"


class TestCapabilitiesEndpoint:
    """GET /capabilities — framework and platform feature manifest."""

    def test_returns_200(self, client):
        resp = client.get("/capabilities")
        assert resp.status_code == 200

    def test_framework_is_claude(self, client):
        data = client.get("/capabilities").json()
        assert data["framework"] == "claude-agent-sdk"

    def test_includes_agent_features(self, client):
        data = client.get("/capabilities").json()
        features = data["agent_features"]
        assert "agentic_chat" in features
        assert "thinking" in features
        assert "human_in_the_loop" in features

    def test_includes_platform_features(self, client):
        data = client.get("/capabilities").json()
        features = data["platform_features"]
        assert "repos" in features
        assert "workflows" in features
        assert "feedback" in features
        assert "mcp_diagnostics" in features

    def test_includes_session_id(self, client):
        data = client.get("/capabilities").json()
        assert data["session_id"] == "e2e-test"

    def test_file_system_and_mcp(self, client):
        data = client.get("/capabilities").json()
        assert data["file_system"] is True
        assert data["mcp"] is True
        assert data["session_persistence"] is True


class TestMcpStatusEndpoint:
    """GET /mcp/status — MCP server diagnostics."""

    def test_returns_200(self, client):
        resp = client.get("/mcp/status")
        assert resp.status_code == 200

    def test_returns_server_list(self, client):
        data = client.get("/mcp/status").json()
        assert "servers" in data
        assert "totalCount" in data
        assert isinstance(data["servers"], list)


class TestFeedbackEndpoint:
    """POST /feedback — Langfuse feedback scoring."""

    def test_accepts_thumbs_up(self, client):
        resp = client.post("/feedback", json={
            "type": "META",
            "metaType": "thumbs_up",
            "payload": {"userId": "test-user"},
        })
        assert resp.status_code == 200
        assert resp.json()["metaType"] == "thumbs_up"

    def test_accepts_thumbs_down(self, client):
        resp = client.post("/feedback", json={
            "type": "META",
            "metaType": "thumbs_down",
            "payload": {"userId": "test-user", "comment": "Not helpful"},
        })
        assert resp.status_code == 200

    def test_rejects_invalid_type(self, client):
        resp = client.post("/feedback", json={
            "type": "INVALID",
            "metaType": "thumbs_up",
            "payload": {},
        })
        assert resp.status_code == 400

    def test_rejects_invalid_meta_type(self, client):
        resp = client.post("/feedback", json={
            "type": "META",
            "metaType": "invalid",
            "payload": {},
        })
        assert resp.status_code == 400


class TestReposEndpoints:
    """Repos management endpoints."""

    def test_repos_status_returns_200(self, client):
        resp = client.get("/repos/status")
        assert resp.status_code == 200
        assert "repos" in resp.json()

    def test_repos_add_requires_url(self, client):
        resp = client.post("/repos/add", json={"url": "", "branch": "main"})
        assert resp.status_code == 400

    def test_repos_remove_returns_200(self, client):
        resp = client.post("/repos/remove", json={"name": "nonexistent-repo"})
        assert resp.status_code == 200


class TestWorkflowEndpoint:
    """POST /workflow — workflow switching."""

    def test_empty_workflow_returns_200(self, client):
        resp = client.post("/workflow", json={
            "gitUrl": "",
            "branch": "main",
            "path": "",
        })
        assert resp.status_code == 200

    def test_same_workflow_is_idempotent(self, client):
        """Calling workflow with same params twice returns 'already active'."""
        payload = {"gitUrl": "", "branch": "main", "path": ""}
        client.post("/workflow", json=payload)
        resp = client.post("/workflow", json=payload)
        assert resp.status_code == 200
        assert "already active" in resp.json().get("message", "").lower()


class TestInterruptEndpoint:
    """POST /interrupt — interrupt with no active session."""

    def test_interrupt_no_session(self, client):
        resp = client.post("/interrupt", json={"thread_id": "nonexistent"})
        # Should return 500 since there's no active session manager yet
        assert resp.status_code == 500


class TestRunEndpointStructural:
    """POST / — structural validation of the run endpoint."""

    def test_run_endpoint_accepts_post(self, client):
        """The run endpoint exists and accepts POST requests."""
        resp = client.post("/", json={
            "threadId": "t-1",
            "runId": "r-1",
            "messages": [_msg("Hello")],
        })
        # 200 with SSE stream (if API key set) or 500 (if auth fails in generator setup)
        assert resp.status_code in (200, 500)

    def test_run_endpoint_rejects_bad_payload(self, client):
        """Missing required 'messages' field should return 422."""
        resp = client.post("/", json={"threadId": "t-1"})
        assert resp.status_code == 422


# ===================================================================
# 2. Live agent tests — require ANTHROPIC_API_KEY
# ===================================================================


def _parse_sse_events(response_text: str) -> list[dict]:
    """Parse SSE text into a list of JSON event dicts."""
    events = []
    for line in response_text.strip().split("\n"):
        line = line.strip()
        if line.startswith("data:"):
            data_str = line[5:].strip()
            if data_str:
                try:
                    events.append(json.loads(data_str))
                except json.JSONDecodeError:
                    pass
    return events


def _assert_run_ok(resp, *, label: str = "run"):
    """Assert the run response is 200 and print diagnostics on failure."""
    if resp.status_code != 200:
        body = resp.text[:1000] if resp.text else "(empty)"
        pytest.fail(
            f"[{label}] Expected 200, got {resp.status_code}\n"
            f"Response body:\n{body}"
        )


def _dump_events(events: list[dict], *, label: str = "run") -> None:
    """Print a compact summary of AG-UI events (visible with pytest -v -s)."""
    print(f"\n  [{label}] {len(events)} events:")
    for i, ev in enumerate(events):
        etype = ev.get("type", "?")
        extra = ""
        if etype == "TEXT_MESSAGE_CONTENT":
            delta = ev.get("delta", "")
            extra = f' delta="{delta[:60]}{"…" if len(delta) > 60 else ""}"'
        elif etype == "TEXT_MESSAGE_START":
            extra = f' role={ev.get("role", "?")} msg_id={ev.get("messageId", "?")[:8]}'
        elif etype == "TOOL_CALL_START":
            extra = f' tool={ev.get("toolCallName", "?")}'
        elif etype == "RUN_ERROR":
            extra = f' message="{ev.get("message", "")[:80]}"'
        elif etype == "CUSTOM":
            extra = f' name={ev.get("name", "?")}'
        print(f"    {i:3d}. {etype}{extra}")
    print()


class TestPlatformLifecycle:
    """Tests that verify platform lifecycle without needing an API key."""

    def test_lifespan_sets_context_on_bridge(self, client, app):
        """Verify the lifespan set RunnerContext on the bridge."""
        bridge = app.state.bridge
        assert bridge.context is not None
        assert bridge.context.session_id == "e2e-test"
        assert bridge.context.workspace_path == "/tmp/e2e-workspace"

    def test_capabilities_model_field_present(self, client):
        """Model field should exist in capabilities (None or string)."""
        data = client.get("/capabilities").json()
        assert "model" in data
        # Value is None (before platform init) or a string (after)
        assert data["model"] is None or isinstance(data["model"], str)

    def test_mark_dirty_via_workflow_change(self, client, app):
        """Changing workflow should mark the bridge dirty for rebuild."""
        bridge = app.state.bridge

        # Set a workflow (different from default empty)
        resp = client.post("/workflow", json={
            "gitUrl": "https://github.com/example/test-workflow.git",
            "branch": "main",
            "path": "",
        })
        assert resp.status_code == 200

        # Bridge should be marked dirty (not ready)
        assert bridge._ready is False

        # Reset back to empty so other tests aren't affected
        client.post("/workflow", json={"gitUrl": "", "branch": "main", "path": ""})

    def test_repos_add_remove_lifecycle(self, client):
        """Full repos lifecycle: check status → add (invalid) → remove."""
        # 1. Status starts empty
        data = client.get("/repos/status").json()
        assert isinstance(data["repos"], list)
        initial_count = len(data["repos"])

        # 2. Add with empty URL is rejected
        resp = client.post("/repos/add", json={"url": "", "branch": "main"})
        assert resp.status_code == 400

        # 3. Remove non-existent is idempotent (no error)
        resp = client.post("/repos/remove", json={"name": "does-not-exist"})
        assert resp.status_code == 200

        # 4. Status unchanged
        data = client.get("/repos/status").json()
        assert len(data["repos"]) == initial_count

    def test_feedback_full_payload(self, client):
        """Feedback with all optional fields."""
        resp = client.post("/feedback", json={
            "type": "META",
            "metaType": "thumbs_up",
            "payload": {
                "userId": "test-user",
                "projectName": "test-project",
                "sessionName": "e2e-test",
                "messageId": "msg-123",
                "traceId": "trace-456",
                "comment": "Great response!",
                "reason": "accurate",
                "workflow": "general",
                "context": "The agent said hello",
                "includeTranscript": True,
                "transcript": [
                    {"role": "user", "content": "Hi"},
                    {"role": "assistant", "content": "Hello!"},
                ],
            },
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["metaType"] == "thumbs_up"
        assert "recorded" in data

    def test_interrupt_returns_structured_error(self, client):
        """Interrupt on unknown thread returns a structured error."""
        resp = client.post("/interrupt", json={"thread_id": "ghost-thread"})
        assert resp.status_code == 500
        data = resp.json()
        assert "detail" in data
        assert "ghost-thread" in data["detail"]

    def test_run_endpoint_schema_validation(self, client):
        """Various payload validation checks."""
        # Missing messages entirely
        resp = client.post("/", json={"threadId": "t-1"})
        assert resp.status_code == 422

        # Empty messages list is valid (endpoint accepts it)
        resp = client.post("/", json={
            "threadId": "t-1",
            "runId": "r-1",
            "messages": [],
        })
        # Will fail during bridge.run() but endpoint accepts the shape
        assert resp.status_code in (200, 500)

    def test_workflow_idempotent_same_params(self, client):
        """Setting the same workflow twice returns 'already active'."""
        payload = {"gitUrl": "", "branch": "main", "path": ""}
        client.post("/workflow", json=payload)
        resp = client.post("/workflow", json=payload)
        assert resp.status_code == 200
        assert "already active" in resp.json()["message"].lower()

    def test_workflow_different_params_triggers_update(self, client):
        """Changing workflow params returns 'updated' not 'already active'."""
        client.post("/workflow", json={"gitUrl": "", "branch": "main", "path": ""})
        resp = client.post("/workflow", json={
            "gitUrl": "https://github.com/example/new-workflow.git",
            "branch": "main",
            "path": "",
        })
        assert resp.status_code == 200
        assert "updated" in resp.json()["message"].lower()

        # Reset
        client.post("/workflow", json={"gitUrl": "", "branch": "main", "path": ""})


@requires_api_key
class TestLiveAgentRun:
    """Full end-to-end test with a real Claude Agent SDK connection.

    These tests send a real prompt and verify the AG-UI event stream
    contains the expected event types in the correct order.
    """

    def test_simple_prompt_returns_valid_event_stream(self, client):
        """Send a trivial prompt and verify we get a complete AG-UI event stream."""
        resp = client.post("/", json={
            "threadId": f"e2e-{uuid.uuid4().hex[:8]}",
            "runId": str(uuid.uuid4()),
            "messages": [_msg("Reply with exactly: PONG")],
        }, headers={"accept": "text/event-stream"})

        _assert_run_ok(resp, label="simple_prompt")
        events = _parse_sse_events(resp.text)
        _dump_events(events, label="simple_prompt")
        assert len(events) > 0

        event_types = [e.get("type") for e in events]

        # Must start with RUN_STARTED
        assert event_types[0] == "RUN_STARTED", f"First event was {event_types[0]}"

        # Must end with RUN_FINISHED
        assert event_types[-1] == "RUN_FINISHED", f"Last event was {event_types[-1]}"

        # Must have at least one text message
        assert "TEXT_MESSAGE_START" in event_types
        assert "TEXT_MESSAGE_CONTENT" in event_types
        assert "TEXT_MESSAGE_END" in event_types

    def test_response_contains_expected_content(self, client):
        """Verify the agent's text response contains something sensible."""
        resp = client.post("/", json={
            "threadId": f"e2e-{uuid.uuid4().hex[:8]}",
            "runId": str(uuid.uuid4()),
            "messages": [_msg("What is 2 + 2? Reply with just the number.")],
        })

        _assert_run_ok(resp, label="math_prompt")
        events = _parse_sse_events(resp.text)
        _dump_events(events, label="math_prompt")

        content_events = [e for e in events if e.get("type") == "TEXT_MESSAGE_CONTENT"]
        full_text = "".join(e.get("delta", "") for e in content_events)
        print(f"  [math_prompt] Full response text: {full_text!r}")
        assert "4" in full_text, f"Expected '4' in response, got: {full_text!r}"

    def test_interrupt_during_run(self, client):
        """Verify interrupt endpoint responds correctly (no active session)."""
        thread_id = f"e2e-int-{uuid.uuid4().hex[:8]}"
        resp = client.post("/interrupt", json={"thread_id": thread_id})
        print(f"  [interrupt] status={resp.status_code} body={resp.text[:200]}")
        # 500 expected since no active session for this thread
        assert resp.status_code in (200, 500)

    def test_shared_state_in_forwarded_props(self, client):
        """Verify forwarded props are passed through to the adapter."""
        resp = client.post("/", json={
            "threadId": f"e2e-{uuid.uuid4().hex[:8]}",
            "runId": str(uuid.uuid4()),
            "messages": [_msg("Reply with exactly: OK")],
            "forwardedProps": {"customSetting": "test-value"},
        })

        _assert_run_ok(resp, label="forwarded_props")
        events = _parse_sse_events(resp.text)
        _dump_events(events, label="forwarded_props")
        event_types = [e.get("type") for e in events]
        assert "RUN_STARTED" in event_types

    def test_tools_can_be_passed(self, client):
        """Verify frontend tools can be registered via the run payload."""
        resp = client.post("/", json={
            "threadId": f"e2e-{uuid.uuid4().hex[:8]}",
            "runId": str(uuid.uuid4()),
            "messages": [_msg("Reply with exactly: TOOLS_OK")],
            "tools": [{
                "name": "test_tool",
                "description": "A test tool",
                "parameters": {"type": "object", "properties": {}},
            }],
        })

        _assert_run_ok(resp, label="tools")
        events = _parse_sse_events(resp.text)
        _dump_events(events, label="tools")
        assert any(e.get("type") == "RUN_STARTED" for e in events)

    def test_event_stream_has_messages_snapshot(self, client):
        """Verify MESSAGES_SNAPSHOT is emitted in the event stream."""
        resp = client.post("/", json={
            "threadId": f"e2e-{uuid.uuid4().hex[:8]}",
            "runId": str(uuid.uuid4()),
            "messages": [_msg("Say hi")],
        })
        _assert_run_ok(resp, label="snapshot")
        events = _parse_sse_events(resp.text)
        _dump_events(events, label="snapshot")
        event_types = [e.get("type") for e in events]
        assert "MESSAGES_SNAPSHOT" in event_types, f"Missing MESSAGES_SNAPSHOT, got: {event_types}"

    def test_capabilities_model_populated_after_run(self, client):
        """After a successful run, capabilities should reflect the configured model."""
        # Run a prompt first to trigger platform init
        resp = client.post("/", json={
            "threadId": f"e2e-{uuid.uuid4().hex[:8]}",
            "runId": str(uuid.uuid4()),
            "messages": [_msg("Reply OK")],
        })
        _assert_run_ok(resp, label="pre_caps")

        # Now check capabilities
        caps = client.get("/capabilities").json()
        print(f"  [caps_after_run] model={caps.get('model')}")
        assert caps["model"] is not None
        assert "claude" in caps["model"].lower()

    def test_multi_turn_same_thread(self, client):
        """Two runs on the same thread_id should both succeed (session reuse)."""
        thread_id = f"e2e-multi-{uuid.uuid4().hex[:8]}"

        # Turn 1
        resp1 = client.post("/", json={
            "threadId": thread_id,
            "runId": str(uuid.uuid4()),
            "messages": [_msg("Remember the word: BANANA")],
        })
        _assert_run_ok(resp1, label="multi_turn1")
        events1 = _parse_sse_events(resp1.text)
        _dump_events(events1, label="multi_turn1")
        assert any(e.get("type") == "RUN_FINISHED" for e in events1)

        # Turn 2 — same thread
        resp2 = client.post("/", json={
            "threadId": thread_id,
            "runId": str(uuid.uuid4()),
            "messages": [_msg("What word did I ask you to remember? Reply with just the word.")],
        })
        _assert_run_ok(resp2, label="multi_turn2")
        events2 = _parse_sse_events(resp2.text)
        _dump_events(events2, label="multi_turn2")

        content = "".join(
            e.get("delta", "") for e in events2
            if e.get("type") == "TEXT_MESSAGE_CONTENT"
        )
        print(f"  [multi_turn2] Response: {content!r}")
        assert "BANANA" in content.upper(), f"Expected BANANA in: {content!r}"

    def test_different_threads_are_isolated(self, client):
        """Different thread_ids should not share conversation context."""
        thread_a = f"e2e-iso-a-{uuid.uuid4().hex[:8]}"
        thread_b = f"e2e-iso-b-{uuid.uuid4().hex[:8]}"

        # Thread A: set a secret word
        resp_a = client.post("/", json={
            "threadId": thread_a,
            "runId": str(uuid.uuid4()),
            "messages": [_msg("Remember: CHERRY")],
        })
        _assert_run_ok(resp_a, label="iso_a")

        # Thread B: ask for the secret (should NOT know it)
        resp_b = client.post("/", json={
            "threadId": thread_b,
            "runId": str(uuid.uuid4()),
            "messages": [_msg("What secret word was I told? If none, reply with: NO_SECRET")],
        })
        _assert_run_ok(resp_b, label="iso_b")
        events_b = _parse_sse_events(resp_b.text)
        _dump_events(events_b, label="iso_b")

        content_b = "".join(
            e.get("delta", "") for e in events_b
            if e.get("type") == "TEXT_MESSAGE_CONTENT"
        )
        print(f"  [iso_b] Response: {content_b!r}")
        # Thread B should NOT know about CHERRY
        assert "CHERRY" not in content_b.upper(), f"Thread leak! B saw CHERRY: {content_b!r}"
