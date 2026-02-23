#!/usr/bin/env python3
"""
Test corrections feedback MCP tool.

Validates:
1. Tool creation and schema structure
2. Target map building from env vars
3. Dynamic schema generation with target enum
4. Langfuse score creation with correct parameters
5. Auto-capture of session context from environment
6. Error handling for missing Langfuse / credentials
7. Input validation and truncation
8. Source field (human vs rubric)
"""

import json
import os
import sys
from pathlib import Path
from unittest.mock import MagicMock, patch

sys.path.insert(0, str(Path(__file__).parent.parent))

from ambient_runner.bridges.claude.corrections import (
    BASE_CORRECTION_PROPERTIES,
    BASE_REQUIRED_FIELDS,
    CORRECTION_SOURCES,
    CORRECTION_TOOL_DESCRIPTION_BASE,
    CORRECTION_TYPES,
    RUBRIC_CORRECTION_ADDENDUM,
    _get_session_context,
    _log_correction_to_langfuse,
    _repo_name,
    _resolve_target,
    build_correction_schema,
    build_target_map,
    create_correction_mcp_tool,
)


# ------------------------------------------------------------------
# Schema validation
# ------------------------------------------------------------------


def test_base_schema_has_all_correction_types():
    """Base properties include all correction types."""
    schema_types = BASE_CORRECTION_PROPERTIES["correction_type"]["enum"]
    assert schema_types == CORRECTION_TYPES
    assert len(schema_types) == 4


def test_base_schema_source_values():
    """Source enum includes human and rubric."""
    source_enum = BASE_CORRECTION_PROPERTIES["source"]["enum"]
    assert source_enum == CORRECTION_SOURCES
    assert "human" in source_enum
    assert "rubric" in source_enum


def test_base_required_fields():
    """Required fields are correction_type, agent_action, user_correction."""
    assert "correction_type" in BASE_REQUIRED_FIELDS
    assert "agent_action" in BASE_REQUIRED_FIELDS
    assert "user_correction" in BASE_REQUIRED_FIELDS
    assert "target" not in BASE_REQUIRED_FIELDS


# ------------------------------------------------------------------
# Target map
# ------------------------------------------------------------------


def test_repo_name_extracts_from_url():
    """_repo_name extracts short name from various URL formats."""
    assert _repo_name("https://github.com/org/my-repo.git") == "my-repo"
    assert _repo_name("https://github.com/org/my-repo") == "my-repo"
    assert _repo_name("https://github.com/org/my-repo/") == "my-repo"


def test_build_target_map_workflow_only():
    """Target map with only a workflow."""
    context = {
        "workflow": {
            "repo_url": "https://github.com/org/workflows.git",
            "branch": "main",
            "path": "workflows/bug-fix",
        },
        "repos": [],
    }
    targets = build_target_map(context)
    assert len(targets) == 1
    assert "bug-fix" in targets
    assert targets["bug-fix"]["target_type"] == "workflow"
    assert targets["bug-fix"]["target_repo_url"] == "https://github.com/org/workflows.git"
    assert targets["bug-fix"]["target_branch"] == "main"
    assert targets["bug-fix"]["target_path"] == "workflows/bug-fix"


def test_build_target_map_repos_only():
    """Target map with only repos, no workflow."""
    context = {
        "workflow": {"repo_url": "", "branch": "", "path": ""},
        "repos": [
            {"url": "https://github.com/org/app.git", "branch": "main"},
            {"url": "https://github.com/org/lib.git", "branch": "dev"},
        ],
    }
    targets = build_target_map(context)
    assert len(targets) == 2
    assert "app" in targets
    assert "lib" in targets
    assert targets["app"]["target_type"] == "repo"
    assert targets["lib"]["target_branch"] == "dev"


def test_build_target_map_workflow_and_repos():
    """Target map with both workflow and repos."""
    context = {
        "workflow": {
            "repo_url": "https://github.com/org/workflows.git",
            "branch": "main",
            "path": "workflows/joker",
        },
        "repos": [
            {"url": "https://github.com/org/my-app.git", "branch": "main"},
        ],
    }
    targets = build_target_map(context)
    assert len(targets) == 2
    assert "joker" in targets
    assert "my-app" in targets
    assert targets["joker"]["target_type"] == "workflow"
    assert targets["my-app"]["target_type"] == "repo"


def test_build_target_map_label_collision():
    """Repo label gets '-repo' suffix when it collides with workflow label."""
    context = {
        "workflow": {
            "repo_url": "https://github.com/org/collider.git",
            "branch": "main",
            "path": "",
        },
        "repos": [
            {"url": "https://github.com/other/collider.git", "branch": "main"},
        ],
    }
    targets = build_target_map(context)
    assert len(targets) == 2
    assert "collider" in targets
    assert "collider-repo" in targets
    assert targets["collider"]["target_type"] == "workflow"
    assert targets["collider-repo"]["target_type"] == "repo"


def test_build_target_map_empty():
    """Empty target map when no workflow and no repos."""
    context = {
        "workflow": {"repo_url": "", "branch": "", "path": ""},
        "repos": [],
    }
    targets = build_target_map(context)
    assert len(targets) == 0


def test_build_target_map_workflow_label_from_path():
    """Workflow label uses last segment of path."""
    context = {
        "workflow": {
            "repo_url": "https://github.com/org/wf.git",
            "branch": "main",
            "path": "workflows/deep/nested/my-flow",
        },
        "repos": [],
    }
    targets = build_target_map(context)
    assert "my-flow" in targets


def test_build_target_map_workflow_label_from_repo_when_no_path():
    """Workflow label falls back to repo name when path is empty."""
    context = {
        "workflow": {
            "repo_url": "https://github.com/org/my-workflows.git",
            "branch": "main",
            "path": "",
        },
        "repos": [],
    }
    targets = build_target_map(context)
    assert "my-workflows" in targets


# ------------------------------------------------------------------
# Dynamic schema
# ------------------------------------------------------------------


def test_schema_with_multiple_targets():
    """Schema includes target as required enum when multiple targets exist."""
    schema = build_correction_schema(["bug-fix", "my-app"])
    assert "target" in schema["properties"]
    assert schema["properties"]["target"]["enum"] == ["bug-fix", "my-app"]
    assert "target" in schema["required"]


def test_schema_with_single_target():
    """Schema includes target as optional when only one target exists."""
    schema = build_correction_schema(["bug-fix"])
    assert "target" in schema["properties"]
    assert "target" not in schema["required"]


def test_schema_with_no_targets():
    """Schema omits target entirely when no targets available."""
    schema = build_correction_schema([])
    assert "target" not in schema["properties"]


# ------------------------------------------------------------------
# Target resolution
# ------------------------------------------------------------------


def test_resolve_target_by_label():
    """_resolve_target finds the correct target by label."""
    target_map = {
        "bug-fix": {
            "target_type": "workflow",
            "target_repo_url": "https://github.com/org/wf.git",
            "target_branch": "main",
            "target_path": "workflows/bug-fix",
        },
        "my-app": {
            "target_type": "repo",
            "target_repo_url": "https://github.com/org/app.git",
            "target_branch": "dev",
            "target_path": "",
        },
    }
    result = _resolve_target("my-app", target_map)
    assert result["target_type"] == "repo"
    assert result["target_repo_url"] == "https://github.com/org/app.git"


def test_resolve_target_auto_fills_single():
    """Auto-fills when label is empty and only one target exists."""
    target_map = {
        "bug-fix": {
            "target_type": "workflow",
            "target_repo_url": "https://github.com/org/wf.git",
            "target_branch": "main",
            "target_path": "workflows/bug-fix",
        },
    }
    result = _resolve_target("", target_map)
    assert result["target_type"] == "workflow"


def test_resolve_target_empty_map():
    """Returns empty target dict when map is empty."""
    result = _resolve_target("anything", {})
    assert result["target_type"] == ""
    assert result["target_repo_url"] == ""


# ------------------------------------------------------------------
# Context auto-capture
# ------------------------------------------------------------------


@patch.dict(
    os.environ,
    {
        "ACTIVE_WORKFLOW_GIT_URL": "https://github.com/org/workflow.git",
        "ACTIVE_WORKFLOW_BRANCH": "main",
        "ACTIVE_WORKFLOW_PATH": "workflows/bug-fix",
        "AGENTIC_SESSION_NAME": "session-12345",
        "AGENTIC_SESSION_NAMESPACE": "test-project",
    },
)
def test_captures_context_from_env():
    """Session context is captured from environment variables."""
    ctx = _get_session_context()
    assert ctx["workflow"]["repo_url"] == "https://github.com/org/workflow.git"
    assert ctx["workflow"]["branch"] == "main"
    assert ctx["workflow"]["path"] == "workflows/bug-fix"
    assert ctx["session_name"] == "session-12345"
    assert ctx["project"] == "test-project"


@patch.dict(os.environ, {}, clear=True)
def test_handles_missing_env_vars():
    """Missing env vars result in empty strings, not errors."""
    ctx = _get_session_context()
    assert ctx["workflow"]["repo_url"] == ""
    assert ctx["workflow"]["branch"] == ""
    assert ctx["workflow"]["path"] == ""
    assert ctx["session_name"] == ""
    assert ctx["project"] == ""


# ------------------------------------------------------------------
# Langfuse logging
# ------------------------------------------------------------------


def test_successful_logging():
    """Score is created with correct name, value, and target metadata."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = "trace-abc"

    target_map = {
        "bug-fix": {
            "target_type": "workflow",
            "target_repo_url": "https://github.com/org/wf.git",
            "target_branch": "main",
            "target_path": "workflows/bug-fix",
        },
    }

    with patch.dict(
        os.environ,
        {
            "AGENTIC_SESSION_NAME": "session-1",
            "AGENTIC_SESSION_NAMESPACE": "my-project",
        },
    ):
        success, error = _log_correction_to_langfuse(
            correction_type="incorrect",
            agent_action="Used if/else for error handling",
            user_correction="Should have used try/except",
            target_label="bug-fix",
            target_map=target_map,
            obs=mock_obs,
            session_id="session-1",
        )

    assert success is True
    assert error is None

    mock_obs.langfuse_client.create_score.assert_called_once()
    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]

    assert call_kwargs["name"] == "session-correction"
    assert call_kwargs["value"] == "incorrect"
    assert call_kwargs["data_type"] == "CATEGORICAL"
    assert call_kwargs["trace_id"] == "trace-abc"
    assert "Used if/else" in call_kwargs["comment"]
    assert "try/except" in call_kwargs["comment"]

    metadata = call_kwargs["metadata"]
    assert metadata["correction_type"] == "incorrect"
    assert metadata["source"] == "human"
    assert metadata["target_type"] == "workflow"
    assert metadata["target_repo_url"] == "https://github.com/org/wf.git"
    assert metadata["target_branch"] == "main"
    assert metadata["target_path"] == "workflows/bug-fix"
    assert metadata["session_name"] == "session-1"
    assert metadata["project"] == "my-project"

    mock_obs.langfuse_client.flush.assert_called_once()


def test_rubric_source_logging():
    """Source='rubric' is stored in metadata for rubric-derived corrections."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = "trace-xyz"

    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_correction_to_langfuse(
            correction_type="style",
            agent_action="Originality scored 2/5 - used predictable puns",
            user_correction="Rubric requires fresh, unexpected humor",
            target_label="",
            target_map={},
            obs=mock_obs,
            session_id="session-1",
            source="rubric",
        )

    assert success is True
    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert call_kwargs["metadata"]["source"] == "rubric"
    assert call_kwargs["metadata"]["correction_type"] == "style"


def test_logging_without_trace_id():
    """Score created without trace_id when not available."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_correction_to_langfuse(
            correction_type="style",
            agent_action="Wrong code style",
            user_correction="Use consistent formatting",
            target_label="",
            target_map={},
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
            correction_type="incorrect",
            agent_action="test",
            user_correction="test",
            target_label="",
            target_map={},
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
                correction_type="incorrect",
                agent_action="test",
                user_correction="test",
                target_label="",
                target_map={},
                obs=mock_obs,
                session_id="session-1",
            )

    assert success is False
    assert "credentials missing" in error.lower()


def test_agent_action_truncation():
    """Agent action is truncated to 500 chars in metadata."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    long_action = "x" * 1000

    with patch.dict(os.environ, {}, clear=True):
        _log_correction_to_langfuse(
            correction_type="incorrect",
            agent_action=long_action,
            user_correction="fix it",
            target_label="",
            target_map={},
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert len(call_kwargs["metadata"]["agent_action"]) == 500


def test_user_correction_truncation():
    """User correction is truncated to 500 chars in metadata."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    long_correction = "y" * 1000

    with patch.dict(os.environ, {}, clear=True):
        _log_correction_to_langfuse(
            correction_type="incorrect",
            agent_action="did something",
            user_correction=long_correction,
            target_label="",
            target_map={},
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert len(call_kwargs["metadata"]["user_correction"]) == 500


def test_logging_with_no_obs():
    """Returns failure when obs is None and Langfuse not enabled."""
    with patch.dict(os.environ, {}, clear=True):
        success, error = _log_correction_to_langfuse(
            correction_type="incorrect",
            agent_action="test",
            user_correction="test",
            target_label="",
            target_map={},
            obs=None,
            session_id="session-1",
        )

    assert success is False


def test_default_source_is_human():
    """Source defaults to 'human' when not specified."""
    mock_obs = MagicMock()
    mock_obs.langfuse_client = MagicMock()
    mock_obs.get_current_trace_id.return_value = None
    mock_obs.last_trace_id = None

    with patch.dict(os.environ, {}, clear=True):
        _log_correction_to_langfuse(
            correction_type="style",
            agent_action="test action",
            user_correction="test correction",
            target_label="",
            target_map={},
            obs=mock_obs,
            session_id="session-1",
        )

    call_kwargs = mock_obs.langfuse_client.create_score.call_args[1]
    assert call_kwargs["metadata"]["source"] == "human"


# ------------------------------------------------------------------
# Tool creation
# ------------------------------------------------------------------


def test_tool_creation():
    """Tool is created with correct name via decorator."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    with patch.dict(os.environ, {}, clear=True):
        tool = create_correction_mcp_tool(
            obs=MagicMock(),
            session_id="session-1",
            sdk_tool_decorator=mock_decorator,
        )

    assert tool is not None
    mock_decorator.assert_called_once()
    call_args = mock_decorator.call_args[0]
    assert call_args[0] == "log_correction"


def test_tool_description_without_rubric():
    """Without rubric, description is the base description only."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    with patch.dict(os.environ, {}, clear=True):
        create_correction_mcp_tool(
            obs=MagicMock(),
            session_id="session-1",
            sdk_tool_decorator=mock_decorator,
            has_rubric=False,
        )

    description = mock_decorator.call_args[0][1]
    assert CORRECTION_TOOL_DESCRIPTION_BASE in description
    assert "Post-Rubric" not in description


def test_tool_description_with_rubric():
    """With rubric, description includes rubric addendum."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    with patch.dict(os.environ, {}, clear=True):
        create_correction_mcp_tool(
            obs=MagicMock(),
            session_id="session-1",
            sdk_tool_decorator=mock_decorator,
            has_rubric=True,
        )

    description = mock_decorator.call_args[0][1]
    assert CORRECTION_TOOL_DESCRIPTION_BASE in description
    assert RUBRIC_CORRECTION_ADDENDUM in description
    assert "Post-Rubric" in description
    assert "source: 'rubric'" in description


def test_tool_schema_includes_targets_from_env():
    """Tool schema has dynamic target enum from env vars."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    repos_json = json.dumps([
        {"url": "https://github.com/org/app.git", "branch": "main"},
    ])

    with patch.dict(
        os.environ,
        {
            "ACTIVE_WORKFLOW_GIT_URL": "https://github.com/org/wf.git",
            "ACTIVE_WORKFLOW_BRANCH": "main",
            "ACTIVE_WORKFLOW_PATH": "workflows/joker",
            "REPOS_JSON": repos_json,
        },
        clear=True,
    ):
        create_correction_mcp_tool(
            obs=MagicMock(),
            session_id="session-1",
            sdk_tool_decorator=mock_decorator,
        )

    schema = mock_decorator.call_args[0][2]
    target_enum = schema["properties"]["target"]["enum"]
    assert "joker" in target_enum
    assert "app" in target_enum
    assert "target" in schema["required"]


def test_tool_description_lists_available_targets():
    """Tool description includes available target labels."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    with patch.dict(
        os.environ,
        {
            "ACTIVE_WORKFLOW_GIT_URL": "https://github.com/org/wf.git",
            "ACTIVE_WORKFLOW_BRANCH": "main",
            "ACTIVE_WORKFLOW_PATH": "workflows/joker",
        },
        clear=True,
    ):
        create_correction_mcp_tool(
            obs=MagicMock(),
            session_id="session-1",
            sdk_tool_decorator=mock_decorator,
        )

    description = mock_decorator.call_args[0][1]
    assert "Available Targets" in description
    assert "`joker` (workflow)" in description


# ------------------------------------------------------------------
# Runner
# ------------------------------------------------------------------


if __name__ == "__main__":
    print("Testing corrections feedback MCP tool...")
    print("=" * 60)

    tests = [
        ("Schema: correction types", test_base_schema_has_all_correction_types),
        ("Schema: source values", test_base_schema_source_values),
        ("Schema: required fields", test_base_required_fields),
        ("Target map: repo name", test_repo_name_extracts_from_url),
        ("Target map: workflow only", test_build_target_map_workflow_only),
        ("Target map: repos only", test_build_target_map_repos_only),
        ("Target map: workflow + repos", test_build_target_map_workflow_and_repos),
        ("Target map: label collision", test_build_target_map_label_collision),
        ("Target map: empty", test_build_target_map_empty),
        ("Target map: label from path", test_build_target_map_workflow_label_from_path),
        ("Target map: label from repo when no path", test_build_target_map_workflow_label_from_repo_when_no_path),
        ("Schema: multiple targets", test_schema_with_multiple_targets),
        ("Schema: single target", test_schema_with_single_target),
        ("Schema: no targets", test_schema_with_no_targets),
        ("Resolve: by label", test_resolve_target_by_label),
        ("Resolve: auto-fill single", test_resolve_target_auto_fills_single),
        ("Resolve: empty map", test_resolve_target_empty_map),
        ("Context: captures from env", test_captures_context_from_env),
        ("Context: handles missing env", test_handles_missing_env_vars),
        ("Logging: successful", test_successful_logging),
        ("Logging: rubric source", test_rubric_source_logging),
        ("Logging: no trace_id", test_logging_without_trace_id),
        ("Logging: not enabled", test_logging_without_langfuse_enabled),
        ("Logging: no credentials", test_logging_without_credentials),
        ("Logging: agent_action truncation", test_agent_action_truncation),
        ("Logging: user_correction truncation", test_user_correction_truncation),
        ("Logging: no obs", test_logging_with_no_obs),
        ("Logging: default source is human", test_default_source_is_human),
        ("Tool: creation", test_tool_creation),
        ("Tool: description without rubric", test_tool_description_without_rubric),
        ("Tool: description with rubric", test_tool_description_with_rubric),
        ("Tool: schema includes targets from env", test_tool_schema_includes_targets_from_env),
        ("Tool: description lists available targets", test_tool_description_lists_available_targets),
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
