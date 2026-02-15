"""
Claude-specific MCP server building and authentication checks.

Assembles the full MCP server dict (external servers from .mcp.json +
platform tools like restart_session and rubric evaluation) and provides
a pre-flight auth check that logs status without emitting events.
"""

import json
import logging
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict

from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


DEFAULT_ALLOWED_TOOLS = [
    "Read", "Write", "Bash", "Glob", "Grep", "Edit",
    "MultiEdit", "WebSearch",
]


def build_mcp_servers(context: RunnerContext, cwd_path: str, obs: Any = None) -> dict:
    """Build the full MCP server config dict including platform tools.

    Args:
        context: Runner context.
        cwd_path: Working directory (used to find rubric files).
        obs: Optional ObservabilityManager (passed to rubric tool).

    Returns:
        Dict of MCP server name -> server config.
    """
    from claude_agent_sdk import create_sdk_mcp_server
    from claude_agent_sdk import tool as sdk_tool

    from ambient_runner.platform.config import load_mcp_config
    from ambient_runner.bridges.claude.tools import (
        create_restart_session_tool,
        create_rubric_mcp_tool,
        load_rubric_content,
    )

    mcp_servers = load_mcp_config(context, cwd_path) or {}

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
    for server_name in mcp_servers.keys():
        is_auth, msg = check_mcp_authentication(server_name)
        if is_auth is False:
            logger.warning(f"MCP auth: {server_name}: {msg}")
        elif is_auth is None and msg:
            logger.info(f"MCP auth: {server_name}: {msg}")


# ---------------------------------------------------------------------------
# MCP authentication checks (also used by /mcp/status endpoint)
# ---------------------------------------------------------------------------


def _read_google_credentials(workspace_path: Path, secret_path: Path) -> Dict[str, Any] | None:
    cred_path = workspace_path if workspace_path.exists() else secret_path
    if not cred_path.exists():
        return None
    try:
        if cred_path.stat().st_size == 0:
            return None
        with open(cred_path, "r") as f:
            return json.load(f)
    except (json.JSONDecodeError, OSError) as e:
        logger.warning(f"Failed to read Google credentials: {e}")
        return None


def _parse_token_expiry(expiry_str: str) -> datetime | None:
    try:
        expiry_str = expiry_str.replace("Z", "+00:00")
        dt = datetime.fromisoformat(expiry_str)
        if dt.tzinfo is None:
            dt = dt.replace(tzinfo=timezone.utc)
        return dt
    except (ValueError, TypeError) as e:
        logger.warning(f"Could not parse token expiry '{expiry_str}': {e}")
        return None


def _validate_google_token(user_creds: Dict[str, Any], user_email: str) -> tuple[bool | None, str]:
    if not user_creds.get("access_token") or not user_creds.get("refresh_token"):
        return False, "Google OAuth credentials incomplete - missing or empty tokens"

    if "token_expiry" in user_creds and user_creds["token_expiry"]:
        expiry = _parse_token_expiry(user_creds["token_expiry"])
        if expiry is None:
            return None, f"Google OAuth authenticated as {user_email} (token expiry format invalid)"

        now = datetime.now(timezone.utc)
        if expiry <= now and not user_creds.get("refresh_token"):
            return False, "Google OAuth token expired - re-authenticate"
        if expiry <= now:
            return None, f"Google OAuth authenticated as {user_email} (token refresh needed)"

    return True, f"Google OAuth authenticated as {user_email}"


def check_mcp_authentication(server_name: str) -> tuple[bool | None, str | None]:
    """Check if credentials are available and valid for known MCP servers."""
    if server_name == "google-workspace":
        workspace_path = Path("/workspace/.google_workspace_mcp/credentials/credentials.json")
        secret_path = Path("/app/.google_workspace_mcp/credentials/credentials.json")
        creds = _read_google_credentials(workspace_path, secret_path)
        if creds is None:
            return False, "Google OAuth not configured - authenticate via Integrations page"

        try:
            user_email = os.environ.get("USER_GOOGLE_EMAIL", "")
            if not user_email or user_email == "user@example.com":
                return False, "Google OAuth not configured - USER_GOOGLE_EMAIL not set"

            user_creds = {
                "access_token": creds.get("token", ""),
                "refresh_token": creds.get("refresh_token", ""),
                "token_expiry": creds.get("expiry", ""),
            }
            return _validate_google_token(user_creds, user_email)
        except KeyError as e:
            return False, f"Google OAuth credentials corrupted: {str(e)}"

    if server_name in ("mcp-atlassian", "jira"):
        jira_url = os.getenv("JIRA_URL", "").strip()
        jira_token = os.getenv("JIRA_API_TOKEN", "").strip()
        if jira_url and jira_token:
            return True, "Jira credentials configured"

        try:
            import urllib.request as _urllib_request

            base = os.getenv("BACKEND_API_URL", "").rstrip("/")
            project = os.getenv("PROJECT_NAME") or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
            session_id = os.getenv("SESSION_ID", "")

            if base and project and session_id:
                url = f"{base}/projects/{project.strip()}/agentic-sessions/{session_id}/credentials/jira"
                req = _urllib_request.Request(url, method="GET")
                bot = (os.getenv("BOT_TOKEN") or "").strip()
                if bot:
                    req.add_header("Authorization", f"Bearer {bot}")
                try:
                    with _urllib_request.urlopen(req, timeout=3) as resp:
                        data = json.loads(resp.read())
                        if data.get("apiToken"):
                            return True, "Jira credentials available (not yet loaded in session)"
                except Exception:
                    pass
        except Exception:
            pass

        return False, "Jira not configured - connect on Integrations page"

    return None, None
