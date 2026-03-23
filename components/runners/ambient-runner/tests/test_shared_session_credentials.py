"""Tests for shared session credential scoping and cleanup."""

import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer
from threading import Thread
from unittest.mock import patch

import pytest

from ambient_runner.platform.auth import (
    _fetch_credential,
    clear_runtime_credentials,
    populate_runtime_credentials,
    sanitize_user_context,
)
from ambient_runner.platform.context import RunnerContext


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_context(
    session_id: str = "test-session",
    current_user_id: str = "",
    current_user_name: str = "",
    **env_overrides,
) -> RunnerContext:
    """Create a RunnerContext with optional current user and env overrides."""
    ctx = RunnerContext(
        session_id=session_id,
        workspace_path="/tmp/test",
        environment=env_overrides,
    )
    if current_user_id:
        ctx.set_current_user(current_user_id, current_user_name)
    return ctx


class _CredentialHandler(BaseHTTPRequestHandler):
    """HTTP handler that records request headers and returns canned credentials."""

    captured_headers: dict = {}
    response_body: dict = {}

    def do_GET(self):
        _CredentialHandler.captured_headers = dict(self.headers)
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        self.wfile.write(json.dumps(_CredentialHandler.response_body).encode())

    def log_message(self, format, *args):
        pass  # suppress server logs in test output


# ---------------------------------------------------------------------------
# RunnerContext.set_current_user
# ---------------------------------------------------------------------------


class TestSetCurrentUser:
    def test_set_current_user_stores_values(self):
        ctx = _make_context()
        ctx.set_current_user("user-123", "Alice")
        assert ctx.current_user_id == "user-123"
        assert ctx.current_user_name == "Alice"

    def test_set_current_user_can_clear(self):
        ctx = _make_context(current_user_id="user-123", current_user_name="Alice")
        ctx.set_current_user("", "")
        assert ctx.current_user_id == ""
        assert ctx.current_user_name == ""


# ---------------------------------------------------------------------------
# sanitize_user_context
# ---------------------------------------------------------------------------


class TestSanitizeUserContext:
    def test_sanitize_normal_values(self):
        uid, uname = sanitize_user_context("user@example.com", "Alice Smith")
        assert uid == "user@example.com"
        assert uname == "Alice Smith"

    def test_sanitize_strips_control_chars(self):
        uid, uname = sanitize_user_context("user\x00id", "Al\x1fice")
        assert "\x00" not in uid
        assert "\x1f" not in uname

    def test_sanitize_truncates_long_values(self):
        long_id = "a" * 300
        uid, _ = sanitize_user_context(long_id, "")
        assert len(uid) <= 255

    def test_sanitize_empty_values(self):
        uid, uname = sanitize_user_context("", "")
        assert uid == ""
        assert uname == ""


# ---------------------------------------------------------------------------
# clear_runtime_credentials
# ---------------------------------------------------------------------------


class TestClearRuntimeCredentials:
    def test_clears_all_credential_env_vars(self):
        keys = [
            "GITHUB_TOKEN",
            "GITLAB_TOKEN",
            "JIRA_API_TOKEN",
            "JIRA_URL",
            "JIRA_EMAIL",
            "USER_GOOGLE_EMAIL",
        ]
        try:
            for key in keys:
                os.environ[key] = "test-value"

            clear_runtime_credentials()

            for key in keys:
                assert key not in os.environ, f"{key} should be cleared"
        finally:
            for key in keys:
                os.environ.pop(key, None)

    def test_does_not_crash_when_vars_absent(self):
        for key in ["GITHUB_TOKEN", "GITLAB_TOKEN", "JIRA_API_TOKEN"]:
            os.environ.pop(key, None)
        # Should not raise
        clear_runtime_credentials()

    def test_does_not_clear_unrelated_vars(self):
        try:
            os.environ["PATH_BACKUP_TEST"] = "keep-me"
            os.environ["GITHUB_TOKEN"] = "remove-me"

            clear_runtime_credentials()

            assert "PATH_BACKUP_TEST" in os.environ
            assert os.environ["PATH_BACKUP_TEST"] == "keep-me"
        finally:
            os.environ.pop("PATH_BACKUP_TEST", None)
            os.environ.pop("GITHUB_TOKEN", None)


# ---------------------------------------------------------------------------
# _fetch_credential — X-Runner-Current-User header
# ---------------------------------------------------------------------------


class TestFetchCredentialHeaders:
    @pytest.mark.asyncio
    async def test_sends_current_user_header_when_set(self):
        """Verify _fetch_credential uses caller token and sends X-Runner-Current-User when context has both."""
        server = HTTPServer(("127.0.0.1", 0), _CredentialHandler)
        port = server.server_address[1]
        thread = Thread(target=server.handle_request, daemon=True)
        thread.start()

        _CredentialHandler.response_body = {"token": "gh-token-for-userB"}
        _CredentialHandler.captured_headers = {}

        try:
            with patch.dict(
                os.environ,
                {
                    "BACKEND_API_URL": f"http://127.0.0.1:{port}/api",
                    "PROJECT_NAME": "test-project",
                    "BOT_TOKEN": "fake-bot-token",
                },
            ):
                ctx = _make_context(
                    current_user_id="userB@example.com",
                    current_user_name="User B",
                )
                # Set caller token — runner uses this instead of BOT_TOKEN
                ctx.caller_token = "Bearer userB-oauth-token"
                result = await _fetch_credential(ctx, "github")

            assert result.get("token") == "gh-token-for-userB"
            assert _CredentialHandler.captured_headers.get("X-Runner-Current-User") == "userB@example.com"
            # Should use caller token, not BOT_TOKEN
            assert "Bearer userB-oauth-token" in _CredentialHandler.captured_headers.get("Authorization", "")
        finally:
            server.server_close()
            thread.join(timeout=2)

    @pytest.mark.asyncio
    async def test_omits_current_user_header_when_not_set(self):
        """Verify _fetch_credential omits X-Runner-Current-User for automated sessions."""
        server = HTTPServer(("127.0.0.1", 0), _CredentialHandler)
        port = server.server_address[1]
        thread = Thread(target=server.handle_request, daemon=True)
        thread.start()

        _CredentialHandler.response_body = {"token": "owner-token"}
        _CredentialHandler.captured_headers = {}

        try:
            with patch.dict(
                os.environ,
                {
                    "BACKEND_API_URL": f"http://127.0.0.1:{port}/api",
                    "PROJECT_NAME": "test-project",
                    "BOT_TOKEN": "fake-bot-token",
                },
            ):
                ctx = _make_context()  # no current_user_id
                result = await _fetch_credential(ctx, "github")

            assert result.get("token") == "owner-token"
            # Header should NOT be present
            assert "X-Runner-Current-User" not in _CredentialHandler.captured_headers
        finally:
            server.server_close()
            thread.join(timeout=2)

    @pytest.mark.asyncio
    async def test_returns_empty_when_backend_unavailable(self):
        """Verify graceful fallback when backend is unreachable."""
        with patch.dict(
            os.environ,
            {
                "BACKEND_API_URL": "http://127.0.0.1:1/api",
                "PROJECT_NAME": "test-project",
            },
        ):
            ctx = _make_context(current_user_id="user-123")
            result = await _fetch_credential(ctx, "github")

        assert result == {}


# ---------------------------------------------------------------------------
# populate_runtime_credentials + clear round-trip
# ---------------------------------------------------------------------------


class TestCredentialLifecycle:
    @pytest.mark.asyncio
    async def test_credentials_populated_then_cleared(self):
        """Simulate a turn: populate credentials, then clear after turn."""
        server = HTTPServer(("127.0.0.1", 0), _CredentialHandler)
        port = server.server_address[1]

        # We need to handle multiple requests (github, google, jira, gitlab)
        call_count = [0]
        responses = {
            "/github": {"token": "gh-tok"},
            "/google": {},
            "/jira": {"apiToken": "jira-tok", "url": "https://jira.example.com", "email": "j@example.com"},
            "/gitlab": {"token": "gl-tok"},
        }

        class MultiHandler(BaseHTTPRequestHandler):
            def do_GET(self):
                call_count[0] += 1
                # Extract credential type from URL path
                for key, resp in responses.items():
                    if key in self.path:
                        self.send_response(200)
                        self.send_header("Content-Type", "application/json")
                        self.end_headers()
                        self.wfile.write(json.dumps(resp).encode())
                        return
                self.send_response(404)
                self.end_headers()

            def log_message(self, format, *args):
                pass

        server = HTTPServer(("127.0.0.1", 0), MultiHandler)
        port = server.server_address[1]
        thread = Thread(target=lambda: [server.handle_request() for _ in range(4)], daemon=True)
        thread.start()

        try:
            with patch.dict(
                os.environ,
                {
                    "BACKEND_API_URL": f"http://127.0.0.1:{port}/api",
                    "PROJECT_NAME": "test-project",
                    "BOT_TOKEN": "fake-bot",
                },
            ):
                ctx = _make_context(current_user_id="userB")

                # Populate (simulates start of turn)
                await populate_runtime_credentials(ctx)

                # Verify credentials are set
                assert os.environ.get("GITHUB_TOKEN") == "gh-tok"
                assert os.environ.get("JIRA_API_TOKEN") == "jira-tok"
                assert os.environ.get("GITLAB_TOKEN") == "gl-tok"

                # Clear (simulates end of turn)
                clear_runtime_credentials()

                # Verify credentials are removed
                assert "GITHUB_TOKEN" not in os.environ
                assert "JIRA_API_TOKEN" not in os.environ
                assert "GITLAB_TOKEN" not in os.environ
                assert "JIRA_URL" not in os.environ
                assert "JIRA_EMAIL" not in os.environ
        finally:
            server.server_close()
            thread.join(timeout=2)
            # Cleanup any leaked env vars
            for key in ["GITHUB_TOKEN", "GITLAB_TOKEN", "JIRA_API_TOKEN", "JIRA_URL", "JIRA_EMAIL", "GIT_USER_NAME", "GIT_USER_EMAIL"]:
                os.environ.pop(key, None)
