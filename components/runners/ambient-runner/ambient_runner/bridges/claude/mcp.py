"""
Claude-specific MCP server building and authentication checks.

Assembles the full MCP server dict (external servers from .mcp.json +
platform tools like refresh_credentials and rubric evaluation) and provides
a pre-flight auth check that logs status without emitting events.
"""

import json
import logging
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict

from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.utils import get_bot_token

logger = logging.getLogger(__name__)


DEFAULT_ALLOWED_TOOLS = [
    "Read",
    "Write",
    "Bash",
    "Glob",
    "Grep",
    "Edit",
    "MultiEdit",
    "WebSearch",
]


def generate_gerrit_config(instances: list[dict]) -> None:
    """Generate gerrit_config.json and gitcookies file from fetched credentials.

    Creates /tmp/gerrit-mcp/ directory with:
    - gerrit_config.json: Native Gerrit MCP server config
    - .gitcookies: Combined gitcookies content (if any instances use git_cookies auth)

    Sets GERRIT_CONFIG_PATH env var to point to the generated config.
    """
    config_dir = Path("/tmp/gerrit-mcp")

    # Always clean up old config to prevent stale credentials
    if config_dir.exists():
        import shutil

        shutil.rmtree(config_dir)

    if not instances:
        os.environ.pop("GERRIT_CONFIG_PATH", None)
        return

    config_dir.mkdir(parents=True, exist_ok=True)

    gerrit_hosts = []
    gitcookies_lines: list[str] = []

    for inst in instances:
        host_entry: dict = {
            "name": inst.get("instanceName", ""),
            "external_url": inst.get("url", ""),
        }

        auth_method = inst.get("authMethod", "")
        if auth_method == "http_basic":
            host_entry["authentication"] = {
                "type": "http_basic",
                "username": inst.get("username", ""),
                "auth_token": inst.get("httpToken", ""),
            }
        elif auth_method == "git_cookies":
            gitcookies_path = str(config_dir / ".gitcookies")
            content = inst.get("gitcookiesContent", "")
            if content:
                gitcookies_lines.append(content.rstrip("\n"))
            host_entry["authentication"] = {
                "type": "git_cookies",
                "gitcookies_path": gitcookies_path,
            }

        gerrit_hosts.append(host_entry)

    # Write combined gitcookies file if any instances use git_cookies auth
    if gitcookies_lines:
        gitcookies_path = config_dir / ".gitcookies"
        with open(gitcookies_path, "w") as f:
            f.write("\n".join(gitcookies_lines) + "\n")
        gitcookies_path.chmod(0o600)
        logger.info("Wrote combined gitcookies file for Gerrit instances")

    # Build gerrit_config.json
    gerrit_config: dict = {
        "gerrit_hosts": gerrit_hosts,
    }
    if gerrit_hosts:
        gerrit_config["default_gerrit_base_url"] = gerrit_hosts[0].get("external_url", "")

    config_path = config_dir / "gerrit_config.json"
    with open(config_path, "w") as f:
        json.dump(gerrit_config, f, indent=2)
    config_path.chmod(0o600)

    os.environ["GERRIT_CONFIG_PATH"] = str(config_path)
    logger.info(f"Generated Gerrit config with {len(gerrit_hosts)} host(s) at {config_path}")


def build_mcp_servers(
    context: RunnerContext,
    cwd_path: str,
    obs: Any = None,
) -> dict:
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
        create_refresh_credentials_tool,
        create_rubric_mcp_tool,
        load_rubric_content,
    )
    from ambient_runner.bridges.claude.corrections import create_correction_mcp_tool
    from ambient_runner.bridges.claude.backend_tools import create_backend_mcp_tools

    mcp_servers = load_mcp_config(context, cwd_path) or {}

    # Session control tools
    refresh_creds_tool = create_refresh_credentials_tool(context, sdk_tool)
    session_server = create_sdk_mcp_server(
        name="session", version="1.0.0", tools=[refresh_creds_tool]
    )
    mcp_servers["session"] = session_server
    logger.info("Added session control MCP tools (refresh_credentials)")

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

    # Corrections feedback tool (always available)
    has_rubric = "rubric" in mcp_servers
    correction_tool = create_correction_mcp_tool(
        obs=obs,
        session_id=context.session_id,
        sdk_tool_decorator=sdk_tool,
        has_rubric=has_rubric,
    )
    if correction_tool:
        correction_server = create_sdk_mcp_server(
            name="corrections", version="1.0.0", tools=[correction_tool]
        )
        mcp_servers["corrections"] = correction_server
        logger.info("Added corrections feedback MCP tool (log_correction)")

    # Backend API tools (session management)
    backend_tools = create_backend_mcp_tools(sdk_tool_decorator=sdk_tool)
    if backend_tools:
        backend_server = create_sdk_mcp_server(
            name="acp", version="1.0.0", tools=backend_tools
        )
        mcp_servers["acp"] = backend_server
        logger.info(
            f"Added backend API MCP tools ({len(backend_tools)}): "
            "acp_list_sessions, acp_get_session, acp_create_session, "
            "acp_stop_session, acp_send_message, acp_get_api_reference"
        )

    return mcp_servers


def build_allowed_tools(mcp_servers: dict) -> list[str]:
    """Build the list of allowed tool names from default tools + MCP servers."""
    allowed = list(DEFAULT_ALLOWED_TOOLS)
    for server_name in mcp_servers.keys():
        allowed.append(f"mcp__{server_name}__*")
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


def _read_google_credentials(
    workspace_path: Path, secret_path: Path
) -> Dict[str, Any] | None:
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


def _validate_google_token(
    user_creds: Dict[str, Any], user_email: str
) -> tuple[bool | None, str]:
    if not user_creds.get("access_token") or not user_creds.get("refresh_token"):
        return False, "Google OAuth credentials incomplete - missing or empty tokens"

    if "token_expiry" in user_creds and user_creds["token_expiry"]:
        expiry = _parse_token_expiry(user_creds["token_expiry"])
        if expiry is None:
            return (
                None,
                f"Google OAuth authenticated as {user_email} (token expiry format invalid)",
            )

        now = datetime.now(timezone.utc)
        if expiry <= now and not user_creds.get("refresh_token"):
            return False, "Google OAuth token expired - re-authenticate"
        if expiry <= now:
            return (
                None,
                f"Google OAuth authenticated as {user_email} (token refresh needed)",
            )

    return True, f"Google OAuth authenticated as {user_email}"


def check_mcp_authentication(server_name: str) -> tuple[bool | None, str | None]:
    """Check if credentials are available and valid for known MCP servers."""
    if server_name == "google-workspace":
        workspace_path = Path(
            "/workspace/.google_workspace_mcp/credentials/credentials.json"
        )
        secret_path = Path("/app/.google_workspace_mcp/credentials/credentials.json")
        creds = _read_google_credentials(workspace_path, secret_path)
        if creds is None:
            return (
                False,
                "Google OAuth not configured - authenticate via Integrations page",
            )

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
            project = os.getenv("PROJECT_NAME") or os.getenv(
                "AGENTIC_SESSION_NAMESPACE", ""
            )
            session_id = os.getenv("SESSION_ID", "")

            if base and project and session_id:
                url = f"{base}/projects/{project.strip()}/agentic-sessions/{session_id}/credentials/jira"
                req = _urllib_request.Request(url, method="GET")
                bot = get_bot_token()
                if bot:
                    req.add_header("Authorization", f"Bearer {bot}")
                try:
                    with _urllib_request.urlopen(req, timeout=3) as resp:
                        data = json.loads(resp.read())
                        if data.get("apiToken"):
                            return (
                                True,
                                "Jira credentials available (not yet loaded in session)",
                            )
                except Exception:
                    pass
        except Exception:
            pass

        return False, "Jira not configured - connect on Integrations page"

    if server_name == "gerrit":
        config_path = os.getenv("GERRIT_CONFIG_PATH", "")
        if config_path and Path(config_path).exists():
            return True, "Gerrit credentials configured"

        # Fallback: check if backend has credentials available
        try:
            import urllib.request as _urllib_request

            base = os.getenv("BACKEND_API_URL", "").rstrip("/")
            project = os.getenv("PROJECT_NAME") or os.getenv(
                "AGENTIC_SESSION_NAMESPACE", ""
            )
            session_id = os.getenv("SESSION_ID", "")

            if base and project and session_id:
                url = f"{base}/projects/{project.strip()}/agentic-sessions/{session_id}/credentials/gerrit"
                req = _urllib_request.Request(url, method="GET")
                bot = (os.getenv("BOT_TOKEN") or "").strip()
                if bot:
                    req.add_header("Authorization", f"Bearer {bot}")
                try:
                    with _urllib_request.urlopen(req, timeout=3) as resp:
                        data = json.loads(resp.read())
                        if data.get("instances"):
                            return (
                                True,
                                "Gerrit credentials available (not yet loaded in session)",
                            )
                except Exception as e:
                    logger.debug(f"Gerrit credential probe failed: {e}")
        except Exception as e:
            logger.debug(f"Gerrit credential check setup failed: {e}")

        return False, "Gerrit not configured - connect on Integrations page"

    # Generic fallback: check if MCP_{SERVER_NAME}_* env vars are populated
    sanitized = server_name.upper().replace("-", "_")
    prefix = f"MCP_{sanitized}_"
    has_creds = any(k.startswith(prefix) for k in os.environ)
    if has_creds:
        return True, f"MCP credentials configured for {server_name}"

    return None, None
