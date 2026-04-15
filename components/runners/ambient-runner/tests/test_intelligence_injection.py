"""Unit tests for proactive intelligence context injection into system prompt."""

from unittest.mock import MagicMock, patch

import pytest


INTEL_CLIENT_PATH = "ambient_runner.tools.intelligence_api.IntelligenceAPIClient"


class TestBuildIntelligenceContextSection:
    """Test _build_intelligence_context_section in prompts.py."""

    def test_empty_repos_returns_empty(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        assert _build_intelligence_context_section([]) == ""
        assert _build_intelligence_context_section(None) == ""

    def test_repos_without_urls_returns_empty(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        result = _build_intelligence_context_section([{"name": "repo1"}])
        assert result == ""

    def test_no_intelligence_returns_unanalyzed_hint(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        mock_client = MagicMock()
        mock_client.intelligence_exists.return_value = False

        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            result = _build_intelligence_context_section(
                [{"url": "https://github.com/org/repo", "name": "repo"}]
            )
        assert "Unanalyzed" in result
        assert "repo" in result
        assert "memory_store" in result

    def test_injects_context_when_intelligence_exists(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        mock_client = MagicMock()
        mock_client.project_id = "test-project"
        mock_client.intelligence_exists.return_value = True
        mock_client._make_request.return_value = {
            "intelligences": [{"id": "intel-1"}],
            "findings": [],
            "injected_context": "<!-- BEGIN -->\n## Project Intelligence\nSome context\n<!-- END -->\n",
        }

        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            result = _build_intelligence_context_section(
                [{"url": "https://github.com/org/repo", "name": "repo"}]
            )

        assert "Project Intelligence" in result
        assert "Some context" in result
        assert "<!-- BEGIN -->" in result

    def test_empty_injected_context_shows_unanalyzed_hint(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        mock_client = MagicMock()
        mock_client.project_id = "test-project"
        mock_client.intelligence_exists.return_value = True
        mock_client._make_request.return_value = {
            "intelligences": [],
            "findings": [],
            "injected_context": "",
        }

        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            result = _build_intelligence_context_section(
                [{"url": "https://github.com/org/repo", "name": "repo"}]
            )
        # No injected context but no unanalyzed hint either (intelligence_exists=True)
        assert result == ""

    def test_api_error_returns_empty(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        with patch(INTEL_CLIENT_PATH, side_effect=ValueError("no URL")):
            result = _build_intelligence_context_section(
                [{"url": "https://github.com/org/repo", "name": "repo"}]
            )
        assert result == ""

    def test_network_error_returns_empty(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        mock_client = MagicMock()
        mock_client.intelligence_exists.side_effect = Exception("connection refused")

        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            result = _build_intelligence_context_section(
                [{"url": "https://github.com/org/repo", "name": "repo"}]
            )
        assert result == ""

    def test_multiple_repos_passes_analyzed_urls(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        mock_client = MagicMock()
        mock_client.project_id = "proj"
        mock_client.intelligence_exists.return_value = True
        mock_client._make_request.return_value = {
            "intelligences": [],
            "findings": [],
            "injected_context": "## Intelligence\ndata\n",
        }

        repos = [
            {"url": "https://github.com/org/repo1", "name": "repo1"},
            {"url": "https://github.com/org/repo2", "name": "repo2"},
        ]
        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            result = _build_intelligence_context_section(repos)

        assert result != ""
        call_args = mock_client._make_request.call_args
        assert "repo1" in call_args[0][1]
        assert "repo2" in call_args[0][1]

    def test_mixed_analyzed_and_unanalyzed_repos(self):
        from ambient_runner.platform.prompts import _build_intelligence_context_section

        mock_client = MagicMock()
        mock_client.project_id = "proj"
        # repo1 has intelligence, repo2 does not
        mock_client.intelligence_exists.side_effect = [True, False]
        mock_client._make_request.return_value = {
            "intelligences": [{"id": "intel-1"}],
            "findings": [],
            "injected_context": "## Intelligence\nrepo1 data\n",
        }

        repos = [
            {"url": "https://github.com/org/repo1", "name": "repo1"},
            {"url": "https://github.com/org/repo2", "name": "repo2"},
        ]
        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            result = _build_intelligence_context_section(repos)

        assert "repo1 data" in result
        assert "Unanalyzed" in result
        assert "repo2" in result


class TestBuildWorkspaceContextPromptIntegration:
    """Test that intelligence section is included in the full prompt."""

    def test_prompt_includes_intelligence_section(self):
        from ambient_runner.platform.prompts import build_workspace_context_prompt

        mock_client = MagicMock()
        mock_client.project_id = "proj"
        mock_client.intelligence_exists.return_value = True
        mock_client._make_request.return_value = {
            "intelligences": [{"id": "intel-1"}],
            "findings": [],
            "injected_context": "<!-- BEGIN PROJECT MEMORY -->\n## Known caveats\n- Don't touch main\n<!-- END -->\n",
        }

        with patch(INTEL_CLIENT_PATH, return_value=mock_client):
            prompt = build_workspace_context_prompt(
                repos_cfg=[{"url": "https://github.com/org/repo", "name": "repo"}],
                workflow_name=None,
                artifacts_path="artifacts",
                ambient_config={},
                workspace_path="/workspace",
            )

        assert "Known caveats" in prompt
        assert "Don't touch main" in prompt
        # Should appear after other sections
        assert "Workspace Structure" in prompt

    def test_prompt_works_without_intelligence(self, monkeypatch):
        from ambient_runner.platform.prompts import build_workspace_context_prompt

        monkeypatch.delenv("API_SERVER_URL", raising=False)
        monkeypatch.delenv("BACKEND_API_URL", raising=False)

        prompt = build_workspace_context_prompt(
            repos_cfg=[{"url": "https://github.com/org/repo", "name": "repo"}],
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={},
            workspace_path="/workspace",
        )

        # Prompt should still be valid without intelligence
        assert "Workspace Structure" in prompt
        assert "Repositories" in prompt
