"""
Platform authentication — credential fetching from the Ambient backend API.

Framework-agnostic: GitHub, Google, Jira, GitLab credential fetching,
user context sanitization, and environment population.
"""

import asyncio
import json as _json
import logging
import os
import re
import time
from datetime import datetime
from pathlib import Path
from urllib import request as _urllib_request
from urllib.parse import urlparse

from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)

# Placeholder email used by the platform when no real email is available.
_PLACEHOLDER_EMAIL = "user@example.com"

# Tracks credential expiry timestamps (epoch seconds) by provider name.
_credential_expiry: dict[str, float] = {}

# How many seconds before expiry to trigger a proactive refresh.
_EXPIRY_BUFFER_SEC = 5 * 60


# ---------------------------------------------------------------------------
# Vertex AI credential validation (shared across all bridges)
# ---------------------------------------------------------------------------


def validate_vertex_credentials_file(context: RunnerContext) -> str:
    """Validate that GOOGLE_APPLICATION_CREDENTIALS is set and the file exists.

    Shared by all bridge auth modules so the check and error messages are
    consistent regardless of which runner is in use.

    Args:
        context: Runner context used to resolve the env var.

    Returns:
        The resolved credentials file path.

    Raises:
        RuntimeError: If the env var is unset or the file does not exist.
    """
    path = context.get_env("GOOGLE_APPLICATION_CREDENTIALS", "").strip()
    if not path:
        raise RuntimeError(
            "GOOGLE_APPLICATION_CREDENTIALS must be set when USE_VERTEX is enabled"
        )
    if not Path(path).exists():
        raise RuntimeError(f"Service account key file not found at {path}")
    return path


# ---------------------------------------------------------------------------
# User context sanitization
# ---------------------------------------------------------------------------


def sanitize_user_context(user_id: str, user_name: str) -> tuple[str, str]:
    """Validate and sanitize user context fields to prevent injection attacks."""
    if user_id:
        user_id = str(user_id).strip()
        if len(user_id) > 255:
            user_id = user_id[:255]
        user_id = re.sub(r"[^a-zA-Z0-9@._-]", "", user_id)

    if user_name:
        user_name = str(user_name).strip()
        if len(user_name) > 255:
            user_name = user_name[:255]
        user_name = re.sub(r"[\x00-\x1f\x7f-\x9f]", "", user_name)

    return user_id, user_name


# ---------------------------------------------------------------------------
# Backend credential fetching
# ---------------------------------------------------------------------------


async def _fetch_credential(context: RunnerContext, credential_type: str) -> dict:
    """Fetch credentials from backend API at runtime."""
    base = os.getenv("BACKEND_API_URL", "").rstrip("/")
    project = os.getenv("PROJECT_NAME") or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
    project = project.strip()
    session_id = context.session_id

    if not base or not project or not session_id:
        logger.warning(
            f"Cannot fetch {credential_type} credentials: missing environment "
            f"variables (base={base}, project={project}, session={session_id})"
        )
        return {}

    url = f"{base}/projects/{project}/agentic-sessions/{session_id}/credentials/{credential_type}"
    logger.info(f"Fetching fresh {credential_type} credentials from: {url}")

    req = _urllib_request.Request(url, method="GET")
    bot = (os.getenv("BOT_TOKEN") or "").strip()
    if bot:
        req.add_header("Authorization", f"Bearer {bot}")

    loop = asyncio.get_running_loop()

    def _do_req():
        try:
            with _urllib_request.urlopen(req, timeout=10) as resp:
                return resp.read().decode("utf-8", errors="replace")
        except Exception as e:
            logger.warning(f"{credential_type} credential fetch failed: {e}")
            return ""

    resp_text = await loop.run_in_executor(None, _do_req)
    if not resp_text:
        return {}

    try:
        data = _json.loads(resp_text)
        logger.info(f"Successfully fetched {credential_type} credentials from backend")
        return data
    except Exception as e:
        logger.error(f"Failed to parse {credential_type} credential response: {e}")
        return {}


async def fetch_github_credentials(context: RunnerContext) -> dict:
    """Fetch GitHub credentials from backend API (always fresh — PAT or minted App token).

    Returns dict with: token, userName, email, provider, and optionally expiresAt
    """
    data = await _fetch_credential(context, "github")
    if data.get("token"):
        logger.info(
            f"Using fresh GitHub credentials from backend "
            f"(user: {data.get('userName', 'unknown')}, hasEmail: {bool(data.get('email'))})"
        )
        if data.get("expiresAt"):
            try:
                exp_dt = datetime.fromisoformat(data["expiresAt"])
                _credential_expiry["github"] = exp_dt.timestamp()
                logger.info(f"GitHub token expires at {data['expiresAt']}")
            except (ValueError, TypeError) as e:
                logger.warning(f"Failed to parse GitHub expiresAt: {e}")
        else:
            # PAT or legacy token without expiry — clear any stale tracking
            _credential_expiry.pop("github", None)
    return data


async def fetch_github_token(context: RunnerContext) -> str:
    """Fetch GitHub token from backend API (always fresh — PAT or minted App token)."""
    data = await fetch_github_credentials(context)
    return data.get("token", "")


def github_token_expiring_soon() -> bool:
    """Return True if the cached GitHub token will expire within the buffer window."""
    expiry = _credential_expiry.get("github")
    if not expiry:
        return False
    return time.time() > expiry - _EXPIRY_BUFFER_SEC


async def fetch_google_credentials(context: RunnerContext) -> dict:
    """Fetch Google OAuth credentials from backend API."""
    data = await _fetch_credential(context, "google")
    if data.get("accessToken"):
        logger.info(
            f"Using fresh Google credentials (email: {data.get('email', 'unknown')})"
        )
    return data


async def fetch_jira_credentials(context: RunnerContext) -> dict:
    """Fetch Jira credentials from backend API."""
    data = await _fetch_credential(context, "jira")
    if data.get("apiToken"):
        logger.info(f"Using Jira credentials (url: {data.get('url', 'unknown')})")
    return data


async def fetch_gitlab_credentials(context: RunnerContext) -> dict:
    """Fetch GitLab credentials from backend API.

    Returns dict with: token, instanceUrl, userName, email, provider
    """
    data = await _fetch_credential(context, "gitlab")
    if data.get("token"):
        logger.info(
            f"Using fresh GitLab credentials from backend "
            f"(instance: {data.get('instanceUrl', 'unknown')}, "
            f"user: {data.get('userName', 'unknown')}, hasEmail: {bool(data.get('email'))})"
        )
    return data


async def fetch_gitlab_token(context: RunnerContext) -> str:
    """Fetch GitLab token from backend API."""
    data = await fetch_gitlab_credentials(context)
    return data.get("token", "")


async def fetch_token_for_url(context: RunnerContext, url: str) -> str:
    """Fetch appropriate token based on repository URL host."""
    try:
        parsed = urlparse(url)
        hostname = parsed.hostname or ""
        if "gitlab" in hostname.lower():
            return await fetch_gitlab_token(context) or ""
        return await fetch_github_token(context)
    except Exception as e:
        logger.warning(f"Failed to parse URL {url}: {e}, falling back to GitHub token")
        return os.getenv("GITHUB_TOKEN") or await fetch_github_token(context)


async def populate_runtime_credentials(context: RunnerContext) -> None:
    """Fetch all credentials from backend and populate environment variables.

    Called before each SDK run to ensure MCP servers have fresh tokens.
    Also configures git identity from GitHub/GitLab credentials.
    """
    logger.info("Fetching fresh credentials from backend API...")

    # Track git identity from provider credentials
    git_user_name = ""
    git_user_email = ""

    # Google credentials
    try:
        google_creds = await fetch_google_credentials(context)
        if google_creds.get("accessToken"):
            creds_dir = Path("/workspace/.google_workspace_mcp/credentials")
            creds_dir.mkdir(parents=True, exist_ok=True)
            creds_file = creds_dir / "credentials.json"

            client_id = os.getenv("GOOGLE_OAUTH_CLIENT_ID", "")
            client_secret = os.getenv("GOOGLE_OAUTH_CLIENT_SECRET", "")

            # The refresh token is written to disk because workspace-mcp
            # runs as a child process and cannot call back to the platform
            # backend to obtain fresh access tokens on its own.  Without it,
            # Google API access silently breaks after the ~1h access-token
            # lifetime.  The file is owner-only (0o600) and lives inside a
            # short-lived Job pod with no shared volume mounts.
            creds_data = {
                "token": google_creds.get("accessToken"),
                "refresh_token": google_creds.get("refreshToken", ""),
                "token_uri": "https://oauth2.googleapis.com/token",
                "client_id": client_id,
                "client_secret": client_secret,
                "scopes": google_creds.get("scopes", []),
                "expiry": google_creds.get("expiresAt", ""),
            }

            with open(creds_file, "w") as f:
                _json.dump(creds_data, f, indent=2)
            creds_file.chmod(0o600)
            logger.info("Updated Google credentials file for workspace-mcp")

            user_email = google_creds.get("email", "")
            if user_email and user_email != _PLACEHOLDER_EMAIL:
                os.environ["USER_GOOGLE_EMAIL"] = user_email
                logger.info(f"Set USER_GOOGLE_EMAIL to {user_email} for workspace-mcp")
    except Exception as e:
        logger.warning(f"Failed to refresh Google credentials: {e}")

    # Jira credentials
    try:
        jira_creds = await fetch_jira_credentials(context)
        if jira_creds.get("apiToken"):
            os.environ["JIRA_URL"] = jira_creds.get("url", "")
            os.environ["JIRA_API_TOKEN"] = jira_creds.get("apiToken", "")
            os.environ["JIRA_EMAIL"] = jira_creds.get("email", "")
            logger.info("Updated Jira credentials in environment")
    except Exception as e:
        logger.warning(f"Failed to refresh Jira credentials: {e}")

    # GitLab credentials (with user identity)
    try:
        gitlab_creds = await fetch_gitlab_credentials(context)
        if gitlab_creds.get("token"):
            os.environ["GITLAB_TOKEN"] = gitlab_creds["token"]
            logger.info("Updated GitLab token in environment")
            # Use GitLab identity if available (can be overridden by GitHub below)
            if gitlab_creds.get("userName"):
                git_user_name = gitlab_creds["userName"]
            if gitlab_creds.get("email"):
                git_user_email = gitlab_creds["email"]
    except Exception as e:
        logger.warning(f"Failed to refresh GitLab credentials: {e}")

    # GitHub credentials (with user identity — takes precedence)
    try:
        github_creds = await fetch_github_credentials(context)
        if github_creds.get("token"):
            os.environ["GITHUB_TOKEN"] = github_creds["token"]
            logger.info("Updated GitHub token in environment")
            # GitHub identity takes precedence over GitLab
            if github_creds.get("userName"):
                git_user_name = github_creds["userName"]
            if github_creds.get("email"):
                git_user_email = github_creds["email"]
    except Exception as e:
        logger.warning(f"Failed to refresh GitHub credentials: {e}")

    # Configure git identity from provider credentials
    await configure_git_identity(git_user_name, git_user_email)

    logger.info("Runtime credentials populated successfully")


async def _fetch_mcp_credentials(context: RunnerContext, server_name: str) -> dict:
    """Fetch generic MCP server credentials from backend API."""
    data = await _fetch_credential(context, f"mcp/{server_name}")
    if data.get("fields"):
        logger.info(f"Fetched MCP credentials for server {server_name}")
    return data


async def populate_mcp_server_credentials(context: RunnerContext) -> None:
    """Fetch and inject credentials for MCP servers that use ${MCP_*} env var patterns.

    Reads the raw .mcp.json to find servers with env blocks referencing
    ${MCP_*} variables, fetches credentials from the backend, and sets
    the corresponding environment variables before env var expansion.
    """
    mcp_config_file = os.getenv("MCP_CONFIG_FILE", "/app/ambient-runner/.mcp.json")
    config_path = Path(mcp_config_file)
    if not config_path.exists():
        return

    try:
        with open(config_path, "r") as f:
            config = _json.load(f)
        mcp_servers = config.get("mcpServers", {})
    except Exception as e:
        logger.warning(f"Failed to read MCP config for credential population: {e}")
        return

    mcp_env_pattern = re.compile(r"\$\{(MCP_[A-Z0-9_]+)")

    for server_name, server_config in mcp_servers.items():
        env_block = server_config.get("env", {})
        if not env_block:
            continue

        # Check if any env value references ${MCP_*} pattern
        needs_creds = any(
            isinstance(v, str) and mcp_env_pattern.search(v) for v in env_block.values()
        )
        if not needs_creds:
            continue

        try:
            data = await _fetch_mcp_credentials(context, server_name)
            fields = data.get("fields", {})
            if not fields:
                logger.warning(
                    f"No MCP credentials found for server {server_name} — "
                    f"tools requiring auth may not work"
                )
                continue

            # Set env vars using convention: MCP_{SERVER_NAME}_{FIELD_NAME}
            sanitized_name = server_name.upper().replace("-", "_")
            for field_name, field_value in fields.items():
                env_key = f"MCP_{sanitized_name}_{field_name.upper()}"
                os.environ[env_key] = field_value
                logger.info(f"Set {env_key} for MCP server {server_name}")
        except Exception as e:
            logger.warning(f"Failed to fetch MCP credentials for {server_name}: {e}")


async def configure_git_identity(user_name: str, user_email: str) -> None:
    """Configure git user.name and user.email from provider credentials.

    Falls back to defaults if not provided. This ensures commits are
    attributed to the correct user rather than the default bot identity.
    """
    import subprocess

    final_name = user_name.strip() if user_name else "Ambient Code Bot"
    final_email = user_email.strip() if user_email else "bot@ambient-code.local"

    # Also set environment variables for git operations in subprocesses
    os.environ["GIT_USER_NAME"] = final_name
    os.environ["GIT_USER_EMAIL"] = final_email

    try:
        subprocess.run(
            ["git", "config", "--global", "user.name", final_name],
            capture_output=True,
            timeout=5,
        )
        subprocess.run(
            ["git", "config", "--global", "user.email", final_email],
            capture_output=True,
            timeout=5,
        )
        logger.info(f"Configured git identity: {final_name} <{final_email}>")
    except (
        subprocess.TimeoutExpired,
        subprocess.CalledProcessError,
        FileNotFoundError,
    ) as e:
        logger.warning(f"Failed to configure git identity: {e}")
    except Exception as e:
        logger.error(f"Unexpected error configuring git identity: {e}", exc_info=True)
