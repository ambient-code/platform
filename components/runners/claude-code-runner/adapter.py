#!/usr/bin/env python3
"""
Claude Code platform adapter — configures and creates the upstream
``ag_ui_claude_sdk.ClaudeAgentAdapter``.

Orchestrates platform modules (auth, workspace, mcp, prompts) to build
SDK options, then returns a ready-to-use adapter.

Usage in ``main.py`` follows the standard AG-UI pattern::

    adapter = build_adapter(context, ...)
    async for event in adapter.run(input_data):
        yield encoder.encode(event)
"""

import logging
import os
from typing import Any

os.umask(0o022)

from ag_ui_claude_sdk import ClaudeAgentAdapter

import auth
import mcp as mcp_mod
import prompts
import workspace
from context import RunnerContext

logger = logging.getLogger(__name__)


async def setup_platform(context: RunnerContext) -> tuple[str, dict]:
    """Run all platform setup before creating the adapter.

    Returns:
        (configured_model, platform_info)
    """
    _api_key, _use_vertex, configured_model = await auth.setup_sdk_authentication(context)
    await auth.populate_runtime_credentials(context)
    cwd_path, add_dirs = workspace.resolve_sdk_paths(context)

    return configured_model, {
        "cwd_path": cwd_path,
        "add_dirs": add_dirs,
    }


def build_adapter(
    context: RunnerContext,
    *,
    configured_model: str,
    cwd_path: str,
    add_dirs: list[str],
    first_run: bool = True,
    obs=None,
) -> ClaudeAgentAdapter:
    """Build and return a configured ``ClaudeAgentAdapter``."""
    mcp_servers = mcp_mod.build_mcp_servers(context, cwd_path, obs)
    mcp_mod.log_auth_status(mcp_servers)

    system_prompt_config = prompts.build_sdk_system_prompt(context.workspace_path, cwd_path)
    allowed_tools = mcp_mod.build_allowed_tools(mcp_servers)

    def sdk_stderr_handler(line: str):
        logger.warning(f"[SDK stderr] {line.rstrip()}")

    options = _build_options(
        context=context,
        cwd_path=cwd_path,
        add_dirs=add_dirs,
        configured_model=configured_model,
        allowed_tools=allowed_tools,
        mcp_servers=mcp_servers,
        system_prompt_config=system_prompt_config,
        sdk_stderr_handler=sdk_stderr_handler,
        first_run=first_run,
    )

    return ClaudeAgentAdapter(
        name="claude_code_runner",
        description="Ambient Code Platform Claude session",
        options=options,
    )


def _build_options(
    *,
    context: RunnerContext,
    cwd_path: str,
    add_dirs: list[str],
    configured_model: str,
    allowed_tools: list[str],
    mcp_servers: dict,
    system_prompt_config: dict,
    sdk_stderr_handler,
    first_run: bool,
) -> dict[str, Any]:
    """Build the options dict for ``ClaudeAgentAdapter``."""
    is_continuation = context.get_env("IS_RESUME", "").strip().lower() == "true"

    options: dict[str, Any] = {
        "cwd": cwd_path,
        "permission_mode": "acceptEdits",
        "allowed_tools": allowed_tools,
        "mcp_servers": mcp_servers,
        "setting_sources": ["project"],
        "system_prompt": system_prompt_config,
        "include_partial_messages": True,
        "stderr": sdk_stderr_handler,
    }

    if add_dirs:
        options["add_dirs"] = add_dirs

    if configured_model:
        options["model"] = configured_model

    max_tokens_env = context.get_env("LLM_MAX_TOKENS") or context.get_env("MAX_TOKENS")
    if max_tokens_env:
        try:
            options["max_tokens"] = int(max_tokens_env)
        except (ValueError, TypeError):
            pass

    temperature_env = context.get_env("LLM_TEMPERATURE") or context.get_env("TEMPERATURE")
    if temperature_env:
        try:
            options["temperature"] = float(temperature_env)
        except (ValueError, TypeError):
            pass

    if not first_run or is_continuation:
        options["continue_conversation"] = True
        logger.info("Enabled continue_conversation (resume from disk state)")

    return options
