#!/usr/bin/env python3
"""
Tests for the suggest_memory MCP tool.

Validates:
1. Tool creation and schema structure
2. Input validation (empty content, invalid type, CL-disabled)
3. File path generation (slugification, date-prefixed)
4. Markdown frontmatter generation
5. Git operations (branch creation, commit, push)
6. GitHub PR creation
7. Error handling (git failures are non-fatal)
"""

import os
import re
import sys
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

sys.path.insert(0, str(Path(__file__).parent.parent))

from ambient_runner.bridges.claude.memory import (
    MEMORY_TYPES,
    SUGGEST_MEMORY_TOOL_DESCRIPTION,
    _generate_file_path,
    _generate_frontmatter,
    _slugify,
    _check_cl_enabled,
    create_suggest_memory_tool,
)


# ------------------------------------------------------------------
# Slugification
# ------------------------------------------------------------------


def test_slugify_simple():
    assert _slugify("Fix auth token handling") == "fix-auth-token-handling"


def test_slugify_special_chars():
    assert _slugify("Use try/except (not if/else)!") == "use-tryexcept-not-ifelse"


def test_slugify_long_string():
    result = _slugify("a" * 200)
    assert len(result) <= 60


def test_slugify_empty():
    assert _slugify("") == "untitled"


def test_slugify_only_special():
    assert _slugify("!!!???") == "untitled"


# ------------------------------------------------------------------
# File path generation
# ------------------------------------------------------------------


def test_generate_file_path_correction():
    path = _generate_file_path("correction", "Fix auth token handling")
    assert path.startswith("docs/learned/corrections/")
    assert path.endswith(".md")
    assert "fix-auth-token-handling" in path
    # Date prefix: YYYY-MM-DD
    filename = path.split("/")[-1]
    assert re.match(r"\d{4}-\d{2}-\d{2}-", filename)


def test_generate_file_path_pattern():
    path = _generate_file_path("pattern", "Always use uv instead of pip")
    assert path.startswith("docs/learned/patterns/")
    assert "always-use-uv-instead-of-pip" in path


# ------------------------------------------------------------------
# Frontmatter generation
# ------------------------------------------------------------------


def test_generate_frontmatter_correction():
    fm = _generate_frontmatter(
        title="Fix auth token handling",
        memory_type="correction",
        session_name="session-123",
    )
    assert "title: Fix auth token handling" in fm
    assert "type: correction" in fm
    assert "source: agent" in fm
    assert "session: session-123" in fm
    assert "---" in fm


def test_generate_frontmatter_pattern():
    fm = _generate_frontmatter(
        title="Use uv for Python packages",
        memory_type="pattern",
        session_name="session-456",
    )
    assert "type: pattern" in fm
    assert "source: agent" in fm


# ------------------------------------------------------------------
# CL enabled check
# ------------------------------------------------------------------


def test_cl_enabled_when_config_exists(tmp_path):
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text('{"learning": {"enabled": true}}')
    assert _check_cl_enabled(str(tmp_path)) is True


def test_cl_disabled_when_config_says_false(tmp_path):
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text('{"learning": {"enabled": false}}')
    assert _check_cl_enabled(str(tmp_path)) is False


def test_cl_disabled_when_no_config(tmp_path):
    assert _check_cl_enabled(str(tmp_path)) is False


def test_cl_disabled_when_no_learning_key(tmp_path):
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text('{"other": "value"}')
    assert _check_cl_enabled(str(tmp_path)) is False


def test_cl_disabled_when_invalid_json(tmp_path):
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text("not json")
    assert _check_cl_enabled(str(tmp_path)) is False


# ------------------------------------------------------------------
# Tool creation
# ------------------------------------------------------------------


def test_tool_creation():
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    with patch.dict(os.environ, {}, clear=True):
        tool = create_suggest_memory_tool(
            sdk_tool_decorator=mock_decorator,
            cwd_path="/workspace/repos/my-app",
            session_name="session-123",
        )

    assert tool is not None
    mock_decorator.assert_called_once()
    call_args = mock_decorator.call_args[0]
    assert call_args[0] == "suggest_memory"


def test_tool_schema_has_required_fields():
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    with patch.dict(os.environ, {}, clear=True):
        create_suggest_memory_tool(
            sdk_tool_decorator=mock_decorator,
            cwd_path="/workspace/repos/my-app",
            session_name="session-123",
        )

    schema = mock_decorator.call_args[0][2]
    assert "content" in schema["properties"]
    assert "type" in schema["properties"]
    assert "title" in schema["properties"]
    assert schema["properties"]["type"]["enum"] == MEMORY_TYPES
    assert set(schema["required"]) == {"content", "type", "title"}


def test_tool_description_is_informative():
    assert "suggest" in SUGGEST_MEMORY_TOOL_DESCRIPTION.lower()
    assert "correction" in SUGGEST_MEMORY_TOOL_DESCRIPTION.lower()
    assert "pattern" in SUGGEST_MEMORY_TOOL_DESCRIPTION.lower()


# ------------------------------------------------------------------
# Tool invocation -- validation
# ------------------------------------------------------------------


@pytest.mark.asyncio
async def test_tool_rejects_invalid_type():
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path="/workspace/repos/my-app",
        session_name="session-123",
    )

    result = await tool({"content": "some content", "type": "invalid", "title": "Test"})
    assert result["isError"] is True
    assert "correction" in result["content"][0]["text"]
    assert "pattern" in result["content"][0]["text"]


@pytest.mark.asyncio
async def test_tool_rejects_empty_content():
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path="/workspace/repos/my-app",
        session_name="session-123",
    )

    result = await tool({"content": "", "type": "correction", "title": "Test"})
    assert result["isError"] is True
    assert "empty" in result["content"][0]["text"].lower()


@pytest.mark.asyncio
async def test_tool_rejects_empty_title():
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path="/workspace/repos/my-app",
        session_name="session-123",
    )

    result = await tool({"content": "some content", "type": "correction", "title": ""})
    assert result["isError"] is True
    assert "title" in result["content"][0]["text"].lower()


@pytest.mark.asyncio
async def test_tool_rejects_when_cl_disabled(tmp_path):
    """When .ambient/config.json has learning.enabled=false, tool returns error."""
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text('{"learning": {"enabled": false}}')

    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path=str(tmp_path),
        session_name="session-123",
    )

    result = await tool(
        {"content": "some content", "type": "correction", "title": "Test"}
    )
    assert result["isError"] is True
    assert "not enabled" in result["content"][0]["text"].lower()


@pytest.mark.asyncio
async def test_tool_rejects_when_no_config(tmp_path):
    """When .ambient/config.json does not exist, tool returns error."""
    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path=str(tmp_path),
        session_name="session-123",
    )

    result = await tool(
        {"content": "some content", "type": "correction", "title": "Test"}
    )
    assert result["isError"] is True
    assert "not enabled" in result["content"][0]["text"].lower()


# ------------------------------------------------------------------
# Tool invocation -- git operations
# ------------------------------------------------------------------


@pytest.mark.asyncio
async def test_tool_success_creates_branch_and_pr(tmp_path):
    """Full happy path: CL enabled, git operations succeed, PR created."""
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text('{"learning": {"enabled": true}}')

    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path=str(tmp_path),
        session_name="session-123",
    )

    with (
        patch(
            "ambient_runner.bridges.claude.memory._run_git_command",
            new_callable=AsyncMock,
            return_value=(True, "main"),
        ) as mock_git,
        patch(
            "ambient_runner.bridges.claude.memory._create_github_pr",
            new_callable=AsyncMock,
            return_value=(True, "https://github.com/org/repo/pull/42"),
        ),
    ):
        result = await tool(
            {
                "content": "Always use uv instead of pip",
                "type": "pattern",
                "title": "Use uv for Python packages",
            }
        )

    assert "isError" not in result or result.get("isError") is not True
    text = result["content"][0]["text"]
    assert "pull/42" in text or "draft PR" in text.lower()

    # Verify git was called to create branch, add, commit, push
    git_calls = mock_git.call_args_list
    # Should have: rev-parse, checkout -b, add, commit, push, checkout back
    assert len(git_calls) >= 5

    # Verify the file was written to disk
    learned_dir = tmp_path / "docs" / "learned" / "patterns"
    assert learned_dir.exists()
    md_files = list(learned_dir.glob("*.md"))
    assert len(md_files) == 1
    content = md_files[0].read_text()
    assert "title: Use uv for Python packages" in content
    assert "type: pattern" in content
    assert "Always use uv instead of pip" in content


@pytest.mark.asyncio
async def test_tool_handles_git_failure_gracefully(tmp_path):
    """Git failure returns error but does not crash."""
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text('{"learning": {"enabled": true}}')

    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path=str(tmp_path),
        session_name="session-123",
    )

    # First call (rev-parse) succeeds, second call (checkout -b) fails
    call_count = 0

    async def mock_git(args, cwd):
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            return (True, "main")
        return (False, "fatal: not a git repository")

    with patch(
        "ambient_runner.bridges.claude.memory._run_git_command",
        side_effect=mock_git,
    ):
        result = await tool(
            {
                "content": "Some content",
                "type": "correction",
                "title": "Fix something",
            }
        )

    assert result["isError"] is True
    text = result["content"][0]["text"].lower()
    assert "failed" in text or "git" in text


@pytest.mark.asyncio
async def test_tool_handles_pr_failure_after_push(tmp_path):
    """Branch pushed but PR creation fails -- returns warning, not error."""
    config_dir = tmp_path / ".ambient"
    config_dir.mkdir()
    config_file = config_dir / "config.json"
    config_file.write_text('{"learning": {"enabled": true}}')

    mock_decorator = MagicMock()
    mock_decorator.return_value = lambda fn: fn

    tool = create_suggest_memory_tool(
        sdk_tool_decorator=mock_decorator,
        cwd_path=str(tmp_path),
        session_name="session-123",
    )

    with (
        patch(
            "ambient_runner.bridges.claude.memory._run_git_command",
            new_callable=AsyncMock,
            return_value=(True, "main"),
        ),
        patch(
            "ambient_runner.bridges.claude.memory._create_github_pr",
            new_callable=AsyncMock,
            return_value=(False, "gh: command not found"),
        ),
    ):
        result = await tool(
            {
                "content": "Some content",
                "type": "pattern",
                "title": "Test pattern",
            }
        )

    # Should NOT be an error (branch was pushed)
    assert result.get("isError") is not True
    text = result["content"][0]["text"]
    assert "pushed successfully" in text
    assert "PR creation failed" in text


# ------------------------------------------------------------------
# Constants
# ------------------------------------------------------------------


def test_memory_types():
    assert "correction" in MEMORY_TYPES
    assert "pattern" in MEMORY_TYPES
    assert len(MEMORY_TYPES) == 2
