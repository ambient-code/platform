"""Unit tests for MCP authentication checks in claude_mcp."""

import json
from datetime import datetime, timedelta, timezone

from ambient_runner.bridges.claude.mcp import check_mcp_authentication


class TestGoogleWorkspaceAuth:
    """Test check_mcp_authentication for google-workspace."""

    def test_no_credentials_file(self, tmp_path, monkeypatch):
        empty_dir = tmp_path / "empty"
        empty_dir.mkdir()
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._WORKSPACE_CREDS_DIR", empty_dir
        )
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._SECRET_CREDS_DIR", empty_dir
        )
        is_auth, msg = check_mcp_authentication("google-workspace")
        assert is_auth is False
        assert "not configured" in msg

    def test_valid_credentials(self, tmp_path, monkeypatch):
        creds_dir = tmp_path / "workspace_creds"
        creds_dir.mkdir(parents=True)
        creds_file = creds_dir / "user@example.org.json"

        future = (datetime.now(timezone.utc) + timedelta(hours=1)).isoformat()
        creds_file.write_text(
            json.dumps(
                {
                    "token": "ya29.valid",
                    "refresh_token": "1//refresh",
                    "expiry": future,
                }
            )
        )

        monkeypatch.setenv("USER_GOOGLE_EMAIL", "user@example.org")
        empty_dir = tmp_path / "empty"
        empty_dir.mkdir()
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._WORKSPACE_CREDS_DIR", creds_dir
        )
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._SECRET_CREDS_DIR", empty_dir
        )

        is_auth, msg = check_mcp_authentication("google-workspace")
        assert is_auth is True
        assert "user@example.org" in msg

    def test_placeholder_email_rejected(self, tmp_path, monkeypatch):
        creds_dir = tmp_path / "workspace_creds"
        creds_dir.mkdir(parents=True)
        creds_file = creds_dir / "credentials.json"
        creds_file.write_text(json.dumps({"token": "t", "refresh_token": "r"}))

        monkeypatch.setenv("USER_GOOGLE_EMAIL", "user@example.com")
        empty_dir = tmp_path / "empty"
        empty_dir.mkdir()
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._WORKSPACE_CREDS_DIR", creds_dir
        )
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._SECRET_CREDS_DIR", empty_dir
        )

        is_auth, msg = check_mcp_authentication("google-workspace")
        assert is_auth is False
        assert "USER_GOOGLE_EMAIL" in msg

    def test_missing_tokens(self, tmp_path, monkeypatch):
        creds_dir = tmp_path / "workspace_creds"
        creds_dir.mkdir(parents=True)
        creds_file = creds_dir / "real@user.com.json"
        creds_file.write_text(json.dumps({"token": "", "refresh_token": ""}))

        monkeypatch.setenv("USER_GOOGLE_EMAIL", "real@user.com")
        empty_dir = tmp_path / "empty"
        empty_dir.mkdir()
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._WORKSPACE_CREDS_DIR", creds_dir
        )
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._SECRET_CREDS_DIR", empty_dir
        )

        is_auth, msg = check_mcp_authentication("google-workspace")
        assert is_auth is False
        assert "incomplete" in msg.lower()

    def test_expired_token_with_refresh(self, tmp_path, monkeypatch):
        creds_dir = tmp_path / "workspace_creds"
        creds_dir.mkdir(parents=True)
        creds_file = creds_dir / "user@corp.com.json"

        past = (datetime.now(timezone.utc) - timedelta(hours=1)).isoformat()
        creds_file.write_text(
            json.dumps({"token": "t", "refresh_token": "r", "expiry": past})
        )

        monkeypatch.setenv("USER_GOOGLE_EMAIL", "user@corp.com")
        empty_dir = tmp_path / "empty"
        empty_dir.mkdir()
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._WORKSPACE_CREDS_DIR", creds_dir
        )
        monkeypatch.setattr(
            "ambient_runner.bridges.claude.mcp._SECRET_CREDS_DIR", empty_dir
        )

        is_auth, msg = check_mcp_authentication("google-workspace")
        # Should be None (needs refresh) not False
        assert is_auth is None
        assert "refresh" in msg.lower()


class TestJiraAuth:
    """Test check_mcp_authentication for jira/mcp-atlassian."""

    def test_jira_credentials_present(self, monkeypatch):
        monkeypatch.setenv("JIRA_URL", "https://jira.example.com")
        monkeypatch.setenv("JIRA_API_TOKEN", "token-123")

        is_auth, msg = check_mcp_authentication("jira")
        assert is_auth is True
        assert "configured" in msg.lower()

    def test_jira_no_credentials(self, monkeypatch):
        monkeypatch.delenv("JIRA_URL", raising=False)
        monkeypatch.delenv("JIRA_API_TOKEN", raising=False)
        monkeypatch.delenv("BACKEND_API_URL", raising=False)

        is_auth, msg = check_mcp_authentication("jira")
        assert is_auth is False
        assert "not configured" in msg.lower()

    def test_mcp_atlassian_alias(self, monkeypatch):
        monkeypatch.setenv("JIRA_URL", "https://jira.example.com")
        monkeypatch.setenv("JIRA_API_TOKEN", "token")

        is_auth, msg = check_mcp_authentication("mcp-atlassian")
        assert is_auth is True


class TestUnknownServer:
    """Test check_mcp_authentication for unknown servers."""

    def test_unknown_returns_none(self):
        is_auth, msg = check_mcp_authentication("some-random-server")
        assert is_auth is None
        assert msg is None
