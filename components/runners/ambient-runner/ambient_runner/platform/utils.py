"""
General utility functions for the Claude Code runner.

Pure functions with no business-logic dependencies — URL parsing,
secret redaction, subprocess helpers, environment variable expansion.
"""

import asyncio
import logging
import os
import re
import warnings
from datetime import datetime, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlparse, urlunparse

logger = logging.getLogger(__name__)

_TRUTHY_VALUES = frozenset({"1", "true", "yes"})

# Canonical path where the operator mounts the runner-token Secret as a file.
# Kubelet automatically refreshes this file when the Secret is updated.
_BOT_TOKEN_FILE = Path("/var/run/secrets/ambient/bot-token")

# K8s SA token mounted in every pod by the kubelet.
_SA_TOKEN_FILE = Path("/var/run/secrets/kubernetes.io/serviceaccount/token")

# In-process cache for the token fetched from the CP token endpoint.
# Set once at startup by _grpc_client.py after a successful CP token fetch.
_cp_fetched_token: str = ""


def get_sa_token() -> str:
    """Return the Kubernetes ServiceAccount token mounted in the pod.

    This is a long-lived K8s-managed token that authenticates to the K8s API
    as system:serviceaccount:<namespace>:<sa-name>. The backend's
    enforceCredentialRBAC classifies this as isBotToken=true, which grants
    access to the session owner's credentials without an owner-match check.
    """
    try:
        if _SA_TOKEN_FILE.exists():
            return _SA_TOKEN_FILE.read_text().strip()
    except OSError:
        pass
    return ""


def set_bot_token(token: str) -> None:
    """Store a token fetched from the CP token endpoint for use by get_bot_token()."""
    global _cp_fetched_token
    _cp_fetched_token = token.strip()


def get_bot_token() -> str:
    """Return the current BOT_TOKEN.

    Priority:
    1. Token fetched from CP token endpoint (set via set_bot_token()).
    2. File mount at _BOT_TOKEN_FILE (kubelet-refreshed Secret).
    3. BOT_TOKEN env var (local / non-Kubernetes fallback).
    """
    if _cp_fetched_token:
        return _cp_fetched_token
    try:
        if _BOT_TOKEN_FILE.exists():
            return _BOT_TOKEN_FILE.read_text().strip()
    except OSError:
        pass
    return (os.getenv("BOT_TOKEN") or "").strip()


def refresh_bot_token() -> str:
    """Fetch a fresh token from the CP token endpoint and update the in-process cache.

    Returns the new token, or the current cached token if the CP endpoint is not
    configured (local dev mode). Raises RuntimeError if the CP fetch fails.
    """
    cp_token_url = os.getenv("AMBIENT_CP_TOKEN_URL", "")
    if not cp_token_url:
        return get_bot_token()

    public_key_pem = os.getenv("AMBIENT_CP_TOKEN_PUBLIC_KEY", "")
    session_id = os.getenv("SESSION_ID", "")
    if not public_key_pem or not session_id:
        logger.warning("refresh_bot_token: CP env vars incomplete, skipping refresh")
        return get_bot_token()

    from ambient_runner._grpc_client import _fetch_token_from_cp

    return _fetch_token_from_cp(cp_token_url, public_key_pem, session_id)


def is_env_truthy(value: str) -> bool:
    """Return True for "1", "true", or "yes" (case-insensitive)."""
    return value.strip().lower() in _TRUTHY_VALUES


def is_vertex_enabled(
    legacy_var: str = "CLAUDE_CODE_USE_VERTEX",
    context: Any | None = None,
) -> bool:
    """Check whether Vertex AI is enabled via environment.

    Checks ``USE_VERTEX`` first (unified name). Falls back to *legacy_var*
    with a deprecation warning. Reads from *context* if provided (via
    ``context.get_env``), otherwise from ``os.getenv``.
    """

    def _get(key: str) -> str:
        if context is not None and hasattr(context, "get_env"):
            return context.get_env(key, "")
        return os.getenv(key, "")

    if is_env_truthy(_get("USE_VERTEX")):
        return True

    legacy = _get(legacy_var)
    if is_env_truthy(legacy):
        warnings.warn(
            f"{legacy_var} is deprecated, use USE_VERTEX instead",
            DeprecationWarning,
            stacklevel=2,
        )
        logger.warning("%s is deprecated, use USE_VERTEX instead", legacy_var)
        return True

    return False


def timestamp() -> str:
    """Return current UTC timestamp in ISO format."""
    return datetime.now(timezone.utc).isoformat()


_REDACT_PATTERNS = [
    (re.compile(r"gh[pousr]_[a-zA-Z0-9]{36,255}"), "gh*_***REDACTED***"),
    (re.compile(r"sk-ant-[a-zA-Z0-9\-_]{30,200}"), "sk-ant-***REDACTED***"),
    (re.compile(r"pk-lf-[a-zA-Z0-9\-_]{10,100}"), "pk-lf-***REDACTED***"),
    (re.compile(r"sk-lf-[a-zA-Z0-9\-_]{10,100}"), "sk-lf-***REDACTED***"),
    (re.compile(r"x-access-token:[^@\s]+@"), "x-access-token:***REDACTED***@"),
    (re.compile(r"oauth2:[^@\s]+@"), "oauth2:***REDACTED***@"),
    (re.compile(r"://[^:@\s]+:[^@\s]+@"), "://***REDACTED***@"),
    (re.compile(r"AIza[a-zA-Z0-9\-_]{30,}"), "AIza***REDACTED***"),
    (
        re.compile(
            r"(ANTHROPIC_API_KEY|LANGFUSE_SECRET_KEY|LANGFUSE_PUBLIC_KEY|BOT_TOKEN|GIT_TOKEN|GEMINI_API_KEY|GOOGLE_API_KEY)\s*=\s*[^\s\'\"]+",
        ),
        r"\1=***REDACTED***",
    ),
]


def redact_secrets(text: str) -> str:
    """Redact tokens and secrets from text for safe logging."""
    if not text:
        return text

    for pattern, replacement in _REDACT_PATTERNS:
        text = pattern.sub(replacement, text)
    return text


def url_with_token(url: str, token: str) -> str:
    """Add authentication token to a git URL.

    Uses x-access-token for GitHub, oauth2 for GitLab.
    """
    if not token or not url.lower().startswith("http"):
        return url
    try:
        parsed = urlparse(url)
        netloc = parsed.netloc
        if "@" in netloc:
            netloc = netloc.split("@", 1)[1]

        hostname = parsed.hostname or ""
        if "gitlab" in hostname.lower():
            auth = f"oauth2:{token}@"
        else:
            auth = f"x-access-token:{token}@"

        new_netloc = auth + netloc
        return urlunparse(
            (
                parsed.scheme,
                new_netloc,
                parsed.path,
                parsed.params,
                parsed.query,
                parsed.fragment,
            )
        )
    except Exception:
        return url


def derive_workflow_name(url: str) -> str:
    """Extract the workflow name from a git URL.

    Returns the last path segment with ``.git`` suffix removed.

    >>> derive_workflow_name("https://github.com/org/my-workflow.git")
    'my-workflow'
    """
    return url.split("/")[-1].removesuffix(".git")


def get_active_integrations() -> list[str]:
    """Return a list of currently active integration names based on env vars."""
    integrations: list[str] = []
    if os.getenv("GITHUB_TOKEN"):
        integrations.append("GitHub")
    if os.getenv("GITLAB_TOKEN"):
        integrations.append("GitLab")
    if os.getenv("JIRA_API_TOKEN"):
        integrations.append("Jira")
    if os.getenv("USER_GOOGLE_EMAIL"):
        integrations.append("Google")
    return integrations


def parse_owner_repo(url: str) -> tuple[str, str, str]:
    """Return (owner, name, host) from various git URL formats.

    Supports HTTPS, SSH, and shorthand owner/repo formats.
    """
    s = (url or "").strip()
    s = s.removesuffix(".git")
    host = "github.com"
    try:
        if s.startswith("http://") or s.startswith("https://"):
            p = urlparse(s)
            host = p.netloc
            parts = [pt for pt in p.path.split("/") if pt]
            if len(parts) >= 2:
                return parts[0], parts[1], host
        if s.startswith("git@") or ":" in s:
            s2 = s
            if s2.startswith("git@"):
                s2 = s2.replace(":", "/", 1)
                s2 = s2.replace("git@", "ssh://git@", 1)
            p = urlparse(s2)
            host = p.hostname or host
            parts = [pt for pt in (p.path or "").split("/") if pt]
            if len(parts) >= 2:
                return parts[-2], parts[-1], host
        parts = [pt for pt in s.split("/") if pt]
        if len(parts) == 2:
            return parts[0], parts[1], host
    except Exception:
        return "", "", host
    return "", "", host


def expand_env_vars(value: Any) -> Any:
    """Recursively expand ${VAR} and ${VAR:-default} patterns in config values."""
    if isinstance(value, str):
        pattern = r"\$\{([^}:]+)(?::-([^}]*))?\}"

        def replace_var(match):
            var_name = match.group(1)
            default_val = match.group(2) if match.group(2) is not None else ""
            return os.environ.get(var_name, default_val)

        return re.sub(pattern, replace_var, value)
    elif isinstance(value, dict):
        return {k: expand_env_vars(v) for k, v in value.items()}
    elif isinstance(value, list):
        return [expand_env_vars(item) for item in value]
    return value


async def run_cmd(
    cmd: list,
    cwd: str | None = None,
    capture_stdout: bool = False,
    ignore_errors: bool = False,
) -> str:
    """Run a subprocess command asynchronously.

    Args:
        cmd: Command and arguments list.
        cwd: Working directory (defaults to current directory).
        capture_stdout: If True, return stdout text.
        ignore_errors: If True, don't raise on non-zero exit.

    Returns:
        stdout text if capture_stdout is True, else empty string.

    Raises:
        RuntimeError: If command fails and ignore_errors is False.
    """
    cmd_safe = [redact_secrets(str(arg)) for arg in cmd]
    logger.info(f"Running command: {' '.join(cmd_safe)}")

    proc = await asyncio.create_subprocess_exec(
        *cmd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
        cwd=cwd,
    )
    stdout_data, stderr_data = await proc.communicate()
    stdout_text = stdout_data.decode("utf-8", errors="replace")
    stderr_text = stderr_data.decode("utf-8", errors="replace")

    if stdout_text.strip():
        logger.info(f"Command stdout: {redact_secrets(stdout_text.strip())}")
    if stderr_text.strip():
        logger.info(f"Command stderr: {redact_secrets(stderr_text.strip())}")

    if proc.returncode != 0 and not ignore_errors:
        raise RuntimeError(stderr_text or f"Command failed: {' '.join(cmd_safe)}")

    if capture_stdout:
        return stdout_text
    return ""
