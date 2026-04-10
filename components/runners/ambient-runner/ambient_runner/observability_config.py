"""Which observability backends are active (Langfuse, MLflow)."""

from __future__ import annotations

import os


def _truthy_env(name: str) -> bool:
    """Return True when the given env var is set to a common affirmative value."""
    return os.getenv(name, "").strip().lower() in ("1", "true", "yes")


def observability_backend_names() -> frozenset[str]:
    """Parsed OBSERVABILITY_BACKENDS, or default ``langfuse`` only.

    Values (comma-separated, case-insensitive): ``langfuse``, ``mlflow``.
    Empty/unset means **langfuse** only for backward compatibility.
    """
    raw = os.getenv("OBSERVABILITY_BACKENDS", "").strip().lower()
    if not raw:
        return frozenset({"langfuse"})
    parts = {p.strip() for p in raw.split(",") if p.strip()}
    allowed = {"langfuse", "mlflow"}
    return frozenset(p for p in parts if p in allowed)


def use_langfuse_backend() -> bool:
    """True when ``langfuse`` is included in the active observability backends."""
    return "langfuse" in observability_backend_names()


def use_mlflow_backend() -> bool:
    """True when MLflow is selected, tracing is enabled, and a tracking URI is set."""
    if "mlflow" not in observability_backend_names():
        return False
    if not _truthy_env("MLFLOW_TRACING_ENABLED"):
        return False
    return bool(os.getenv("MLFLOW_TRACKING_URI", "").strip())
