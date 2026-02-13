"""
Unit tests for Git identity configuration from provider credentials.

Tests the configure_git_identity() function and the credential fetching
functions that now return user identity (userName, email) in addition to tokens.

Bug Fix: GitHub credentials aren't mounted to session - need git identity
         Also adds provider distinction (github vs gitlab)
"""

import asyncio
import os
import subprocess
import sys
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

# Add parent directory to path to import auth module
sys.path.insert(0, str(Path(__file__).parent.parent))


class TestConfigureGitIdentity:
    """Test configure_git_identity function."""

    @pytest.fixture(autouse=True)
    def setup_env(self):
        """Save and restore environment variables."""
        original_env = os.environ.copy()
        yield
        os.environ.clear()
        os.environ.update(original_env)

    @pytest.mark.asyncio
    async def test_configure_git_identity_with_valid_credentials(self):
        """Test git identity is configured with provided user name and email."""
        from auth import configure_git_identity

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(returncode=0)

            await configure_git_identity("John Doe", "john@example.com")

            # Verify git config commands were called
            assert mock_run.call_count == 2

            # Check user.name was set
            name_call = mock_run.call_args_list[0]
            assert "user.name" in name_call[0][0]
            assert "John Doe" in name_call[0][0]

            # Check user.email was set
            email_call = mock_run.call_args_list[1]
            assert "user.email" in email_call[0][0]
            assert "john@example.com" in email_call[0][0]

            # Verify environment variables were set
            assert os.environ.get("GIT_USER_NAME") == "John Doe"
            assert os.environ.get("GIT_USER_EMAIL") == "john@example.com"

    @pytest.mark.asyncio
    async def test_configure_git_identity_falls_back_to_defaults(self):
        """Test git identity uses defaults when credentials are empty."""
        from auth import configure_git_identity

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(returncode=0)

            await configure_git_identity("", "")

            # Verify defaults were used
            assert os.environ.get("GIT_USER_NAME") == "Ambient Code Bot"
            assert os.environ.get("GIT_USER_EMAIL") == "bot@ambient-code.local"

            # Check git config was called with defaults
            name_call = mock_run.call_args_list[0]
            assert "Ambient Code Bot" in name_call[0][0]

    @pytest.mark.asyncio
    async def test_configure_git_identity_strips_whitespace(self):
        """Test git identity strips whitespace from values."""
        from auth import configure_git_identity

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(returncode=0)

            await configure_git_identity("  Jane Doe  ", "  jane@example.com  ")

            assert os.environ.get("GIT_USER_NAME") == "Jane Doe"
            assert os.environ.get("GIT_USER_EMAIL") == "jane@example.com"

    @pytest.mark.asyncio
    async def test_configure_git_identity_handles_subprocess_error(self):
        """Test git identity handles subprocess errors gracefully."""
        from auth import configure_git_identity

        with patch("subprocess.run") as mock_run:
            mock_run.side_effect = subprocess.TimeoutExpired("git", 5)

            # Should not raise, just log warning
            await configure_git_identity("Test User", "test@example.com")

            # Environment variables should still be set even if git config fails
            assert os.environ.get("GIT_USER_NAME") == "Test User"
            assert os.environ.get("GIT_USER_EMAIL") == "test@example.com"


class TestFetchGitHubCredentials:
    """Test fetch_github_credentials function returns identity."""

    @pytest.fixture(autouse=True)
    def setup_env(self):
        """Set up environment variables."""
        original_env = os.environ.copy()
        os.environ["BACKEND_API_URL"] = "http://test-backend:8080/api"
        os.environ["PROJECT_NAME"] = "test-project"
        yield
        os.environ.clear()
        os.environ.update(original_env)

    @pytest.mark.asyncio
    async def test_fetch_github_credentials_returns_identity(self):
        """Test that fetch_github_credentials returns userName and email."""
        from auth import fetch_github_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        mock_response = {
            "token": "ghp_test_token",
            "userName": "Test User",
            "email": "test@github.com",
            "provider": "github",
        }

        with patch("auth._fetch_credential", new_callable=AsyncMock) as mock_fetch:
            mock_fetch.return_value = mock_response

            result = await fetch_github_credentials(mock_context)

            assert result["token"] == "ghp_test_token"
            assert result["userName"] == "Test User"
            assert result["email"] == "test@github.com"
            assert result["provider"] == "github"

    @pytest.mark.asyncio
    async def test_fetch_github_token_delegates_to_fetch_github_credentials(self):
        """Test that fetch_github_token uses fetch_github_credentials."""
        from auth import fetch_github_token
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch("auth.fetch_github_credentials", new_callable=AsyncMock) as mock_fetch:
            mock_fetch.return_value = {"token": "ghp_test_token", "userName": "Test"}

            result = await fetch_github_token(mock_context)

            assert result == "ghp_test_token"
            mock_fetch.assert_called_once_with(mock_context)


class TestFetchGitLabCredentials:
    """Test fetch_gitlab_credentials function returns identity."""

    @pytest.fixture(autouse=True)
    def setup_env(self):
        """Set up environment variables."""
        original_env = os.environ.copy()
        os.environ["BACKEND_API_URL"] = "http://test-backend:8080/api"
        os.environ["PROJECT_NAME"] = "test-project"
        yield
        os.environ.clear()
        os.environ.update(original_env)

    @pytest.mark.asyncio
    async def test_fetch_gitlab_credentials_returns_identity(self):
        """Test that fetch_gitlab_credentials returns userName and email."""
        from auth import fetch_gitlab_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        mock_response = {
            "token": "glpat-test_token",
            "instanceUrl": "https://gitlab.com",
            "userName": "Test GitLab User",
            "email": "test@gitlab.com",
            "provider": "gitlab",
        }

        with patch("auth._fetch_credential", new_callable=AsyncMock) as mock_fetch:
            mock_fetch.return_value = mock_response

            result = await fetch_gitlab_credentials(mock_context)

            assert result["token"] == "glpat-test_token"
            assert result["instanceUrl"] == "https://gitlab.com"
            assert result["userName"] == "Test GitLab User"
            assert result["email"] == "test@gitlab.com"
            assert result["provider"] == "gitlab"

    @pytest.mark.asyncio
    async def test_fetch_gitlab_token_delegates_to_fetch_gitlab_credentials(self):
        """Test that fetch_gitlab_token uses fetch_gitlab_credentials."""
        from auth import fetch_gitlab_token
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch("auth.fetch_gitlab_credentials", new_callable=AsyncMock) as mock_fetch:
            mock_fetch.return_value = {"token": "glpat-test_token"}

            result = await fetch_gitlab_token(mock_context)

            assert result == "glpat-test_token"
            mock_fetch.assert_called_once_with(mock_context)


class TestPopulateRuntimeCredentialsGitIdentity:
    """Test that populate_runtime_credentials configures git identity."""

    @pytest.fixture(autouse=True)
    def setup_env(self):
        """Set up environment variables."""
        original_env = os.environ.copy()
        os.environ["BACKEND_API_URL"] = "http://test-backend:8080/api"
        os.environ["PROJECT_NAME"] = "test-project"
        yield
        os.environ.clear()
        os.environ.update(original_env)

    @pytest.mark.asyncio
    async def test_git_identity_from_github(self):
        """Test git identity is configured from GitHub credentials."""
        from auth import populate_runtime_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        github_creds = {
            "token": "ghp_test",
            "userName": "GitHub User",
            "email": "user@github.com",
            "provider": "github",
        }

        with patch("auth.fetch_google_credentials", new_callable=AsyncMock) as mock_google, \
             patch("auth.fetch_jira_credentials", new_callable=AsyncMock) as mock_jira, \
             patch("auth.fetch_gitlab_credentials", new_callable=AsyncMock) as mock_gitlab, \
             patch("auth.fetch_github_credentials", new_callable=AsyncMock) as mock_github, \
             patch("auth.configure_git_identity", new_callable=AsyncMock) as mock_config:

            mock_google.return_value = {}
            mock_jira.return_value = {}
            mock_gitlab.return_value = {}
            mock_github.return_value = github_creds

            await populate_runtime_credentials(mock_context)

            # Verify configure_git_identity was called with GitHub user info
            mock_config.assert_called_once_with("GitHub User", "user@github.com")

    @pytest.mark.asyncio
    async def test_git_identity_from_gitlab_when_no_github(self):
        """Test git identity is configured from GitLab when GitHub not available."""
        from auth import populate_runtime_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        gitlab_creds = {
            "token": "glpat-test",
            "userName": "GitLab User",
            "email": "user@gitlab.com",
            "provider": "gitlab",
        }

        with patch("auth.fetch_google_credentials", new_callable=AsyncMock) as mock_google, \
             patch("auth.fetch_jira_credentials", new_callable=AsyncMock) as mock_jira, \
             patch("auth.fetch_gitlab_credentials", new_callable=AsyncMock) as mock_gitlab, \
             patch("auth.fetch_github_credentials", new_callable=AsyncMock) as mock_github, \
             patch("auth.configure_git_identity", new_callable=AsyncMock) as mock_config:

            mock_google.return_value = {}
            mock_jira.return_value = {}
            mock_gitlab.return_value = gitlab_creds
            mock_github.return_value = {}  # No GitHub credentials

            await populate_runtime_credentials(mock_context)

            # Verify configure_git_identity was called with GitLab user info
            mock_config.assert_called_once_with("GitLab User", "user@gitlab.com")

    @pytest.mark.asyncio
    async def test_github_takes_precedence_over_gitlab(self):
        """Test GitHub identity takes precedence when both are available."""
        from auth import populate_runtime_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        gitlab_creds = {
            "token": "glpat-test",
            "userName": "GitLab User",
            "email": "user@gitlab.com",
            "provider": "gitlab",
        }
        github_creds = {
            "token": "ghp_test",
            "userName": "GitHub User",
            "email": "user@github.com",
            "provider": "github",
        }

        with patch("auth.fetch_google_credentials", new_callable=AsyncMock) as mock_google, \
             patch("auth.fetch_jira_credentials", new_callable=AsyncMock) as mock_jira, \
             patch("auth.fetch_gitlab_credentials", new_callable=AsyncMock) as mock_gitlab, \
             patch("auth.fetch_github_credentials", new_callable=AsyncMock) as mock_github, \
             patch("auth.configure_git_identity", new_callable=AsyncMock) as mock_config:

            mock_google.return_value = {}
            mock_jira.return_value = {}
            mock_gitlab.return_value = gitlab_creds
            mock_github.return_value = github_creds

            await populate_runtime_credentials(mock_context)

            # GitHub should take precedence
            mock_config.assert_called_once_with("GitHub User", "user@github.com")

    @pytest.mark.asyncio
    async def test_defaults_when_no_credentials(self):
        """Test defaults are used when no credentials have identity."""
        from auth import populate_runtime_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch("auth.fetch_google_credentials", new_callable=AsyncMock) as mock_google, \
             patch("auth.fetch_jira_credentials", new_callable=AsyncMock) as mock_jira, \
             patch("auth.fetch_gitlab_credentials", new_callable=AsyncMock) as mock_gitlab, \
             patch("auth.fetch_github_credentials", new_callable=AsyncMock) as mock_github, \
             patch("auth.configure_git_identity", new_callable=AsyncMock) as mock_config:

            mock_google.return_value = {}
            mock_jira.return_value = {}
            mock_gitlab.return_value = {}
            mock_github.return_value = {}

            await populate_runtime_credentials(mock_context)

            # Should be called with empty strings (configure_git_identity handles defaults)
            mock_config.assert_called_once_with("", "")


class TestProviderDistinction:
    """Test provider field is correctly returned and used."""

    @pytest.mark.asyncio
    async def test_github_provider_field(self):
        """Test GitHub credentials include provider='github'."""
        from auth import fetch_github_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch("auth._fetch_credential", new_callable=AsyncMock) as mock_fetch:
            mock_fetch.return_value = {
                "token": "ghp_test",
                "provider": "github",
            }

            result = await fetch_github_credentials(mock_context)
            assert result.get("provider") == "github"

    @pytest.mark.asyncio
    async def test_gitlab_provider_field(self):
        """Test GitLab credentials include provider='gitlab'."""
        from auth import fetch_gitlab_credentials
        from context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch("auth._fetch_credential", new_callable=AsyncMock) as mock_fetch:
            mock_fetch.return_value = {
                "token": "glpat-test",
                "provider": "gitlab",
            }

            result = await fetch_gitlab_credentials(mock_context)
            assert result.get("provider") == "gitlab"
