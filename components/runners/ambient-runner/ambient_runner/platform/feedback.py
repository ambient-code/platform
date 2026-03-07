"""Platform-level feedback utilities shared across all bridges.

Provides public wrappers around Langfuse score logging so that any
bridge (Claude, Gemini, future) can log rubric evaluations and
corrections without depending on Claude-bridge internals.
"""

from ambient_runner.bridges.claude.corrections import (
    CORRECTION_SOURCES,
    CORRECTION_TYPES,
    _get_session_context,
    _log_correction_to_langfuse,
    build_target_map,
)
from ambient_runner.bridges.claude.tools import _log_to_langfuse

__all__ = [
    "CORRECTION_SOURCES",
    "CORRECTION_TYPES",
    "log_rubric_score",
    "log_correction",
    "get_session_context",
    "build_target_map",
]

# Module-level cache for get_session_context() — the function runs subprocess
# git commands that are slow and stable within a single process lifetime.
_session_context_cache: dict | None = None


def get_session_context() -> dict:
    """Return auto-captured session context (repos, workflow, session name).

    Result is cached for the process lifetime since repos and workflow info
    don't change mid-session. Reads from environment variables and falls back
    to workspace filesystem scanning when REPOS_JSON is not set.
    """
    global _session_context_cache
    if _session_context_cache is None:
        _session_context_cache = _get_session_context()
    return _session_context_cache


def log_rubric_score(
    score: float | None,
    comment: str,
    session_id: str,
    obs=None,
    metadata=None,
) -> tuple[bool, str | None]:
    """Log a rubric evaluation score to Langfuse.

    Args:
        score: Numeric evaluation score.
        comment: Evaluation reasoning.
        session_id: Current session ID.
        obs: Optional ObservabilityManager for trace correlation.
        metadata: Optional schema-driven metadata dict.

    Returns:
        (success, error_message) tuple.
    """
    return _log_to_langfuse(
        score=score,
        comment=comment,
        metadata=metadata,
        obs=obs,
        session_id=session_id,
    )


def log_correction(
    correction_type: str,
    agent_action: str,
    user_correction: str,
    session_id: str,
    target_label: str = "",
    obs=None,
    source: str = "human",
) -> tuple[bool, str | None]:
    """Log a correction to Langfuse for the improvement feedback loop.

    Args:
        correction_type: One of CORRECTION_TYPES.
        agent_action: What the agent did or assumed.
        user_correction: What the user said should have happened instead.
        session_id: Current session ID.
        target_label: Optional target label resolved against the target map.
        obs: Optional ObservabilityManager for trace correlation.
        source: One of CORRECTION_SOURCES ('human' or 'rubric').

    Returns:
        (success, error_message) tuple.
    """
    target_map = build_target_map(get_session_context())
    return _log_correction_to_langfuse(
        correction_type=correction_type,
        agent_action=agent_action,
        user_correction=user_correction,
        target_label=target_label,
        target_map=target_map,
        obs=obs,
        session_id=session_id,
        source=source,
    )
