"""GET /mcp/status — MCP server connection diagnostics.

Also contains the ``_check_mcp_authentication`` helper used by
``mcp.log_auth_status`` (imported from here to avoid circular deps with main).
"""

import json
import logging
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict

from fastapi import APIRouter

from endpoints import state

logger = logging.getLogger(__name__)

router = APIRouter()


# ------------------------------------------------------------------
# MCP authentication helpers (used by mcp.py via import)
# ------------------------------------------------------------------


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


# ------------------------------------------------------------------
# Endpoint
# ------------------------------------------------------------------


@router.get("/mcp/status")
async def get_mcp_status():
    """Returns MCP server connection status via the SDK's get_mcp_status()."""
    try:
        if not state.context:
            return {"servers": [], "totalCount": 0, "message": "Context not initialized yet"}

        from claude_agent_sdk import ClaudeAgentOptions, ClaudeSDKClient
        import config as runner_config

        workspace_path = state.context.workspace_path or "/workspace"
        active_workflow_url = os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip()
        cwd_path = workspace_path

        if active_workflow_url:
            workflow_name = active_workflow_url.split("/")[-1].removesuffix(".git")
            workflow_path = os.path.join(workspace_path, "workflows", workflow_name)
            if os.path.exists(workflow_path):
                cwd_path = workflow_path

        mcp_servers = runner_config.load_mcp_config(state.context, cwd_path) or {}

        options = ClaudeAgentOptions(
            cwd=cwd_path,
            permission_mode="acceptEdits",
            mcp_servers=mcp_servers,
        )

        client = ClaudeSDKClient(options=options)
        try:
            logger.info("MCP Status: Connecting ephemeral SDK client...")
            await client.connect()

            sdk_status = await client.get_mcp_status()
            logger.info("MCP Status: SDK returned:\n%s", json.dumps(sdk_status, indent=2, default=str))

            raw_servers = []
            if isinstance(sdk_status, dict):
                raw_servers = sdk_status.get("mcpServers", [])
            elif isinstance(sdk_status, list):
                raw_servers = sdk_status

            servers_list = []
            for srv in raw_servers:
                if not isinstance(srv, dict):
                    continue
                server_info = srv.get("serverInfo") or {}
                raw_tools = srv.get("tools") or []
                tools = [
                    {
                        "name": t.get("name", ""),
                        "annotations": {k: v for k, v in (t.get("annotations") or {}).items()},
                    }
                    for t in raw_tools
                    if isinstance(t, dict)
                ]
                servers_list.append({
                    "name": srv.get("name", ""),
                    "displayName": server_info.get("name", srv.get("name", "")),
                    "status": srv.get("status", "unknown"),
                    "version": server_info.get("version", ""),
                    "tools": tools,
                })

            return {"servers": servers_list, "totalCount": len(servers_list)}
        finally:
            logger.info("MCP Status: Disconnecting ephemeral SDK client...")
            await client.disconnect()

    except Exception as e:
        logger.error(f"Failed to get MCP status: {e}", exc_info=True)
        return {"servers": [], "totalCount": 0, "error": str(e)}
