# components/runners/ambient-runner/ambient_runner/bridges/cursor_cli/bridge.py
"""CursorCLIBridge -- full-lifecycle PlatformBridge for the Cursor CLI.

Owns the Cursor CLI session lifecycle:
- Platform setup (auth, workspace, observability)
- Adapter creation
- Session worker management (one invocation per turn)
- Tracing middleware integration
- Interrupt and graceful shutdown
"""

import asyncio
import json
import logging
import os
import time
from pathlib import Path
from typing import Any, AsyncIterator

from ag_ui.core import BaseEvent, RunAgentInput
from ag_ui_cursor_cli import CursorCLIAdapter
from ag_ui_cursor_cli.types import InitEvent, parse_event
from ag_ui_cursor_cli.utils import extract_user_message

from ambient_runner.bridge import (
    FrameworkCapabilities,
    PlatformBridge,
    _async_safe_manager_shutdown,
    setup_bridge_observability,
)
from ambient_runner.bridges.cursor_cli.session import (
    SHUTDOWN_TIMEOUT_SEC,
    CursorSessionManager,
)
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


class CursorCLIBridge(PlatformBridge):
    """Bridge between the Ambient platform and the Cursor CLI."""

    def __init__(self) -> None:
        super().__init__()
        self._session_manager: CursorSessionManager | None = None
        self._adapter: CursorCLIAdapter | None = None
        self._obs: Any = None

        self._configured_model: str = ""
        self._api_key: str = ""
        self._cwd_path: str = ""
        self._mcp_settings_path: str | None = None
        self._mcp_status_cache: dict | None = None

    def capabilities(self) -> FrameworkCapabilities:
        has_tracing = (
            self._obs is not None
            and hasattr(self._obs, "langfuse_client")
            and self._obs.langfuse_client is not None
        )
        return FrameworkCapabilities(
            framework="cursor-cli",
            agent_features=["agentic_chat", "backend_tool_rendering"],
            file_system=True,
            mcp=True,
            tracing="langfuse" if has_tracing else None,
        )

    async def run(self, input_data: RunAgentInput, **kwargs) -> AsyncIterator[BaseEvent]:
        await self._ensure_ready()
        await self._refresh_credentials_if_stale()

        user_msg = extract_user_message(input_data)

        thread_id = input_data.thread_id or self._context.session_id
        worker = self._session_manager.get_or_create_worker(
            thread_id,
            model=self._configured_model,
            api_key=self._api_key,
            cwd=self._cwd_path,
        )

        session_id = self._session_manager.get_session_id(thread_id)

        async def _line_stream_with_capture():
            async for line in worker.query(user_msg, session_id=session_id):
                event = parse_event(line)
                if isinstance(event, InitEvent) and event.session_id:
                    self._session_manager.set_session_id(thread_id, event.session_id)
                yield line

        async with self._session_manager.get_lock(thread_id):
            from ambient_runner.middleware import (
                secret_redaction_middleware,
                tracing_middleware,
            )

            wrapped_stream = tracing_middleware(
                secret_redaction_middleware(
                    self._adapter.run(input_data, line_stream=_line_stream_with_capture()),
                ),
                obs=self._obs,
                model=self._configured_model,
                prompt=user_msg,
            )

            async for event in wrapped_stream:
                yield event

    async def interrupt(self, thread_id: str | None = None) -> None:
        if not self._session_manager:
            raise RuntimeError("No active session manager")

        tid = thread_id or (self._context.session_id if self._context else None)
        if not tid:
            raise RuntimeError("No thread_id available")

        logger.info("Interrupt request for thread=%s", tid)
        await self._session_manager.interrupt(tid)

    async def shutdown(self) -> None:
        if self._session_manager:
            try:
                await asyncio.wait_for(
                    self._session_manager.shutdown(),
                    timeout=SHUTDOWN_TIMEOUT_SEC * 3,
                )
            except asyncio.TimeoutError:
                logger.warning(
                    "CursorCLIBridge: manager shutdown timed out after %ds",
                    SHUTDOWN_TIMEOUT_SEC * 3,
                )
        if self._obs:
            await self._obs.finalize()
        logger.info("CursorCLIBridge: shutdown complete")

    def mark_dirty(self) -> None:
        self._ready = False
        self._adapter = None
        self._mcp_status_cache = None
        if self._session_manager:
            self._session_manager.clear_session_ids()
            manager = self._session_manager
            self._session_manager = None
            _async_safe_manager_shutdown(manager)
        logger.info("CursorCLIBridge: marked dirty -- will reinitialise on next run")

    def get_error_context(self) -> str:
        if not self._session_manager:
            return ""
        all_lines = self._session_manager.get_all_stderr(max_per_worker=10)
        if all_lines:
            return "Cursor CLI stderr:\n" + "\n".join(all_lines[-20:])
        return ""

    async def get_mcp_status(self) -> dict:
        if self._mcp_status_cache is not None:
            return self._mcp_status_cache

        empty: dict = {"servers": [], "totalCount": 0}
        if not self._mcp_settings_path:
            return empty

        try:
            mcp_path = Path(self._mcp_settings_path)
            if not mcp_path.exists():
                return empty

            with open(mcp_path) as f:
                settings = json.load(f)

            mcp_servers = settings.get("mcpServers", {})
            servers_list = []
            for name, config in mcp_servers.items():
                transport = "stdio"
                if config.get("httpUrl"):
                    transport = "http"
                elif config.get("url"):
                    transport = "sse"
                servers_list.append(
                    {
                        "name": name,
                        "displayName": name,
                        "status": "configured",
                        "transport": transport,
                        "tools": [],
                    }
                )

            result = {"servers": servers_list, "totalCount": len(servers_list)}
            self._mcp_status_cache = result
            return result
        except Exception as e:
            logger.error("Failed to get MCP status: %s", e, exc_info=True)
            return {"servers": [], "totalCount": 0, "error": str(e)}

    @property
    def context(self) -> RunnerContext | None:
        return self._context

    @property
    def configured_model(self) -> str:
        return self._configured_model

    @property
    def obs(self) -> Any:
        return self._obs

    async def _setup_platform(self) -> None:
        if self._session_manager is None:
            state_dir = os.path.join(
                os.getenv("WORKSPACE_PATH", "/workspace"),
                os.getenv("RUNNER_STATE_DIR", ".cursor"),
            )
            self._session_manager = CursorSessionManager(state_dir=state_dir)

        from ambient_runner.bridges.cursor_cli.auth import setup_cursor_cli_auth
        from ambient_runner.bridges.cursor_cli.mcp import setup_cursor_mcp
        from ambient_runner.platform.auth import populate_runtime_credentials
        from ambient_runner.platform.workspace import resolve_workspace_paths

        model, api_key = await setup_cursor_cli_auth(self._context)

        cwd_path, _ = resolve_workspace_paths(self._context)

        # Run credential refresh and observability setup concurrently — independent ops.
        _, self._obs = await asyncio.gather(
            populate_runtime_credentials(self._context),
            setup_bridge_observability(self._context, model),
        )
        self._last_creds_refresh = time.monotonic()

        mcp_settings_path = setup_cursor_mcp(self._context, cwd_path)

        self._configured_model = model
        self._api_key = api_key
        self._cwd_path = cwd_path
        self._mcp_settings_path = mcp_settings_path
        self._adapter = CursorCLIAdapter()
