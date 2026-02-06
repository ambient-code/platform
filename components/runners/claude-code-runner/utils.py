"""
General utility functions for the Claude Code runner.

Pure functions with no business-logic dependencies — URL parsing,
secret redaction, subprocess helpers, environment variable expansion.
"""

import asyncio
import logging
import os
import re
from datetime import datetime, timezone
from typing import Any
from urllib.parse import urlparse, urlunparse

logger = logging.getLogger(__name__)


def timestamp() -> str:
    """Return current UTC timestamp in ISO format."""
    return datetime.now(timezone.utc).isoformat()


def redact_secrets(text: str) -> str:
    """Redact tokens and secrets from text for safe logging."""
    if not text:
        return text

    text = re.sub(r"gh[pousr]_[a-zA-Z0-9]{36,255}", "gh*_***REDACTED***", text)
    text = re.sub(r"sk-ant-[a-zA-Z0-9\-_]{30,200}", "sk-ant-***REDACTED***", text)
    text = re.sub(r"pk-lf-[a-zA-Z0-9\-_]{10,100}", "pk-lf-***REDACTED***", text)
    text = re.sub(r"sk-lf-[a-zA-Z0-9\-_]{10,100}", "sk-lf-***REDACTED***", text)
    text = re.sub(
        r"x-access-token:[^@\s]+@", "x-access-token:***REDACTED***@", text
    )
    text = re.sub(r"oauth2:[^@\s]+@", "oauth2:***REDACTED***@", text)
    text = re.sub(r"://[^:@\s]+:[^@\s]+@", "://***REDACTED***@", text)
    text = re.sub(
        r'(ANTHROPIC_API_KEY|LANGFUSE_SECRET_KEY|LANGFUSE_PUBLIC_KEY|BOT_TOKEN|GIT_TOKEN)\s*=\s*[^\s\'"]+',
        r"\1=***REDACTED***",
        text,
    )
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
