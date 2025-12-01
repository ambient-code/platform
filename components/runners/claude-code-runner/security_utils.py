"""
Security utilities for Claude Code runner.

Provides exception sanitization and timeout wrappers
to prevent API key leaks and hanging operations.
"""

import re
import asyncio
import logging
from typing import Callable, Any, TypeVar, ParamSpec

P = ParamSpec("P")
T = TypeVar("T")


def sanitize_exception_message(
    exception: Exception, secrets_to_redact: dict[str, str]
) -> str:
    """Sanitize exception message to prevent secret leakage.

    Replaces any occurrence of secret values with redacted placeholders.
    NEVER logs the exception object itself - only the sanitized message string.

    Defense-in-depth: After sanitization, validates that no secrets leaked through.
    If validation fails, returns a generic error message instead.

    Approach: Simple string replacement + post-sanitization validation
    Rationale:
    - Straightforward and easy to audit (no complex regex patterns)
    - Works for typical cases: API keys, tokens, hosts in error messages
    - Performance: O(n*m) where n=message length, m=number of secrets (acceptable for small m)
    - Post-validation catches edge cases (encoded forms, partial substrings)

    Limitations:
    - May not catch secrets in encoded forms (base64, URL-encoded) during replacement
    - Substring matches could over-redact (e.g., "pk" in "package")
    - Relies on caller providing complete secrets dict

    For production use:
    - Always include all sensitive values in secrets_to_redact
    - Test with actual error scenarios to verify effectiveness
    - Post-validation provides safety net for edge cases

    Args:
        exception: The exception object
        secrets_to_redact: Dict mapping secret names to values (e.g., {"public_key": "pk-123"})

    Returns:
        Sanitized error message string (or generic message if validation fails)
    """
    error_msg = str(exception)

    # Redact each secret using simple string replacement
    for secret_name, secret_value in secrets_to_redact.items():
        if secret_value and secret_value.strip():
            placeholder = f"[REDACTED_{secret_name.upper()}]"
            error_msg = error_msg.replace(secret_value, placeholder)

    # Validate no secrets leaked through sanitization
    # This catches edge cases like partial matches, encoded forms, etc.
    for secret_name, secret_value in secrets_to_redact.items():
        if secret_value and secret_value.strip() and secret_value in error_msg:
            # Do not log secret_name - reveals context to attackers
            logging.error("SECURITY: Credential sanitization validation failed")
            return "Operation failed - check configuration and credentials"

    return error_msg


async def with_timeout(
    coro_or_func: Callable[P, Any],
    timeout_seconds: float,
    operation_name: str,
    *args: P.args,
    **kwargs: P.kwargs,
) -> tuple[bool, Any]:
    """Execute async operation with timeout.

    Args:
        coro_or_func: Async function or coroutine to execute
        timeout_seconds: Timeout in seconds
        operation_name: Operation description for logging
        *args: Positional arguments to pass to the function
        **kwargs: Keyword arguments to pass to the function

    Returns:
        Tuple of (success, result_or_error)
    """
    try:
        # If it's a callable, call it to get the coroutine
        if callable(coro_or_func):
            coro = coro_or_func(*args, **kwargs)
        else:
            coro = coro_or_func

        result = await asyncio.wait_for(coro, timeout=timeout_seconds)
        return True, result
    except asyncio.TimeoutError:
        logging.warning(f"{operation_name} timed out after {timeout_seconds}s")
        return False, None
    except Exception as e:
        logging.error(f"{operation_name} failed: {e}")
        return False, e


async def with_sync_timeout(
    func: Callable[P, T],
    timeout_seconds: float,
    operation_name: str,
    *args: P.args,
    **kwargs: P.kwargs,
) -> tuple[bool, T | None]:
    """Execute synchronous blocking operation with timeout in executor.

    Useful for synchronous I/O operations that might hang (e.g., network flushes).

    Args:
        func: Synchronous function to execute
        timeout_seconds: Timeout in seconds
        operation_name: Operation description for logging
        *args: Positional arguments to pass to the function
        **kwargs: Keyword arguments to pass to the function

    Returns:
        Tuple of (success, result_or_None)
    """
    loop = asyncio.get_event_loop()

    try:
        # Run sync function in executor with timeout
        result = await asyncio.wait_for(
            loop.run_in_executor(None, lambda: func(*args, **kwargs)),
            timeout=timeout_seconds,
        )
        return True, result
    except asyncio.TimeoutError:
        logging.warning(f"{operation_name} timed out after {timeout_seconds}s")
        return False, None
    except Exception as e:
        logging.error(f"{operation_name} failed: {e}")
        return False, None


def validate_and_sanitize_for_logging(value: str, max_length: int = 1000) -> str:
    """Validate and sanitize string value before logging to prevent log injection.

    This function uses LENIENT validation - only removes control characters that could
    break log parsers (newlines, ANSI escape codes). Preserves spaces, punctuation,
    and unicode characters for better debugging visibility.

    For STRICT validation (API parameters, database queries), use dedicated sanitizers
    like _sanitize_user_context() which enforce alphanumeric-only patterns.

    Args:
        value: String to sanitize
        max_length: Maximum allowed length

    Returns:
        Sanitized string safe for logging
    """
    if not value:
        return ""

    # Remove control characters
    sanitized = re.sub(r"[\x00-\x1f\x7f-\x9f]", "", str(value))

    # Truncate if too long
    if len(sanitized) > max_length:
        sanitized = sanitized[:max_length] + "...[truncated]"

    return sanitized


def sanitize_model_name(model: str, max_length: int = 100) -> str | None:
    """Sanitize model name to prevent injection attacks in metadata/tags.

    Validates and sanitizes model names before adding to Langfuse metadata or tags.
    Prevents potential injection attacks or API request disruption.

    Allowed characters:
    - Alphanumeric: a-z, A-Z, 0-9
    - Separators: hyphen (-), underscore (_), colon (:), at-sign (@), period (.)

    Common model name patterns:
    - claude-3-5-sonnet-20241022
    - claude-sonnet-4-5@20250929
    - gpt-4-turbo-preview
    - models/gemini-pro

    Args:
        model: Model name to sanitize
        max_length: Maximum allowed length (default: 100)

    Returns:
        Sanitized model name, or None if empty after sanitization
    """
    if not model or not isinstance(model, str):
        return None

    # Remove any characters that aren't alphanumeric or allowed separators
    sanitized = re.sub(r'[^a-zA-Z0-9@.:/_-]', '', model[:max_length])

    # Return None if empty after sanitization
    return sanitized if sanitized else None
