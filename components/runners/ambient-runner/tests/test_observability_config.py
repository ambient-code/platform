"""Tests for observability backend selection."""

import os
from unittest.mock import patch

from ambient_runner.observability_config import (
    observability_backend_names,
    use_langfuse_backend,
    use_mlflow_backend,
)


def test_default_backends_is_langfuse_only():
    with patch.dict(os.environ, {}, clear=True):
        assert observability_backend_names() == frozenset({"langfuse"})
        assert use_langfuse_backend() is True
        assert use_mlflow_backend() is False


def test_observability_backends_parsing():
    with patch.dict(
        os.environ,
        {"OBSERVABILITY_BACKENDS": "mlflow, langfuse ,unknown"},
        clear=True,
    ):
        assert observability_backend_names() == frozenset({"langfuse", "mlflow"})


def test_use_mlflow_requires_flags_and_uri():
    with patch.dict(
        os.environ,
        {
            "OBSERVABILITY_BACKENDS": "mlflow",
            "MLFLOW_TRACING_ENABLED": "true",
            "MLFLOW_TRACKING_URI": "http://mlflow:5000",
        },
        clear=True,
    ):
        assert use_mlflow_backend() is True

    with patch.dict(
        os.environ,
        {
            "OBSERVABILITY_BACKENDS": "mlflow",
            "MLFLOW_TRACING_ENABLED": "true",
            "MLFLOW_TRACKING_URI": "",
        },
        clear=True,
    ):
        assert use_mlflow_backend() is False


def test_mlflow_backend_ignored_without_backend_list():
    with patch.dict(
        os.environ,
        {
            "OBSERVABILITY_BACKENDS": "langfuse",
            "MLFLOW_TRACING_ENABLED": "true",
            "MLFLOW_TRACKING_URI": "http://mlflow:5000",
        },
        clear=True,
    ):
        assert use_mlflow_backend() is False
