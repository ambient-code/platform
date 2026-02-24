#!/usr/bin/env python3
"""
Test global feedback MCP tool.

Validates:
1. Tool creation and schema structure
2. Langfuse score creation with correct parameters
3. Error handling for missing Langfuse / credentials
4. Comment truncation
5. Rating to boolean mapping
"""

import os
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

sys.path.insert(0, str(Path(__file__).parent.parent))

from ambient_runner.bridges.claude.feedback_tool import (
    FEEDBACK_INPUT_SCHEMA,
    FEEDBACK_RATINGS,
    FEEDBACK_TOOL_DESCRIPTION,
    _log_feedback_to_langfuse,
    create_feedback_mcp_tool,
)


# ------------------------------------------------------------------
# Schema validation
# ------------------------------------------------------------------


def test_schema_has_rating_and_comment():
    """Input schema includes both rating and comment fields."""
    props = FEEDBACK_INPUT_SCHEMA["properties"]
    assert "rating" in props
    assert "comment" in props


def test_schema_rating_enum():
    """Rating enum contains positive and negative."""
    enum = FEEDBACK_INPUT_SCHEMA["properties"]["rating"]["enum"]
    assert "positive" in enum
    assert "negative" in enum
    assert enum == FEEDBACK_RATINGS


def test_schema_required_fields():
    """Both rating and comment are required."""
    assert "rating" in FEEDBACK_INPUT_SCHEMA["required"]
    assert "comment" in FEEDBACK_INPUT_SCHEMA["required"]


# ------------------------------------------------------------------
# Tool creation
# ------------------------------------------------------------------


def test_tool_creation():
    """Tool is created with correct name via decorator."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_feedback_mcp_tool(
        obs=MagicMock(),
        session_id="session-1",
        sdk_tool_decorator=mock_decorator,
    )

    assert tool is not None
    mock_decorator.assert_called_once()
    call_args = mock_decorator.call_args[0]
    assert call_args[0] == "submit_feedback"


def test_tool_description_content():
    """Tool description explains when to call submit_feedback."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    create_feedback_mcp_tool(
        obs=MagicMock(),
        session_id="session-1",
        sdk_tool_decorator=mock_decorator,
    )

    description = mock_decorator.call_args[0][1]
    assert (
        "submit_feedback" not in description or FEEDBACK_TOOL_DESCRIPTION in description
    )
    assert "positive" in description or "rating" in description


def test_tool_schema_passed_to_decorator():
    """Full input schema is passed as third arg to decorator."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    create_feedback_mcp_tool(
        obs=MagicMock(),
        session_id="session-1",
        sdk_tool_decorator=mock_decorator,
    )

    schema = mock_decorator.call_args[0][2]
    assert schema["type"] == "object"
    assert "rating" in schema["properties"]
    assert "comment" in schema["properties"]


# ------------------------------------------------------------------
# Langfuse logging
# ------------------------------------------------------------------


def test_positive_rating_maps_to_true():
    """Positive rating logs value=True to Langfuse."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = "trace-abc"

    with patch.dict(
        os.environ,
        {
            "AGENTIC_SESSION_NAME": "session-1",
            "AGENTIC_SESSION_NAMESPACE": "my-project",
        },
    ):
        success, error = _log_feedback_to_langfuse(
            rating="positive",
            comment="Great job!",
            obs=mock_obs,
            session_id="session-1",
        )

    assert success is True
    assert error is None

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert call_kwargs["value"] is True
    assert call_kwargs["name"] == "session-feedback"
    assert call_kwargs["data_type"] == "BOOLEAN"
    assert call_kwargs["trace_id"] == "trace-abc"
    mock_obs.langfuse_client.flush.assert_called_once()


def test_negative_rating_maps_to_false():
    """Negative rating logs value=False to Langfuse."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = "trace-xyz"

    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_feedback_to_langfuse(
            rating="negative",
            comment="Needs improvement.",
            obs=mock_obs,
            session_id="session-1",
        )

    assert success is True
    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert call_kwargs["value"] is False


def test_logging_metadata_includes_session_context():
    """Metadata captures session_id, session_name, project, and rating."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    with patch.dict(
        os.environ,
        {
            "AGENTIC_SESSION_NAME": "my-session",
            "AGENTIC_SESSION_NAMESPACE": "my-project",
        },
    ):
        _log_feedback_to_langfuse(
            rating="positive",
            comment="Excellent output.",
            obs=mock_obs,
            session_id="session-42",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    metadata = call_kwargs["metadata"]
    assert metadata["rating"] == "positive"
    assert metadata["session_id"] == "session-42"
    assert metadata["session_name"] == "my-session"
    assert metadata["project"] == "my-project"


def test_logging_without_trace_id():
    """Score is created without trace_id when none is available."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_feedback_to_langfuse(
            rating="positive",
            comment="All good.",
            obs=mock_obs,
            session_id="session-1",
        )

    assert success is True
    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert "trace_id" not in call_kwargs


def test_logging_uses_last_trace_id_fallback():
    """Falls back to obs.last_trace_id when get_current_trace_id returns None."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = "last-trace-999"

    with patch.dict(os.environ, {}, clear=True):
        _log_feedback_to_langfuse(
            rating="negative",
            comment="Not what I expected.",
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert call_kwargs["trace_id"] == "last-trace-999"


def test_logging_without_langfuse_enabled():
    """Returns failure when Langfuse not enabled and no obs client."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = None

    with patch.dict(os.environ, {"LANGFUSE_ENABLED": "false"}, clear=True):
        success, error = _log_feedback_to_langfuse(
            rating="positive",
            comment="Great!",
            obs=mock_obs,
            session_id="session-1",
        )

    assert success is False
    assert "not enabled" in error


def test_logging_without_credentials():
    """Returns failure when Langfuse enabled but credentials missing."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = None

    with patch.dict(os.environ, {"LANGFUSE_ENABLED": "true"}, clear=True):
        with patch.dict("sys.modules", {"langfuse": MagicMock()}):
            success, error = _log_feedback_to_langfuse(
                rating="positive",
                comment="Nice work.",
                obs=mock_obs,
                session_id="session-1",
            )

    assert success is False
    assert "credentials missing" in error.lower()


def test_logging_with_no_obs():
    """Returns failure when obs is None and Langfuse not enabled."""
    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_feedback_to_langfuse(
            rating="positive",
            comment="Great.",
            obs=None,
            session_id="session-1",
        )

    assert success is False


def test_comment_truncation():
    """Comment is truncated to 500 chars."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    long_comment = "x" * 1000

    with patch.dict(os.environ, {}, clear=True):
        _log_feedback_to_langfuse(
            rating="positive",
            comment=long_comment,
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert len(call_kwargs["comment"]) == 500


# ------------------------------------------------------------------
# Runner
# ------------------------------------------------------------------


if __name__ == "__main__":
    print("Testing global feedback MCP tool...")
    print("=" * 60)

    tests = [
        ("Schema: rating and comment fields", test_schema_has_rating_and_comment),
        ("Schema: rating enum values", test_schema_rating_enum),
        ("Schema: required fields", test_schema_required_fields),
        ("Tool: creation", test_tool_creation),
        ("Tool: description content", test_tool_description_content),
        ("Tool: schema passed to decorator", test_tool_schema_passed_to_decorator),
        ("Logging: positive maps to True", test_positive_rating_maps_to_true),
        ("Logging: negative maps to False", test_negative_rating_maps_to_false),
        (
            "Logging: metadata has session context",
            test_logging_metadata_includes_session_context,
        ),
        ("Logging: no trace_id", test_logging_without_trace_id),
        ("Logging: last_trace_id fallback", test_logging_uses_last_trace_id_fallback),
        ("Logging: not enabled", test_logging_without_langfuse_enabled),
        ("Logging: no credentials", test_logging_without_credentials),
        ("Logging: no obs", test_logging_with_no_obs),
        ("Logging: comment truncation", test_comment_truncation),
    ]

    passed = 0
    failed = 0

    for test_name, test_func in tests:
        try:
            test_func()
            print(f"  PASS  {test_name}")
            passed += 1
        except AssertionError as e:
            print(f"  FAIL  {test_name}: {e}")
            failed += 1
        except Exception as e:
            print(f"  FAIL  {test_name}: Unexpected error: {e}")
            failed += 1

    print("=" * 60)
    print(f"Results: {passed} passed, {failed} failed")

    if failed > 0:
        sys.exit(1)
