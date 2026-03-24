"""Unit tests for app.py initial prompt dispatch functions."""

import json
import os
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ambient_runner.app import (
    _auto_execute_initial_prompt,
    _push_initial_prompt_via_grpc,
    _push_initial_prompt_via_http,
)


# ---------------------------------------------------------------------------
# _push_initial_prompt_via_grpc
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestPushInitialPromptViaGRPC:
    async def test_pushes_user_event_with_prompt_content(self):
        mock_result = MagicMock()
        mock_result.seq = 42

        mock_client = MagicMock()
        mock_client.session_messages.push.return_value = mock_result
        mock_client.close = MagicMock()

        mock_cls = MagicMock()
        mock_cls.from_env.return_value = mock_client

        with patch("ambient_runner._grpc_client.AmbientGRPCClient", mock_cls):
            await _push_initial_prompt_via_grpc("hello world", "sess-1")

        mock_client.session_messages.push.assert_called_once()
        call = mock_client.session_messages.push.call_args
        assert call[0][0] == "sess-1"
        assert call[1]["event_type"] == "user"
        payload = json.loads(call[1]["payload"])
        assert payload["threadId"] == "sess-1"
        assert "runId" in payload
        assert len(payload["messages"]) == 1
        assert payload["messages"][0]["role"] == "user"
        assert payload["messages"][0]["content"] == "hello world"

    async def test_closes_client_after_push(self):
        mock_result = MagicMock()
        mock_result.seq = 1
        mock_client = MagicMock()
        mock_client.session_messages.push.return_value = mock_result
        mock_client.close = MagicMock()

        mock_cls = MagicMock()
        mock_cls.from_env.return_value = mock_client

        with patch("ambient_runner._grpc_client.AmbientGRPCClient", mock_cls):
            await _push_initial_prompt_via_grpc("prompt", "sess-close")

        mock_client.close.assert_called_once()

    async def test_does_not_raise_on_grpc_error(self):
        mock_cls = MagicMock()
        mock_cls.from_env.side_effect = RuntimeError("connection refused")

        with patch("ambient_runner._grpc_client.AmbientGRPCClient", mock_cls):
            await _push_initial_prompt_via_grpc("prompt", "sess-err")

    async def test_handles_none_push_result(self):
        mock_client = MagicMock()
        mock_client.session_messages.push.return_value = None
        mock_client.close = MagicMock()

        mock_cls = MagicMock()
        mock_cls.from_env.return_value = mock_client

        with patch("ambient_runner._grpc_client.AmbientGRPCClient", mock_cls):
            await _push_initial_prompt_via_grpc("prompt", "sess-none")

        mock_client.close.assert_called_once()


# ---------------------------------------------------------------------------
# _push_initial_prompt_via_http
# ---------------------------------------------------------------------------


def _make_aiohttp_session(status: int = 200, text: str = "ok"):
    """Build a mock aiohttp.ClientSession that works with async-with on both
    the session itself and session.post(...)."""
    mock_resp = AsyncMock()
    mock_resp.status = status
    mock_resp.text = AsyncMock(return_value=text)

    post_ctx = MagicMock()
    post_ctx.__aenter__ = AsyncMock(return_value=mock_resp)
    post_ctx.__aexit__ = AsyncMock(return_value=False)

    mock_session = MagicMock()
    mock_session.__aenter__ = AsyncMock(return_value=mock_session)
    mock_session.__aexit__ = AsyncMock(return_value=False)
    mock_session.post = MagicMock(return_value=post_ctx)

    return mock_session


@pytest.mark.asyncio
class TestPushInitialPromptViaHTTP:
    async def test_posts_to_localhost_agui_port(self):
        mock_session = _make_aiohttp_session()

        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(os.environ, {"INITIAL_PROMPT_DELAY_SECONDS": "0"}),
        ):
            await _push_initial_prompt_via_http("hi", "sess-http")

        mock_session.post.assert_called_once()
        call_url = mock_session.post.call_args[0][0]
        assert "localhost" in call_url

    async def test_includes_bot_token_in_auth_header_when_present(self):
        mock_session = _make_aiohttp_session()

        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(
                os.environ,
                {"BOT_TOKEN": "tok-abc", "INITIAL_PROMPT_DELAY_SECONDS": "0"},
            ),
        ):
            await _push_initial_prompt_via_http("hi", "sess-token")

        headers = mock_session.post.call_args[1]["headers"]
        assert headers.get("Authorization") == "Bearer tok-abc"

    async def test_no_auth_header_when_bot_token_absent(self):
        mock_session = _make_aiohttp_session()

        env_without_token = {k: v for k, v in os.environ.items() if k != "BOT_TOKEN"}
        env_without_token["INITIAL_PROMPT_DELAY_SECONDS"] = "0"
        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(os.environ, env_without_token, clear=True),
        ):
            await _push_initial_prompt_via_http("hi", "sess-no-token")

        headers = mock_session.post.call_args[1]["headers"]
        assert "Authorization" not in headers

    async def test_returns_after_max_retries_on_failure(self):
        mock_session = MagicMock()
        mock_session.__aenter__ = AsyncMock(return_value=mock_session)
        mock_session.__aexit__ = AsyncMock(return_value=False)
        mock_session.post = MagicMock(side_effect=Exception("connection refused"))

        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch("asyncio.sleep", new_callable=AsyncMock),
            patch.dict(os.environ, {"INITIAL_PROMPT_DELAY_SECONDS": "0"}),
        ):
            await _push_initial_prompt_via_http("hi", "sess-retry")

        assert mock_session.post.call_count == 8


# ---------------------------------------------------------------------------
# _auto_execute_initial_prompt — routing
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestAutoExecuteInitialPrompt:
    async def test_routes_to_grpc_when_url_set(self):
        with (
            patch(
                "ambient_runner.app._push_initial_prompt_via_grpc",
                new_callable=AsyncMock,
            ) as mock_grpc,
            patch(
                "ambient_runner.app._push_initial_prompt_via_http",
                new_callable=AsyncMock,
            ) as mock_http,
            patch.dict(os.environ, {"INITIAL_PROMPT_DELAY_SECONDS": "0"}),
        ):
            await _auto_execute_initial_prompt(
                "hello", "sess-1", grpc_url="localhost:9000"
            )

        mock_grpc.assert_awaited_once_with("hello", "sess-1")
        mock_http.assert_not_awaited()

    async def test_routes_to_http_when_no_grpc_url(self):
        with (
            patch(
                "ambient_runner.app._push_initial_prompt_via_grpc",
                new_callable=AsyncMock,
            ) as mock_grpc,
            patch(
                "ambient_runner.app._push_initial_prompt_via_http",
                new_callable=AsyncMock,
            ) as mock_http,
            patch.dict(os.environ, {"INITIAL_PROMPT_DELAY_SECONDS": "0"}),
        ):
            await _auto_execute_initial_prompt("hello", "sess-1", grpc_url="")

        mock_http.assert_awaited_once_with("hello", "sess-1")
        mock_grpc.assert_not_awaited()

    async def test_routes_to_http_when_grpc_url_default(self):
        with (
            patch(
                "ambient_runner.app._push_initial_prompt_via_grpc",
                new_callable=AsyncMock,
            ) as mock_grpc,
            patch(
                "ambient_runner.app._push_initial_prompt_via_http",
                new_callable=AsyncMock,
            ) as mock_http,
            patch.dict(os.environ, {"INITIAL_PROMPT_DELAY_SECONDS": "0"}),
        ):
            await _auto_execute_initial_prompt("hello", "sess-1")

        mock_http.assert_awaited_once()
        mock_grpc.assert_not_awaited()
