#!/usr/bin/env python3
"""
Tests for the feedback loop query/aggregation script.

Validates:
1. Score grouping by workflow (repo_url, branch, path)
2. Prompt generation content and structure
3. Session creation API calls
4. Dry run mode
"""

import json
import sys
from datetime import datetime, timezone
from pathlib import Path
from unittest.mock import MagicMock, patch

sys.path.insert(0, str(Path(__file__).parent))

from query_corrections import (
    build_improvement_prompt,
    create_improvement_session,
    group_corrections,
)


# ------------------------------------------------------------------
# Sample data
# ------------------------------------------------------------------


def _make_score(
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
    """Helper to create a test score object matching the new schema."""
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
        "value": correction_type,  # CATEGORICAL: value is the type string
        "comment": f"Agent did: {agent_action}\nUser corrected to: {user_correction}",
        "traceId": trace_id,
        "metadata": metadata,
        "createdAt": "2026-02-15T10:00:00Z",
    }


# ------------------------------------------------------------------
# Grouping tests
# ------------------------------------------------------------------


def test_groups_by_workflow():
    """Scores grouped into (workflow_repo_url, workflow_branch, workflow_path) buckets."""
    scores = [
        _make_score(workflow_path="workflows/wf-1"),
        _make_score(workflow_path="workflows/wf-1"),
        _make_score(workflow_path="workflows/wf-2"),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 2

    group_1 = next(g for g in groups if g["workflow_path"] == "workflows/wf-1")
    group_2 = next(g for g in groups if g["workflow_path"] == "workflows/wf-2")

    assert group_1["total_count"] == 2
    assert group_2["total_count"] == 1


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


def test_deduplicates_repos():
    """Same repo URL appearing in multiple corrections is deduplicated."""
    repo = [{"url": "https://github.com/org/repo", "branch": "main"}]
    scores = [
        _make_score(repos=repo),
        _make_score(repos=repo),
        _make_score(repos=repo),
    ]
    groups = group_corrections(scores)
    assert len(groups[0]["repos"]) == 1


def test_collects_repos_across_corrections():
    """Repos from different corrections in same group are collected."""
    scores = [
        _make_score(repos=[{"url": "https://github.com/org/repo-a", "branch": "main"}]),
        _make_score(repos=[{"url": "https://github.com/org/repo-b", "branch": "main"}]),
    ]
    groups = group_corrections(scores)
    repo_urls = {r["url"] for r in groups[0]["repos"]}
    assert "https://github.com/org/repo-a" in repo_urls
    assert "https://github.com/org/repo-b" in repo_urls


def test_handles_missing_metadata():
    """Scores with missing metadata fields are handled gracefully."""
    scores = [
        {"value": "incorrect", "comment": "test", "metadata": None},
        {"value": "incomplete", "comment": "test2", "metadata": {}},
    ]
    groups = group_corrections(scores)
    assert len(groups) == 1
    assert groups[0]["workflow_repo_url"] == ""
    assert groups[0]["workflow_path"] == ""
    assert groups[0]["total_count"] == 2


def test_sorted_by_count_descending():
    """Groups sorted by total_count descending."""
    scores = [
        _make_score(workflow_path="workflows/small"),
        _make_score(workflow_path="workflows/big"),
        _make_score(workflow_path="workflows/big"),
        _make_score(workflow_path="workflows/big"),
    ]
    groups = group_corrections(scores)
    assert groups[0]["workflow_path"] == "workflows/big"
    assert groups[0]["total_count"] == 3


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


def test_correction_type_from_score_value():
    """correction_type is taken from the score's value field (CATEGORICAL)."""
    scores = [{"value": "out_of_scope", "metadata": {}, "traceId": "t1"}]
    groups = group_corrections(scores)
    correction = groups[0]["corrections"][0]
    assert correction["correction_type"] == "out_of_scope"


# ------------------------------------------------------------------
# Prompt generation tests
# ------------------------------------------------------------------


def test_prompt_includes_workflow_info():
    """Prompt includes workflow repo, branch, and path."""
    group = {
        "workflow_repo_url": "https://github.com/org/workflows",
        "workflow_branch": "main",
        "workflow_path": "workflows/bug-fix",
        "repos": [],
        "corrections": [],
        "total_count": 2,
        "correction_type_counts": {"incomplete": 2},
    }
    prompt = build_improvement_prompt(group)
    assert "https://github.com/org/workflows" in prompt
    assert "workflows/bug-fix" in prompt
    assert "main" in prompt


def test_prompt_includes_all_corrections():
    """Prompt includes agent_action and user_correction for each correction."""
    group = {
        "workflow_repo_url": "https://github.com/org/workflows",
        "workflow_branch": "main",
        "workflow_path": "workflows/test",
        "repos": [],
        "corrections": [
            {
                "correction_type": "incorrect",
                "agent_action": "Used wrong pattern",
                "user_correction": "Should use factory pattern",
                "session_name": "session-1",
                "trace_id": "trace-1",
            },
            {
                "correction_type": "incomplete",
                "agent_action": "Forgot to update tests",
                "user_correction": "Always update tests when changing logic",
                "session_name": "session-2",
                "trace_id": "trace-2",
            },
        ],
        "total_count": 2,
        "correction_type_counts": {"incorrect": 1, "incomplete": 1},
    }

    prompt = build_improvement_prompt(group)

    assert "Used wrong pattern" in prompt
    assert "Should use factory pattern" in prompt
    assert "Forgot to update tests" in prompt
    assert "Always update tests when changing logic" in prompt


def test_prompt_identifies_top_correction_type():
    """Prompt highlights the most common correction type."""
    group = {
        "workflow_repo_url": "https://github.com/org/workflows",
        "workflow_branch": "main",
        "workflow_path": "wf",
        "repos": [],
        "corrections": [],
        "total_count": 5,
        "correction_type_counts": {"incomplete": 3, "incorrect": 2},
    }

    prompt = build_improvement_prompt(group)
    assert "incomplete" in prompt
    assert "3 occurrences" in prompt


def test_prompt_includes_repos_section():
    """Prompt includes target repos when present."""
    group = {
        "workflow_repo_url": "https://github.com/org/workflows",
        "workflow_branch": "main",
        "workflow_path": "wf",
        "repos": [
            {"url": "https://github.com/org/my-repo", "branch": "main"},
        ],
        "corrections": [],
        "total_count": 2,
        "correction_type_counts": {"incorrect": 2},
    }

    prompt = build_improvement_prompt(group)
    assert "https://github.com/org/my-repo" in prompt
    assert "Target Repositories" in prompt


def test_prompt_no_repos_section_when_empty():
    """Prompt omits Target Repositories section when repos list is empty."""
    group = {
        "workflow_repo_url": "https://github.com/org/workflows",
        "workflow_branch": "main",
        "workflow_path": "wf",
        "repos": [],
        "corrections": [],
        "total_count": 2,
        "correction_type_counts": {"incorrect": 2},
    }

    prompt = build_improvement_prompt(group)
    assert "Target Repositories" not in prompt


# ------------------------------------------------------------------
# Session creation tests
# ------------------------------------------------------------------


@patch("query_corrections.requests.post")
def test_sends_correct_api_request(mock_post):
    """POST request has correct structure."""
    mock_resp = MagicMock()
    mock_resp.json.return_value = {"name": "session-123", "uid": "uid-456"}
    mock_resp.raise_for_status = MagicMock()
    mock_post.return_value = mock_resp

    group = {
        "workflow_repo_url": "https://github.com/org/workflows",
        "workflow_branch": "main",
        "workflow_path": "workflows/test",
        "repos": [{"url": "https://github.com/org/my-repo", "branch": "main"}],
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

    call_args, call_kwargs = mock_post.call_args

    url = call_args[0] if call_args else call_kwargs.get("url", "")
    assert "test-project" in url

    headers = call_kwargs.get("headers", {})
    assert headers["Authorization"] == "Bearer bot-token-123"

    body = call_kwargs["json"]
    assert body["initialPrompt"] == "Test prompt content"
    assert body["environmentVariables"]["LANGFUSE_MASK_MESSAGES"] == "false"
    assert body["labels"]["feedback-loop"] == "true"

    # Workflow repo is first, target repo is second
    assert body["repos"][0]["url"] == "https://github.com/org/workflows"
    assert body["repos"][0]["branch"] == "main"
    assert body["repos"][1]["url"] == "https://github.com/org/my-repo"


@patch("query_corrections.requests.post")
def test_handles_api_errors(mock_post):
    """API errors are logged and do not crash."""
    import requests as _requests
    mock_post.side_effect = _requests.RequestException("Connection refused")

    group = {
        "workflow_repo_url": "https://github.com/org/workflows",
        "workflow_branch": "main",
        "workflow_path": "wf",
        "repos": [],
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


def test_no_repos_when_workflow_url_invalid():
    """repos field omitted when workflow_repo_url is not a valid HTTP URL."""
    with patch("query_corrections.requests.post") as mock_post:
        mock_resp = MagicMock()
        mock_resp.json.return_value = {"name": "session-1"}
        mock_resp.raise_for_status = MagicMock()
        mock_post.return_value = mock_resp

        group = {
            "workflow_repo_url": "",
            "workflow_branch": "",
            "workflow_path": "wf",
            "repos": [],
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


def test_workflow_branch_included_in_repo():
    """workflow_branch is passed on the workflow repo entry."""
    with patch("query_corrections.requests.post") as mock_post:
        mock_resp = MagicMock()
        mock_resp.json.return_value = {"name": "session-1"}
        mock_resp.raise_for_status = MagicMock()
        mock_post.return_value = mock_resp

        group = {
            "workflow_repo_url": "https://github.com/org/workflows",
            "workflow_branch": "feat/my-branch",
            "workflow_path": "wf",
            "repos": [],
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
        assert body["repos"][0]["branch"] == "feat/my-branch"


# ------------------------------------------------------------------
# Runner
# ------------------------------------------------------------------


if __name__ == "__main__":
    print("Testing feedback loop query script...")
    print("=" * 60)

    tests = [
        ("Grouping: by workflow", test_groups_by_workflow),
        ("Grouping: correction type counts", test_counts_correction_types),
        ("Grouping: deduplicates repos", test_deduplicates_repos),
        ("Grouping: collects repos across corrections", test_collects_repos_across_corrections),
        ("Grouping: missing metadata", test_handles_missing_metadata),
        ("Grouping: sorted descending", test_sorted_by_count_descending),
        ("Grouping: agent_action and user_correction extracted", test_extracts_agent_action_and_user_correction),
        ("Grouping: correction_type from score value", test_correction_type_from_score_value),
        ("Prompt: includes workflow info", test_prompt_includes_workflow_info),
        ("Prompt: includes all corrections", test_prompt_includes_all_corrections),
        ("Prompt: top correction type", test_prompt_identifies_top_correction_type),
        ("Prompt: repos section present", test_prompt_includes_repos_section),
        ("Prompt: no repos section when empty", test_prompt_no_repos_section_when_empty),
        ("Session: correct API request", test_sends_correct_api_request),
        ("Session: handles API errors", test_handles_api_errors),
        ("Session: no repos for invalid URL", test_no_repos_when_workflow_url_invalid),
        ("Session: workflow branch in repo", test_workflow_branch_included_in_repo),
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
