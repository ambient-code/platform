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
from pathlib import Path
from urllib import request as _urllib_request
from urllib.parse import urlparse

from ambient_runner.platform.context import RunnerContext

logger = logging.getLogger(__name__)


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


async def fetch_github_token(context: RunnerContext) -> str:
    """Fetch GitHub token from backend API."""
    data = await _fetch_credential(context, "github")
    token = data.get("token", "")
    if token:
        logger.info("Using fresh GitHub token from backend")
    return token


async def fetch_google_credentials(context: RunnerContext) -> dict:
    """Fetch Google OAuth credentials from backend API."""
    data = await _fetch_credential(context, "google")
    if data.get("accessToken"):
        logger.info(f"Using fresh Google credentials (email: {data.get('email', 'unknown')})")
    return data


async def fetch_jira_credentials(context: RunnerContext) -> dict:
    """Fetch Jira credentials from backend API."""
    data = await _fetch_credential(context, "jira")
    if data.get("apiToken"):
        logger.info(f"Using Jira credentials (url: {data.get('url', 'unknown')})")
    return data


async def fetch_gitlab_token(context: RunnerContext) -> str:
    """Fetch GitLab token from backend API."""
    data = await _fetch_credential(context, "gitlab")
    token = data.get("token", "")
    if token:
        logger.info(f"Using fresh GitLab token (instance: {data.get('instanceUrl', 'unknown')})")
    return token


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
    """Fetch all credentials from backend and populate environment variables."""
    logger.info("Fetching fresh credentials from backend API...")

    # Google credentials
    google_creds = await fetch_google_credentials(context)
    if google_creds.get("accessToken"):
        creds_dir = Path("/workspace/.google_workspace_mcp/credentials")
        creds_dir.mkdir(parents=True, exist_ok=True)
        creds_file = creds_dir / "credentials.json"

        creds_data = {
            "token": google_creds.get("accessToken"),
            "refresh_token": "",
            "token_uri": "https://oauth2.googleapis.com/token",
            "client_id": os.getenv("GOOGLE_OAUTH_CLIENT_ID", ""),
            "client_secret": os.getenv("GOOGLE_OAUTH_CLIENT_SECRET", ""),
            "scopes": google_creds.get("scopes", []),
            "expiry": google_creds.get("expiresAt", ""),
        }

        with open(creds_file, "w") as f:
            _json.dump(creds_data, f, indent=2)
        creds_file.chmod(0o644)
        logger.info("Updated Google credentials file for workspace-mcp")

        user_email = google_creds.get("email", "")
        if user_email and user_email != "user@example.com":
            os.environ["USER_GOOGLE_EMAIL"] = user_email

    # Jira credentials
    jira_creds = await fetch_jira_credentials(context)
    if jira_creds.get("apiToken"):
        os.environ["JIRA_URL"] = jira_creds.get("url", "")
        os.environ["JIRA_API_TOKEN"] = jira_creds.get("apiToken", "")
        os.environ["JIRA_EMAIL"] = jira_creds.get("email", "")

    # GitLab token
    gitlab_token = await fetch_gitlab_token(context)
    if gitlab_token:
        os.environ["GITLAB_TOKEN"] = gitlab_token

    # GitHub token
    github_token = await fetch_github_token(context)
    if github_token:
        os.environ["GITHUB_TOKEN"] = github_token

    logger.info("Runtime credentials populated successfully")
