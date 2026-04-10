"""MLflow GenAI tracing for ambient-runner (parallel to Langfuse)."""

from __future__ import annotations

import logging
import os
from typing import Any, Callable

logger = logging.getLogger(__name__)


class MLflowSessionTracer:
    """Mirrors turn/tool boundaries from ObservabilityManager into MLflow spans."""

    def __init__(self, session_id: str, user_id: str, user_name: str) -> None:
        self.session_id = session_id
        self.user_id = user_id
        self.user_name = user_name
        self._enabled = False
        self._namespace = ""
        self._mask_fn: Callable[[Any], Any] | None = None
        self._turn_gen: Any = None
        self._turn_span: Any = None
        self._tool_ctx: dict[str, tuple[Any, Any]] = {}

    @property
    def enabled(self) -> bool:
        return self._enabled

    @property
    def has_active_turn(self) -> bool:
        return self._turn_span is not None

    def initialize(
        self,
        *,
        prompt: str,
        namespace: str,
        model: str | None,
        workflow_url: str,
        workflow_branch: str,
        workflow_path: str,
        mask_fn: Callable[[Any], Any] | None,
    ) -> bool:
        """Configure tracking URI and experiment. Returns True on success."""
        try:
            import mlflow
            from mlflow.entities import SpanStatusCode
        except ImportError:
            logger.debug("MLflow not installed — tracing disabled")
            return False

        _ = SpanStatusCode  # noqa: F841 — import check

        tracking_uri = os.getenv("MLFLOW_TRACKING_URI", "").strip()
        if not tracking_uri:
            logger.warning(
                "MLFLOW_TRACING_ENABLED but MLFLOW_TRACKING_URI is empty — MLflow tracing disabled"
            )
            return False

        try:
            mlflow.set_tracking_uri(tracking_uri)
            exp_name = os.getenv("MLFLOW_EXPERIMENT_NAME", "ambient-code-sessions").strip()
            if not exp_name:
                exp_name = "ambient-code-sessions"
            mlflow.set_experiment(exp_name)
        except Exception as e:
            logger.warning("MLflow: failed to set tracking URI or experiment: %s", e)
            return False

        auth_mode = os.getenv("MLFLOW_TRACKING_AUTH", "").strip()
        if auth_mode:
            logger.info("MLflow: MLFLOW_TRACKING_AUTH=%s", auth_mode)
        if os.getenv("MLFLOW_WORKSPACE", "").strip():
            logger.info("MLflow: MLFLOW_WORKSPACE override is set")
        if auth_mode in ("kubernetes", "kubernetes-namespaced"):
            token_path = "/var/run/secrets/kubernetes.io/serviceaccount/token"
            if not os.path.isfile(token_path):
                logger.warning(
                    "MLflow: %s auth expects in-cluster service account token at %s (missing); "
                    "ensure the runner pod mounts the session service account token",
                    auth_mode,
                    token_path,
                )

        self._namespace = namespace
        self._mask_fn = mask_fn
        self._enabled = True
        logger.info(
            "MLflow: session tracing enabled (session_id=%s, experiment=%s)",
            self.session_id,
            exp_name,
        )
        _ = (prompt, model, workflow_url, workflow_branch, workflow_path)  # reserved for tags
        return True

    def _apply_mask(self, value: Any) -> Any:
        if self._mask_fn is None:
            return value
        try:
            return self._mask_fn(value)
        except Exception:
            return value

    def start_turn(self, model: str, user_input: str | None) -> None:
        if not self._enabled:
            return
        if self._turn_span is not None:
            logger.debug("MLflow: turn already active, skipping duplicate start_turn")
            return
        try:
            import mlflow
            from mlflow.entities import SpanType

            text_in = user_input if user_input is not None else "User input"
            text_in = self._apply_mask(text_in)

            gen = mlflow.start_span(
                name="claude_interaction",
                span_type=SpanType.CHAIN,
                attributes={
                    "ambient.session_id": self.session_id,
                    "ambient.user_id": self.user_id,
                    "ambient.namespace": self._namespace,
                    "llm.model_name": model,
                },
            )
            self._turn_gen = gen
            self._turn_span = gen.__enter__()
            self._turn_span.set_inputs(
                {"model": model, "messages": [{"role": "user", "content": text_in}]}
            )
            try:
                mlflow.update_current_trace(
                    tags={
                        "ambient.session_id": self.session_id,
                        "ambient.user_id": self.user_id,
                        "ambient.namespace": self._namespace,
                    },
                    metadata={
                        "user_name": self.user_name,
                    },
                )
            except Exception as te:
                logger.debug("MLflow: update_current_trace skipped: %s", te)
            logger.info("MLflow: started turn span (model=%s)", model)
        except Exception as e:
            logger.warning("MLflow: start_turn failed: %s", e, exc_info=True)
            self._reset_turn()

    def close_turn(
        self,
        turn_count: int,
        output_text: str,
        usage_details: dict[str, int] | None,
    ) -> None:
        if not self._enabled or self._turn_gen is None:
            return
        try:
            out = self._apply_mask(output_text or "(no text output)")
            attrs: dict[str, str] = {"turn": str(turn_count)}
            if usage_details:
                for k, v in usage_details.items():
                    attrs[f"usage.{k}"] = str(v)
            self._turn_span.set_outputs({"text": out})
            self._turn_span.set_attributes(attrs)
        except Exception as e:
            logger.warning("MLflow: close_turn update failed: %s", e)
        finally:
            try:
                self._turn_gen.__exit__(None, None, None)
            except Exception as e:
                logger.debug("MLflow: turn context exit: %s", e)
            self._reset_turn()
            try:
                import mlflow

                mlflow.flush_trace_async_logging()
            except Exception as fe:
                logger.debug("MLflow: flush after turn: %s", fe)

    def track_tool_use(self, tool_name: str, tool_id: str, tool_input: dict) -> None:
        if not self._enabled or self._turn_span is None:
            return
        try:
            import mlflow
            from mlflow.entities import SpanType

            inp = self._apply_mask(dict(tool_input) if tool_input else {})
            gen = mlflow.start_span(
                name=f"tool_{tool_name}",
                span_type=SpanType.TOOL,
                attributes={"tool_id": tool_id, "tool_name": tool_name},
            )
            span = gen.__enter__()
            span.set_inputs(inp)
            self._tool_ctx[tool_id] = (gen, span)
            logger.debug("MLflow: tool span started %s (%s)", tool_name, tool_id)
        except Exception as e:
            logger.debug("MLflow: track_tool_use failed: %s", e)

    def track_tool_result(self, tool_use_id: str, content: Any, is_error: bool) -> None:
        ctx = self._tool_ctx.pop(tool_use_id, None)
        if not ctx:
            return
        gen, span = ctx
        try:
            from mlflow.entities import SpanStatusCode

            result_text = str(content) if content else "No output"
            if len(result_text) > 500:
                result_text = result_text[:500] + "...[truncated]"
            result_text = self._apply_mask(result_text)
            span.set_outputs({"result": result_text})
            if is_error:
                span.set_status(SpanStatusCode.ERROR, description="tool_error")
        except Exception as e:
            logger.debug("MLflow: track_tool_result failed: %s", e)
        finally:
            try:
                gen.__exit__(None, None, None)
            except Exception as e:
                logger.debug("MLflow: tool context exit: %s", e)

    def record_turn_error(self) -> None:
        if not self._enabled or self._turn_span is None:
            return
        try:
            from mlflow.entities import SpanStatusCode

            self._turn_span.set_status(SpanStatusCode.ERROR, description="run_error")
        except Exception as e:
            logger.debug("MLflow: record_turn_error: %s", e)

    def emit_session_summary_span(self, metadata: dict[str, Any]) -> None:
        if not self._enabled:
            return
        try:
            import mlflow
            from mlflow.entities import SpanType

            with mlflow.start_span(
                name="Claude Code - Session Metrics",
                span_type=SpanType.CHAIN,
                attributes={"ambient.source": "claude-code-metrics"},
            ) as span:
                span.set_inputs(
                    {
                        "session_id": self.session_id,
                        "user_id": self.user_id,
                        "namespace": self._namespace,
                    }
                )
                span.set_outputs(metadata)
        except Exception as e:
            logger.warning("MLflow: session summary span failed: %s", e)

    def cleanup_on_error(self) -> None:
        """Close open spans after a run failure (mirrors Langfuse error cleanup)."""
        if not self._enabled:
            return
        for _tool_id, (gen, span) in list(self._tool_ctx.items()):
            try:
                from mlflow.entities import SpanStatusCode

                span.set_status(SpanStatusCode.ERROR, description="run_error")
            except Exception:
                pass
            try:
                gen.__exit__(None, None, None)
            except Exception:
                pass
        self._tool_ctx.clear()
        if self._turn_gen is not None:
            try:
                self.record_turn_error()
            except Exception:
                pass
            try:
                self._turn_gen.__exit__(None, None, None)
            except Exception:
                pass
            self._reset_turn()
        try:
            import mlflow

            mlflow.flush_trace_async_logging()
        except Exception:
            pass

    def finalize(self) -> None:
        if not self._enabled:
            return
        for tool_id, (gen, span) in list(self._tool_ctx.items()):
            try:
                from mlflow.entities import SpanStatusCode

                span.set_status(SpanStatusCode.ERROR, description="finalize_cleanup")
            except Exception:
                pass
            try:
                gen.__exit__(None, None, None)
            except Exception:
                pass
        self._tool_ctx.clear()

        if self._turn_gen is not None:
            try:
                self._turn_gen.__exit__(None, None, None)
            except Exception:
                pass
            self._reset_turn()

        try:
            import mlflow

            mlflow.flush_trace_async_logging()
        except Exception as e:
            logger.debug("MLflow: finalize flush: %s", e)

    def _reset_turn(self) -> None:
        self._turn_gen = None
        self._turn_span = None
