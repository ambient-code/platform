"""
Unit tests for Google Workspace MCP authentication validation.

Tests the enhanced _check_mcp_authentication() logic that validates
token structure, expiry, and refresh token availability.
"""

import pytest
import json
import tempfile
import os
import sys
from pathlib import Path
from datetime import datetime, timedelta, timezone
from unittest.mock import patch

# Add parent directory to path to import main module
sys.path.insert(0, str(Path(__file__).parent.parent))


@pytest.fixture
def temp_workspace_creds(tmp_path):
    """Create temporary workspace credentials directory."""
    workspace_dir = tmp_path / ".google_workspace_mcp" / "credentials"
    workspace_dir.mkdir(parents=True)
    return workspace_dir / "credentials.json"


@pytest.fixture
def temp_secret_creds(tmp_path):
    """Create temporary secret mount credentials directory."""
    secret_dir = tmp_path / "app_secret" / ".google_workspace_mcp" / "credentials"
    secret_dir.mkdir(parents=True)
    return secret_dir / "credentials.json"


class TestGoogleAuthValidation:
    """Test Google Workspace MCP authentication validation."""

    def test_no_credentials_file(self, temp_workspace_creds, temp_secret_creds):
        """Test when credentials.json doesn't exist."""
        from main import _check_mcp_authentication

        # Patch Path to point to our temp locations (but files don't exist yet)
        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                elif "/app/.google_workspace_mcp" in str(path_str):
                    return temp_secret_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is False
        assert "not configured" in msg.lower()

    def test_empty_credentials_file(self, temp_workspace_creds):
        """Test when credentials.json exists but is empty."""
        from main import _check_mcp_authentication

        # Create empty file
        temp_workspace_creds.touch()

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is False
        assert "not configured" in msg.lower()  # Empty file treated as not configured

    def test_valid_unexpired_tokens(self, temp_workspace_creds):
        """Test with valid, unexpired tokens."""
        from main import _check_mcp_authentication

        # Create valid credentials
        creds = {
            "user@gmail.com": {
                "access_token": "ya29.valid_token",
                "refresh_token": "1//valid_refresh",
                "token_expiry": (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat(),
                "email": "user@gmail.com"
            }
        }

        temp_workspace_creds.write_text(json.dumps(creds))

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is True
        assert "user@gmail.com" in msg

    def test_valid_tokens_with_z_suffix(self, temp_workspace_creds):
        """Test timestamp with Z suffix is handled correctly."""
        from main import _check_mcp_authentication

        creds = {
            "user@gmail.com": {
                "access_token": "ya29.valid_token",
                "refresh_token": "1//valid_refresh",
                "token_expiry": "2026-12-31T23:59:59Z",  # Z-suffix format
                "email": "user@gmail.com"
            }
        }

        temp_workspace_creds.write_text(json.dumps(creds))

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is True
        assert "user@gmail.com" in msg

    def test_expired_token_with_refresh(self, temp_workspace_creds):
        """Test expired access token but valid refresh token."""
        from main import _check_mcp_authentication

        creds = {
            "user@gmail.com": {
                "access_token": "ya29.expired_token",
                "refresh_token": "1//valid_refresh",
                "token_expiry": (datetime.now(timezone.utc) - timedelta(hours=1)).isoformat(),
                "email": "user@gmail.com"
            }
        }

        temp_workspace_creds.write_text(json.dumps(creds))

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is None  # Needs refresh
        assert "refresh needed" in msg.lower()

    def test_expired_token_no_refresh(self, temp_workspace_creds):
        """Test expired token with no refresh token."""
        from main import _check_mcp_authentication

        creds = {
            "user@gmail.com": {
                "access_token": "ya29.expired_token",
                "refresh_token": "",
                "token_expiry": (datetime.now(timezone.utc) - timedelta(hours=1)).isoformat(),
                "email": "user@gmail.com"
            }
        }

        temp_workspace_creds.write_text(json.dumps(creds))

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is False
        assert "empty" in msg.lower()

    def test_missing_required_fields(self, temp_workspace_creds):
        """Test credentials missing required fields."""
        from main import _check_mcp_authentication

        creds = {
            "user@gmail.com": {
                "access_token": "ya29.token"
                # Missing refresh_token
            }
        }

        temp_workspace_creds.write_text(json.dumps(creds))

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is False
        assert "incomplete" in msg.lower()

    def test_corrupted_json(self, temp_workspace_creds):
        """Test corrupted JSON file."""
        from main import _check_mcp_authentication

        temp_workspace_creds.write_text("{ invalid json")

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is False
        assert "corrupted" in msg.lower()

    def test_placeholder_email_rejected(self, temp_workspace_creds):
        """Test that placeholder email 'user@example.com' is rejected."""
        from main import _check_mcp_authentication

        creds = {
            "user@example.com": {
                "access_token": "ya29.token",
                "refresh_token": "1//refresh",
                "email": "user@example.com"
            }
        }

        temp_workspace_creds.write_text(json.dumps(creds))

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is False
        assert "placeholder" in msg.lower()

    def test_malformed_timestamp(self, temp_workspace_creds):
        """Test that malformed timestamp returns None (uncertain) not True."""
        from main import _check_mcp_authentication

        creds = {
            "user@gmail.com": {
                "access_token": "ya29.token",
                "refresh_token": "1//refresh",
                "token_expiry": "not-a-valid-timestamp",
                "email": "user@gmail.com"
            }
        }

        temp_workspace_creds.write_text(json.dumps(creds))

        with patch('main.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            is_auth, msg = _check_mcp_authentication("google-workspace")

        assert is_auth is None  # Uncertain, not True
        assert "invalid" in msg.lower()


class TestUserEmailExtraction:
    """Test USER_GOOGLE_EMAIL environment variable setting."""

    @pytest.mark.asyncio
    async def test_email_extracted_and_set(self, temp_workspace_creds, monkeypatch):
        """Test email is extracted from credentials and set as env var."""
        from adapter import ClaudeCodeAdapter

        # Ensure env var starts clean
        monkeypatch.delenv("USER_GOOGLE_EMAIL", raising=False)

        # Create valid credentials
        creds = {
            "test.user@gmail.com": {
                "access_token": "ya29.token",
                "refresh_token": "1//refresh",
                "email": "test.user@gmail.com"
            }
        }
        temp_workspace_creds.write_text(json.dumps(creds))

        adapter = ClaudeCodeAdapter()

        with patch('adapter.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            await adapter._set_google_user_email()

        assert os.getenv("USER_GOOGLE_EMAIL") == "test.user@gmail.com"

    @pytest.mark.asyncio
    async def test_placeholder_email_skipped(self, temp_workspace_creds, monkeypatch):
        """Test that placeholder email is not set as env var."""
        from adapter import ClaudeCodeAdapter

        creds = {
            "user@example.com": {
                "access_token": "ya29.token",
                "refresh_token": "1//refresh",
                "email": "user@example.com"
            }
        }
        temp_workspace_creds.write_text(json.dumps(creds))

        # Clear env var using monkeypatch (auto-restored after test)
        monkeypatch.delenv("USER_GOOGLE_EMAIL", raising=False)

        adapter = ClaudeCodeAdapter()

        with patch('adapter.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            await adapter._set_google_user_email()

        # Should not be set to placeholder
        assert os.getenv("USER_GOOGLE_EMAIL") != "user@example.com"

    @pytest.mark.asyncio
    async def test_empty_dict_handled(self, temp_workspace_creds, monkeypatch):
        """Test that empty credentials dict is handled gracefully."""
        from adapter import ClaudeCodeAdapter

        # Ensure env var starts clean
        monkeypatch.delenv("USER_GOOGLE_EMAIL", raising=False)

        # Empty dict
        creds = {}
        temp_workspace_creds.write_text(json.dumps(creds))

        adapter = ClaudeCodeAdapter()

        with patch('adapter.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            # Should not raise exception
            await adapter._set_google_user_email()

    @pytest.mark.asyncio
    async def test_non_dict_structure_handled(self, temp_workspace_creds, monkeypatch):
        """Test that non-dict JSON structure is handled gracefully."""
        from adapter import ClaudeCodeAdapter

        # Ensure env var starts clean
        monkeypatch.delenv("USER_GOOGLE_EMAIL", raising=False)

        # List instead of dict
        temp_workspace_creds.write_text(json.dumps(["not", "a", "dict"]))

        adapter = ClaudeCodeAdapter()

        with patch('adapter.Path') as mock_path_class:
            def path_side_effect(path_str):
                if "/workspace/.google_workspace_mcp" in str(path_str):
                    return temp_workspace_creds
                return Path(path_str)

            mock_path_class.side_effect = path_side_effect
            # Should not raise exception
            await adapter._set_google_user_email()
