#!/usr/bin/env python3
"""
Tests for the feedback loop query/aggregation script.

Validates:
1. Score grouping by target (type, repo_url, branch, path)
2. Backward compatibility with old workflow-based schema
3. Prompt generation content and structure
4. Session creation API calls
5. Dry run mode
"""

import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from unittest.mock import MagicMock, patch

sys.path.insert(0, str(Path(__file__).parent))

from query_corrections import (
    _extract_target_fields,
    _repo_short_name,
    build_improvement_prompt,
    create_improvement_session,
    group_corrections,
)


# ------------------------------------------------------------------
# Sample data helpers
# ------------------------------------------------------------------


def _make_score(
    correction_type="incorrect",
    agent_action="Used wrong approach",
    user_correction="Should have done it this way",
    target_type="workflow",
    target_repo_url="https://github.com/org/workflows",
    target_branch="main",
    target_path="workflows/bug-fix",
    session_name="session-1",
    trace_id="trace-abc",
    source="human",
):
    """Helper to create a test score with the new target-based schema."""
    metadata = {
        "correction_type": correction_type,
        "source": source,
        "agent_action": agent_action,
        "user_correction": user_correction,
        "target_type": target_type,
        "target_repo_url": target_repo_url,
        "target_branch": target_branch,
        "target_path": target_path,
        "session_name": session_name,
    }
    return {
        "value": correction_type,
        "comment": f"Agent did: {agent_action}\nUser corrected to: {user_correction}",
        "traceId": trace_id,
        "metadata": metadata,
        "createdAt": "2026-02-15T10:00:00Z",
    }


def _make_old_schema_score(
    correction_type="incorrect",
    agent_action="Used wrong approach",
    user_correction="Should have done it this way",
    workflow_repo_url="https://github.com/org/workflows",
    workflow_branch="main",
    workflow_path="workflows/bug-fix",
    repos=None,
    session_name="session-1",
    trace_id="trace-abc",
):
    """Helper to create a score with the OLD workflow-based schema."""
    if repos is None:
        repos = [{"url": "https://github.com/org/repo", "branch": "main"}]
    metadata = {
        "correction_type": correction_type,
        "agent_action": agent_action,
        "user_correction": user_correction,
        "workflow_repo_url": workflow_repo_url,
        "workflow_branch": workflow_branch,
        "workflow_path": workflow_path,
        "repos": json.dumps(repos),
        "session_name": session_name,
    }
    return {
        "value": correction_type,
        "comment": f"Agent did: {agent_action}\nUser corrected to: {user_correction}",
        "traceId": trace_id,
        "metadata": metadata,
        "createdAt": "2026-02-15T10:00:00Z",
    }


# ------------------------------------------------------------------
# _extract_target_fields tests
# ------------------------------------------------------------------


def test_extract_new_schema():
    """New schema target_* fields are extracted directly."""
    meta = {
        "target_type": "repo",
        "target_repo_url": "https://github.com/org/app",
        "target_branch": "dev",
        "target_path": "",
    }
    result = _extract_target_fields(meta)
    assert result == ("repo", "https://github.com/org/app", "dev", "")


def test_extract_old_schema_workflow():
    """Old schema workflow_* fields are migrated to target_* with type=workflow."""
    meta = {
        "workflow_repo_url": "https://github.com/org/wf",
        "workflow_branch": "main",
        "workflow_path": "workflows/test",
    }
    result = _extract_target_fields(meta)
    assert result == ("workflow", "https://github.com/org/wf", "main", "workflows/test")


def test_extract_old_schema_no_workflow():
    """Old schema with empty workflow fields defaults to type=repo."""
    meta = {"workflow_repo_url": "", "workflow_branch": "", "workflow_path": ""}
    result = _extract_target_fields(meta)
    assert result[0] == "repo"


def test_extract_empty_metadata():
    """Empty metadata defaults to type=repo with empty fields."""
    result = _extract_target_fields({})
    assert result == ("repo", "", "", "")


# ------------------------------------------------------------------
# Grouping tests
# ------------------------------------------------------------------


def test_groups_by_target():
    """Scores grouped into (target_type, target_repo_url, branch, path) buckets."""
    scores = [
        _make_score(target_type="workflow", target_path="workflows/wf-1"),
        _make_score(target_type="workflow", target_path="workflows/wf-1"),
        _make_score(target_type="repo", target_repo_url="https://github.com/org/app", target_path=""),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 2

    wf_group = next(g for g in groups if g["target_type"] == "workflow")
    repo_group = next(g for g in groups if g["target_type"] == "repo")

    assert wf_group["total_count"] == 2
    assert repo_group["total_count"] == 1


def test_separate_groups_for_workflow_and_repo():
    """Same repo URL produces separate groups for workflow vs repo target types."""
    scores = [
        _make_score(
            target_type="workflow",
            target_repo_url="https://github.com/org/wf",
            target_path="workflows/test",
        ),
        _make_score(
            target_type="repo",
            target_repo_url="https://github.com/org/wf",
            target_path="",
        ),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 2


def test_counts_correction_types():
    """Correction type counts are accurate."""
    scores = [
        _make_score(correction_type="incomplete"),
        _make_score(correction_type="incomplete"),
        _make_score(correction_type="incorrect"),
    ]
    groups = group_corrections(scores)
    counts = groups[0]["correction_type_counts"]
    assert counts["incomplete"] == 2
    assert counts["incorrect"] == 1


def test_repo_branches_grouped_together():
    """Repo corrections from different branches are grouped into one."""
    scores = [
        _make_score(
            target_type="repo",
            target_repo_url="https://github.com/org/app",
            target_branch="feat/login",
            target_path="",
        ),
        _make_score(
            target_type="repo",
            target_repo_url="https://github.com/org/app",
            target_branch="fix/bug-123",
            target_path="",
        ),
        _make_score(
            target_type="repo",
            target_repo_url="https://github.com/org/app",
            target_branch="main",
            target_path="",
        ),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 1
    assert groups[0]["total_count"] == 3


def test_workflow_branches_stay_separate():
    """Workflow corrections on different branches remain separate groups."""
    scores = [
        _make_score(
            target_type="workflow",
            target_repo_url="https://github.com/org/wf",
            target_branch="main",
            target_path="workflows/test",
        ),
        _make_score(
            target_type="workflow",
            target_repo_url="https://github.com/org/wf",
            target_branch="feat/new-workflow",
            target_path="workflows/test",
        ),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 2


def test_handles_missing_metadata():
    """Scores with missing metadata fields are handled gracefully."""
    scores = [
        {"value": "incorrect", "comment": "test", "metadata": None},
        {"value": "incomplete", "comment": "test2", "metadata": {}},
    ]
    groups = group_corrections(scores)
    assert len(groups) == 1
    assert groups[0]["target_type"] == "repo"
    assert groups[0]["target_repo_url"] == ""
    assert groups[0]["total_count"] == 2


def test_sorted_by_count_descending():
    """Groups sorted by total_count descending."""
    scores = [
        _make_score(target_path="workflows/small"),
        _make_score(target_path="workflows/big"),
        _make_score(target_path="workflows/big"),
        _make_score(target_path="workflows/big"),
    ]
    groups = group_corrections(scores)
    assert groups[0]["target_path"] == "workflows/big"
    assert groups[0]["total_count"] == 3


def test_backward_compat_old_scores():
    """Old-schema scores (workflow_repo_url) are migrated and grouped correctly."""
    scores = [
        _make_old_schema_score(workflow_path="workflows/wf-1"),
        _make_old_schema_score(workflow_path="workflows/wf-1"),
        _make_old_schema_score(workflow_path="workflows/wf-2"),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 2

    wf1 = next(g for g in groups if g["target_path"] == "workflows/wf-1")
    assert wf1["target_type"] == "workflow"
    assert wf1["total_count"] == 2


def test_extracts_agent_action_and_user_correction():
    """agent_action and user_correction are extracted from metadata."""
    scores = [
        _make_score(
            agent_action="I modified the wrong file",
            user_correction="Should have edited config.py not main.py",
        )
    ]
    groups = group_corrections(scores)
    correction = groups[0]["corrections"][0]
    assert correction["agent_action"] == "I modified the wrong file"
    assert correction["user_correction"] == "Should have edited config.py not main.py"


# ------------------------------------------------------------------
# Prompt generation tests
# ------------------------------------------------------------------


def test_prompt_workflow_target():
    """Workflow prompt includes workflow-specific info and instructions."""
    group = {
        "target_type": "workflow",
        "target_repo_url": "https://github.com/org/workflows",
        "target_branch": "main",
        "target_path": "workflows/bug-fix",
        "corrections": [],
        "total_count": 2,
        "correction_type_counts": {"incomplete": 2},
        "source_counts": {"human": 2},
    }
    prompt = build_improvement_prompt(group)
    assert "workflow" in prompt.lower()
    assert "workflows/bug-fix" in prompt
    assert "https://github.com/org/workflows" in prompt
    assert "workflow files" in prompt.lower()


def test_prompt_repo_target():
    """Repo prompt includes repo-specific info and instructions."""
    group = {
        "target_type": "repo",
        "target_repo_url": "https://github.com/org/my-app",
        "target_branch": "main",
        "target_path": "",
        "corrections": [],
        "total_count": 3,
        "correction_type_counts": {"incorrect": 3},
        "source_counts": {"human": 3},
    }
    prompt = build_improvement_prompt(group)
    assert "repository" in prompt.lower()
    assert "https://github.com/org/my-app" in prompt
    assert "CLAUDE.md" in prompt


def test_prompt_includes_all_corrections():
    """Prompt includes agent_action and user_correction for each correction."""
    group = {
        "target_type": "workflow",
        "target_repo_url": "https://github.com/org/workflows",
        "target_branch": "main",
        "target_path": "workflows/test",
        "corrections": [
            {
                "correction_type": "incorrect",
                "source": "human",
                "agent_action": "Used wrong pattern",
                "user_correction": "Should use factory pattern",
                "session_name": "session-1",
                "trace_id": "trace-1",
            },
            {
                "correction_type": "incomplete",
                "source": "human",
                "agent_action": "Forgot to update tests",
                "user_correction": "Always update tests when changing logic",
                "session_name": "session-2",
                "trace_id": "trace-2",
            },
        ],
        "total_count": 2,
        "correction_type_counts": {"incorrect": 1, "incomplete": 1},
        "source_counts": {"human": 2},
    }

    prompt = build_improvement_prompt(group)

    assert "Used wrong pattern" in prompt
    assert "Should use factory pattern" in prompt
    assert "Forgot to update tests" in prompt
    assert "Always update tests when changing logic" in prompt


def test_prompt_identifies_top_correction_type():
    """Prompt highlights the most common correction type."""
    group = {
        "target_type": "workflow",
        "target_repo_url": "https://github.com/org/workflows",
        "target_branch": "main",
        "target_path": "wf",
        "corrections": [],
        "total_count": 5,
        "correction_type_counts": {"incomplete": 3, "incorrect": 2},
        "source_counts": {"human": 5},
    }

    prompt = build_improvement_prompt(group)
    assert "incomplete" in prompt
    assert "3 occurrences" in prompt


# ------------------------------------------------------------------
# Session creation tests
# ------------------------------------------------------------------


def test_repo_short_name():
    """_repo_short_name extracts name from URL."""
    assert _repo_short_name("https://github.com/org/my-repo.git") == "my-repo"
    assert _repo_short_name("https://github.com/org/my-repo") == "my-repo"


@patch("query_corrections.requests.post")
def test_sends_correct_api_request(mock_post):
    """POST request has correct structure for target-based groups."""
    mock_resp = MagicMock()
    mock_resp.json.return_value = {"name": "session-123", "uid": "uid-456"}
    mock_resp.raise_for_status = MagicMock()
    mock_post.return_value = mock_resp

    group = {
        "target_type": "workflow",
        "target_repo_url": "https://github.com/org/workflows",
        "target_branch": "main",
        "target_path": "workflows/test",
        "total_count": 3,
    }

    result = create_improvement_session(
        api_url="https://ambient.example.com/api",
        api_token="bot-token-123",
        project="test-project",
        prompt="Test prompt content",
        group=group,
    )

    assert result is not None
    assert result["name"] == "session-123"

    mock_post.assert_called_once()
    _, call_kwargs = mock_post.call_args

    url = call_kwargs.get("url", mock_post.call_args[0][0] if mock_post.call_args[0] else "")
    assert "test-project" in url

    headers = call_kwargs.get("headers", {})
    assert headers["Authorization"] == "Bearer bot-token-123"

    body = call_kwargs["json"]
    assert body["initialPrompt"] == "Test prompt content"
    assert body["labels"]["feedback-loop"] == "true"
    assert body["labels"]["target-type"] == "workflow"

    assert body["repos"][0]["url"] == "https://github.com/org/workflows"
    assert body["repos"][0]["branch"] == "main"
    assert body["repos"][0]["autoPush"] is True


@patch("query_corrections.requests.post")
def test_repo_target_session(mock_post):
    """Repo-targeted session has correct display name and labels."""
    mock_resp = MagicMock()
    mock_resp.json.return_value = {"name": "session-456"}
    mock_resp.raise_for_status = MagicMock()
    mock_post.return_value = mock_resp

    group = {
        "target_type": "repo",
        "target_repo_url": "https://github.com/org/my-app.git",
        "target_branch": "dev",
        "target_path": "",
        "total_count": 2,
    }

    create_improvement_session(
        api_url="https://api.example.com",
        api_token="token",
        project="proj",
        prompt="prompt",
        group=group,
    )

    body = mock_post.call_args[1]["json"]
    assert "repo" in body["displayName"].lower()
    assert body["labels"]["target-type"] == "repo"
    assert body["repos"][0]["url"] == "https://github.com/org/my-app.git"


@patch("query_corrections.requests.post")
def test_handles_api_errors(mock_post):
    """API errors are logged and do not crash."""
    import requests as _requests
    mock_post.side_effect = _requests.RequestException("Connection refused")

    group = {
        "target_type": "workflow",
        "target_repo_url": "https://github.com/org/workflows",
        "target_branch": "main",
        "target_path": "wf",
        "total_count": 2,
    }

    result = create_improvement_session(
        api_url="https://ambient.example.com/api",
        api_token="token",
        project="proj",
        prompt="prompt",
        group=group,
    )

    assert result is None


def test_no_repos_when_url_invalid():
    """repos field omitted when target_repo_url is not a valid HTTP URL."""
    with patch("query_corrections.requests.post") as mock_post:
        mock_resp = MagicMock()
        mock_resp.json.return_value = {"name": "session-1"}
        mock_resp.raise_for_status = MagicMock()
        mock_post.return_value = mock_resp

        group = {
            "target_type": "repo",
            "target_repo_url": "",
            "target_branch": "",
            "target_path": "",
            "total_count": 2,
        }

        create_improvement_session(
            api_url="https://api.example.com",
            api_token="token",
            project="proj",
            prompt="prompt",
            group=group,
        )

        body = mock_post.call_args[1]["json"]
        assert "repos" not in body


# ------------------------------------------------------------------
# Runner
# ------------------------------------------------------------------


if __name__ == "__main__":
    print("Testing feedback loop query script...")
    print("=" * 60)

    tests = [
        ("Extract: new schema", test_extract_new_schema),
        ("Extract: old schema workflow", test_extract_old_schema_workflow),
        ("Extract: old schema no workflow", test_extract_old_schema_no_workflow),
        ("Extract: empty metadata", test_extract_empty_metadata),
        ("Grouping: by target", test_groups_by_target),
        ("Grouping: separate workflow vs repo", test_separate_groups_for_workflow_and_repo),
        ("Grouping: correction type counts", test_counts_correction_types),
        ("Grouping: repo branches grouped", test_repo_branches_grouped_together),
        ("Grouping: workflow branches separate", test_workflow_branches_stay_separate),
        ("Grouping: missing metadata", test_handles_missing_metadata),
        ("Grouping: sorted descending", test_sorted_by_count_descending),
        ("Grouping: backward compat old scores", test_backward_compat_old_scores),
        ("Grouping: agent_action and user_correction", test_extracts_agent_action_and_user_correction),
        ("Prompt: workflow target", test_prompt_workflow_target),
        ("Prompt: repo target", test_prompt_repo_target),
        ("Prompt: includes all corrections", test_prompt_includes_all_corrections),
        ("Prompt: top correction type", test_prompt_identifies_top_correction_type),
        ("Session: repo short name", test_repo_short_name),
        ("Session: correct API request", test_sends_correct_api_request),
        ("Session: repo target session", test_repo_target_session),
        ("Session: handles API errors", test_handles_api_errors),
        ("Session: no repos for invalid URL", test_no_repos_when_url_invalid),
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
