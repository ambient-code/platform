"""Tests for shared session credential scoping and cleanup."""

import json
import os
from http.server import BaseHTTPRequestHandler, HTTPServer
from io import BytesIO
from threading import Thread
from unittest.mock import AsyncMock, MagicMock, patch
from urllib.error import HTTPError

import pytest

from ambient_runner.platform.auth import (
    _GITHUB_TOKEN_FILE,
    _GITLAB_TOKEN_FILE,
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
# Token file lifecycle (mid-run refresh support)
# ---------------------------------------------------------------------------


class TestTokenFiles:
    """Token files let the git credential helper pick up mid-run refreshes.

    The CLI subprocess is spawned once and its environment is fixed at that
    point. Updating os.environ later does not propagate into the subprocess.
    Writing tokens to files allows the credential helper (which runs fresh for
    every git operation) to always use the latest token.
    """

    def _cleanup(self):
        """Remove token files created during tests."""
        _GITHUB_TOKEN_FILE.unlink(missing_ok=True)
        _GITLAB_TOKEN_FILE.unlink(missing_ok=True)

    @pytest.mark.asyncio
    async def test_populate_writes_github_token_file(self):
        """populate_runtime_credentials writes GITHUB_TOKEN to the token file."""
        self._cleanup()
        try:
            with patch("ambient_runner.platform.auth._fetch_credential") as mock_fetch:

                async def _creds(ctx, ctype):
                    if ctype == "github":
                        return {
                            "token": "gh-mid-run-token",
                            "userName": "user",
                            "email": "u@example.com",
                        }
                    return {}

                mock_fetch.side_effect = _creds
                ctx = _make_context()
                await populate_runtime_credentials(ctx)

            assert _GITHUB_TOKEN_FILE.exists()
            assert _GITHUB_TOKEN_FILE.read_text() == "gh-mid-run-token"
        finally:
            self._cleanup()
            for key in ["GITHUB_TOKEN", "GIT_USER_NAME", "GIT_USER_EMAIL"]:
                os.environ.pop(key, None)

    @pytest.mark.asyncio
    async def test_populate_writes_gitlab_token_file(self):
        """populate_runtime_credentials writes GITLAB_TOKEN to the token file."""
        self._cleanup()
        try:
            with patch("ambient_runner.platform.auth._fetch_credential") as mock_fetch:

                async def _creds(ctx, ctype):
                    if ctype == "gitlab":
                        return {
                            "token": "gl-mid-run-token",
                            "userName": "user",
                            "email": "u@example.com",
                        }
                    return {}

                mock_fetch.side_effect = _creds
                ctx = _make_context()
                await populate_runtime_credentials(ctx)

            assert _GITLAB_TOKEN_FILE.exists()
            assert _GITLAB_TOKEN_FILE.read_text() == "gl-mid-run-token"
        finally:
            self._cleanup()
            for key in ["GITLAB_TOKEN", "GIT_USER_NAME", "GIT_USER_EMAIL"]:
                os.environ.pop(key, None)

    def test_clear_removes_token_files(self):
        """clear_runtime_credentials removes the token files written at populate time."""
        _GITHUB_TOKEN_FILE.write_text("old-token")
        _GITLAB_TOKEN_FILE.write_text("old-gl-token")
        try:
            clear_runtime_credentials()
            assert not _GITHUB_TOKEN_FILE.exists(), (
                "GitHub token file should be removed"
            )
            assert not _GITLAB_TOKEN_FILE.exists(), (
                "GitLab token file should be removed"
            )
        finally:
            self._cleanup()

    def test_clear_does_not_crash_when_token_files_absent(self):
        """clear_runtime_credentials succeeds even if the token files don't exist."""
        self._cleanup()
        # Should not raise
        clear_runtime_credentials()

    @pytest.mark.asyncio
    async def test_second_populate_overwrites_token_file(self):
        """A second populate_runtime_credentials call overwrites the stale token file.

        This is the mid-run refresh scenario: the MCP tool calls populate again
        with a fresh token and the file must reflect the new value.
        """
        self._cleanup()
        try:
            call_num = [0]

            async def _creds(ctx, ctype):
                if ctype == "github":
                    call_num[0] += 1
                    return {
                        "token": f"gh-token-{call_num[0]}",
                        "userName": "u",
                        "email": "u@e.com",
                    }
                return {}

            with patch(
                "ambient_runner.platform.auth._fetch_credential", side_effect=_creds
            ):
                ctx = _make_context()
                await populate_runtime_credentials(ctx)
                assert _GITHUB_TOKEN_FILE.read_text() == "gh-token-1"

                await populate_runtime_credentials(ctx)
                assert _GITHUB_TOKEN_FILE.read_text() == "gh-token-2"
        finally:
            self._cleanup()
            for key in ["GITHUB_TOKEN", "GIT_USER_NAME", "GIT_USER_EMAIL"]:
                os.environ.pop(key, None)


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
            assert (
                _CredentialHandler.captured_headers.get("X-Runner-Current-User")
                == "userB@example.com"
            )
            # Should use caller token, not BOT_TOKEN
            assert (
                "Bearer userB-oauth-token"
                in _CredentialHandler.captured_headers.get("Authorization", "")
            )
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
            "/jira": {
                "apiToken": "jira-tok",
                "url": "https://jira.example.com",
                "email": "j@example.com",
            },
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
        thread = Thread(
            target=lambda: [server.handle_request() for _ in range(4)], daemon=True
        )
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
            for key in [
                "GITHUB_TOKEN",
                "GITLAB_TOKEN",
                "JIRA_API_TOKEN",
                "JIRA_URL",
                "JIRA_EMAIL",
                "GIT_USER_NAME",
                "GIT_USER_EMAIL",
            ]:
                os.environ.pop(key, None)


# ---------------------------------------------------------------------------
# _fetch_credential — auth failure propagation (issue #1043)
# ---------------------------------------------------------------------------


class TestFetchCredentialAuthFailures:
    @pytest.mark.asyncio
    async def test_raises_permission_error_on_401_without_caller_token(
        self, monkeypatch
    ):
        """_fetch_credential raises PermissionError when backend returns 401 with BOT_TOKEN."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "bot-token")

        ctx = _make_context(session_id="sess-1")
        # No caller token — uses BOT_TOKEN directly

        err = HTTPError(
            "http://backend.svc.cluster.local/api/...",
            401,
            "Unauthorized",
            {},
            BytesIO(b""),
        )
        with patch("urllib.request.urlopen", side_effect=err):
            with pytest.raises(
                PermissionError, match="authentication failed with HTTP 401"
            ):
                await _fetch_credential(ctx, "github")

    @pytest.mark.asyncio
    async def test_raises_permission_error_on_403_without_caller_token(
        self, monkeypatch
    ):
        """_fetch_credential raises PermissionError when backend returns 403 with BOT_TOKEN."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "bot-token")

        ctx = _make_context(session_id="sess-1")

        err = HTTPError(
            "http://backend.svc.cluster.local/api/...",
            403,
            "Forbidden",
            {},
            BytesIO(b""),
        )
        with patch("urllib.request.urlopen", side_effect=err):
            with pytest.raises(
                PermissionError, match="authentication failed with HTTP 403"
            ):
                await _fetch_credential(ctx, "google")

    @pytest.mark.asyncio
    async def test_raises_permission_error_when_caller_and_bot_both_fail(
        self, monkeypatch
    ):
        """_fetch_credential raises PermissionError when caller token 401s and BOT_TOKEN also fails."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "bot-token")

        ctx = _make_context(session_id="sess-1", current_user_id="user@example.com")
        ctx.caller_token = "Bearer expired-caller-token"

        caller_err = HTTPError("http://...", 401, "Unauthorized", {}, BytesIO(b""))
        fallback_err = HTTPError("http://...", 403, "Forbidden", {}, BytesIO(b""))

        with patch("urllib.request.urlopen", side_effect=[caller_err, fallback_err]):
            with pytest.raises(
                PermissionError,
                match="caller token expired and BOT_TOKEN fallback also failed",
            ):
                await _fetch_credential(ctx, "github")

    @pytest.mark.asyncio
    async def test_does_not_raise_on_non_auth_http_errors(self, monkeypatch):
        """_fetch_credential returns {} for non-auth HTTP errors (404, 500, etc.)."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        ctx = _make_context(session_id="sess-1")

        err = HTTPError("http://...", 404, "Not Found", {}, BytesIO(b""))
        with patch("urllib.request.urlopen", side_effect=err):
            result = await _fetch_credential(ctx, "github")

        assert result == {}

    @pytest.mark.asyncio
    async def test_caller_token_fallback_succeeds_when_bot_token_works(
        self, monkeypatch
    ):
        """_fetch_credential returns data when caller token 401s but BOT_TOKEN fallback succeeds."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")
        monkeypatch.setenv("BOT_TOKEN", "valid-bot-token")

        ctx = _make_context(session_id="sess-1", current_user_id="user@example.com")
        ctx.caller_token = "Bearer expired-caller-token"

        caller_err = HTTPError("http://...", 401, "Unauthorized", {}, BytesIO(b""))

        mock_response = MagicMock()
        mock_response.read.return_value = json.dumps(
            {"token": "gh-tok-via-bot"}
        ).encode()
        mock_response.__enter__ = lambda s: s
        mock_response.__exit__ = MagicMock(return_value=False)

        with patch("urllib.request.urlopen", side_effect=[caller_err, mock_response]):
            result = await _fetch_credential(ctx, "github")

        assert result.get("token") == "gh-tok-via-bot"


# ---------------------------------------------------------------------------
# populate_runtime_credentials — raises on auth failure (issue #1043)
# ---------------------------------------------------------------------------


class TestPopulateRuntimeCredentialsAuthFailures:
    @pytest.mark.asyncio
    async def test_raises_when_github_auth_fails(self, monkeypatch):
        """populate_runtime_credentials raises PermissionError when GitHub auth fails."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        ctx = _make_context(session_id="sess-1")

        async def _fail_github(context, cred_type):
            if cred_type == "github":
                raise PermissionError("github authentication failed with HTTP 401")
            return {}

        with patch(
            "ambient_runner.platform.auth._fetch_credential", side_effect=_fail_github
        ):
            with pytest.raises(
                PermissionError,
                match="Credential refresh failed due to authentication errors",
            ):
                await populate_runtime_credentials(ctx)

    @pytest.mark.asyncio
    async def test_raises_when_multiple_providers_fail(self, monkeypatch):
        """populate_runtime_credentials raises PermissionError listing all auth failures."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        ctx = _make_context(session_id="sess-1")

        async def _fail_all(context, cred_type):
            raise PermissionError(f"{cred_type} authentication failed with HTTP 401")

        with patch(
            "ambient_runner.platform.auth._fetch_credential", side_effect=_fail_all
        ):
            with pytest.raises(PermissionError) as exc_info:
                await populate_runtime_credentials(ctx)

        msg = str(exc_info.value)
        assert "authentication errors" in msg

    @pytest.mark.asyncio
    async def test_succeeds_when_all_credentials_empty_no_auth_error(self, monkeypatch):
        """populate_runtime_credentials does not raise when credentials are simply missing (not auth failures)."""
        monkeypatch.setenv("BACKEND_API_URL", "http://backend.svc.cluster.local/api")
        monkeypatch.setenv("PROJECT_NAME", "test-project")

        ctx = _make_context(session_id="sess-1")

        with patch("ambient_runner.platform.auth._fetch_credential", return_value={}):
            # Should not raise — empty credentials just means no integrations configured
            await populate_runtime_credentials(ctx)


# ---------------------------------------------------------------------------
# refresh_credentials_tool — reports isError on auth failure (issue #1043)
# ---------------------------------------------------------------------------


class TestRefreshCredentialsTool:
    def _make_tool_decorator(self):
        """Create a mock sdk_tool decorator that preserves the function."""

        def mock_tool(name, description, schema):
            def decorator(func):
                return func

            return decorator

        return mock_tool

    @pytest.mark.asyncio
    async def test_returns_is_error_on_auth_failure(self):
        """refresh_credentials_tool returns isError=True when populate_runtime_credentials raises PermissionError."""
        from ambient_runner.bridges.claude.tools import create_refresh_credentials_tool

        mock_context = MagicMock()
        tool_fn = create_refresh_credentials_tool(
            mock_context, self._make_tool_decorator()
        )

        with patch(
            "ambient_runner.platform.auth.populate_runtime_credentials",
            new_callable=AsyncMock,
            side_effect=PermissionError("github authentication failed with HTTP 401"),
        ):
            result = await tool_fn({})

        assert result.get("isError") is True
        assert "github authentication failed" in result["content"][0]["text"]

    @pytest.mark.asyncio
    async def test_returns_success_on_successful_refresh(self):
        """refresh_credentials_tool returns success message when credentials refresh succeeds."""
        from ambient_runner.bridges.claude.tools import create_refresh_credentials_tool

        mock_context = MagicMock()
        tool_fn = create_refresh_credentials_tool(
            mock_context, self._make_tool_decorator()
        )

        with (
            patch(
                "ambient_runner.platform.auth.populate_runtime_credentials",
                new_callable=AsyncMock,
            ),
            patch(
                "ambient_runner.platform.utils.get_active_integrations",
                return_value=["github", "jira"],
            ),
        ):
            result = await tool_fn({})

        assert result.get("isError") is None or result.get("isError") is False
        assert "successfully" in result["content"][0]["text"].lower()
