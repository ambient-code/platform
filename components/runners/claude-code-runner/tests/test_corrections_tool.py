#!/usr/bin/env python3
"""
Test corrections feedback MCP tool.

Validates:
1. Tool creation and schema structure
2. Langfuse score creation with correct parameters
3. Auto-capture of session context from environment
4. Error handling for missing Langfuse / credentials
5. Input validation and truncation
"""

import os
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from ambient_runner.bridges.claude.corrections import (
    CORRECTION_CATEGORIES,
    CORRECTION_INPUT_SCHEMA,
    CORRECTION_SCOPES,
    _get_session_context,
    _log_correction_to_langfuse,
    create_correction_mcp_tool,
)


# ------------------------------------------------------------------
# Schema validation
# ------------------------------------------------------------------


def test_schema_has_all_categories():
    """Input schema includes all 9 correction categories."""
    schema_categories = CORRECTION_INPUT_SCHEMA["properties"]["category"]["enum"]
    assert schema_categories == CORRECTION_CATEGORIES
    assert len(schema_categories) == 9


def test_schema_severity_values():
    """Severity enum is [1, 2, 3]."""
    severity_enum = CORRECTION_INPUT_SCHEMA["properties"]["severity"]["enum"]
    assert severity_enum == [1, 2, 3]


def test_schema_scope_values():
    """Scope enum includes all 4 scope values."""
    scope_enum = CORRECTION_INPUT_SCHEMA["properties"]["scope"]["enum"]
    assert scope_enum == CORRECTION_SCOPES
    assert len(scope_enum) == 4


def test_schema_required_fields():
    """Required fields are category, severity, scope, description."""
    required = CORRECTION_INPUT_SCHEMA["required"]
    assert "category" in required
    assert "severity" in required
    assert "scope" in required
    assert "description" in required
    assert "correction_details" not in required


# ------------------------------------------------------------------
# Context auto-capture
# ------------------------------------------------------------------


@patch.dict(
    os.environ,
    {
        "ACTIVE_WORKFLOW_GIT_URL": "https://github.com/org/workflow.git",
        "ACTIVE_WORKFLOW_PATH": "workflows/bug-fix",
        "AGENTIC_SESSION_NAME": "session-12345",
        "AGENTIC_SESSION_NAMESPACE": "test-project",
    },
)
def test_captures_context_from_env():
    """Session context is captured from environment variables."""
    ctx = _get_session_context()
    assert ctx["repo_url"] == "https://github.com/org/workflow.git"
    assert ctx["workflow"] == "workflows/bug-fix"
    assert ctx["session_name"] == "session-12345"
    assert ctx["project"] == "test-project"


@patch.dict(os.environ, {}, clear=True)
def test_handles_missing_env_vars():
    """Missing env vars result in empty strings, not errors."""
    ctx = _get_session_context()
    assert ctx["repo_url"] == ""
    assert ctx["workflow"] == ""
    assert ctx["session_name"] == ""
    assert ctx["project"] == ""


# ------------------------------------------------------------------
# Langfuse logging
# ------------------------------------------------------------------


def test_successful_logging():
    """Score is created with correct name, value, and metadata."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = "trace-abc"

    with patch.dict(
        os.environ,
        {
            "ACTIVE_WORKFLOW_GIT_URL": "https://github.com/org/repo.git",
            "ACTIVE_WORKFLOW_PATH": "workflows/test",
            "AGENTIC_SESSION_NAME": "session-1",
            "AGENTIC_SESSION_NAMESPACE": "my-project",
        },
    ):
        success, error = _log_correction_to_langfuse(
            category="wrong_approach",
            severity=3,
            scope="repo_context",
            description="Agent took wrong approach to error handling",
            correction_details="Should have used try/except not if/else",
            obs=mock_obs,
            session_id="session-1",
        )

    assert success is True
    assert error is None

    mock_obs.langfuse_client.create_score.assert_called_once()
    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]

    assert call_kwargs["name"] == "session-correction"
    assert call_kwargs["value"] == 3
    assert call_kwargs["data_type"] == "NUMERIC"
    assert call_kwargs["trace_id"] == "trace-abc"
    assert call_kwargs["comment"] == "Agent took wrong approach to error handling"

    metadata = call_kwargs["metadata"]
    assert metadata["category"] == "wrong_approach"
    assert metadata["scope"] == "repo_context"
    assert metadata["session_id"] == "session-1"
    assert metadata["repo_url"] == "https://github.com/org/repo.git"
    assert metadata["workflow"] == "workflows/test"
    assert metadata["session_name"] == "session-1"
    assert metadata["project"] == "my-project"
    assert metadata["correction_details"] == "Should have used try/except not if/else"

    mock_obs.langfuse_client.flush.assert_called_once()


def test_logging_without_trace_id():
    """Score created without trace_id when not available."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_correction_to_langfuse(
            category="style_violation",
            severity=1,
            scope="code_patterns",
            description="Wrong code style",
            correction_details=None,
            obs=mock_obs,
            session_id="session-1",
        )

    assert success is True
    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert "trace_id" not in call_kwargs


def test_logging_without_langfuse_enabled():
    """Returns failure when Langfuse not enabled and no obs client."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = None

    with patch.dict(os.environ, {"LANGFUSE_ENABLED": "false"}, clear=True):
        success, error = _log_correction_to_langfuse(
            category="wrong_approach",
            severity=2,
            scope="repo_context",
            description="test",
            correction_details=None,
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
            success, error = _log_correction_to_langfuse(
                category="wrong_approach",
                severity=2,
                scope="repo_context",
                description="test",
                correction_details=None,
                obs=mock_obs,
                session_id="session-1",
            )

    assert success is False
    assert "credentials missing" in error.lower()


def test_description_truncation():
    """Description is truncated to 500 chars in comment."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None

    long_desc = "x" * 1000

    with patch.dict(os.environ, {}, clear=True):
        _log_correction_to_langfuse(
            category="wrong_approach",
            severity=2,
            scope="repo_context",
            description=long_desc,
            correction_details=None,
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert len(call_kwargs["comment"]) == 500


def test_correction_details_truncation():
    """Correction details are truncated to 500 chars in metadata."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None

    long_details = "y" * 1000

    with patch.dict(os.environ, {}, clear=True):
        _log_correction_to_langfuse(
            category="wrong_approach",
            severity=2,
            scope="repo_context",
            description="test",
            correction_details=long_details,
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert len(call_kwargs["metadata"]["correction_details"]) == 500


def test_no_correction_details_omitted_from_metadata():
    """correction_details key absent from metadata when not provided."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None

    with patch.dict(os.environ, {}, clear=True):
        _log_correction_to_langfuse(
            category="wrong_approach",
            severity=2,
            scope="repo_context",
            description="test",
            correction_details=None,
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert "correction_details" not in call_kwargs["metadata"]


def test_logging_with_no_obs():
    """Returns failure when obs is None and Langfuse not enabled."""
    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_correction_to_langfuse(
            category="wrong_approach",
            severity=2,
            scope="repo_context",
            description="test",
            correction_details=None,
            obs=None,
            session_id="session-1",
        )

    assert success is False


# ------------------------------------------------------------------
# Tool creation
# ------------------------------------------------------------------


def test_tool_creation():
    """Tool is created with correct name via decorator."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_correction_mcp_tool(
        obs=MagicMock(),
        session_id="session-1",
        sdk_tool_decorator=mock_decorator,
    )

    assert tool is not None
    mock_decorator.assert_called_once()
    call_args = mock_decorator.call_args[0]
    assert call_args[0] == "log_correction"


# ------------------------------------------------------------------
# Runner
# ------------------------------------------------------------------


if __name__ == "__main__":
    print("Testing corrections feedback MCP tool...")
    print("=" * 60)

    tests = [
        ("Schema: all categories", test_schema_has_all_categories),
        ("Schema: severity values", test_schema_severity_values),
        ("Schema: scope values", test_schema_scope_values),
        ("Schema: required fields", test_schema_required_fields),
        ("Context: captures from env", test_captures_context_from_env),
        ("Context: handles missing env", test_handles_missing_env_vars),
        ("Logging: successful", test_successful_logging),
        ("Logging: no trace_id", test_logging_without_trace_id),
        ("Logging: not enabled", test_logging_without_langfuse_enabled),
        ("Logging: no credentials", test_logging_without_credentials),
        ("Logging: description truncation", test_description_truncation),
        ("Logging: details truncation", test_correction_details_truncation),
        ("Logging: no details omitted", test_no_correction_details_omitted_from_metadata),
        ("Logging: no obs", test_logging_with_no_obs),
        ("Tool: creation", test_tool_creation),
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
