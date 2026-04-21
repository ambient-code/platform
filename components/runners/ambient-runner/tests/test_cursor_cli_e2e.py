"""E2E smoke test: runs real cursor-agent binary and validates parse_event().

Requires:
- cursor-agent binary in PATH and authenticated (cursor-agent login)
- Network access to Cursor API

Skip with: pytest -m "not e2e"
"""

import subprocess
import pytest

from ag_ui_cursor_cli.types import (
    InitEvent,
    MessageEvent,
    ResultEvent,
    ToolCallCompletedEvent,
    ToolCallStartEvent,
    parse_event,
)

pytestmark = pytest.mark.e2e


def _has_cursor_agent() -> bool:
    try:
        result = subprocess.run(
            ["cursor-agent", "--version"],
            capture_output=True,
            text=True,
            timeout=5,
        )
        return result.returncode == 0
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return False


def _run_cursor_agent(prompt: str, cwd: str = "/tmp") -> list[str]:
    result = subprocess.run(
        [
            "cursor-agent",
            "--print",
            "--force",
            "--output-format",
            "stream-json",
            prompt,
        ],
        capture_output=True,
        text=True,
        timeout=60,
        cwd=cwd,
    )
    assert result.returncode == 0, f"cursor-agent failed: {result.stderr}"
    return [line for line in result.stdout.strip().split("\n") if line.strip()]


@pytest.mark.skipif(
    not _has_cursor_agent(), reason="cursor-agent not installed or not authenticated"
)
class TestCursorAgentE2E:
    def test_simple_prompt_parses_all_events(self):
        lines = _run_cursor_agent("respond with exactly: hello world")

        assert len(lines) >= 3, (
            f"Expected at least init+message+result, got {len(lines)}"
        )

        events = [parse_event(line) for line in lines]
        none_count = sum(1 for e in events if e is None)
        assert none_count == 0, (
            f"{none_count}/{len(lines)} lines returned None from parse_event. "
            f"Unparsed lines: {[line for line, ev in zip(lines, events) if ev is None]}"
        )

        # Verify event type sequence
        types = [(type(e).__name__, getattr(e, "type", "")) for e in events]
        assert types[0] == ("InitEvent", "system"), (
            f"First event should be InitEvent, got {types[0]}"
        )
        assert types[-1] == ("ResultEvent", "result"), (
            f"Last event should be ResultEvent, got {types[-1]}"
        )

        # Verify init has session_id
        init = events[0]
        assert isinstance(init, InitEvent)
        assert init.session_id, "InitEvent.session_id should not be empty"
        assert init.model, "InitEvent.model should not be empty"

        # Verify result
        result = events[-1]
        assert isinstance(result, ResultEvent)
        assert not result.is_error
        assert result.duration_ms > 0

        # Verify at least one assistant message with content
        assistant_msgs = [
            e for e in events if isinstance(e, MessageEvent) and e.type == "assistant"
        ]
        assert len(assistant_msgs) > 0, "Expected at least one assistant message"
        full_text = "".join(m.content for m in assistant_msgs)
        assert "hello" in full_text.lower(), (
            f"Expected 'hello' in response, got: {full_text}"
        )

    def test_tool_use_prompt_parses_tool_events(self, tmp_path):
        test_file = tmp_path / "test.txt"
        test_file.write_text("sentinel-value-12345\n")

        lines = _run_cursor_agent(
            f"read the file {test_file} and tell me what it says",
            cwd=str(tmp_path),
        )

        events = [parse_event(line) for line in lines]
        none_count = sum(1 for e in events if e is None)
        assert none_count == 0, (
            f"{none_count}/{len(lines)} lines returned None. "
            f"Unparsed: {[line[:120] for line, ev in zip(lines, events) if ev is None]}"
        )

        tool_starts = [e for e in events if isinstance(e, ToolCallStartEvent)]
        tool_completes = [e for e in events if isinstance(e, ToolCallCompletedEvent)]

        assert len(tool_starts) > 0, "Expected at least one ToolCallStartEvent"
        assert len(tool_completes) > 0, "Expected at least one ToolCallCompletedEvent"

        # Verify tool call has an ID and name
        tc = tool_starts[0]
        assert tc.tool_id, "tool_id should not be empty"
        assert tc.tool_name, "tool_name should not be empty"

        # Verify tool completion has output
        tcd = tool_completes[0]
        assert tcd.tool_id, "completed tool_id should not be empty"
        assert tcd.output or tcd.error, "completed tool should have output or error"
