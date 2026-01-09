"""Unit tests for autoPush functionality in adapter.py."""

import pytest
import json
import os
import sys
from unittest.mock import MagicMock, patch, Mock

# Mock ag_ui module before importing adapter
sys.modules['ag_ui'] = Mock()
sys.modules['ag_ui.core'] = Mock()
sys.modules['context'] = Mock()


class TestGetReposConfig:
    """Tests for _get_repos_config method."""

    def setup_method(self):
        """Set up test fixtures."""
        # Import here after mocking dependencies
        from adapter import ClaudeCodeAdapter
        self.adapter = ClaudeCodeAdapter()

    def test_parse_simple_repo_with_autopush_true(self):
        """Test parsing repo with autoPush=true."""
        repos_json = json.dumps([
            {
                "url": "https://github.com/owner/repo.git",
                "branch": "main",
                "autoPush": True
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 1
        assert result[0]["url"] == "https://github.com/owner/repo.git"
        assert result[0]["branch"] == "main"
        assert result[0]["autoPush"] is True
        assert "repo" in result[0]["name"]

    def test_parse_simple_repo_with_autopush_false(self):
        """Test parsing repo with autoPush=false."""
        repos_json = json.dumps([
            {
                "url": "https://github.com/owner/repo.git",
                "branch": "develop",
                "autoPush": False
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 1
        assert result[0]["autoPush"] is False

    def test_parse_repo_without_autopush(self):
        """Test parsing repo without autoPush field defaults to False."""
        repos_json = json.dumps([
            {
                "url": "https://github.com/owner/repo.git",
                "branch": "main"
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 1
        assert result[0]["autoPush"] is False

    def test_parse_multiple_repos_mixed_autopush(self):
        """Test parsing multiple repos with mixed autoPush settings."""
        repos_json = json.dumps([
            {
                "url": "https://github.com/owner/repo1.git",
                "branch": "main",
                "autoPush": True
            },
            {
                "url": "https://github.com/owner/repo2.git",
                "branch": "develop",
                "autoPush": False
            },
            {
                "url": "https://github.com/owner/repo3.git",
                "branch": "feature"
                # No autoPush field
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 3
        assert result[0]["autoPush"] is True
        assert result[1]["autoPush"] is False
        assert result[2]["autoPush"] is False  # Default

    def test_parse_repo_with_explicit_name(self):
        """Test parsing repo with explicit name field."""
        repos_json = json.dumps([
            {
                "name": "my-custom-repo",
                "url": "https://github.com/owner/repo.git",
                "branch": "main",
                "autoPush": True
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 1
        assert result[0]["name"] == "my-custom-repo"
        assert result[0]["autoPush"] is True

    def test_parse_empty_repos_json(self):
        """Test parsing empty REPOS_JSON."""
        with patch.dict(os.environ, {"REPOS_JSON": ""}):
            result = self.adapter._get_repos_config()

        assert result == []

    def test_parse_missing_repos_json(self):
        """Test parsing when REPOS_JSON not set."""
        with patch.dict(os.environ, {}, clear=True):
            result = self.adapter._get_repos_config()

        assert result == []

    def test_parse_invalid_json(self):
        """Test parsing invalid JSON returns empty list."""
        with patch.dict(os.environ, {"REPOS_JSON": "invalid-json{"}):
            result = self.adapter._get_repos_config()

        assert result == []

    def test_parse_non_list_json(self):
        """Test parsing non-list JSON returns empty list."""
        repos_json = json.dumps({"url": "https://github.com/owner/repo.git"})

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert result == []

    def test_parse_repo_without_url(self):
        """Test that repos without URL are skipped."""
        repos_json = json.dumps([
            {
                "branch": "main",
                "autoPush": True
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert result == []

    def test_derive_repo_name_from_url(self):
        """Test automatic derivation of repo name from URL."""
        test_cases = [
            ("https://github.com/owner/my-repo.git", "my-repo"),
            ("https://github.com/owner/my-repo", "my-repo"),
            ("git@github.com:owner/another-repo.git", "another-repo"),
        ]

        for url, expected_name in test_cases:
            repos_json = json.dumps([{"url": url, "autoPush": True}])

            with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
                result = self.adapter._get_repos_config()

            assert len(result) == 1
            assert result[0]["name"] == expected_name

    def test_autopush_with_invalid_string_type(self):
        """Test that string autoPush values default to False."""
        repos_json = json.dumps([
            {
                "url": "https://github.com/owner/repo.git",
                "autoPush": "true"  # String instead of boolean
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 1
        # Invalid type should default to False
        assert result[0]["autoPush"] is False

    def test_autopush_with_invalid_number_type(self):
        """Test that numeric autoPush values default to False."""
        repos_json = json.dumps([
            {
                "url": "https://github.com/owner/repo.git",
                "autoPush": 1  # Number instead of boolean
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 1
        # Invalid type should default to False
        assert result[0]["autoPush"] is False

    def test_autopush_with_null_value(self):
        """Test that null autoPush values default to False."""
        repos_json = json.dumps([
            {
                "url": "https://github.com/owner/repo.git",
                "autoPush": None  # null in JSON
            }
        ])

        with patch.dict(os.environ, {"REPOS_JSON": repos_json}):
            result = self.adapter._get_repos_config()

        assert len(result) == 1
        # null should default to False
        assert result[0]["autoPush"] is False


class TestBuildWorkspaceContextPrompt:
    """Tests for _build_workspace_context_prompt method."""

    def setup_method(self):
        """Set up test fixtures."""
        # Import here after mocking dependencies
        from adapter import ClaudeCodeAdapter

        self.adapter = ClaudeCodeAdapter()

        # Create a mock context
        mock_context = MagicMock()
        mock_context.workspace_path = "/workspace"
        self.adapter.context = mock_context

    def test_prompt_includes_git_instructions_with_autopush(self):
        """Test that git push instructions are included when autoPush=true."""
        repos_cfg = [
            {
                "name": "my-repo",
                "url": "https://github.com/owner/my-repo.git",
                "branch": "main",
                "autoPush": True
            }
        ]

        prompt = self.adapter._build_workspace_context_prompt(
            repos_cfg=repos_cfg,
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Verify git instructions are present
        assert "Git Push Instructions" in prompt
        assert "repos/my-repo/" in prompt
        assert "branch: main" in prompt
        assert "git add" in prompt
        assert "git commit" in prompt
        assert "git push origin" in prompt

    def test_prompt_excludes_git_instructions_without_autopush(self):
        """Test that git push instructions are excluded when autoPush=false."""
        repos_cfg = [
            {
                "name": "my-repo",
                "url": "https://github.com/owner/my-repo.git",
                "branch": "main",
                "autoPush": False
            }
        ]

        prompt = self.adapter._build_workspace_context_prompt(
            repos_cfg=repos_cfg,
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Verify git instructions are NOT present
        assert "Git Push Instructions" not in prompt
        assert "git add" not in prompt
        assert "git commit" not in prompt
        assert "git push origin" not in prompt

    def test_prompt_includes_multiple_autopush_repos(self):
        """Test that all autoPush repos are listed in instructions."""
        repos_cfg = [
            {
                "name": "repo1",
                "url": "https://github.com/owner/repo1.git",
                "branch": "main",
                "autoPush": True
            },
            {
                "name": "repo2",
                "url": "https://github.com/owner/repo2.git",
                "branch": "develop",
                "autoPush": True
            },
            {
                "name": "repo3",
                "url": "https://github.com/owner/repo3.git",
                "branch": "feature",
                "autoPush": False
            }
        ]

        prompt = self.adapter._build_workspace_context_prompt(
            repos_cfg=repos_cfg,
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Verify both autoPush repos are listed
        assert "repos/repo1/" in prompt
        assert "branch: main" in prompt
        assert "repos/repo2/" in prompt
        assert "branch: develop" in prompt
        # repo3 should not be in git instructions since autoPush=false
        # (but it will be in the general repos list)

    def test_prompt_without_repos(self):
        """Test prompt generation when no repos are configured."""
        prompt = self.adapter._build_workspace_context_prompt(
            repos_cfg=[],
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Should not include git instructions
        assert "Git Push Instructions" not in prompt
        # Should still include other sections
        assert "Workspace Structure" in prompt
        assert "Artifacts" in prompt

    def test_prompt_with_workflow(self):
        """Test prompt generation with workflow context."""
        repos_cfg = [
            {
                "name": "my-repo",
                "url": "https://github.com/owner/my-repo.git",
                "branch": "main",
                "autoPush": True
            }
        ]

        prompt = self.adapter._build_workspace_context_prompt(
            repos_cfg=repos_cfg,
            workflow_name="test-workflow",
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Should include both workflow info and git instructions
        assert "workflows/test-workflow/" in prompt
        assert "Git Push Instructions" in prompt
        assert "repos/my-repo/" in prompt
