"""
Unit tests for Git identity configuration from provider credentials.

Tests the configure_git_identity() function and the credential fetching
functions that now return user identity (userName, email) in addition to tokens.

Bug Fix: GitHub credentials aren't mounted to session - need git identity
         Also adds provider distinction (github vs gitlab)
"""

import os
import subprocess
from unittest.mock import AsyncMock, MagicMock, patch

import pytest


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
        from ambient_runner.platform.auth import configure_git_identity

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
        from ambient_runner.platform.auth import configure_git_identity

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
        from ambient_runner.platform.auth import configure_git_identity

        with patch("subprocess.run") as mock_run:
            mock_run.return_value = MagicMock(returncode=0)

            await configure_git_identity("  Jane Doe  ", "  jane@example.com  ")

            assert os.environ.get("GIT_USER_NAME") == "Jane Doe"
            assert os.environ.get("GIT_USER_EMAIL") == "jane@example.com"

    @pytest.mark.asyncio
    async def test_configure_git_identity_handles_subprocess_error(self):
        """Test git identity handles subprocess errors gracefully."""
        from ambient_runner.platform.auth import configure_git_identity

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
        from ambient_runner.platform.auth import fetch_github_credentials
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        mock_response = {
            "token": "ghp_test_token",
            "userName": "Test User",
            "email": "test@github.com",
            "provider": "github",
        }

        with patch(
            "ambient_runner.platform.auth._fetch_credential", new_callable=AsyncMock
        ) as mock_fetch:
            mock_fetch.return_value = mock_response

            result = await fetch_github_credentials(mock_context)

            assert result["token"] == "ghp_test_token"
            assert result["userName"] == "Test User"
            assert result["email"] == "test@github.com"
            assert result["provider"] == "github"

    @pytest.mark.asyncio
    async def test_fetch_github_token_delegates_to_fetch_github_credentials(self):
        """Test that fetch_github_token uses fetch_github_credentials."""
        from ambient_runner.platform.auth import fetch_github_token
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch(
            "ambient_runner.platform.auth.fetch_github_credentials",
            new_callable=AsyncMock,
        ) as mock_fetch:
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
        from ambient_runner.platform.auth import fetch_gitlab_credentials
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        mock_response = {
            "token": "glpat-test_token",
            "instanceUrl": "https://gitlab.com",
            "userName": "Test GitLab User",
            "email": "test@gitlab.com",
            "provider": "gitlab",
        }

        with patch(
            "ambient_runner.platform.auth._fetch_credential", new_callable=AsyncMock
        ) as mock_fetch:
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
        from ambient_runner.platform.auth import fetch_gitlab_token
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch(
            "ambient_runner.platform.auth.fetch_gitlab_credentials",
            new_callable=AsyncMock,
        ) as mock_fetch:
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
        from ambient_runner.platform.auth import populate_runtime_credentials
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        github_creds = {
            "token": "ghp_test",
            "userName": "GitHub User",
            "email": "user@github.com",
            "provider": "github",
        }

        with (
            patch(
                "ambient_runner.platform.auth.fetch_google_credentials",
                new_callable=AsyncMock,
            ) as mock_google,
            patch(
                "ambient_runner.platform.auth.fetch_jira_credentials",
                new_callable=AsyncMock,
            ) as mock_jira,
            patch(
                "ambient_runner.platform.auth.fetch_gitlab_credentials",
                new_callable=AsyncMock,
            ) as mock_gitlab,
            patch(
                "ambient_runner.platform.auth.fetch_github_credentials",
                new_callable=AsyncMock,
            ) as mock_github,
            patch(
                "ambient_runner.platform.auth.configure_git_identity",
                new_callable=AsyncMock,
            ) as mock_config,
            patch(
                "ambient_runner.platform.auth.install_git_credential_helper",
            ) as mock_cred_helper,
        ):
            mock_google.return_value = {}
            mock_jira.return_value = {}
            mock_gitlab.return_value = {}
            mock_github.return_value = github_creds

            await populate_runtime_credentials(mock_context)

            # Verify configure_git_identity was called with GitHub user info
            mock_config.assert_called_once_with("GitHub User", "user@github.com")
            # Verify credential helper was installed
            mock_cred_helper.assert_called_once()

    @pytest.mark.asyncio
    async def test_git_identity_from_gitlab_when_no_github(self):
        """Test git identity is configured from GitLab when GitHub not available."""
        from ambient_runner.platform.auth import populate_runtime_credentials
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        gitlab_creds = {
            "token": "glpat-test",
            "userName": "GitLab User",
            "email": "user@gitlab.com",
            "provider": "gitlab",
        }

        with (
            patch(
                "ambient_runner.platform.auth.fetch_google_credentials",
                new_callable=AsyncMock,
            ) as mock_google,
            patch(
                "ambient_runner.platform.auth.fetch_jira_credentials",
                new_callable=AsyncMock,
            ) as mock_jira,
            patch(
                "ambient_runner.platform.auth.fetch_gitlab_credentials",
                new_callable=AsyncMock,
            ) as mock_gitlab,
            patch(
                "ambient_runner.platform.auth.fetch_github_credentials",
                new_callable=AsyncMock,
            ) as mock_github,
            patch(
                "ambient_runner.platform.auth.configure_git_identity",
                new_callable=AsyncMock,
            ) as mock_config,
            patch(
                "ambient_runner.platform.auth.install_git_credential_helper",
            ),
        ):
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
        from ambient_runner.platform.auth import populate_runtime_credentials
        from ambient_runner.platform.context import RunnerContext

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

        with (
            patch(
                "ambient_runner.platform.auth.fetch_google_credentials",
                new_callable=AsyncMock,
            ) as mock_google,
            patch(
                "ambient_runner.platform.auth.fetch_jira_credentials",
                new_callable=AsyncMock,
            ) as mock_jira,
            patch(
                "ambient_runner.platform.auth.fetch_gitlab_credentials",
                new_callable=AsyncMock,
            ) as mock_gitlab,
            patch(
                "ambient_runner.platform.auth.fetch_github_credentials",
                new_callable=AsyncMock,
            ) as mock_github,
            patch(
                "ambient_runner.platform.auth.configure_git_identity",
                new_callable=AsyncMock,
            ) as mock_config,
            patch(
                "ambient_runner.platform.auth.install_git_credential_helper",
            ),
        ):
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
        from ambient_runner.platform.auth import populate_runtime_credentials
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with (
            patch(
                "ambient_runner.platform.auth.fetch_google_credentials",
                new_callable=AsyncMock,
            ) as mock_google,
            patch(
                "ambient_runner.platform.auth.fetch_jira_credentials",
                new_callable=AsyncMock,
            ) as mock_jira,
            patch(
                "ambient_runner.platform.auth.fetch_gitlab_credentials",
                new_callable=AsyncMock,
            ) as mock_gitlab,
            patch(
                "ambient_runner.platform.auth.fetch_github_credentials",
                new_callable=AsyncMock,
            ) as mock_github,
            patch(
                "ambient_runner.platform.auth.configure_git_identity",
                new_callable=AsyncMock,
            ) as mock_config,
            patch(
                "ambient_runner.platform.auth.install_git_credential_helper",
            ),
        ):
            mock_google.return_value = {}
            mock_jira.return_value = {}
            mock_gitlab.return_value = {}
            mock_github.return_value = {}

            await populate_runtime_credentials(mock_context)

            # Should be called with empty strings (configure_git_identity handles defaults)
            mock_config.assert_called_once_with("", "")


class TestInstallGitCredentialHelper:
    """Test install_git_credential_helper function."""

    @pytest.fixture(autouse=True)
    def reset_guard(self):
        """Reset the module-level installation guard between tests."""
        import ambient_runner.platform.auth as auth_mod

        auth_mod._credential_helper_installed = False
        yield
        auth_mod._credential_helper_installed = False

    def test_creates_helper_script_and_configures_git(self):
        """Test that the credential helper script is written and git is configured."""
        from ambient_runner.platform.auth import (
            install_git_credential_helper,
            _GIT_CREDENTIAL_HELPER_PATH,
        )

        with (
            patch("subprocess.run") as mock_run,
            patch("pathlib.Path.write_text") as mock_write,
            patch("pathlib.Path.chmod") as mock_chmod,
        ):
            mock_run.return_value = MagicMock(returncode=0)

            install_git_credential_helper()

            # Verify script was written
            mock_write.assert_called_once()
            script_content = mock_write.call_args[0][0]
            assert "GITHUB_TOKEN" in script_content
            assert "GITLAB_TOKEN" in script_content
            assert "x-access-token" in script_content
            assert "oauth2" in script_content

            # Verify chmod was called (755)
            mock_chmod.assert_called_once()

            # Verify git config was called
            mock_run.assert_called_once()
            assert "credential.helper" in mock_run.call_args[0][0]
            assert _GIT_CREDENTIAL_HELPER_PATH in mock_run.call_args[0][0]

    def test_skips_when_already_installed(self):
        """Test that repeated calls are no-ops after first install."""
        import ambient_runner.platform.auth as auth_mod

        auth_mod._credential_helper_installed = True

        with patch("pathlib.Path.write_text") as mock_write:
            auth_mod.install_git_credential_helper()
            mock_write.assert_not_called()

    def test_handles_errors_gracefully(self):
        """Test that errors during installation are caught and logged."""
        from ambient_runner.platform.auth import install_git_credential_helper

        with patch("pathlib.Path.write_text", side_effect=OSError("permission denied")):
            # Should not raise
            install_git_credential_helper()


class TestProviderDistinction:
    """Test provider field is correctly returned and used."""

    @pytest.mark.asyncio
    async def test_github_provider_field(self):
        """Test GitHub credentials include provider='github'."""
        from ambient_runner.platform.auth import fetch_github_credentials
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch(
            "ambient_runner.platform.auth._fetch_credential", new_callable=AsyncMock
        ) as mock_fetch:
            mock_fetch.return_value = {
                "token": "ghp_test",
                "provider": "github",
            }

            result = await fetch_github_credentials(mock_context)
            assert result.get("provider") == "github"

    @pytest.mark.asyncio
    async def test_gitlab_provider_field(self):
        """Test GitLab credentials include provider='gitlab'."""
        from ambient_runner.platform.auth import fetch_gitlab_credentials
        from ambient_runner.platform.context import RunnerContext

        mock_context = MagicMock(spec=RunnerContext)
        mock_context.session_id = "test-session"

        with patch(
            "ambient_runner.platform.auth._fetch_credential", new_callable=AsyncMock
        ) as mock_fetch:
            mock_fetch.return_value = {
                "token": "glpat-test",
                "provider": "gitlab",
            }

            result = await fetch_gitlab_credentials(mock_context)
            assert result.get("provider") == "gitlab"
