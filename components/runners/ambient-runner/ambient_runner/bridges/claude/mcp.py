"""
Claude-specific MCP server building and authentication checks.

Assembles the full MCP server dict (external servers from .mcp.json +
platform tools like refresh_credentials and rubric evaluation) and provides
a pre-flight auth check that logs status without emitting events.
"""

import json
import logging
import os
import shutil
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.utils import get_bot_token

logger = logging.getLogger(__name__)

_WORKSPACE_CREDS_DIR = Path("/workspace/.google_workspace_mcp/credentials")
_SECRET_CREDS_DIR = Path("/app/.google_workspace_mcp/credentials")


DEFAULT_ALLOWED_TOOLS = [
    "Read",
    "Write",
    "Bash",
    "Glob",
    "Grep",
    "Edit",
    "MultiEdit",
    "WebSearch",
    "Skill",
    "Agent",
]


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

    mcp_servers = load_mcp_config(context, cwd_path) or {}

    # Ambient MCP sidecar (SSE transport, injected when annotation ambient-code.io/mcp-sidecar=true)
    ambient_mcp_url = os.getenv("AMBIENT_MCP_URL", "").strip()
    if ambient_mcp_url:
        mcp_servers["ambient"] = {
            "type": "sse",
            "url": f"{ambient_mcp_url.rstrip('/')}/sse",
        }
        logger.info("Added ambient MCP sidecar server (SSE): %s", ambient_mcp_url)

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

    # Credential-aware MCP servers (dynamically configured per bound credentials)
    credential_mcp_servers = build_credential_mcp_servers()
    mcp_servers.update(credential_mcp_servers)
    if credential_mcp_servers:
        logger.info(
            "Added credential MCP servers: %s", list(credential_mcp_servers.keys())
        )

    # Gerrit MCP server (only if credentials are configured)
    gerrit_config = os.environ.get("GERRIT_CONFIG_PATH", "")
    if gerrit_config and Path(gerrit_config).exists():
        mcp_servers["gerrit"] = {
            "command": "/opt/gerrit-mcp-server/.venv/bin/python",
            "args": ["/opt/gerrit-mcp-server/gerrit_mcp_server/main.py", "stdio"],
            "env": {
                "PYTHONPATH": "/opt/gerrit-mcp-server/",
                "GERRIT_CONFIG_PATH": gerrit_config,
            },
        }
        logger.info("Added Gerrit MCP server (credentials configured)")

    return mcp_servers


_CREDENTIAL_MCP_REGISTRY: dict[str, dict] = {
    "jira": {
        "command": "uvx",
        "args": ["mcp-atlassian"],
        "env": {
            "JIRA_URL": "${JIRA_URL}",
            "JIRA_USERNAME": "${JIRA_EMAIL}",
            "JIRA_API_TOKEN": "${JIRA_API_TOKEN}",
            "JIRA_SSL_VERIFY": "true",
            "READ_ONLY_MODE": "${JIRA_READ_ONLY_MODE:-true}",
        },
        "server_name": "mcp-atlassian",
    },
    "kubeconfig": {
        "command": "uvx",
        "args": [
            "kubernetes-mcp-server@latest",
            "--kubeconfig",
            "/tmp/.ambient_kubeconfig",
            "--disable-multi-cluster",
        ],
        "server_name": "openshift",
    },
    "google": {
        "command": "uvx",
        "args": ["workspace-mcp@1.17.1", "--permissions", "gmail:send", "drive:full"],
        "env": {
            "GOOGLE_MCP_CREDENTIALS_DIR": "${GOOGLE_MCP_CREDENTIALS_DIR}",
            "MCP_SINGLE_USER_MODE": "1",
            "GOOGLE_OAUTH_CLIENT_ID": "${GOOGLE_OAUTH_CLIENT_ID}",
            "GOOGLE_OAUTH_CLIENT_SECRET": "${GOOGLE_OAUTH_CLIENT_SECRET}",
            "GOOGLE_OAUTH_REDIRECT_URI": "${GOOGLE_OAUTH_REDIRECT_URI}",
            "USER_GOOGLE_EMAIL": "${USER_GOOGLE_EMAIL:-user@example.com}",
        },
        "server_name": "google-workspace",
    },
}


def _expand_env_vars(value: str) -> str:
    import re as _re

    def _replace(match: _re.Match) -> str:
        expr = match.group(1)
        if ":-" in expr:
            var, default = expr.split(":-", 1)
            return os.environ.get(var, default)
        return os.environ.get(expr, match.group(0))

    return _re.sub(r"\$\{([^}]+)}", _replace, value)


_CREDENTIAL_SIDECAR_REGISTRY: dict[str, dict[str, str]] = {
    "github": {
        "server_name": "github",
        "type": "sse",
        "path": "/sse",
    },
    "jira": {
        "server_name": "mcp-atlassian",
        "type": "sse",
        "path": "/sse",
    },
    "kubeconfig": {
        "server_name": "openshift",
        "type": "sse",
        "path": "/sse",
    },
    "google": {
        "server_name": "google-workspace",
        "type": "sse",
        "path": "/sse",
    },
}


def build_credential_mcp_servers() -> dict:
    credential_mcp_urls_raw = os.getenv("CREDENTIAL_MCP_URLS", "").strip()
    if credential_mcp_urls_raw:
        return _build_sidecar_mcp_servers(credential_mcp_urls_raw)
    return _build_subprocess_mcp_servers()


def _build_sidecar_mcp_servers(credential_mcp_urls_raw: str) -> dict:
    try:
        credential_mcp_urls = json.loads(credential_mcp_urls_raw)
    except (json.JSONDecodeError, TypeError):
        logger.warning("Failed to parse CREDENTIAL_MCP_URLS — skipping credential MCP servers")
        return {}

    if not isinstance(credential_mcp_urls, dict):
        logger.warning("CREDENTIAL_MCP_URLS is not a JSON object — skipping credential MCP servers")
        return {}

    servers: dict = {}
    for provider, url in credential_mcp_urls.items():
        if not isinstance(url, str) or not url.strip():
            logger.warning("Skipping credential sidecar %s: invalid URL", provider)
            continue
        spec = _CREDENTIAL_SIDECAR_REGISTRY.get(provider, {})
        server_name = spec.get("server_name", provider)
        transport_type = spec.get("type", "sse")
        path = spec.get("path", "/sse")
        servers[server_name] = {
            "type": transport_type,
            "url": f"{url.rstrip('/')}{path}",
        }
        logger.info(
            "Configured %s credential sidecar (%s) at %s",
            server_name,
            transport_type,
            url,
        )

    _wait_for_sidecar_readiness(servers)
    return servers


def _wait_for_sidecar_readiness(
    servers: dict,
    timeout: float = 60.0,
    poll_interval: float = 1.0,
) -> None:
    import socket
    import time
    from urllib.parse import urlparse

    if not servers:
        return

    endpoints: list[tuple[str, str, int]] = []
    for name, cfg in servers.items():
        url = cfg.get("url", "")
        parsed = urlparse(url)
        host = parsed.hostname or "127.0.0.1"
        port = parsed.port
        if port:
            endpoints.append((name, host, port))

    if not endpoints:
        return

    logger.info("Waiting for %d credential sidecar(s) to become ready (timeout=%ds)", len(endpoints), int(timeout))
    deadline = time.monotonic() + timeout
    pending = list(endpoints)

    while pending and time.monotonic() < deadline:
        still_pending = []
        for name, host, port in pending:
            try:
                with socket.create_connection((host, port), timeout=1.0):
                    logger.info("Credential sidecar %s ready at %s:%d", name, host, port)
            except (ConnectionRefusedError, OSError, socket.timeout):
                still_pending.append((name, host, port))
        pending = still_pending
        if pending:
            time.sleep(poll_interval)

    if pending:
        names = [p[0] for p in pending]
        logger.warning("Credential sidecar(s) not ready after %ds: %s", int(timeout), names)


def _build_subprocess_mcp_servers() -> dict:
    credential_ids_raw = os.getenv("CREDENTIAL_IDS", "").strip()
    if not credential_ids_raw:
        return {}

    try:
        credential_ids = json.loads(credential_ids_raw)
    except (json.JSONDecodeError, TypeError):
        logger.warning("Failed to parse CREDENTIAL_IDS — skipping credential MCP servers")
        return {}

    servers: dict = {}
    for provider, _cred_id in credential_ids.items():
        registry_entry = _CREDENTIAL_MCP_REGISTRY.get(provider)
        if not registry_entry:
            continue

        server_name = registry_entry["server_name"]
        server_config: dict = {
            "command": registry_entry["command"],
            "args": list(registry_entry["args"]),
        }

        if "env" in registry_entry:
            server_config["env"] = {
                k: _expand_env_vars(v) for k, v in registry_entry["env"].items()
            }

        servers[server_name] = server_config
        logger.info(
            "Configured %s MCP server for bound credential %s (provider: %s)",
            server_name,
            _cred_id,
            provider,
        )

    return servers


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


def _load_credential_file(path: Path) -> dict[str, Any] | None:
    if not path.exists():
        return None
    try:
        if path.stat().st_size == 0:
            return None
        with open(path, "r") as f:
            return json.load(f)
    except (json.JSONDecodeError, OSError) as e:
        logger.warning(f"Failed to read Google credentials from {path.name}: {e}")
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
    user_creds: dict[str, Any], user_email: str
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
        creds = None
        for creds_dir in (_WORKSPACE_CREDS_DIR, _SECRET_CREDS_DIR):
            if creds_dir.is_dir():
                for json_file in sorted(creds_dir.glob("*.json")):
                    candidate = _load_credential_file(json_file)
                    if candidate is not None:
                        logger.debug("Using Google credentials from %s", json_file)
                        creds = candidate
                        break
            if creds is not None:
                break
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
        config_path = os.environ.get("GERRIT_CONFIG_PATH", "")
        if config_path and Path(config_path).exists():
            return True, "Gerrit credentials configured"
        return False, "Gerrit not configured - connect on Integrations page"

    # Generic fallback: check if MCP_{SERVER_NAME}_* env vars are populated
    sanitized = server_name.upper().replace("-", "_")
    prefix = f"MCP_{sanitized}_"
    has_creds = any(k.startswith(prefix) for k in os.environ)
    if has_creds:
        return True, f"MCP credentials configured for {server_name}"

    return None, None


# ---------------------------------------------------------------------------
# Gerrit MCP config generation
# ---------------------------------------------------------------------------


def generate_gerrit_config(instances: list[dict]) -> None:
    """Generate Gerrit MCP server configuration from credential instances."""
    config_dir = Path("/tmp/gerrit-mcp")
    config_file = config_dir / "gerrit_config.json"
    gitcookies_file = config_dir / ".gitcookies"

    # Clean up stale config
    if config_dir.exists():
        shutil.rmtree(config_dir)

    if not instances:
        os.environ.pop("GERRIT_CONFIG_PATH", None)
        return

    config_dir.mkdir(parents=True, exist_ok=True)

    gerrit_hosts = []
    gitcookies_lines = []

    for inst in instances:
        host_entry = {
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
            host_entry["authentication"] = {
                "type": "git_cookies",
                "gitcookies_path": str(gitcookies_file),
            }
            content = inst.get("gitcookiesContent", "")
            if content:
                gitcookies_lines.append(content.strip())

        gerrit_hosts.append(host_entry)

    # Write combined .gitcookies if any instances use git_cookies auth
    if gitcookies_lines:
        gitcookies_file.write_text("\n".join(gitcookies_lines) + "\n")
        gitcookies_file.chmod(0o600)

    config = {
        "gerrit_hosts": gerrit_hosts,
        "default_gerrit_base_url": instances[0].get("url", "") if instances else "",
    }

    config_file.write_text(json.dumps(config, indent=2))
    config_file.chmod(0o600)

    os.environ["GERRIT_CONFIG_PATH"] = str(config_file)
    logger.info(f"Generated Gerrit config with {len(gerrit_hosts)} host(s)")
