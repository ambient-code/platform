"""ObservabilityManager + MLflow wiring (mocked)."""

import os
import sys
import types
from unittest.mock import MagicMock, patch

import pytest

if "langfuse" not in sys.modules:
    _mf = types.ModuleType("langfuse")
    _mf.Langfuse = MagicMock
    _mf.propagate_attributes = MagicMock
    sys.modules["langfuse"] = _mf


@pytest.mark.asyncio
async def test_initialize_mlflow_only_invokes_tracer():
    from ambient_runner.observability import ObservabilityManager

    env = {
        "OBSERVABILITY_BACKENDS": "mlflow",
        "MLFLOW_TRACING_ENABLED": "true",
        "MLFLOW_TRACKING_URI": "file:///tmp/mlflow-obs-test",
        "LANGFUSE_ENABLED": "false",
    }
    with patch.dict(os.environ, env, clear=True):
        with patch(
            "ambient_runner.observability.MLflowSessionTracer",
        ) as MockTracer:
            inst = MockTracer.return_value
            inst.initialize.return_value = True

            manager = ObservabilityManager("s1", "u1", "n1")
            ok = await manager.initialize("prompt", "ns", model="m1")

            assert ok is True
            assert manager.langfuse_client is None
            MockTracer.assert_called_once_with(
                session_id="s1", user_id="u1", user_name="n1"
            )
            inst.initialize.assert_called_once()


@pytest.mark.asyncio
async def test_initialize_langfuse_only_does_not_construct_mlflow_tracer():
    from ambient_runner.observability import ObservabilityManager

    env = {
        "LANGFUSE_ENABLED": "true",
        "LANGFUSE_PUBLIC_KEY": "pk",
        "LANGFUSE_SECRET_KEY": "sk",
        "LANGFUSE_HOST": "http://localhost:3000",
    }
    with patch.dict(os.environ, env, clear=True):
        with patch("ambient_runner.observability.MLflowSessionTracer") as MockTracer:
            with patch("langfuse.Langfuse") as MockLangfuse:
                with patch("langfuse.propagate_attributes") as mock_prop:
                    cm = MagicMock()
                    mock_prop.return_value = cm
                    cm.__enter__ = MagicMock(return_value=None)
                    cm.__exit__ = MagicMock(return_value=None)

                    manager = ObservabilityManager("s1", "u1", "n1")
                    ok = await manager.initialize("p", "ns")

                    assert ok is True
                    MockLangfuse.assert_called_once()
                    MockTracer.assert_not_called()
