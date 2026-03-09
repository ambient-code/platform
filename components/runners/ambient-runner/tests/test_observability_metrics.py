"""Unit tests for session-level metrics integration.

Tests the observability_models module (tool classification, cost estimation,
Pydantic models) and the metrics tracking added to ObservabilityManager.
"""

import sys
import types
from unittest.mock import Mock

import pytest

# Ensure a mock 'langfuse' module exists so imports succeed in test env
if "langfuse" not in sys.modules:
    _mock_langfuse = types.ModuleType("langfuse")
    _mock_langfuse.Langfuse = Mock
    _mock_langfuse.propagate_attributes = Mock
    sys.modules["langfuse"] = _mock_langfuse

from ambient_runner.observability import ObservabilityManager
from ambient_runner.observability_models import (
    SessionMetric,
    ToolCallType,
    classify_tool,
    estimate_cost,
    is_clarification_request,
)
from tests.conftest import make_text_start, make_tool_end, make_tool_start


# ------------------------------------------------------------------
# Fixtures
# ------------------------------------------------------------------


@pytest.fixture
def manager():
    """Create an ObservabilityManager with metrics state."""
    return ObservabilityManager(
        session_id="test-session", user_id="user-1", user_name="Test User"
    )


@pytest.fixture
def manager_with_langfuse(manager):
    """ObservabilityManager with a mocked Langfuse client."""
    mock_client = Mock()
    mock_ctx = Mock()
    mock_ctx.__enter__ = Mock(return_value=Mock(trace_id="trace-123"))
    mock_ctx.__exit__ = Mock(return_value=False)
    mock_client.start_as_current_observation.return_value = mock_ctx
    mock_client.start_as_current_span.return_value = mock_ctx
    mock_client.flush = Mock()

    manager.langfuse_client = mock_client
    manager.init_event_tracking("claude-sonnet-4", "test prompt")
    return manager


# ------------------------------------------------------------------
# Tool Classification
# ------------------------------------------------------------------


class TestToolClassification:
    """Tests for ToolCallType enum and classify_tool()."""

    def test_core_tools(self):
        assert classify_tool("Read") == ToolCallType.READ
        assert classify_tool("Write") == ToolCallType.WRITE
        assert classify_tool("Bash") == ToolCallType.BASH
        assert classify_tool("Edit") == ToolCallType.EDIT
        assert classify_tool("Glob") == ToolCallType.GLOB
        assert classify_tool("Grep") == ToolCallType.GREP

    def test_interaction_tools(self):
        assert classify_tool("AskUserQuestion") == ToolCallType.ASK_USER_QUESTION
        assert classify_tool("Task") == ToolCallType.TASK
        assert classify_tool("Skill") == ToolCallType.SKILL

    def test_mcp_github(self):
        assert (
            classify_tool("mcp__github-readonly__github_pr_list")
            == ToolCallType.MCP_GITHUB
        )

    def test_mcp_jira(self):
        assert (
            classify_tool("mcp__mcp-atlassian__jira_get_issue") == ToolCallType.MCP_JIRA
        )

    def test_mcp_confluence(self):
        assert (
            classify_tool("mcp__mcp-atlassian__confluence_search")
            == ToolCallType.MCP_CONFLUENCE
        )

    def test_mcp_atlassian_generic(self):
        assert (
            classify_tool("mcp__mcp-atlassian__some_other_tool")
            == ToolCallType.MCP_ATLASSIAN
        )

    def test_mcp_ide(self):
        assert classify_tool("mcp__ide__getDiagnostics") == ToolCallType.MCP_IDE

    def test_mcp_other(self):
        assert classify_tool("mcp__custom-server__tool") == ToolCallType.MCP_OTHER

    def test_unknown_tool(self):
        assert classify_tool("SomeUnknownTool") == ToolCallType.OTHER
        assert classify_tool("") == ToolCallType.OTHER

    def test_case_sensitivity(self):
        # Direct match is case-sensitive
        assert classify_tool("read") == ToolCallType.OTHER
        # MCP prefix matching is case-insensitive
        assert classify_tool("MCP__GitHub__list") == ToolCallType.MCP_GITHUB


# ------------------------------------------------------------------
# Cost Estimation
# ------------------------------------------------------------------


class TestCostEstimation:
    """Tests for estimate_cost()."""

    def test_sonnet_pricing(self):
        usage = {"input_tokens": 1_000_000, "output_tokens": 1_000_000}
        cost = estimate_cost(usage, "claude-sonnet-4-20250514")
        # input: $3.00 + output: $15.00 = $18.00
        assert cost == 18.0

    def test_opus_pricing(self):
        usage = {"input_tokens": 1_000_000, "output_tokens": 1_000_000}
        cost = estimate_cost(usage, "claude-opus-4-20250514")
        # input: $15.00 + output: $75.00 = $90.00
        assert cost == 90.0

    def test_cache_tokens(self):
        usage = {
            "input_tokens": 0,
            "output_tokens": 0,
            "cache_read_input_tokens": 1_000_000,
            "cache_creation_input_tokens": 1_000_000,
        }
        cost = estimate_cost(usage, "claude-sonnet-4")
        # cache_read: $0.30 + cache_creation (at input price): $3.00 = $3.30
        assert cost == 3.3

    def test_unknown_model_defaults_to_sonnet(self):
        usage = {"input_tokens": 1_000_000, "output_tokens": 0}
        cost = estimate_cost(usage, "some-unknown-model")
        assert cost == 3.0  # Sonnet input price

    def test_empty_usage(self):
        cost = estimate_cost({}, "claude-sonnet-4")
        assert cost == 0.0

    def test_rounding(self):
        usage = {"input_tokens": 100, "output_tokens": 50}
        cost = estimate_cost(usage, "claude-sonnet-4")
        # Tiny amounts should be rounded to 6 decimal places
        assert isinstance(cost, float)
        assert cost > 0


# ------------------------------------------------------------------
# Clarification Detection
# ------------------------------------------------------------------


class TestClarificationDetection:
    """Tests for is_clarification_request()."""

    def test_short_question(self):
        assert is_clarification_request("What file should I modify?") is True

    def test_short_non_question(self):
        assert is_clarification_request("Done.") is False

    def test_empty_string(self):
        assert is_clarification_request("") is False

    def test_no_question_mark(self):
        assert is_clarification_request("I will now edit the file.") is False

    def test_code_block_not_clarification(self):
        text = "Here's the fix:\n```python\nprint('hello')\n```\nDoes this look right?"
        assert is_clarification_request(text) is False

    def test_long_response_with_trailing_question(self):
        text = "I've analyzed the codebase. " * 10 + "\nWhich approach do you prefer?"
        assert is_clarification_request(text) is True

    def test_long_response_no_question(self):
        text = "I've completed the implementation. " * 20
        assert is_clarification_request(text) is False

    def test_long_trailing_line_not_clarification(self):
        # Trailing line > 200 chars with total text > 500 chars shouldn't
        # be treated as clarification (bypasses the short-response check)
        long_prefix = "I've analyzed. " * 40  # > 500 chars
        long_question = "A" * 250 + "?"
        text = long_prefix + "\n" + long_question
        assert is_clarification_request(text) is False


# ------------------------------------------------------------------
# Metrics Tracking via Events
# ------------------------------------------------------------------


class TestMetricsTracking:
    """Tests for in-memory metrics accumulation in ObservabilityManager."""

    def test_init_has_zeroed_metrics(self, manager):
        assert manager._metric_tool_calls == {}
        assert manager._metric_tool_calls_total == 0
        assert manager._metric_tool_failures_total == 0
        assert manager._metric_unclear_context == 0
        assert manager._metric_human_interrupts == 0

    def test_tool_call_counting(self, manager_with_langfuse):
        mgr = manager_with_langfuse

        # Simulate tool call events
        mgr.track_agui_event(make_text_start())
        mgr.track_agui_event(make_tool_start("tc-1", "Read"))
        mgr.track_agui_event(make_tool_end("tc-1"))
        mgr.track_agui_event(make_tool_start("tc-2", "Bash"))
        mgr.track_agui_event(make_tool_end("tc-2"))
        mgr.track_agui_event(make_tool_start("tc-3", "Read"))
        mgr.track_agui_event(make_tool_end("tc-3"))

        assert mgr._metric_tool_calls_total == 3
        assert mgr._metric_tool_calls["Read"] == 2
        assert mgr._metric_tool_calls["Bash"] == 1

    def test_mcp_tool_classification(self, manager_with_langfuse):
        mgr = manager_with_langfuse

        mgr.track_agui_event(make_text_start())
        mgr.track_agui_event(
            make_tool_start("tc-1", "mcp__github-readonly__github_pr_list")
        )
        mgr.track_agui_event(make_tool_end("tc-1"))

        assert mgr._metric_tool_calls.get("mcp_github") == 1

    def test_ask_user_question_increments_unclear_context(self, manager_with_langfuse):
        mgr = manager_with_langfuse

        mgr.track_agui_event(make_text_start())
        mgr.track_agui_event(make_tool_start("tc-1", "AskUserQuestion"))
        mgr.track_agui_event(make_tool_end("tc-1"))

        assert mgr._metric_unclear_context == 1
        assert mgr._metric_tool_calls.get("AskUserQuestion") == 1

    def test_tool_failure_tracking(self, manager_with_langfuse):
        mgr = manager_with_langfuse

        mgr.track_agui_event(make_text_start())
        mgr.track_agui_event(make_tool_start("tc-1", "Bash"))

        # Create a tool end event with an error
        error_event = make_tool_end("tc-1", error="Command not found: foobar")
        mgr.track_agui_event(error_event)

        assert mgr._metric_tool_failures_total == 1
        assert mgr._metric_tool_failure_counts.get("Bash") == 1
        assert any(
            "Bash" in k and "Command not found" in k
            for k in mgr._metric_tool_failure_reasons
        )

    def test_record_interrupt(self, manager):
        assert manager._metric_human_interrupts == 0
        manager.record_interrupt()
        assert manager._metric_human_interrupts == 1
        manager.record_interrupt()
        assert manager._metric_human_interrupts == 2

    def test_accumulate_usage_per_query(self, manager):
        """Test accumulation when SDK reports per-query (non-cumulative) values."""
        usage1 = {
            "input_tokens": 100,
            "output_tokens": 50,
            "cache_read_input_tokens": 20,
        }
        manager._accumulate_usage(usage1, "claude-sonnet-4")

        assert manager._metric_accumulated_usage["input_tokens"] == 100
        assert manager._metric_accumulated_usage["output_tokens"] == 50
        assert manager._metric_accumulated_usage["cache_read_input_tokens"] == 20
        assert manager._metric_models_seen["claude-sonnet-4"] == 1
        assert manager._metric_total_cost_usd > 0

    def test_accumulate_usage_cumulative_computes_deltas(self, manager):
        """Test that cumulative SDK values don't cause double-counting.

        The SDK may report cumulative usage across queries. If query 1
        reports input_tokens=100 and query 2 reports input_tokens=300,
        the actual delta is 200 (not 300). The method should accumulate
        only the delta.
        """
        # Query 1: 100 input, 50 output
        usage1 = {"input_tokens": 100, "output_tokens": 50}
        manager._accumulate_usage(usage1, "claude-sonnet-4")
        assert manager._metric_accumulated_usage["input_tokens"] == 100
        assert manager._metric_accumulated_usage["output_tokens"] == 50

        # Query 2: cumulative 300 input, 150 output (delta: 200 in, 100 out)
        usage2 = {"input_tokens": 300, "output_tokens": 150}
        manager._accumulate_usage(usage2, "claude-sonnet-4")
        assert manager._metric_accumulated_usage["input_tokens"] == 300
        assert manager._metric_accumulated_usage["output_tokens"] == 150

        # Query 3: cumulative 600 input, 200 output (delta: 300 in, 50 out)
        usage3 = {"input_tokens": 600, "output_tokens": 200}
        manager._accumulate_usage(usage3, "claude-sonnet-4")
        assert manager._metric_accumulated_usage["input_tokens"] == 600
        assert manager._metric_accumulated_usage["output_tokens"] == 200

        assert manager._metric_models_seen["claude-sonnet-4"] == 3

    def test_accumulate_usage_none_is_noop(self, manager):
        manager._accumulate_usage(None, "claude-sonnet-4")
        assert manager._metric_accumulated_usage == {}

    def test_clarification_detection_in_turn(self, manager):
        manager._detect_and_record_clarification("What file should I edit?")
        assert manager._metric_unclear_context == 1

        manager._detect_and_record_clarification("Done editing the file.")
        assert manager._metric_unclear_context == 1  # not incremented


# ------------------------------------------------------------------
# Session Summary Emission
# ------------------------------------------------------------------


class TestSessionSummaryEmission:
    """Tests for _emit_session_summary() Langfuse score emission."""

    def test_emit_summary_with_metrics(self, manager_with_langfuse):
        mgr = manager_with_langfuse

        # Populate some metrics
        mgr._metric_tool_calls = {"Read": 3, "Bash": 2}
        mgr._metric_tool_calls_total = 5
        mgr._metric_tool_failures_total = 1
        mgr._metric_tool_failure_counts = {"Bash": 1}
        mgr._metric_tool_failure_reasons = {"Bash: Permission denied": 1}
        mgr._metric_unclear_context = 1
        mgr._metric_human_interrupts = 0
        mgr._metric_accumulated_usage = {
            "input_tokens": 1000,
            "output_tokens": 500,
        }
        mgr._metric_models_seen = {"claude-sonnet-4": 3}
        mgr._metric_total_cost_usd = 0.0105

        mgr._emit_session_summary()

        # Verify Langfuse span was created
        assert mgr.langfuse_client.start_as_current_span.called
        call_kwargs = mgr.langfuse_client.start_as_current_span.call_args[1]
        assert call_kwargs["name"] == "Claude Code - Session Metrics"

        meta = call_kwargs["metadata"]
        assert meta["source"] == "claude-code-metrics"
        # Verify consolidated metadata matches trace-level fields
        assert meta["namespace"] == ""
        assert meta["user_name"] == "Test User"
        assert call_kwargs["input"]["namespace"] == ""
        assert call_kwargs["input"]["user_name"] == "Test User"
        # Verify flat scores are embedded in metadata (not as separate score objects)
        assert meta["tool_calls_total"] == 5.0
        assert meta["tool_calls_Read"] == 3.0
        assert meta["token_input"] == 1000.0
        assert meta["estimated_cost_usd"] == 0.0105

    def test_skip_when_no_metrics(self, manager_with_langfuse):
        mgr = manager_with_langfuse

        # No tool calls, no usage — should skip
        mgr._emit_session_summary()

        assert not mgr.langfuse_client.start_as_current_span.called

    def test_skip_when_no_langfuse(self, manager):
        # No langfuse_client — should not raise
        manager._metric_tool_calls_total = 5
        manager._emit_session_summary()  # Should not raise


# ------------------------------------------------------------------
# SessionMetric Model
# ------------------------------------------------------------------


class TestSessionMetricModel:
    """Tests for SessionMetric Pydantic model and to_flat_scores()."""

    def test_build_and_serialize(self):
        metric = SessionMetric.build(
            session_id="s-1",
            user_id="u-1",
            tool_calls={"Read": 3, "Bash": 2},
            tool_calls_total=5,
            tool_failures_total=1,
            tool_failure_counts={"Bash": 1},
            tool_failure_reasons={"Bash: error": 1},
            unclear_context=2,
            human_interrupts=1,
            accumulated_usage={
                "input_tokens": 1000,
                "output_tokens": 500,
                "cache_read_input_tokens": 200,
            },
            models_seen={"claude-sonnet-4": 3},
            total_cost_usd=0.0105,
        )

        assert metric.session_id == "s-1"
        assert metric.tools_usage_metric.tool_calls_total == 5
        assert metric.interrupt_metric.interrupt_unclear_context == 2
        assert metric.token_metrics.token_total == 1700  # 1000+500+200

        # Verify serialization
        data = metric.model_dump()
        assert data["session_id"] == "s-1"
        assert data["token_metrics"]["estimated_cost_usd"] == 0.0105

    def test_to_flat_scores(self):
        metric = SessionMetric.build(
            session_id="s-1",
            user_id="u-1",
            tool_calls={"Read": 3, "Bash": 2},
            tool_calls_total=5,
            tool_failures_total=1,
            tool_failure_counts={"Bash": 1},
            tool_failure_reasons={},
            unclear_context=0,
            human_interrupts=1,
            accumulated_usage={"input_tokens": 1000, "output_tokens": 500},
            models_seen={},
            total_cost_usd=0.01,
        )

        scores = metric.to_flat_scores()

        assert scores["tool_calls_total"] == 5.0
        assert scores["tool_calls_Read"] == 3.0
        assert scores["tool_calls_Bash"] == 2.0
        assert scores["interrupt_tool_failure_total"] == 1.0
        assert scores["interrupt_tool_failure_Bash"] == 1.0
        assert scores["interrupt_human"] == 1.0
        assert scores["token_input"] == 1000.0
        assert scores["token_output"] == 500.0
        assert scores["estimated_cost_usd"] == 0.01

    def test_flat_scores_empty_metric(self):
        metric = SessionMetric(session_id="s-1", user_id="u-1")
        scores = metric.to_flat_scores()

        assert scores["tool_calls_total"] == 0.0
        assert scores["token_total"] == 0.0
        assert scores["interrupt_human"] == 0.0
        assert all(isinstance(v, float) for v in scores.values())
