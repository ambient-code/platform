"""MCP configuration for the Gemini CLI bridge.

Loads the shared MCP config from the platform layer and writes it
to a ``.gemini/settings.json`` file that the Gemini CLI reads on
startup.  The Gemini CLI discovers MCP servers from the
``mcpServers`` key in its settings file — there is no ``--mcp-config``
CLI flag.

The platform ``load_mcp_config()`` returns server dicts with keys
like ``command``, ``args``, ``env``, ``url``, ``httpUrl`` — these
map directly to the Gemini CLI settings format.
"""

import json
import logging
import os
from pathlib import Path
from typing import Optional

from ambient_runner.platform.config import load_mcp_config
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


def build_gemini_mcp_settings(
    context: RunnerContext,
    cwd_path: str,
) -> Optional[dict]:
    """Load MCP servers from platform config and return Gemini settings dict.

    Args:
        context: Runner context (used by ``load_mcp_config`` for env lookups).
        cwd_path: Working directory.

    Returns:
        A dict suitable for writing to ``.gemini/settings.json``, or None
        if no MCP servers are configured.
    """
    mcp_servers = load_mcp_config(context, cwd_path)
    if not mcp_servers:
        logger.info("No MCP servers configured for Gemini CLI")
        return None

    logger.info(
        "Loaded %d MCP server(s) for Gemini CLI: %s",
        len(mcp_servers),
        list(mcp_servers.keys()),
    )
    return {"mcpServers": mcp_servers}


def write_gemini_settings(
    cwd_path: str,
    settings: dict,
) -> str:
    """Write (or merge) Gemini CLI settings to ``.gemini/settings.json``.

    If a settings file already exists, its ``mcpServers`` key is merged
    with the new servers (new entries take precedence).  Other existing
    keys are preserved.

    Args:
        cwd_path: Working directory (the project root).
        settings: Dict with at least an ``mcpServers`` key.

    Returns:
        Absolute path to the written settings file.
    """
    gemini_dir = Path(cwd_path) / ".gemini"
    gemini_dir.mkdir(parents=True, exist_ok=True)
    settings_path = gemini_dir / "settings.json"

    # Merge with any existing settings
    existing: dict = {}
    if settings_path.exists():
        try:
            with open(settings_path) as f:
                existing = json.load(f)
            logger.debug("Loaded existing .gemini/settings.json")
        except (json.JSONDecodeError, OSError) as exc:
            logger.warning(
                "Could not read existing settings.json, overwriting: %s", exc
            )

    # Merge mcpServers: existing servers as base, new ones override
    merged_servers = existing.get("mcpServers", {})
    merged_servers.update(settings.get("mcpServers", {}))
    existing["mcpServers"] = merged_servers

    with open(settings_path, "w") as f:
        json.dump(existing, f, indent=2)

    # Restrict permissions — settings may contain expanded credentials
    settings_path.chmod(0o600)

    abs_path = str(settings_path.resolve())
    logger.info(
        "Wrote Gemini CLI settings with %d MCP server(s) to %s",
        len(merged_servers),
        abs_path,
    )
    return abs_path


def _build_feedback_server_entry() -> dict:
    """Build the ambient-feedback MCP server entry with Langfuse credentials injected.

    The Gemini CLI strips LANGFUSE_* vars from the subprocess environment via
    the blocklist in session.py (they're runner-internal secrets the AI model
    shouldn't see). MCP server entries can declare their own ``env`` block which
    the CLI passes directly to the server subprocess, bypassing the blocklist.
    We use this to give the feedback_server the Langfuse credentials it needs.
    """
    env: dict[str, str] = {}
    for key in (
        "LANGFUSE_ENABLED",
        "LANGFUSE_PUBLIC_KEY",
        "LANGFUSE_SECRET_KEY",
        "LANGFUSE_HOST",
        "AGENTIC_SESSION_NAME",
        "AGENTIC_SESSION_NAMESPACE",
        "REPOS_JSON",
        "ACTIVE_WORKFLOW_GIT_URL",
        "ACTIVE_WORKFLOW_BRANCH",
        "ACTIVE_WORKFLOW_PATH",
    ):
        val = os.environ.get(key, "")
        if val:
            env[key] = val

    entry: dict = {
        "command": "python",
        "args": ["-m", "ambient_runner.bridges.gemini_cli.feedback_server"],
    }
    if env:
        entry["env"] = env
    return entry


def setup_gemini_mcp(
    context: RunnerContext,
    cwd_path: str,
) -> Optional[str]:
    """End-to-end MCP setup: load config, write settings file, write commands.

    Args:
        context: Runner context.
        cwd_path: Working directory.

    Returns:
        Path to the written settings file, or None if no MCP servers.
    """
    settings = build_gemini_mcp_settings(context, cwd_path) or {"mcpServers": {}}

    # Always register the ambient-feedback server so evaluate_rubric and
    # log_correction tools are available to custom commands.
    settings["mcpServers"]["ambient-feedback"] = _build_feedback_server_entry()

    settings_path = write_gemini_settings(cwd_path, settings)

    # Write /ambient:evaluate-rubric and /ambient:log-correction custom commands.
    from ambient_runner.bridges.gemini_cli.commands import write_gemini_commands

    write_gemini_commands(cwd_path)

    return settings_path
