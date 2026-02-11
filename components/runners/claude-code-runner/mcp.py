"""
MCP server building and authentication checks for the Claude Code runner.

Assembles the full MCP server dict (external servers from .mcp.json +
platform tools like restart_session and rubric evaluation) and provides
a pre-flight auth check that logs status without emitting events.
"""

import logging

import config as runner_config
from context import RunnerContext
from tools import create_restart_session_tool, create_rubric_mcp_tool, load_rubric_content

logger = logging.getLogger(__name__)


DEFAULT_ALLOWED_TOOLS = [
    "Read", "Write", "Bash", "Glob", "Grep", "Edit",
    "MultiEdit", "WebSearch",
]


def build_mcp_servers(context: RunnerContext, cwd_path: str, obs=None) -> dict:
    """Build the full MCP server config dict including platform tools.

    Args:
        context: Runner context.
        cwd_path: Working directory (used to find rubric files).
        obs: Optional ObservabilityManager (passed to rubric tool).

    Returns:
        Dict of MCP server name → server config.
    """
    from claude_agent_sdk import create_sdk_mcp_server
    from claude_agent_sdk import tool as sdk_tool

    mcp_servers = runner_config.load_mcp_config(context, cwd_path) or {}

    # Session control tool
    restart_tool = create_restart_session_tool(None, sdk_tool)
    session_server = create_sdk_mcp_server(
        name="session", version="1.0.0", tools=[restart_tool]
    )
    mcp_servers["session"] = session_server
    logger.info("Added session control MCP tools (restart_session)")

    # Rubric evaluation tool
    rubric_content, rubric_config = load_rubric_content(cwd_path)
    if rubric_content or rubric_config:
        rubric_tool = create_rubric_mcp_tool(
            rubric_content=rubric_content or "",
            rubric_config=rubric_config,
            obs=obs,
            session_id=context.session_id,
            sdk_tool_decorator=sdk_tool,
        )
        if rubric_tool:
            rubric_server = create_sdk_mcp_server(
                name="rubric", version="1.0.0", tools=[rubric_tool]
            )
            mcp_servers["rubric"] = rubric_server
            logger.info(
                f"Added rubric evaluation MCP tool "
                f"(categories: {list(rubric_config.get('schema', {}).keys())})"
            )

    return mcp_servers


def build_allowed_tools(mcp_servers: dict) -> list[str]:
    """Build the list of allowed tool names from default tools + MCP servers."""
    allowed = list(DEFAULT_ALLOWED_TOOLS)
    for server_name in mcp_servers.keys():
        allowed.append(f"mcp__{server_name}")
    logger.info(f"MCP tool permissions granted for servers: {list(mcp_servers.keys())}")
    return allowed


def log_auth_status(mcp_servers: dict) -> None:
    """Log MCP server authentication status (server-side only, no events)."""
    from endpoints.mcp_status import check_mcp_authentication as _check_mcp_authentication

    for server_name in mcp_servers.keys():
        is_auth, msg = _check_mcp_authentication(server_name)
        if is_auth is False:
            logger.warning(f"MCP auth: {server_name}: {msg}")
        elif is_auth is None and msg:
            logger.info(f"MCP auth: {server_name}: {msg}")
