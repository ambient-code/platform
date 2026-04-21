# components/runners/ambient-runner/ambient_runner/bridges/cursor_cli/mcp.py
"""MCP configuration for the Cursor CLI bridge.

Loads the shared MCP config from the platform layer and writes it
to `.cursor/mcp.json` that the Cursor CLI reads on startup.
"""

import json
import logging
from pathlib import Path

from ambient_runner.platform.config import load_mcp_config
from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


def setup_cursor_mcp(
    context: RunnerContext,
    cwd_path: str,
) -> str | None:
    """Load MCP config and write .cursor/mcp.json.

    Returns:
        Path to the written mcp.json, or None if no MCP servers.
    """
    mcp_servers = load_mcp_config(context, cwd_path)
    if not mcp_servers:
        logger.info("No MCP servers configured for Cursor CLI")
        return None

    logger.info(
        "Loaded %d MCP server(s) for Cursor CLI: %s",
        len(mcp_servers),
        list(mcp_servers.keys()),
    )

    cursor_dir = Path(cwd_path) / ".cursor"
    cursor_dir.mkdir(parents=True, exist_ok=True)
    mcp_path = cursor_dir / "mcp.json"

    existing: dict = {}
    try:
        with open(mcp_path) as f:
            existing = json.load(f)
        logger.debug("Loaded existing .cursor/mcp.json")
    except FileNotFoundError:
        pass
    except (json.JSONDecodeError, OSError) as exc:
        logger.warning("Could not read existing mcp.json, overwriting: %s", exc)

    merged_servers = existing.get("mcpServers", {})
    merged_servers.update(mcp_servers)
    existing["mcpServers"] = merged_servers

    with open(mcp_path, "w") as f:
        json.dump(existing, f, indent=2)

    mcp_path.chmod(0o600)

    abs_path = str(mcp_path.resolve())
    logger.info(
        "Wrote Cursor CLI MCP config with %d server(s) to %s",
        len(merged_servers),
        abs_path,
    )
    return abs_path
