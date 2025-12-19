"""
Test cases for repository autoPush flag extraction and handling.

This module tests the _get_repos_config() method's extraction of per-repo
autoPush flags from the REPOS_JSON environment variable.
"""

import json
import os
import pytest
from pathlib import Path
import sys
from unittest.mock import Mock, patch

# Add parent directory to path for importing adapter module
adapter_dir = Path(__file__).parent.parent
if str(adapter_dir) not in sys.path:
    sys.path.insert(0, str(adapter_dir))

from adapter import ClaudeCodeAdapter  # type: ignore[import]


class TestGetReposConfig:
    """Test suite for _get_repos_config method's autoPush extraction"""

    def test_extract_autopush_true(self):
        """Test extraction of autoPush=true from repos config"""
        repos_json = json.dumps([
            {
                "name": "repo1",
                "input": {"url": "https://github.com/org/repo1", "branch": "main"},
                "output": {"url": "https://github.com/user/fork1", "branch": "feature"},
                "autoPush": True
            }
        ])

        with patch.dict(os.environ, {'REPOS_JSON': repos_json}):
            # Create mock context with required attributes
            mock_context = Mock()
            mock_context.workspace_path = "/tmp/workspace"
            mock_context.session_id = "test-session-123"
            mock_context.get_env = lambda k, d=None: os.getenv(k, d)

            adapter = ClaudeCodeAdapter()
            adapter.context = mock_context
            result = adapter._get_repos_config()

            assert len(result) == 1
            assert result[0]['name'] == 'repo1'
            assert result[0]['autoPush'] is True

    def test_extract_autopush_false(self):
        """Test extraction of autoPush=false from repos config"""
        repos_json = json.dumps([
            {
                "name": "repo2",
                "input": {"url": "https://github.com/org/repo2", "branch": "develop"},
                "autoPush": False
            }
        ])

        with patch.dict(os.environ, {'REPOS_JSON': repos_json}):
            mock_context = Mock()
            mock_context.workspace_path = "/tmp/workspace"
            mock_context.session_id = "test-session-456"
            mock_context.get_env = lambda k, d=None: os.getenv(k, d)

            adapter = ClaudeCodeAdapter()
            adapter.context = mock_context
            result = adapter._get_repos_config()

            assert len(result) == 1
            assert result[0]['name'] == 'repo2'
            assert result[0]['autoPush'] is False

    def test_default_autopush_false_when_missing(self):
        """Test that autoPush defaults to False when not specified"""
        repos_json = json.dumps([
            {
                "name": "repo3",
                "input": {"url": "https://github.com/org/repo3", "branch": "main"}
                # No autoPush field
            }
        ])

        with patch.dict(os.environ, {'REPOS_JSON': repos_json}):
            mock_context = Mock()
            mock_context.workspace_path = "/tmp/workspace"
            mock_context.session_id = "test-session-789"
            # mock_context.model_name removed
            # mock_context.system_prompt removed
            mock_context.get_env = lambda k, d=None: os.getenv(k, d)

            adapter = ClaudeCodeAdapter()
            adapter.context = mock_context
            result = adapter._get_repos_config()

            assert len(result) == 1
            assert result[0]['name'] == 'repo3'
            assert result[0]['autoPush'] is False  # Default when not specified

    def test_mixed_autopush_flags(self):
        """Test handling of multiple repos with different autoPush settings"""
        repos_json = json.dumps([
            {
                "name": "repo-push",
                "input": {"url": "https://github.com/org/repo-push", "branch": "main"},
                "autoPush": True
            },
            {
                "name": "repo-no-push",
                "input": {"url": "https://github.com/org/repo-no-push", "branch": "main"},
                "autoPush": False
            },
            {
                "name": "repo-default",
                "input": {"url": "https://github.com/org/repo-default", "branch": "main"}
                # No autoPush field - defaults to False
            }
        ])

        with patch.dict(os.environ, {'REPOS_JSON': repos_json}):
            mock_context = Mock()
            mock_context.workspace_path = "/tmp/workspace"
            mock_context.session_id = "test-session-multi"
            # mock_context.model_name removed
            # mock_context.system_prompt removed
            mock_context.get_env = lambda k, d=None: os.getenv(k, d)

            adapter = ClaudeCodeAdapter()
            adapter.context = mock_context
            result = adapter._get_repos_config()

            assert len(result) == 3

            # Find each repo by name and verify autoPush
            repo_push = next(r for r in result if r['name'] == 'repo-push')
            assert repo_push['autoPush'] is True

            repo_no_push = next(r for r in result if r['name'] == 'repo-no-push')
            assert repo_no_push['autoPush'] is False

            repo_default = next(r for r in result if r['name'] == 'repo-default')
            assert repo_default['autoPush'] is False

    def test_legacy_format_without_autopush(self):
        """Test that legacy format (no input/output structure) still works"""
        # This tests backward compatibility - old format doesn't have autoPush
        repos_json = json.dumps([
            {
                "url": "https://github.com/org/legacy-repo",
                "branch": "main"
            }
        ])

        with patch.dict(os.environ, {'REPOS_JSON': repos_json}):
            mock_context = Mock()
            mock_context.workspace_path = "/tmp/workspace"
            mock_context.session_id = "test-session-legacy"
            # mock_context.model_name removed
            # mock_context.system_prompt removed
            mock_context.get_env = lambda k, d=None: os.getenv(k, d)

            adapter = ClaudeCodeAdapter()
            adapter.context = mock_context
            result = adapter._get_repos_config()

            # Legacy format without input/output structure is intentionally filtered out
            # because autoPush requires the new structured format to distinguish input from output repos
            assert len(result) == 0

    def test_empty_repos_json(self):
        """Test handling of empty REPOS_JSON"""
        with patch.dict(os.environ, {'REPOS_JSON': ''}):
            mock_context = Mock()
            mock_context.workspace_path = "/tmp/workspace"
            mock_context.session_id = "test-session-empty"
            # mock_context.model_name removed
            # mock_context.system_prompt removed
            mock_context.get_env = lambda k, d=None: os.getenv(k, d)

            adapter = ClaudeCodeAdapter()
            adapter.context = mock_context
            result = adapter._get_repos_config()

            assert result == []

    def test_invalid_json(self):
        """Test handling of invalid JSON in REPOS_JSON"""
        with patch.dict(os.environ, {'REPOS_JSON': 'invalid-json-{['}):
            mock_context = Mock()
            mock_context.workspace_path = "/tmp/workspace"
            mock_context.session_id = "test-session-invalid"
            # mock_context.model_name removed
            # mock_context.system_prompt removed
            mock_context.get_env = lambda k, d=None: os.getenv(k, d)

            adapter = ClaudeCodeAdapter()
            adapter.context = mock_context
            result = adapter._get_repos_config()

            # Should return empty list on parse error
            assert result == []


class TestSystemPromptInjection:
    """Test suite for system prompt git instruction injection based on autoPush flag"""

    def test_git_instructions_when_autopush_enabled(self):
        """Test that git push instructions appear when at least one repo has autoPush=true"""
        repos_cfg = [
            {
                'name': 'repo-with-autopush',
                'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                'output': {'url': 'https://github.com/user/fork', 'branch': 'feature'},
                'autoPush': True
            }
        ]

        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        # mock_context.model_name removed
        # mock_context.system_prompt removed
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context
        prompt = adapter._build_workspace_context_prompt(
            repos_cfg=repos_cfg,
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Verify git instructions section is present
        assert "## Git Operations - IMPORTANT" in prompt
        assert "When you complete your work:" in prompt
        assert "1. Commit your changes with a clear, descriptive message" in prompt
        assert "2. Push changes to the configured output branch" in prompt
        assert "3. Confirm the push succeeded" in prompt

        # Verify best practices are included
        assert "Git push best practices:" in prompt
        assert "Use: git push -u origin <branch-name>" in prompt
        assert "Only retry on network errors (up to 4 times with backoff)" in prompt

        # Verify repo list is correct
        assert "Repositories configured for auto-push: repo-with-autopush" in prompt

    def test_git_instructions_omitted_when_autopush_disabled(self):
        """Test that git push instructions are NOT included when all repos have autoPush=false"""
        repos_cfg = [
            {
                'name': 'repo-no-push',
                'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                'autoPush': False
            }
        ]

        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        # mock_context.model_name removed
        # mock_context.system_prompt removed
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context
        prompt = adapter._build_workspace_context_prompt(
            repos_cfg=repos_cfg,
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Verify git instructions section is NOT present
        assert "## Git Operations - IMPORTANT" not in prompt
        assert "When you complete your work:" not in prompt
        assert "Repositories configured for auto-push:" not in prompt

    def test_git_instructions_with_multiple_autopush_repos(self):
        """Test that all repos with autoPush=true are listed in the system prompt"""
        repos_cfg = [
            {
                'name': 'repo1',
                'input': {'url': 'https://github.com/org/repo1', 'branch': 'main'},
                'output': {'url': 'https://github.com/user/fork1', 'branch': 'feature'},
                'autoPush': True
            },
            {
                'name': 'repo2',
                'input': {'url': 'https://github.com/org/repo2', 'branch': 'main'},
                'autoPush': False  # This one should NOT be listed
            },
            {
                'name': 'repo3',
                'input': {'url': 'https://github.com/org/repo3', 'branch': 'main'},
                'output': {'url': 'https://github.com/user/fork3', 'branch': 'feature'},
                'autoPush': True
            }
        ]

        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        # mock_context.model_name removed
        # mock_context.system_prompt removed
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context
        prompt = adapter._build_workspace_context_prompt(
            repos_cfg=repos_cfg,
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Verify git instructions section is present
        assert "## Git Operations - IMPORTANT" in prompt

        # Verify repo list includes ONLY repos with autoPush=true
        assert "Repositories configured for auto-push: repo1, repo3" in prompt
        # Verify repo2 (autoPush=false) is NOT listed
        assert "repo2" not in prompt.split("Repositories configured for auto-push:")[1].split("\n")[0]

    def test_git_instructions_omitted_when_no_repos(self):
        """Test that git push instructions are NOT included when repos_cfg is empty"""
        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        # mock_context.model_name removed
        # mock_context.system_prompt removed
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context
        prompt = adapter._build_workspace_context_prompt(
            repos_cfg=[],
            workflow_name=None,
            artifacts_path="artifacts",
            ambient_config={}
        )

        # Verify git instructions section is NOT present
        assert "## Git Operations - IMPORTANT" not in prompt
        assert "Repositories configured for auto-push:" not in prompt


class TestPushBehavior:
    """Test suite for git push behavior based on autoPush flag"""

    @pytest.mark.asyncio
    async def test_push_skipped_when_autopush_false(self):
        """Test that git push is NOT called when autoPush=false"""
        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        # mock_context.model_name removed
        # mock_context.system_prompt removed
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Mock _run_cmd to track calls
        run_cmd_calls = []
        async def mock_run_cmd(cmd, **kwargs):
            run_cmd_calls.append(cmd)
            # Return empty string for git status (no changes)
            if cmd[0] == 'git' and cmd[1] == 'status':
                return "?? test-file.txt"  # Simulate changes
            return ""

        adapter._run_cmd = mock_run_cmd

        # Mock _get_repos_config to return repo with autoPush=false
        def mock_get_repos_config():
            return [
                {
                    'name': 'repo-no-push',
                    'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork', 'branch': 'feature'},
                    'autoPush': False
                }
            ]
        adapter._get_repos_config = mock_get_repos_config

        # Mock _fetch_token_for_url to avoid actual token fetching
        async def mock_fetch_token(url):
            return "fake-token"
        adapter._fetch_token_for_url = mock_fetch_token

        # Call _push_results_if_any
        await adapter._push_results_if_any()

        # Verify git push was NOT called
        push_commands = [cmd for cmd in run_cmd_calls if 'push' in cmd]
        assert len(push_commands) == 0, "git push should not be called when autoPush=false"

    @pytest.mark.asyncio
    async def test_push_executed_when_autopush_true(self):
        """Test that git push IS called when autoPush=true"""
        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        # mock_context.model_name removed
        # mock_context.system_prompt removed
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Mock _run_cmd to track calls
        run_cmd_calls = []
        async def mock_run_cmd(cmd, **kwargs):
            run_cmd_calls.append(cmd)
            # Return appropriate responses for different git commands
            if cmd[0] == 'git':
                if cmd[1] == 'status' and '--porcelain' in cmd:
                    return "?? test-file.txt"  # Simulate changes
                elif cmd[1] == 'remote':
                    return "output"  # Simulate remote exists
            return ""

        adapter._run_cmd = mock_run_cmd

        # Mock _get_repos_config to return repo with autoPush=true
        def mock_get_repos_config():
            return [
                {
                    'name': 'repo-with-push',
                    'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork', 'branch': 'feature'},
                    'autoPush': True
                }
            ]
        adapter._get_repos_config = mock_get_repos_config

        # Mock _fetch_token_for_url to avoid actual token fetching
        async def mock_fetch_token(url):
            return "fake-token"
        adapter._fetch_token_for_url = mock_fetch_token

        # Mock _url_with_token
        adapter._url_with_token = lambda url, token: url

        # Call _push_results_if_any
        await adapter._push_results_if_any()

        # Verify git push WAS called
        push_commands = [cmd for cmd in run_cmd_calls if 'git' in cmd and 'push' in cmd]
        assert len(push_commands) > 0, "git push should be called when autoPush=true"

    @pytest.mark.asyncio
    async def test_mixed_autopush_settings(self):
        """Test that only repos with autoPush=true are pushed"""
        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        # mock_context.model_name removed
        # mock_context.system_prompt removed
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Track which repos were pushed
        pushed_repos = []
        run_cmd_calls = []

        async def mock_run_cmd(cmd, cwd=None, **kwargs):
            run_cmd_calls.append({'cmd': cmd, 'cwd': cwd})
            # Track push commands by the working directory
            if cmd[0] == 'git' and 'push' in cmd and cwd:
                repo_name = cwd.split('/')[-1]
                pushed_repos.append(repo_name)

            # Return appropriate responses
            if cmd[0] == 'git':
                if cmd[1] == 'status' and '--porcelain' in cmd:
                    return "?? test-file.txt"  # Simulate changes
                elif cmd[1] == 'remote':
                    return "output"  # Simulate remote exists
            return ""

        adapter._run_cmd = mock_run_cmd

        # Mock _get_repos_config with mixed autoPush settings
        def mock_get_repos_config():
            return [
                {
                    'name': 'repo-push',
                    'input': {'url': 'https://github.com/org/repo1', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork1', 'branch': 'feature'},
                    'autoPush': True
                },
                {
                    'name': 'repo-no-push',
                    'input': {'url': 'https://github.com/org/repo2', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork2', 'branch': 'feature'},
                    'autoPush': False
                },
                {
                    'name': 'repo-push-2',
                    'input': {'url': 'https://github.com/org/repo3', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork3', 'branch': 'feature'},
                    'autoPush': True
                }
            ]
        adapter._get_repos_config = mock_get_repos_config

        # Mock _fetch_token_for_url
        async def mock_fetch_token(url):
            return "fake-token"
        adapter._fetch_token_for_url = mock_fetch_token

        # Mock _url_with_token
        adapter._url_with_token = lambda url, token: url

        # Call _push_results_if_any
        await adapter._push_results_if_any()

        # Verify only repos with autoPush=true were pushed
        assert 'repo-push' in pushed_repos, "repo-push should be pushed (autoPush=true)"
        assert 'repo-push-2' in pushed_repos, "repo-push-2 should be pushed (autoPush=true)"
        assert 'repo-no-push' not in pushed_repos, "repo-no-push should NOT be pushed (autoPush=false)"
