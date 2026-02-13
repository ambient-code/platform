"""
Authentication and credential management for the Claude Code runner.

Handles Anthropic API keys, Vertex AI setup, and runtime credential
fetching from the backend API (GitHub, Google, Jira, GitLab).
"""

import asyncio
import json as _json
import logging
import os
import re
from pathlib import Path
from urllib import request as _urllib_request
from urllib.parse import urlparse

from context import RunnerContext

logger = logging.getLogger(__name__)

# Placeholder email used by the platform when no real email is available.
_PLACEHOLDER_EMAIL = "user@example.com"


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
# Model helpers
# ---------------------------------------------------------------------------

# Anthropic API → Vertex AI model name mapping
VERTEX_MODEL_MAP: dict[str, str] = {
    "claude-opus-4-6": "claude-opus-4-6@default",
    "claude-opus-4-5": "claude-opus-4-5@20251101",
    "claude-sonnet-4-5": "claude-sonnet-4-5@20250929",
    "claude-haiku-4-5": "claude-haiku-4-5@20251001",
}


def map_to_vertex_model(model: str) -> str:
    """Map Anthropic API model names to Vertex AI model names."""
    return VERTEX_MODEL_MAP.get(model, model)


async def setup_vertex_credentials(context: RunnerContext) -> dict:
    """Set up Google Cloud Vertex AI credentials from service account.

    Returns:
        Dict with credentials_path, project_id, region.

    Raises:
        RuntimeError: If required environment variables are missing.
    """
    service_account_path = context.get_env("GOOGLE_APPLICATION_CREDENTIALS", "").strip()
    project_id = context.get_env("ANTHROPIC_VERTEX_PROJECT_ID", "").strip()
    region = context.get_env("CLOUD_ML_REGION", "").strip()

    if not service_account_path:
        raise RuntimeError(
            "GOOGLE_APPLICATION_CREDENTIALS must be set when CLAUDE_CODE_USE_VERTEX=1"
        )
    if not project_id:
        raise RuntimeError(
            "ANTHROPIC_VERTEX_PROJECT_ID must be set when CLAUDE_CODE_USE_VERTEX=1"
        )
    if not region:
        raise RuntimeError("CLOUD_ML_REGION must be set when CLAUDE_CODE_USE_VERTEX=1")

    if not Path(service_account_path).exists():
        raise RuntimeError(
            f"Service account key file not found at {service_account_path}"
        )

    logger.info(f"Vertex AI configured: project={project_id}, region={region}")
    return {
        "credentials_path": service_account_path,
        "project_id": project_id,
        "region": region,
    }


# ---------------------------------------------------------------------------
# Backend credential fetching
# ---------------------------------------------------------------------------


async def _fetch_credential(context: RunnerContext, credential_type: str) -> dict:
    """Fetch credentials from backend API at runtime.

    Args:
        context: Runner context with session_id.
        credential_type: One of 'github', 'google', 'jira', 'gitlab'.

    Returns:
        Dictionary with credential data or empty dict if unavailable.
    """
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

    url = (
        f"{base}/projects/{project}/agentic-sessions/"
        f"{session_id}/credentials/{credential_type}"
    )
    logger.info(f"Fetching fresh {credential_type} credentials from: {url}")

    req = _urllib_request.Request(url, method="GET")
    bot = (os.getenv("BOT_TOKEN") or "").strip()
    if bot:
        req.add_header("Authorization", f"Bearer {bot}")

    loop = asyncio.get_event_loop()

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

    Returns dict with: token, userName, email, provider
    """
    data = await _fetch_credential(context, "github")
    if data.get("token"):
        logger.info(
            f"Using fresh GitHub credentials from backend "
            f"(user: {data.get('userName', 'unknown')}, hasEmail: {bool(data.get('email'))})"
        )
    return data


async def fetch_github_token(context: RunnerContext) -> str:
    """Fetch GitHub token from backend API (always fresh — PAT or minted App token)."""
    data = await fetch_github_credentials(context)
    return data.get("token", "")


async def fetch_google_credentials(context: RunnerContext) -> dict:
    """Fetch Google OAuth credentials from backend API."""
    data = await _fetch_credential(context, "google")
    if data.get("accessToken"):
        logger.info(
            f"Using fresh Google credentials from backend "
            f"(email: {data.get('email', 'unknown')})"
        )
    return data


async def fetch_jira_credentials(context: RunnerContext) -> dict:
    """Fetch Jira credentials from backend API."""
    data = await _fetch_credential(context, "jira")
    if data.get("apiToken"):
        logger.info(
            f"Using Jira credentials from backend (url: {data.get('url', 'unknown')})"
        )
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
            token = await fetch_gitlab_token(context)
            if token:
                logger.info(f"Using fresh GitLab token for {hostname}")
                return token
            else:
                logger.warning(f"No GitLab credentials configured for {url}")
                return ""

        token = await fetch_github_token(context)
        if token:
            logger.info(f"Using fresh GitHub token for {hostname}")
        return token

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
    google_creds = await fetch_google_credentials(context)
    if google_creds.get("accessToken"):
        creds_dir = Path("/workspace/.google_workspace_mcp/credentials")
        creds_dir.mkdir(parents=True, exist_ok=True)
        creds_file = creds_dir / "credentials.json"

        client_id = os.getenv("GOOGLE_OAUTH_CLIENT_ID", "")
        client_secret = os.getenv("GOOGLE_OAUTH_CLIENT_SECRET", "")

        creds_data = {
            "token": google_creds.get("accessToken"),
            "refresh_token": "",
            "token_uri": "https://oauth2.googleapis.com/token",
            "client_id": client_id,
            "client_secret": client_secret,
            "scopes": google_creds.get("scopes", []),
            "expiry": google_creds.get("expiresAt", ""),
        }

        with open(creds_file, "w") as f:
            _json.dump(creds_data, f, indent=2)
        creds_file.chmod(0o644)
        logger.info("✓ Updated Google credentials file for workspace-mcp")

        user_email = google_creds.get("email", "")
        if user_email and user_email != _PLACEHOLDER_EMAIL:
            os.environ["USER_GOOGLE_EMAIL"] = user_email
            logger.info(f"✓ Set USER_GOOGLE_EMAIL to {user_email} for workspace-mcp")

    # Jira credentials
    jira_creds = await fetch_jira_credentials(context)
    if jira_creds.get("apiToken"):
        os.environ["JIRA_URL"] = jira_creds.get("url", "")
        os.environ["JIRA_API_TOKEN"] = jira_creds.get("apiToken", "")
        os.environ["JIRA_EMAIL"] = jira_creds.get("email", "")
        logger.info("✓ Updated Jira credentials in environment")

    # GitLab credentials (with user identity)
    gitlab_creds = await fetch_gitlab_credentials(context)
    if gitlab_creds.get("token"):
        os.environ["GITLAB_TOKEN"] = gitlab_creds["token"]
        logger.info("✓ Updated GitLab token in environment")
        # Use GitLab identity if available (can be overridden by GitHub below)
        if gitlab_creds.get("userName"):
            git_user_name = gitlab_creds["userName"]
        if gitlab_creds.get("email"):
            git_user_email = gitlab_creds["email"]

    # GitHub credentials (with user identity - takes precedence)
    github_creds = await fetch_github_credentials(context)
    if github_creds.get("token"):
        os.environ["GITHUB_TOKEN"] = github_creds["token"]
        logger.info("✓ Updated GitHub token in environment")
        # GitHub identity takes precedence over GitLab
        if github_creds.get("userName"):
            git_user_name = github_creds["userName"]
        if github_creds.get("email"):
            git_user_email = github_creds["email"]

    # Configure git identity from provider credentials
    # Fix for: GitHub credentials aren't mounted to session - need git identity
    await configure_git_identity(git_user_name, git_user_email)

    logger.info("Runtime credentials populated successfully")


async def configure_git_identity(user_name: str, user_email: str) -> None:
    """Configure git user.name and user.email from provider credentials.

    Falls back to defaults if not provided. This ensures commits are
    attributed to the correct user rather than the default bot identity.
    """
    import subprocess

    # Use provided values or fall back to defaults
    final_name = user_name.strip() if user_name else "Ambient Code Bot"
    final_email = user_email.strip() if user_email else "bot@ambient-code.local"

    # Also set environment variables for git operations in subprocesses
    os.environ["GIT_USER_NAME"] = final_name
    os.environ["GIT_USER_EMAIL"] = final_email

    try:
        # Configure git globally for this session
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
        logger.info(f"✓ Configured git identity: {final_name} <{final_email}>")
    except (subprocess.TimeoutExpired, subprocess.CalledProcessError, FileNotFoundError) as e:
        logger.warning(f"Failed to configure git identity: {e}")
    except Exception as e:
        logger.error(f"Unexpected error configuring git identity: {e}", exc_info=True)


async def fetch_github_token_legacy(context: RunnerContext) -> str:
    """Legacy method — kept for backward compatibility."""
    base = os.getenv("BACKEND_API_URL", "").rstrip("/")
    project = os.getenv("PROJECT_NAME") or os.getenv("AGENTIC_SESSION_NAMESPACE", "")
    project = project.strip()
    session_id = context.session_id

    if not base or not project or not session_id:
        logger.warning("Cannot fetch GitHub token: missing environment variables")
        return ""

    url = f"{base}/projects/{project}/agentic-sessions/{session_id}/github/token"
    logger.info(f"Fetching GitHub token from legacy endpoint: {url}")

    req = _urllib_request.Request(
        url,
        data=b"{}",
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    bot = (os.getenv("BOT_TOKEN") or "").strip()
    if bot:
        req.add_header("Authorization", f"Bearer {bot}")

    loop = asyncio.get_event_loop()

    def _do_req():
        try:
            with _urllib_request.urlopen(req, timeout=10) as resp:
                return resp.read().decode("utf-8", errors="replace")
        except Exception as e:
            logger.warning(f"GitHub token fetch failed: {e}")
            return ""

    resp_text = await loop.run_in_executor(None, _do_req)
    if not resp_text:
        return ""

    try:
        data = _json.loads(resp_text)
        token = str(data.get("token") or "")
        if token:
            logger.info("Successfully fetched GitHub token from backend")
        return token
    except Exception as e:
        logger.error(f"Failed to parse token response: {e}")
        return ""
