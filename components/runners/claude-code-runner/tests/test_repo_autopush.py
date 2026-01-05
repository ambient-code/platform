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
        assert "Use: git push -u output <branch-name>" in prompt
        assert "For repos with output URLs, push to the 'output' remote" in prompt
        assert "Only retry on network errors (up to 4 times with backoff)" in prompt

        # Verify repo list is correct (now in bulleted format with branch info)
        assert "Repositories configured for auto-push:" in prompt
        assert "- repo-with-autopush: push to branch 'feature'" in prompt
        assert "Note: Only push repositories listed above" in prompt

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

        # Verify repo list includes ONLY repos with autoPush=true (now in bulleted format)
        assert "Repositories configured for auto-push:" in prompt
        assert "- repo1: push to branch 'feature'" in prompt
        assert "- repo3: push to branch 'feature'" in prompt
        # Verify repo2 (autoPush=false) is NOT listed in the autopush section
        autopush_section = prompt.split("Repositories configured for auto-push:")[1].split("Note:")[0]
        assert "repo2" not in autopush_section, "repo2 should not appear in autopush repos list"

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
    """Test suite for agentic git push with lightweight fallback"""

    @pytest.mark.asyncio
    async def test_verification_skipped_when_autopush_false(self):
        """Test that verification is skipped when autoPush=false"""
        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Track git commands
        run_cmd_calls = []
        async def mock_run_cmd(cmd, **kwargs):
            run_cmd_calls.append(cmd)
            return ""
        adapter._run_cmd = mock_run_cmd

        # Mock _get_repos_config to return repo with autoPush=false
        adapter._get_repos_config = lambda: [
            {
                'name': 'repo-no-push',
                'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                'output': {'url': 'https://github.com/user/fork', 'branch': 'feature'},
                'autoPush': False
            }
        ]

        # Mock token fetching
        async def mock_fetch_token():
            return "fake-token"
        adapter._fetch_github_token = mock_fetch_token

        # Consume events
        events = []
        async for event in adapter._ensure_autopush_repos_pushed():
            events.append(event)

        # Verify no git commands were called (verification skipped)
        assert len(run_cmd_calls) == 0, "No git commands should run when autoPush=false"

    @pytest.mark.asyncio
    async def test_no_fallback_when_claude_already_pushed(self):
        """Test that no fallback push occurs when Claude already pushed"""
        from pathlib import Path

        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Mock: Claude already pushed (no unpushed commits)
        async def mock_check_unpushed(repo_dir):
            return False  # No unpushed commits
        adapter._check_if_has_unpushed_commits = mock_check_unpushed

        # Track git commands
        run_cmd_calls = []
        async def mock_run_cmd(cmd, **kwargs):
            run_cmd_calls.append(cmd)
            return ""
        adapter._run_cmd = mock_run_cmd

        # Mock Path methods to simulate repo directory exists
        original_exists = Path.exists
        original_is_dir = Path.is_dir
        Path.exists = lambda self: True
        Path.is_dir = lambda self: True

        try:
            # Mock config
            adapter._get_repos_config = lambda: [
                {
                    'name': 'repo-already-pushed',
                    'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork', 'branch': 'feature'},
                    'autoPush': True
                }
            ]

            # Mock token
            async def mock_fetch_token():
                return "fake-token"
            adapter._fetch_github_token = mock_fetch_token

            # Consume events
            events = []
            async for event in adapter._ensure_autopush_repos_pushed():
                events.append(event)

            # Verify no push commands (Claude already pushed)
            push_commands = [cmd for cmd in run_cmd_calls if 'push' in cmd]
            assert len(push_commands) == 0, "No fallback push should occur when Claude already pushed"

            # Verify success event was emitted
            success_events = [e for e in events if hasattr(e, 'event') and e.event.get('type') == 'autopush_success']
            assert len(success_events) > 0, "Should emit autopush_success event when Claude pushed"
            assert success_events[0].event.get('agent') == 'claude', "Success should be attributed to Claude"

        finally:
            Path.exists = original_exists
            Path.is_dir = original_is_dir

    @pytest.mark.asyncio
    async def test_fallback_push_when_claude_forgot(self):
        """Test that fallback push executes when Claude didn't push"""
        from pathlib import Path

        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Mock: Claude forgot to push (has unpushed commits)
        async def mock_check_unpushed(repo_dir):
            return True  # Has unpushed commits
        adapter._check_if_has_unpushed_commits = mock_check_unpushed

        # Track git commands
        run_cmd_calls = []
        async def mock_run_cmd(cmd, **kwargs):
            run_cmd_calls.append(cmd)
            return ""
        adapter._run_cmd = mock_run_cmd

        # Mock Path.exists to simulate repo directory exists
        original_exists = Path.exists
        def mock_exists(self):
            if str(self).endswith('repo-forgot-push'):
                return True
            return original_exists(self)
        Path.exists = mock_exists

        # Mock Path.is_dir
        original_is_dir = Path.is_dir
        def mock_is_dir(self):
            if str(self).endswith('repo-forgot-push'):
                return True
            return original_is_dir(self)
        Path.is_dir = mock_is_dir

        try:
            # Mock config
            adapter._get_repos_config = lambda: [
                {
                    'name': 'repo-forgot-push',
                    'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork', 'branch': 'feature'},
                    'autoPush': True
                }
            ]

            # Mock token
            async def mock_fetch_token():
                return "fake-token"
            adapter._fetch_github_token = mock_fetch_token
            adapter._url_with_token = lambda url, token: url

            # Consume events
            events = []
            async for event in adapter._ensure_autopush_repos_pushed():
                events.append(event)

            # Verify fallback push was called
            push_commands = [cmd for cmd in run_cmd_calls if 'push' in cmd]
            assert len(push_commands) > 0, "Fallback should push when Claude forgot"

            # Verify remote configuration commands were called
            remote_commands = [cmd for cmd in run_cmd_calls if 'remote' in cmd]
            assert len(remote_commands) > 0, "Fallback should configure remote"

            # Verify fallback event was emitted
            fallback_events = [e for e in events if hasattr(e, 'event') and e.event.get('type') == 'autopush_fallback']
            assert len(fallback_events) > 0, "Should emit autopush_fallback event when Claude forgot to push"

            # Verify success event after fallback completes
            success_events = [e for e in events if hasattr(e, 'event') and e.event.get('type') == 'autopush_success']
            assert len(success_events) > 0, "Should emit autopush_success event after fallback completes"
            fallback_success = [e for e in success_events if e.event.get('agent') == 'fallback']
            assert len(fallback_success) > 0, "Fallback success should be attributed to 'fallback' agent"

        finally:
            # Restore Path methods
            Path.exists = original_exists
            Path.is_dir = original_is_dir

    @pytest.mark.asyncio
    async def test_fallback_does_not_commit(self):
        """Test that fallback only pushes, does NOT stage or commit"""
        from pathlib import Path

        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Mock: Claude forgot to push
        async def mock_check_unpushed(repo_dir):
            return True
        adapter._check_if_has_unpushed_commits = mock_check_unpushed

        # Track git commands
        run_cmd_calls = []
        async def mock_run_cmd(cmd, **kwargs):
            run_cmd_calls.append(cmd)
            return ""
        adapter._run_cmd = mock_run_cmd

        # Mock Path.exists and is_dir
        original_exists = Path.exists
        original_is_dir = Path.is_dir
        Path.exists = lambda self: True if str(self).endswith('repo-test') else original_exists(self)
        Path.is_dir = lambda self: True if str(self).endswith('repo-test') else original_is_dir(self)

        try:
            # Mock config
            adapter._get_repos_config = lambda: [
                {
                    'name': 'repo-test',
                    'input': {'url': 'https://github.com/org/repo', 'branch': 'main'},
                    'output': {'url': 'https://github.com/user/fork', 'branch': 'feature'},
                    'autoPush': True
                }
            ]

            # Mock token
            async def mock_fetch_token():
                return "fake-token"
            adapter._fetch_github_token = mock_fetch_token
            adapter._url_with_token = lambda url, token: url

            # Consume events
            events = []
            async for event in adapter._ensure_autopush_repos_pushed():
                events.append(event)

            # Verify NO staging or committing (fallback assumes Claude committed)
            # Note: 'git remote add' is NOT staging - check for 'git add <files>' specifically
            add_commands = [cmd for cmd in run_cmd_calls if len(cmd) >= 2 and cmd[0] == 'git' and cmd[1] == 'add']
            commit_commands = [cmd for cmd in run_cmd_calls if len(cmd) >= 2 and cmd[0] == 'git' and cmd[1] == 'commit']

            assert len(add_commands) == 0, "Fallback should NOT stage files (assumes Claude committed)"
            assert len(commit_commands) == 0, "Fallback should NOT commit (assumes Claude committed)"

            # Verify push WAS called
            push_commands = [cmd for cmd in run_cmd_calls if 'push' in cmd]
            assert len(push_commands) > 0, "Fallback should push"

        finally:
            Path.exists = original_exists
            Path.is_dir = original_is_dir

    @pytest.mark.asyncio
    async def test_mixed_autopush_only_pushes_enabled_repos(self):
        """Test that only repos with autoPush=true trigger fallback"""
        from pathlib import Path

        mock_context = Mock()
        mock_context.workspace_path = "/tmp/workspace"
        mock_context.session_id = "test-session"
        mock_context.get_env = lambda k, d=None: None

        adapter = ClaudeCodeAdapter()
        adapter.context = mock_context

        # Track which repos were checked
        checked_repos = []
        async def mock_check_unpushed(repo_dir):
            repo_name = str(repo_dir).split('/')[-1]
            checked_repos.append(repo_name)
            return True  # All have unpushed commits
        adapter._check_if_has_unpushed_commits = mock_check_unpushed

        # Track push commands
        pushed_repos = []
        async def mock_run_cmd(cmd, cwd=None, **kwargs):
            if cmd[0] == 'git' and 'push' in cmd and cwd:
                repo_name = cwd.split('/')[-1]
                pushed_repos.append(repo_name)
            return ""
        adapter._run_cmd = mock_run_cmd

        # Mock Path methods
        original_exists = Path.exists
        original_is_dir = Path.is_dir
        Path.exists = lambda self: True
        Path.is_dir = lambda self: True

        try:
            # Mock config with mixed autoPush settings
            adapter._get_repos_config = lambda: [
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

            # Mock token
            async def mock_fetch_token():
                return "fake-token"
            adapter._fetch_github_token = mock_fetch_token
            adapter._url_with_token = lambda url, token: url

            # Consume events
            events = []
            async for event in adapter._ensure_autopush_repos_pushed():
                events.append(event)

            # Verify only repos with autoPush=true were checked and pushed
            assert 'repo-push' in checked_repos, "repo-push should be checked (autoPush=true)"
            assert 'repo-push-2' in checked_repos, "repo-push-2 should be checked (autoPush=true)"
            assert 'repo-no-push' not in checked_repos, "repo-no-push should NOT be checked (autoPush=false)"

            assert 'repo-push' in pushed_repos, "repo-push should be pushed (autoPush=true)"
            assert 'repo-push-2' in pushed_repos, "repo-push-2 should be pushed (autoPush=true)"
            assert 'repo-no-push' not in pushed_repos, "repo-no-push should NOT be pushed (autoPush=false)"

            # Verify summary event was emitted
            summary_events = [e for e in events if hasattr(e, 'event') and e.event.get('type') == 'autopush_summary']
            assert len(summary_events) > 0, "Should emit autopush_summary event after processing multiple repos"
            summary = summary_events[0].event
            # Should have 2 fallback successes (both repos unpushed and pushed by fallback)
            assert summary.get('fallback_success') == 2, f"Expected 2 fallback successes, got {summary.get('fallback_success')}"

        finally:
            Path.exists = original_exists
            Path.is_dir = original_is_dir
