#!/usr/bin/env python3
"""
Tests for the feedback loop query/aggregation script.

Validates:
1. Score grouping by repo_url and workflow
2. Prompt generation content and structure
3. Session creation API calls
4. Dry run mode
"""

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
    category="wrong_approach",
    severity=2,
    scope="repo_context",
    description="Test correction",
    repo_url="https://github.com/org/repo",
    workflow="workflows/bug-fix",
    session_name="session-1",
    trace_id="trace-abc",
    correction_details=None,
):
    """Helper to create a test score object."""
    metadata = {
        "category": category,
        "scope": scope,
        "repo_url": repo_url,
        "workflow": workflow,
        "session_name": session_name,
    }
    if correction_details:
        metadata["correction_details"] = correction_details
    return {
        "value": severity,
        "comment": description,
        "traceId": trace_id,
        "metadata": metadata,
        "createdAt": "2026-02-15T10:00:00Z",
    }


# ------------------------------------------------------------------
# Grouping tests
# ------------------------------------------------------------------


def test_groups_by_repo_and_workflow():
    """Scores grouped into (repo_url, workflow) buckets."""
    scores = [
        _make_score(repo_url="https://github.com/org/repo-a", workflow="wf-1"),
        _make_score(repo_url="https://github.com/org/repo-a", workflow="wf-1"),
        _make_score(repo_url="https://github.com/org/repo-b", workflow="wf-2"),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 2

    # Find groups by repo
    group_a = next(g for g in groups if "repo-a" in g["repo_url"])
    group_b = next(g for g in groups if "repo-b" in g["repo_url"])

    assert group_a["total_count"] == 2
    assert group_b["total_count"] == 1


def test_calculates_avg_severity():
    """Average severity is computed correctly."""
    scores = [
        _make_score(severity=1),
        _make_score(severity=3),
        _make_score(severity=2),
    ]
    groups = group_corrections(scores)
    assert len(groups) == 1
    assert groups[0]["avg_severity"] == 2.0


def test_counts_categories():
    """Category counts are accurate."""
    scores = [
        _make_score(category="wrong_approach"),
        _make_score(category="wrong_approach"),
        _make_score(category="style_violation"),
    ]
    groups = group_corrections(scores)
    counts = groups[0]["category_counts"]
    assert counts["wrong_approach"] == 2
    assert counts["style_violation"] == 1


def test_counts_scopes():
    """Scope counts are accurate."""
    scores = [
        _make_score(scope="repo_context"),
        _make_score(scope="repo_context"),
        _make_score(scope="workflow_instructions"),
    ]
    groups = group_corrections(scores)
    counts = groups[0]["scope_counts"]
    assert counts["repo_context"] == 2
    assert counts["workflow_instructions"] == 1


def test_handles_missing_metadata():
    """Scores with missing metadata fields are handled gracefully."""
    scores = [
        {"value": 2, "comment": "test", "metadata": None},
        {"value": 3, "comment": "test2", "metadata": {}},
    ]
    groups = group_corrections(scores)
    assert len(groups) == 1
    assert groups[0]["repo_url"] == "unknown"
    assert groups[0]["workflow"] == "unknown"
    assert groups[0]["total_count"] == 2


def test_sorted_by_count_descending():
    """Groups sorted by total_count descending."""
    scores = [
        _make_score(repo_url="https://github.com/org/small", workflow="wf"),
        _make_score(repo_url="https://github.com/org/big", workflow="wf"),
        _make_score(repo_url="https://github.com/org/big", workflow="wf"),
        _make_score(repo_url="https://github.com/org/big", workflow="wf"),
    ]
    groups = group_corrections(scores)
    assert groups[0]["repo_url"] == "https://github.com/org/big"
    assert groups[0]["total_count"] == 3


# ------------------------------------------------------------------
# Prompt generation tests
# ------------------------------------------------------------------


def test_prompt_includes_all_corrections():
    """Prompt includes details from all corrections in group."""
    group = {
        "repo_url": "https://github.com/org/repo",
        "workflow": "workflows/test",
        "corrections": [
            {
                "category": "wrong_approach",
                "severity": 3,
                "scope": "repo_context",
                "description": "Used wrong pattern",
                "correction_details": "Should use factory pattern",
                "session_name": "session-1",
                "trace_id": "trace-1",
            },
            {
                "category": "missing_context",
                "severity": 2,
                "scope": "workflow_instructions",
                "description": "Didn't know about auth flow",
                "correction_details": "",
                "session_name": "session-2",
                "trace_id": "trace-2",
            },
        ],
        "total_count": 2,
        "avg_severity": 2.5,
        "category_counts": {"wrong_approach": 1, "missing_context": 1},
        "scope_counts": {"repo_context": 1, "workflow_instructions": 1},
    }

    prompt = build_improvement_prompt(group)

    assert "https://github.com/org/repo" in prompt
    assert "workflows/test" in prompt
    assert "wrong_approach" in prompt
    assert "missing_context" in prompt
    assert "Used wrong pattern" in prompt
    assert "Should use factory pattern" in prompt
    assert "Didn't know about auth flow" in prompt
    assert "2 corrections" in prompt.lower() or "2 user corrections" in prompt.lower()


def test_prompt_identifies_top_category():
    """Prompt highlights the most common category."""
    group = {
        "repo_url": "https://github.com/org/repo",
        "workflow": "wf",
        "corrections": [],
        "total_count": 5,
        "avg_severity": 2.0,
        "category_counts": {"wrong_approach": 3, "style_violation": 2},
        "scope_counts": {"repo_context": 5},
    }

    prompt = build_improvement_prompt(group)
    assert "wrong_approach" in prompt
    assert "3 occurrences" in prompt


def test_prompt_maps_scopes_to_files():
    """Prompt maps scopes to correct file targets."""
    group = {
        "repo_url": "https://github.com/org/repo",
        "workflow": "wf",
        "corrections": [],
        "total_count": 3,
        "avg_severity": 2.0,
        "category_counts": {"wrong_approach": 3},
        "scope_counts": {
            "workflow_instructions": 1,
            "repo_context": 1,
            "code_patterns": 1,
        },
    }

    prompt = build_improvement_prompt(group)
    assert ".ambient/" in prompt
    assert "CLAUDE.md" in prompt
    assert ".claude/patterns/" in prompt


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
        "repo_url": "https://github.com/org/my-repo",
        "workflow": "workflows/test",
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

    # requests.post(url, headers=..., json=..., timeout=...)
    call_args, call_kwargs = mock_post.call_args

    # Check URL (first positional arg)
    url = call_args[0] if call_args else call_kwargs.get("url", "")
    assert "test-project" in url

    # Check auth header
    headers = call_kwargs.get("headers", {})
    assert headers["Authorization"] == "Bearer bot-token-123"

    # Check body
    body = call_kwargs["json"]
    assert body["initialPrompt"] == "Test prompt content"
    assert body["environmentVariables"]["LANGFUSE_MASK_MESSAGES"] == "false"
    assert body["labels"]["feedback-loop"] == "true"
    assert body["repos"][0]["url"] == "https://github.com/org/my-repo"


@patch("query_corrections.requests.post")
def test_handles_api_errors(mock_post):
    """API errors are logged and do not crash."""
    import requests as _requests
    mock_post.side_effect = _requests.RequestException("Connection refused")

    group = {
        "repo_url": "https://github.com/org/repo",
        "workflow": "wf",
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


def test_no_repo_when_url_invalid():
    """repos field omitted when repo_url is not a valid HTTP URL."""
    with patch("query_corrections.requests.post") as mock_post:
        mock_resp = MagicMock()
        mock_resp.json.return_value = {"name": "session-1"}
        mock_resp.raise_for_status = MagicMock()
        mock_post.return_value = mock_resp

        group = {
            "repo_url": "unknown",
            "workflow": "wf",
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
        ("Grouping: by repo and workflow", test_groups_by_repo_and_workflow),
        ("Grouping: avg severity", test_calculates_avg_severity),
        ("Grouping: category counts", test_counts_categories),
        ("Grouping: scope counts", test_counts_scopes),
        ("Grouping: missing metadata", test_handles_missing_metadata),
        ("Grouping: sorted descending", test_sorted_by_count_descending),
        ("Prompt: includes all corrections", test_prompt_includes_all_corrections),
        ("Prompt: top category", test_prompt_identifies_top_category),
        ("Prompt: scope to file map", test_prompt_maps_scopes_to_files),
        ("Session: correct API request", test_sends_correct_api_request),
        ("Session: handles API errors", test_handles_api_errors),
        ("Session: no repo for invalid URL", test_no_repo_when_url_invalid),
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
