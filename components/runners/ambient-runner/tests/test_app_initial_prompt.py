"""Unit tests for app.py initial prompt dispatch functions.

Coverage targets:
- _push_initial_prompt_via_grpc: happy path, push raises (client still closed),
  None result, from_env error, offloaded to executor (non-blocking)
- _push_initial_prompt_via_http: happy path, missing env vars bail, bot token,
  no token, retry-on-failure (8 attempts), non-transient error early return
- _auto_execute_initial_prompt: routes to gRPC when grpc_url set,
  routes to HTTP when grpc_url empty, routes to HTTP when grpc_url defaulted
- create_ambient_app lifespan: gRPC OFF path (no AMBIENT_GRPC_ENABLED env),
  gRPC ON path (AMBIENT_GRPC_ENABLED=true + AMBIENT_GRPC_URL)
"""

import asyncio
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

    async def test_closes_client_even_when_push_raises(self):
        """client.close() must be called in finally even if push() raises."""
        mock_client = MagicMock()
        mock_client.session_messages.push.side_effect = RuntimeError("rpc failed")
        mock_client.close = MagicMock()

        mock_cls = MagicMock()
        mock_cls.from_env.return_value = mock_client

        with patch("ambient_runner._grpc_client.AmbientGRPCClient", mock_cls):
            await _push_initial_prompt_via_grpc("prompt", "sess-push-raises")

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

    async def test_push_offloaded_to_executor(self):
        """The blocking push must be offloaded via run_in_executor, not called inline."""
        mock_client = MagicMock()
        mock_client.session_messages.push.return_value = MagicMock(seq=1)
        mock_client.close = MagicMock()

        mock_cls = MagicMock()
        mock_cls.from_env.return_value = mock_client

        executor_calls = []
        real_loop = asyncio.get_event_loop()

        original_run_in_executor = real_loop.run_in_executor

        async def capturing_executor(executor, fn, *args):
            executor_calls.append(fn)
            return await original_run_in_executor(executor, fn, *args)

        with (
            patch("ambient_runner._grpc_client.AmbientGRPCClient", mock_cls),
            patch.object(real_loop, "run_in_executor", side_effect=capturing_executor),
        ):
            await _push_initial_prompt_via_grpc("prompt", "sess-executor")

        assert len(executor_calls) == 1


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
    async def test_posts_to_backend_url(self):
        mock_session = _make_aiohttp_session()

        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(
                os.environ,
                {
                    "INITIAL_PROMPT_DELAY_SECONDS": "0",
                    "BACKEND_API_URL": "http://backend:8080",
                    "PROJECT_NAME": "ambient-code",
                },
            ),
        ):
            await _push_initial_prompt_via_http("hi", "sess-http")

        mock_session.post.assert_called_once()
        call_url = mock_session.post.call_args[0][0]
        assert "backend:8080" in call_url
        assert "ambient-code" in call_url
        assert "sess-http" in call_url

    async def test_bails_early_when_backend_url_missing(self):
        """If BACKEND_API_URL is not set, function logs error and returns without posting."""
        mock_session = _make_aiohttp_session()

        env = {
            "INITIAL_PROMPT_DELAY_SECONDS": "0",
            "PROJECT_NAME": "ambient-code",
        }
        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(os.environ, env, clear=True),
        ):
            await _push_initial_prompt_via_http("hi", "sess-no-backend")

        mock_session.post.assert_not_called()

    async def test_bails_early_when_project_name_missing(self):
        """If PROJECT_NAME is not set, function logs error and returns without posting."""
        mock_session = _make_aiohttp_session()

        env = {
            "INITIAL_PROMPT_DELAY_SECONDS": "0",
            "BACKEND_API_URL": "http://backend:8080",
        }
        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(os.environ, env, clear=True),
        ):
            await _push_initial_prompt_via_http("hi", "sess-no-project")

        mock_session.post.assert_not_called()

    async def test_includes_bot_token_in_auth_header_when_present(self):
        mock_session = _make_aiohttp_session()

        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(
                os.environ,
                {
                    "BOT_TOKEN": "tok-abc",
                    "INITIAL_PROMPT_DELAY_SECONDS": "0",
                    "BACKEND_API_URL": "http://backend:8080",
                    "PROJECT_NAME": "ambient-code",
                },
            ),
        ):
            await _push_initial_prompt_via_http("hi", "sess-token")

        headers = mock_session.post.call_args[1]["headers"]
        assert headers.get("Authorization") == "Bearer tok-abc"

    async def test_no_auth_header_when_bot_token_absent(self):
        mock_session = _make_aiohttp_session()

        env_without_token = {k: v for k, v in os.environ.items() if k != "BOT_TOKEN"}
        env_without_token["INITIAL_PROMPT_DELAY_SECONDS"] = "0"
        env_without_token["BACKEND_API_URL"] = "http://backend:8080"
        env_without_token["PROJECT_NAME"] = "ambient-code"
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
            patch.dict(
                os.environ,
                {
                    "INITIAL_PROMPT_DELAY_SECONDS": "0",
                    "BACKEND_API_URL": "http://backend:8080",
                    "PROJECT_NAME": "ambient-code",
                },
            ),
        ):
            await _push_initial_prompt_via_http("hi", "sess-retry")

        assert mock_session.post.call_count == 8

    async def test_non_transient_error_exits_early_without_full_retries(self):
        """A 400 response without 'not available' body should not exhaust all retries."""
        mock_session = _make_aiohttp_session(status=400, text="bad request")

        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch("asyncio.sleep", new_callable=AsyncMock),
            patch.dict(
                os.environ,
                {
                    "INITIAL_PROMPT_DELAY_SECONDS": "0",
                    "BACKEND_API_URL": "http://backend:8080",
                    "PROJECT_NAME": "ambient-code",
                },
            ),
        ):
            await _push_initial_prompt_via_http("hi", "sess-400")

        assert mock_session.post.call_count == 1

    async def test_not_available_body_triggers_retry(self):
        """'not available' in response body should retry up to max retries."""
        mock_session = _make_aiohttp_session(status=503, text="runner not available")

        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch("asyncio.sleep", new_callable=AsyncMock),
            patch.dict(
                os.environ,
                {
                    "INITIAL_PROMPT_DELAY_SECONDS": "0",
                    "BACKEND_API_URL": "http://backend:8080",
                    "PROJECT_NAME": "ambient-code",
                },
            ),
        ):
            await _push_initial_prompt_via_http("hi", "sess-not-available")

        assert mock_session.post.call_count == 8

    async def test_uses_agentic_session_namespace_fallback_for_project(self):
        """When PROJECT_NAME is missing but AGENTIC_SESSION_NAMESPACE is set, uses that."""
        mock_session = _make_aiohttp_session()

        env = {
            "INITIAL_PROMPT_DELAY_SECONDS": "0",
            "BACKEND_API_URL": "http://backend:8080",
            "AGENTIC_SESSION_NAMESPACE": "ns-fallback",
        }
        with (
            patch("aiohttp.ClientSession", return_value=mock_session),
            patch.dict(os.environ, env, clear=True),
        ):
            await _push_initial_prompt_via_http("hi", "sess-ns")

        mock_session.post.assert_called_once()
        call_url = mock_session.post.call_args[0][0]
        assert "ns-fallback" in call_url


# ---------------------------------------------------------------------------
# _auto_execute_initial_prompt — routing: gRPC ON vs OFF
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestAutoExecuteInitialPrompt:
    async def test_skips_push_when_grpc_url_set(self):
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

        mock_grpc.assert_not_awaited()
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


# ---------------------------------------------------------------------------
# create_ambient_app lifespan — gRPC OFF path (no env vars)
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestCreateAmbientAppLifespanGRPCOff:
    """Verify gRPC listener is NOT started when AMBIENT_GRPC_ENABLED is absent."""

    async def test_grpc_listener_not_started_without_env(self):
        from ambient_runner.app import create_ambient_app
        from ambient_runner.bridges.claude.bridge import ClaudeBridge

        bridge = ClaudeBridge()
        bridge._active_streams = {}

        env_overrides = {}
        for key in ("AMBIENT_GRPC_ENABLED", "AMBIENT_GRPC_URL", "INITIAL_PROMPT"):
            env_overrides[key] = ""

        app = create_ambient_app(bridge)

        with (
            patch.dict(os.environ, env_overrides),
            patch.object(
                bridge, "start_grpc_listener", new_callable=AsyncMock
            ) as mock_start,
            patch.object(bridge, "shutdown", new_callable=AsyncMock),
        ):
            async with app.router.lifespan_context(app):
                pass

        mock_start.assert_not_called()

    async def test_grpc_listener_not_started_when_only_url_set(self):
        """URL alone (without AMBIENT_GRPC_ENABLED=true) must not start listener."""
        from ambient_runner.app import create_ambient_app
        from ambient_runner.bridges.claude.bridge import ClaudeBridge

        bridge = ClaudeBridge()
        bridge._active_streams = {}

        app = create_ambient_app(bridge)

        with (
            patch.dict(
                os.environ,
                {"AMBIENT_GRPC_URL": "localhost:9000", "INITIAL_PROMPT": ""},
                clear=False,
            ),
            patch.object(
                bridge, "start_grpc_listener", new_callable=AsyncMock
            ) as mock_start,
            patch.object(bridge, "shutdown", new_callable=AsyncMock),
        ):
            async with app.router.lifespan_context(app):
                pass

        mock_start.assert_not_called()


# ---------------------------------------------------------------------------
# create_ambient_app lifespan — gRPC ON path
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
class TestCreateAmbientAppLifespanGRPCOn:
    """Verify gRPC listener IS started when AMBIENT_GRPC_ENABLED=true and URL set."""

    async def test_grpc_listener_started_when_both_env_vars_set(self):
        from ambient_runner.app import create_ambient_app
        from ambient_runner.bridges.claude.bridge import ClaudeBridge

        bridge = ClaudeBridge()
        bridge._active_streams = {}

        mock_listener = MagicMock()
        mock_listener.ready = asyncio.Event()
        mock_listener.ready.set()

        app = create_ambient_app(bridge)

        async def _mock_start_grpc_listener(grpc_url):
            bridge._grpc_listener = mock_listener

        with (
            patch.dict(
                os.environ,
                {
                    "AMBIENT_GRPC_ENABLED": "true",
                    "AMBIENT_GRPC_URL": "localhost:9000",
                    "INITIAL_PROMPT": "",
                    "SESSION_ID": "sess-grpc-on",
                },
            ),
            patch.object(
                bridge, "start_grpc_listener", side_effect=_mock_start_grpc_listener
            ) as mock_start,
            patch.object(bridge, "shutdown", new_callable=AsyncMock),
        ):
            async with app.router.lifespan_context(app):
                pass

        mock_start.assert_called_once_with("localhost:9000")

    async def test_grpc_listener_not_started_when_enabled_but_url_empty(self):
        """AMBIENT_GRPC_ENABLED=true but AMBIENT_GRPC_URL="" must not start listener."""
        from ambient_runner.app import create_ambient_app
        from ambient_runner.bridges.claude.bridge import ClaudeBridge

        bridge = ClaudeBridge()
        bridge._active_streams = {}

        app = create_ambient_app(bridge)

        with (
            patch.dict(
                os.environ,
                {
                    "AMBIENT_GRPC_ENABLED": "true",
                    "AMBIENT_GRPC_URL": "",
                    "INITIAL_PROMPT": "",
                },
            ),
            patch.object(
                bridge, "start_grpc_listener", new_callable=AsyncMock
            ) as mock_start,
            patch.object(bridge, "shutdown", new_callable=AsyncMock),
        ):
            async with app.router.lifespan_context(app):
                pass

        mock_start.assert_not_called()
