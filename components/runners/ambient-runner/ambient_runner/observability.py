"""
Observability manager for Claude Code runner - hybrid Langfuse integration.

Provides Langfuse LLM observability for Claude sessions with trace structure:

1. Turn Traces (top-level generations):
   - ONE trace per turn (SDK sends multiple AssistantMessages during streaming, but guard prevents duplicates)
   - Named: "claude_interaction" (turn number stored in metadata)
   - First AssistantMessage creates trace, subsequent ones ignored until end_turn() clears it
   - Final trace contains authoritative turn number and usage data from ResultMessage
   - Canonical format with separate cache token tracking for accurate cost
   - All traces grouped by session_id via propagate_attributes()

2. Tool Spans (observations within turn traces):
   - Named: tool_Read, tool_Write, tool_Bash, etc.
   - Shows tool execution in real-time
   - NO usage/cost data (prevents inflation from SDK's cumulative metrics)
   - Child observations of their parent turn trace

Architecture:
- Session-based grouping via propagate_attributes() with session_id and user_id
- Each turn creates ONE independent trace (not nested under session)
- Langfuse automatically aggregates tokens and costs across all traces with same session_id
- Filter by session_id, user_id, model, or metadata.turn in Langfuse UI
- Sessions can be paused/resumed: each turn creates a trace regardless of session lifecycle

Trace Hierarchy:
claude_interaction (trace - generation, metadata: {turn: 1})
├── tool_Read (observation - span)
└── tool_Write (observation - span)

claude_interaction (trace - generation, metadata: {turn: 2})
└── tool_Bash (observation - span)

Usage Format:
{
    "input": int,  # Regular input tokens
    "output": int,  # Output tokens
    "cache_read_input_tokens": int,  # Optional, 90% discount
    "cache_creation_input_tokens": int,  # Optional, 25% premium
}

Reference: https://langfuse.com/docs/observability/sdk/python/sdk-v3
"""

import logging
import os
from typing import Any
from urllib.parse import urlparse

from ambient_runner.platform.security_utils import (
    sanitize_exception_message,
    sanitize_model_name,
    validate_and_sanitize_for_logging,
    with_sync_timeout,
)

# Canonical token key names used across usage dicts from the Claude Agent SDK.
_TOKEN_KEYS = (
    "input_tokens",
    "output_tokens",
    "cache_creation_input_tokens",
    "cache_read_input_tokens",
)


def is_langfuse_enabled() -> bool:
    """Check whether Langfuse observability is enabled via env var."""
    return os.getenv("LANGFUSE_ENABLED", "").strip().lower() in ("1", "true", "yes")


def _privacy_masking_function(data: Any, **kwargs) -> Any:
    """Mask sensitive user inputs and outputs while preserving usage metrics.

    This function redacts message content (user prompts and assistant responses)
    to prevent logging potentially sensitive data, while preserving:
    - Usage metrics (token counts, costs)
    - Metadata (model, turn number, timestamps)
    - Session identifiers

    Controlled by LANGFUSE_MASK_MESSAGES environment variable:
    - "true" (default): Redact all message content for privacy
    - "false": Allow full message logging (use only in dev/testing)

    Args:
        data: Data to potentially mask (string, dict, list, or other)
        **kwargs: Additional context (unused but required by Langfuse API)

    Returns:
        Masked data with same structure as input
    """
    if isinstance(data, str):
        # Redact string content (likely message text)
        # Short strings (< 50 chars) might be metadata, keep them
        if len(data) > 50:
            return "[REDACTED FOR PRIVACY]"
        return data
    elif isinstance(data, dict):
        # Recursively process dict, preserving structure
        masked = {}
        for key, value in data.items():
            # Preserve usage and metadata fields - these don't contain sensitive data
            if key in (
                "usage",
                "usage_details",
                "metadata",
                "model",
                "turn",
                "input_tokens",
                "output_tokens",
                "cache_read_input_tokens",
                "cache_creation_input_tokens",
                "total_tokens",
                "cost_usd",
                "duration_ms",
                "duration_api_ms",
                "num_turns",
                "session_id",
                "tool_id",
                "tool_name",
                "is_error",
                "level",
            ):
                masked[key] = value
            # Redact content fields that may contain user data
            elif key in ("content", "text", "input", "output", "prompt", "completion"):
                if isinstance(value, str) and len(value) > 50:
                    masked[key] = "[REDACTED FOR PRIVACY]"
                else:
                    # Short values might be metadata/enums, recurse
                    masked[key] = _privacy_masking_function(value)
            else:
                # Recursively process other fields
                masked[key] = _privacy_masking_function(value)
        return masked
    elif isinstance(data, list):
        # Recursively process list items
        return [_privacy_masking_function(item) for item in data]
    else:
        # Preserve other types (numbers, booleans, None, etc.)
        return data


class ObservabilityManager:
    """Manages Langfuse observability for Claude sessions."""

    def __init__(self, session_id: str, user_id: str, user_name: str):
        """Initialize observability manager.

        Args:
            session_id: Unique session identifier
            user_id: Sanitized user ID
            user_name: Sanitized user name
        """
        self.session_id = session_id
        self.user_id = user_id
        self.user_name = user_name
        self.namespace = ""
        self.langfuse_client = None
        self._propagate_ctx = None
        self._tool_spans: dict[str, Any] = {}  # Stores span objects directly
        self._current_turn_generation = (
            None  # Track active turn for tool span parenting
        )
        self._current_turn_ctx = None  # Track turn context manager for proper cleanup
        self._pending_initial_prompt = None  # Store initial prompt for turn 1
        self._last_trace_id: str | None = None  # Persists after end_turn() for feedback

        # Session-level metrics (in-memory, accumulated across turns)
        self._metric_tool_calls: dict[str, int] = {}  # ToolCallType.value -> count
        self._metric_tool_calls_total: int = 0
        self._metric_tool_failures_total: int = 0
        self._metric_tool_failure_counts: dict[str, int] = {}
        self._metric_tool_failure_reasons: dict[str, int] = {}
        self._metric_unclear_context: int = 0
        self._metric_human_interrupts: int = 0
        self._metric_accumulated_usage: dict[str, int] = {}
        self._metric_models_seen: dict[str, int] = {}
        self._metric_total_cost_usd: float = 0.0
        # Track last seen usage to compute deltas (SDK may report cumulative values)
        self._metric_prev_usage: dict[str, int] = {}

    async def initialize(
        self,
        prompt: str,
        namespace: str,
        model: str = None,
        workflow_url: str = "",
        workflow_branch: str = "",
        workflow_path: str = "",
    ) -> bool:
        """Initialize Langfuse observability.

        Args:
            prompt: Initial prompt for the session
            namespace: Kubernetes namespace
            model: Model name to track in metadata (e.g., 'claude-3-5-sonnet-20241022')
            workflow_url: Active workflow git URL (from ACTIVE_WORKFLOW_GIT_URL)
            workflow_branch: Active workflow branch (from ACTIVE_WORKFLOW_BRANCH)
            workflow_path: Active workflow subpath (from ACTIVE_WORKFLOW_PATH)

        Returns:
            True if Langfuse initialized successfully
        """
        if not is_langfuse_enabled():
            return False

        try:
            from langfuse import Langfuse, propagate_attributes
        except ImportError:
            logging.debug("Langfuse not available - continuing without observability")
            return False

        public_key = os.getenv("LANGFUSE_PUBLIC_KEY", "").strip()
        secret_key = os.getenv("LANGFUSE_SECRET_KEY", "").strip()
        host = os.getenv("LANGFUSE_HOST", "").strip()

        if not public_key or not secret_key:
            logging.warning(
                "LANGFUSE_ENABLED is true but keys are missing. "
                "Create 'ambient-admin-langfuse-secret' with LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY."
            )
            return False

        if not host:
            logging.warning(
                "LANGFUSE_HOST is missing. Add to secret (e.g., http://langfuse:3000)."
            )
            return False

        # Validate host format
        try:
            parsed = urlparse(host)
            if (
                not parsed.scheme
                or not parsed.netloc
                or parsed.scheme not in ("http", "https")
            ):
                logging.warning(f"LANGFUSE_HOST invalid format: {host}")
                return False
        except Exception as e:
            logging.warning(f"Failed to parse LANGFUSE_HOST: {e}")
            return False

        try:
            # Determine if message masking should be enabled
            # Default: MASK messages (privacy-first approach)
            # Set LANGFUSE_MASK_MESSAGES=false to explicitly disable masking (dev/testing only)
            mask_messages_env = (
                os.getenv("LANGFUSE_MASK_MESSAGES", "true").strip().lower()
            )
            enable_masking = mask_messages_env not in ("false", "0", "no")

            if enable_masking:
                logging.info(
                    "Langfuse: Privacy masking ENABLED - user messages and responses will be redacted"
                )
                mask_fn = _privacy_masking_function
            else:
                logging.warning(
                    "Langfuse: Privacy masking DISABLED - full message content will be logged (use only for dev/testing)"
                )
                mask_fn = None

            # Initialize client with optional masking
            self.langfuse_client = Langfuse(
                public_key=public_key, secret_key=secret_key, host=host, mask=mask_fn
            )

            # Store namespace for use in session metrics span
            self.namespace = namespace

            # Build metadata with model information
            metadata = {
                "namespace": namespace,
                "user_name": self.user_name,
                "initial_prompt": prompt[:200] if len(prompt) > 200 else prompt,
            }

            # Build tags list
            tags = ["claude-code", f"namespace:{namespace}"]

            # Add model to metadata and tags if provided (after sanitization)
            # SECURITY: Model name is sanitized via sanitize_model_name() to prevent injection attacks.
            # Only alphanumeric chars and allowed separators (-, _, :, @, ., /) are permitted.
            # This prevents malicious tag values from disrupting Langfuse API or metadata storage.
            if model:
                sanitized_model = sanitize_model_name(model)
                if sanitized_model:
                    metadata["model"] = sanitized_model
                    tags.append(f"model:{sanitized_model}")
                    logging.info(
                        f"Langfuse: Model '{sanitized_model}' added to session metadata and tags"
                    )
                else:
                    logging.warning(
                        f"Langfuse: Model name '{model}' failed sanitization - omitting from metadata"
                    )

            # Add workflow context to metadata and tags when a workflow is active.
            # The operator sets ACTIVE_WORKFLOW_* env vars on runner pods; callers
            # read those and pass the values here so traces can be filtered by workflow.
            workflow_url = (workflow_url or "").strip()
            if workflow_url:
                raw_name = (
                    workflow_url.rstrip("/").split("/")[-1].removesuffix(".git").strip()
                )
                derived_name = sanitize_model_name(raw_name) or ""
                metadata["workflow_name"] = derived_name or "unknown"
                metadata["workflow_url"] = validate_and_sanitize_for_logging(
                    workflow_url
                )
                if workflow_branch:
                    metadata["workflow_branch"] = validate_and_sanitize_for_logging(
                        workflow_branch.strip()
                    )
                if workflow_path:
                    metadata["workflow_path"] = validate_and_sanitize_for_logging(
                        workflow_path.strip()
                    )
                if derived_name:
                    tags.append(f"workflow:{derived_name}")
                    logging.info(
                        f"Langfuse: Workflow '{derived_name}' added to session metadata and tags"
                    )
                else:
                    logging.info(
                        "Langfuse: Workflow added to session metadata (name could not be derived)"
                    )

            # Enter propagate_attributes context - all traces share session_id/user_id/tags/metadata
            # Each turn will be a separate trace, automatically grouped by session_id
            # Wrap context creation and __enter__ to ensure proper cleanup on failure
            try:
                self._propagate_ctx = propagate_attributes(
                    user_id=self.user_id,
                    session_id=self.session_id,
                    tags=tags,
                    metadata=metadata,
                )
                self._propagate_ctx.__enter__()
            except Exception:
                # Cleanup propagate context if __enter__ failed
                if self._propagate_ctx:
                    try:
                        self._propagate_ctx.__exit__(None, None, None)
                    except Exception:
                        pass
                    self._propagate_ctx = None
                raise

            logging.info(
                f"Langfuse: Session tracking enabled (session_id={self.session_id}, user_id={self.user_id}, model={model})"
            )
            return True

        except Exception as e:
            secrets = {"public_key": public_key, "secret_key": secret_key, "host": host}
            error_msg = sanitize_exception_message(e, secrets)
            logging.warning(f"Langfuse init failed: {error_msg}")

            # Cleanup on initialization failure
            if self._propagate_ctx:
                try:
                    self._propagate_ctx.__exit__(None, None, None)
                except Exception:
                    pass

            self.langfuse_client = None
            self._propagate_ctx = None
            return False

    def start_turn(self, model: str, user_input: str | None = None) -> None:
        """Start tracking a new turn as a top-level trace.

        Creates the turn generation as a TRACE (not an observation) so that each turn
        appears as a separate trace in Langfuse. Tools will be observations within the trace.

        Prevents duplicate traces when SDK sends multiple AssistantMessages per turn during
        streaming. Only the first AssistantMessage creates a trace; subsequent ones are ignored
        until end_turn() clears the current trace.

        Cannot use 'with' context managers due to async streaming architecture.
        Messages arrive asynchronously (AssistantMessage → ToolUseBlocks → ResultMessage)
        and the turn context must stay open across multiple async loop iterations.

        Args:
            model: Model name (e.g., "claude-3-5-sonnet-20241022")
            user_input: Optional actual user input/prompt (if available)
        """
        if not self.langfuse_client:
            return

        # Guard: Prevent creating duplicate traces for the same turn
        # SDK sends multiple AssistantMessages during streaming - only create trace once
        if self._current_turn_generation:
            logging.debug(
                "Langfuse: Trace already active for current turn, skipping duplicate start_turn"
            )
            return

        try:
            # Use pending initial prompt for turn 1 if available
            if user_input is None and self._pending_initial_prompt:
                user_input = self._pending_initial_prompt
                self._pending_initial_prompt = None  # Clear after use
                logging.debug("Langfuse: Using pending initial prompt")

            # Use actual user input if provided, otherwise use generic placeholder
            if user_input:
                input_content = [{"role": "user", "content": user_input}]
                logging.info(
                    f"Langfuse: Starting turn trace with model={model} and actual user input"
                )
            else:
                input_content = [{"role": "user", "content": "User input"}]
                logging.info(f"Langfuse: Starting turn trace with model={model}")

            # Create generation as a TRACE using start_as_current_observation()
            # Name doesn't include turn number - that will be added to metadata in end_turn()
            # This makes the trace a top-level observation, not nested
            # Tools will automatically become child observations of this trace
            self._current_turn_ctx = self.langfuse_client.start_as_current_observation(
                as_type="generation",
                name="claude_interaction",  # Generic name, turn number added in metadata
                input=input_content,
                model=model,
                metadata={},  # Turn number will be added in end_turn()
            )
            self._current_turn_generation = self._current_turn_ctx.__enter__()
            # Persist trace ID so it survives end_turn() — needed for feedback
            current_tid = self.get_current_trace_id()
            if current_tid:
                self._last_trace_id = current_tid
            logging.info(
                f"Langfuse: Created new trace (model={model}, trace_id={current_tid})"
            )

        except Exception as e:
            logging.error(f"Langfuse: Failed to start turn: {e}", exc_info=True)

    def get_current_trace_id(self) -> str | None:
        """Get the current turn's trace ID for feedback association.

        Returns:
            The Langfuse trace ID if a turn is active, None otherwise.
        """
        if not self._current_turn_generation:
            return None

        # The generation object has a trace_id attribute
        try:
            return getattr(self._current_turn_generation, "trace_id", None)
        except Exception:
            return None

    @property
    def last_trace_id(self) -> str | None:
        """Most recent Langfuse trace ID (persists after turn ends).

        Used by the feedback endpoint to attach scores to the correct trace
        without requiring the backend to scan the event log.
        """
        return self._last_trace_id

    def end_turn(
        self, turn_count: int, message: Any, usage: dict | None = None
    ) -> None:
        """Complete turn tracking with output and usage data (called when ResultMessage arrives).

        Updates the turn generation with the assistant's output, usage metrics, and SDK's
        authoritative turn number in metadata, then closes it.

        Args:
            turn_count: Current turn number (from SDK's authoritative num_turns in ResultMessage)
            message: AssistantMessage from Claude SDK
            usage: Usage dict from ResultMessage with input_tokens, output_tokens, cache tokens, etc.
        """
        # Return silently if Langfuse not initialized
        if not self.langfuse_client:
            return

        if not self._current_turn_generation:
            logging.debug(
                f"Langfuse: end_turn called but no active turn for turn {turn_count} (may not be initialized)"
            )
            return

        try:
            from claude_agent_sdk import TextBlock

            # Extract text content
            text_content = []
            message_content = getattr(message, "content", []) or []
            for blk in message_content:
                if isinstance(blk, TextBlock):
                    text_content.append(getattr(blk, "text", ""))

            output_text = (
                "\n".join(text_content) if text_content else "(no text output)"
            )

            # Calculate usage_details if we have usage data
            usage_details_dict = self._build_usage_details(usage)

            # Accumulate session-level usage metrics
            evt_model = getattr(self, "_evt_model", "")
            self._accumulate_usage(usage, evt_model)

            # Update with output, usage_details, and turn number in metadata
            # SDK v3 requires 'usage_details' parameter for usage tracking
            update_params = {
                "output": output_text,
                "metadata": {"turn": turn_count},  # Add SDK's authoritative turn number
            }
            if usage_details_dict:
                update_params["usage_details"] = usage_details_dict
            self._current_turn_generation.update(**update_params)

            # Exit the context manager to properly close the trace
            if self._current_turn_ctx:
                self._current_turn_ctx.__exit__(None, None, None)

            # Clear current turn state
            self._current_turn_generation = None
            self._current_turn_ctx = None

            # Flush data to Langfuse immediately after turn completes
            # This ensures traces appear in the UI during long-running sessions
            if self.langfuse_client:
                try:
                    self.langfuse_client.flush()
                    logging.info(f"Langfuse: Flushed turn {turn_count} data")
                except Exception as e:
                    logging.warning(
                        f"Langfuse: Flush failed after turn {turn_count}: {e}"
                    )

            if usage_details_dict:
                input_count = usage_details_dict.get("input", 0)
                output_count = usage_details_dict.get("output", 0)
                cache_read_count = usage_details_dict.get("cache_read_input_tokens", 0)
                cache_creation_count = usage_details_dict.get(
                    "cache_creation_input_tokens", 0
                )
                total_tokens = (
                    input_count + output_count + cache_read_count + cache_creation_count
                )

                log_msg = (
                    f"Langfuse: Completed turn {turn_count} - "
                    f"{input_count} input, {output_count} output"
                )
                if cache_read_count > 0 or cache_creation_count > 0:
                    log_msg += f", {cache_read_count} cache_read, {cache_creation_count} cache_creation"
                log_msg += f" (total: {total_tokens})"
                logging.info(log_msg)
            else:
                logging.info(f"Langfuse: Completed turn {turn_count} (no usage data)")

        except Exception as e:
            logging.error(f"Langfuse: Failed to end turn: {e}", exc_info=True)
            # Clean up turn state even on error
            if self._current_turn_ctx:
                try:
                    self._current_turn_ctx.__exit__(None, None, None)
                except Exception as cleanup_error:
                    logging.warning(
                        f"Langfuse: Cleanup during error failed: {cleanup_error}"
                    )
            self._current_turn_generation = None
            self._current_turn_ctx = None

    def track_tool_use(self, tool_name: str, tool_id: str, tool_input: dict) -> None:
        """Track tool use for visibility in Langfuse UI.

        Creates a span without usage data to show tool execution in real-time.
        Usage/cost tracking is done separately in track_interaction() from ResultMessage.

        Args:
            tool_name: Tool name (e.g., "Read", "Write", "Bash")
            tool_id: Unique tool use ID
            tool_input: Tool input parameters
        """
        if not self.langfuse_client:
            return

        try:
            # Create span as CHILD of current turn trace
            # Since turn is the current observation (via start_as_current_observation),
            # tools created via start_observation automatically become children
            # IMPORTANT: No usage_details parameter - avoids cumulative usage inflation
            if self._current_turn_generation:
                # Create as child of the current turn trace
                span = self._current_turn_generation.start_observation(
                    as_type="span",
                    name=f"tool_{tool_name}",
                    input=tool_input,
                    metadata={"tool_id": tool_id, "tool_name": tool_name},
                )
                self._tool_spans[tool_id] = span
                logging.debug(
                    f"Langfuse: Started tool span for {tool_name} (id={tool_id}) under turn"
                )
            else:
                # Fallback: create orphaned span if no active turn (shouldn't happen)
                logging.warning(
                    f"No active turn for tool {tool_name}, creating orphaned span"
                )
                span = self.langfuse_client.start_observation(
                    as_type="span",
                    name=f"tool_{tool_name}",
                    input=tool_input,
                    metadata={"tool_id": tool_id, "tool_name": tool_name},
                )
                self._tool_spans[tool_id] = span
                logging.debug(
                    f"Langfuse: Started orphaned tool span for {tool_name} (id={tool_id})"
                )
        except Exception as e:
            logging.debug(f"Langfuse: Failed to track tool use: {e}")

    def track_tool_result(self, tool_use_id: str, content: Any, is_error: bool) -> None:
        """Track tool result for visibility in Langfuse UI.

        Updates the tool span with result without adding usage data.

        Args:
            tool_use_id: Tool use ID
            content: Tool result content
            is_error: Whether execution failed
        """
        if tool_use_id not in self._tool_spans:
            return

        try:
            tool_span = self._tool_spans[tool_use_id]

            # Truncate long results for readability
            result_text = str(content) if content else "No output"
            if len(result_text) > 500:
                result_text = result_text[:500] + "...[truncated]"

            # IMPORTANT: No usage_details parameter - only result metadata
            tool_span.update(
                output={"result": result_text},
                level="ERROR" if is_error else "DEFAULT",
                metadata={"is_error": is_error or False},
            )

            # End the span to close it properly
            tool_span.end()

            del self._tool_spans[tool_use_id]
            logging.debug(f"Langfuse: Completed tool span for {tool_use_id}")

        except Exception as e:
            logging.debug(f"Langfuse: Failed to track tool result: {e}")

    # ------------------------------------------------------------------
    # Session-level metrics
    # ------------------------------------------------------------------

    def record_interrupt(self) -> None:
        """Record a human interrupt (called by the bridge on interrupt)."""
        self._metric_human_interrupts += 1
        logging.debug(
            f"Langfuse metrics: human interrupt recorded "
            f"(total={self._metric_human_interrupts})"
        )

    def _track_metrics_from_event(self, event: Any, etype: Any) -> None:
        """Accumulate session-level metrics from an AG-UI event.

        Called inside ``track_agui_event()`` for every event. Classifies
        tool calls by type, counts failures with reasons, and detects
        unclear-context signals (AskUserQuestion tool).

        Args:
            event: An AG-UI ``BaseEvent`` (or subclass).
            etype: The resolved ``EventType`` value (avoids redundant import).
        """
        from ag_ui.core import EventType
        from ambient_runner.observability_models import classify_tool

        if etype == EventType.TOOL_CALL_START:
            tool_name = getattr(event, "tool_call_name", "")
            tool_id = getattr(event, "tool_call_id", "")
            tool_type = classify_tool(tool_name).value

            # Cache the classification so TOOL_CALL_END can reuse it
            self._evt_tool_types[tool_id] = tool_type

            self._metric_tool_calls[tool_type] = (
                self._metric_tool_calls.get(tool_type, 0) + 1
            )
            self._metric_tool_calls_total += 1

            if tool_name == "AskUserQuestion":
                self._metric_unclear_context += 1

            logging.debug(
                f"Langfuse metrics: tool call {tool_name} -> {tool_type} "
                f"(total={self._metric_tool_calls_total})"
            )

        elif etype == EventType.TOOL_CALL_END:
            error = getattr(event, "error", None)
            if error:
                tool_id = getattr(event, "tool_call_id", "")
                # Reuse cached classification from TOOL_CALL_START
                tool_type = self._evt_tool_types.get(tool_id)
                if tool_type is None:
                    tool_name = self._evt_tool_names.get(tool_id, "unknown")
                    tool_type = classify_tool(tool_name).value

                self._metric_tool_failures_total += 1
                self._metric_tool_failure_counts[tool_type] = (
                    self._metric_tool_failure_counts.get(tool_type, 0) + 1
                )

                reason = str(error).split("\n", 1)[0].strip()[:80] or "Unknown"
                reason_key = f"{tool_type}: {reason}"
                self._metric_tool_failure_reasons[reason_key] = (
                    self._metric_tool_failure_reasons.get(reason_key, 0) + 1
                )

                logging.debug(
                    f"Langfuse metrics: tool failure "
                    f"{self._evt_tool_names.get(tool_id, 'unknown')} -> "
                    f"{tool_type}: {reason}"
                )

            # Clean up cached classification
            tool_id = getattr(event, "tool_call_id", "")
            self._evt_tool_types.pop(tool_id, None)

    def _accumulate_usage(self, usage: dict | None, model: str = "") -> None:
        """Accumulate token usage and cost from a completed turn.

        The Claude Agent SDK may report **cumulative** token counts in
        ``ResultMessage.usage`` (i.e., each query's usage includes all
        previous queries in the session).  To avoid double-counting, we
        compute the delta between the current and previous usage values
        and only accumulate the difference.

        If the SDK reports per-query values instead, the delta logic is
        still correct (prev is always zero-like since values only grow).

        Called from ``end_turn()`` and ``_close_turn_from_agui_result()``.
        """
        if not usage or not isinstance(usage, dict):
            return

        # Compute deltas against previous cumulative values
        delta_usage: dict[str, int] = {}
        for key in _TOKEN_KEYS:
            current = int(usage.get(key, 0))
            previous = self._metric_prev_usage.get(key, 0)
            delta = max(current - previous, 0)
            if delta > 0:
                delta_usage[key] = delta
                self._metric_accumulated_usage[key] = (
                    self._metric_accumulated_usage.get(key, 0) + delta
                )

        # Update prev to current for next call
        for key in _TOKEN_KEYS:
            val = int(usage.get(key, 0))
            if val > 0:
                self._metric_prev_usage[key] = val

        if model:
            self._metric_models_seen[model] = self._metric_models_seen.get(model, 0) + 1

        # Estimate cost from the delta, not the cumulative values
        from ambient_runner.observability_models import estimate_cost

        self._metric_total_cost_usd += estimate_cost(delta_usage, model)

    @staticmethod
    def _build_usage_details(usage: dict | None) -> dict | None:
        """Build Langfuse ``usage_details`` dict from SDK usage data.

        Returns a dict with ``input``, ``output``, and optional cache token
        fields, or ``None`` if *usage* is empty/invalid.
        """
        if not usage or not isinstance(usage, dict):
            return None
        input_tokens = usage.get("input_tokens", 0)
        output_tokens = usage.get("output_tokens", 0)
        cache_creation = usage.get("cache_creation_input_tokens", 0)
        cache_read = usage.get("cache_read_input_tokens", 0)
        details: dict[str, int] = {
            "input": input_tokens,
            "output": output_tokens,
        }
        if cache_read > 0:
            details["cache_read_input_tokens"] = cache_read
        if cache_creation > 0:
            details["cache_creation_input_tokens"] = cache_creation
        return details

    def _detect_and_record_clarification(self, text: str) -> None:
        """Check if assistant text is a clarification request and record it."""
        if not text:
            return

        from ambient_runner.observability_models import is_clarification_request

        if is_clarification_request(text):
            self._metric_unclear_context += 1
            logging.debug(
                "Langfuse metrics: clarification request detected in assistant text"
            )

    def _emit_session_summary(self) -> None:
        """Emit session-level summary metrics as Langfuse numeric scores.

        Creates a span with all accumulated metrics flattened into
        numeric scores for Langfuse dashboard visualization.
        """
        if not self.langfuse_client:
            return

        # Skip if no metrics were collected
        if self._metric_tool_calls_total == 0 and not self._metric_accumulated_usage:
            logging.debug("Langfuse metrics: no metrics to emit, skipping summary")
            return

        try:
            from ambient_runner.observability_models import SessionMetric

            metric = SessionMetric.build(
                session_id=self.session_id,
                user_id=self.user_id,
                tool_calls=self._metric_tool_calls,
                tool_calls_total=self._metric_tool_calls_total,
                tool_failures_total=self._metric_tool_failures_total,
                tool_failure_counts=self._metric_tool_failure_counts,
                tool_failure_reasons=self._metric_tool_failure_reasons,
                unclear_context=self._metric_unclear_context,
                human_interrupts=self._metric_human_interrupts,
                accumulated_usage=self._metric_accumulated_usage,
                models_seen=self._metric_models_seen,
                total_cost_usd=self._metric_total_cost_usd,
            )

            scores = metric.to_flat_scores()

            # Merge flat scores into metadata so all metrics are searchable
            # and plottable in the Langfuse dashboard without needing
            # individual score objects.
            span_metadata = {
                "source": "claude-code-metrics",
                "session_id": self.session_id,
                "user_id": self.user_id,
                "namespace": self.namespace,
                "user_name": self.user_name,
                "collected_at": metric.timestamp,
                "models_seen": metric.token_metrics.models_seen,
                **scores,
            }

            with self.langfuse_client.start_as_current_span(
                name="Claude Code - Session Metrics",
                input={
                    "session_id": self.session_id,
                    "user_id": self.user_id,
                    "namespace": self.namespace,
                    "user_name": self.user_name,
                },
                metadata=span_metadata,
            ) as metrics_span:
                metrics_span.update(output=metric.model_dump())

            logging.info(
                f"Langfuse metrics: emitted session summary as metadata "
                f"(tools={self._metric_tool_calls_total}, "
                f"cost=${self._metric_total_cost_usd:.4f})"
            )

        except Exception as e:
            logging.error(
                f"Langfuse metrics: failed to emit session summary: {e}",
                exc_info=True,
            )

    # ------------------------------------------------------------------
    # AG-UI event-driven tracking
    # ------------------------------------------------------------------

    def init_event_tracking(self, model: str, prompt: str) -> None:
        """Prepare the manager to track observability from AG-UI events.

        Call this once per run before feeding events via ``track_agui_event``.

        Args:
            model: Model name for the Langfuse generation.
            prompt: User prompt (used as input for the first turn trace).
        """
        self._evt_model = model
        self._evt_prompt = prompt
        self._evt_turn_started = False
        self._evt_accumulated_text = ""
        self._evt_tool_args: dict[str, str] = {}
        self._evt_tool_names: dict[str, str] = {}
        self._evt_tool_types: dict[str, str] = {}

    def track_agui_event(self, event: Any) -> None:
        """Track a single AG-UI event for Langfuse observability.

        Derives turn boundaries, tool calls, and result data entirely from the
        AG-UI event stream — no raw SDK messages needed.

        Args:
            event: An AG-UI ``BaseEvent`` (or subclass).
        """
        if not self.langfuse_client:
            return

        from ag_ui.core import EventType

        etype = getattr(event, "type", None)

        # Accumulate session-level metrics (tool counts, failures, etc.)
        self._track_metrics_from_event(event, etype)

        # --- Turn start: first assistant text message ----
        if etype == EventType.TEXT_MESSAGE_START:
            role = getattr(event, "role", "")
            if role == "assistant" and not self._evt_turn_started:
                self.start_turn(self._evt_model, user_input=self._evt_prompt)
                self._evt_turn_started = True

        # --- Accumulate streamed text ---
        elif etype == EventType.TEXT_MESSAGE_CONTENT:
            delta = getattr(event, "delta", "")
            if delta:
                self._evt_accumulated_text += delta

        # --- Tool call start ---
        elif etype == EventType.TOOL_CALL_START:
            tool_id = getattr(event, "tool_call_id", "")
            tool_name = getattr(event, "tool_call_name", "")
            self._evt_tool_names[tool_id] = tool_name
            self._evt_tool_args[tool_id] = ""
            # Create Langfuse span immediately (input details arrive later)
            self.track_tool_use(tool_name, tool_id, {})

        # --- Streaming tool arguments ---
        elif etype == EventType.TOOL_CALL_ARGS:
            tool_id = getattr(event, "tool_call_id", "")
            delta = getattr(event, "delta", "")
            if tool_id in self._evt_tool_args:
                self._evt_tool_args[tool_id] += delta

        # --- Tool call end ---
        elif etype == EventType.TOOL_CALL_END:
            tool_id = getattr(event, "tool_call_id", "")
            result = getattr(event, "result", None)
            error = getattr(event, "error", None)
            self.track_tool_result(tool_id, result or error, bool(error))
            self._evt_tool_args.pop(tool_id, None)
            self._evt_tool_names.pop(tool_id, None)

        # --- Run finished: close the turn with result data ---
        elif etype == EventType.RUN_FINISHED:
            self._close_turn_from_agui_result(event)

    def finalize_event_tracking(self) -> None:
        """Safety-net: close any open turn that was not ended by a RUN_FINISHED."""
        if not self.langfuse_client:
            return
        if self._evt_turn_started:
            self._close_turn_with_text(
                turn_count=1,
                text=self._evt_accumulated_text,
                usage=None,
            )
            self._evt_turn_started = False

    # --- private helpers for event tracking ---

    def _close_turn_from_agui_result(self, event: Any) -> None:
        """Extract result data from a ``RUN_FINISHED`` event and close the turn."""
        if not self._evt_turn_started:
            return

        result = getattr(event, "result", None)
        usage = None
        num_turns = 1

        if isinstance(result, dict):
            usage_raw = result.get("usage")
            if usage_raw is not None and not isinstance(usage_raw, dict):
                try:
                    if hasattr(usage_raw, "__dict__"):
                        usage_raw = usage_raw.__dict__
                    elif hasattr(usage_raw, "model_dump"):
                        usage_raw = usage_raw.model_dump()
                except Exception:
                    usage_raw = None
            usage = usage_raw if isinstance(usage_raw, dict) else None
            num_turns = result.get("num_turns", 1) or 1

        # Accumulate session-level usage metrics and check for clarification
        model = (
            result.get("model", self._evt_model)
            if isinstance(result, dict)
            else self._evt_model
        )
        self._accumulate_usage(usage, model)
        self._detect_and_record_clarification(self._evt_accumulated_text)

        self._close_turn_with_text(
            turn_count=num_turns,
            text=self._evt_accumulated_text,
            usage=usage,
        )
        self._evt_turn_started = False

    def _close_turn_with_text(
        self, turn_count: int, text: str, usage: dict | None
    ) -> None:
        """Close the current Langfuse turn using pre-accumulated text.

        This is the event-driven equivalent of ``end_turn`` — it does the same
        Langfuse bookkeeping but takes plain text instead of an SDK message.
        """
        if not self._current_turn_generation:
            return

        try:
            output_text = text or "(no text output)"

            usage_details_dict = self._build_usage_details(usage)

            update_params: dict[str, Any] = {
                "output": output_text,
                "metadata": {"turn": turn_count},
            }
            if usage_details_dict:
                update_params["usage_details"] = usage_details_dict
            self._current_turn_generation.update(**update_params)

            if self._current_turn_ctx:
                self._current_turn_ctx.__exit__(None, None, None)

            self._current_turn_generation = None
            self._current_turn_ctx = None

            if self.langfuse_client:
                try:
                    self.langfuse_client.flush()
                    logging.info(f"Langfuse: Flushed turn {turn_count} data")
                except Exception as e:
                    logging.warning(
                        f"Langfuse: Flush failed after turn {turn_count}: {e}"
                    )

            if usage_details_dict:
                total = sum(usage_details_dict.values())
                logging.info(
                    f"Langfuse: Completed turn {turn_count} "
                    f"({usage_details_dict.get('input', 0)} input, "
                    f"{usage_details_dict.get('output', 0)} output, "
                    f"total: {total})"
                )
            else:
                logging.info(f"Langfuse: Completed turn {turn_count} (no usage data)")

        except Exception as e:
            logging.error(f"Langfuse: Failed to close turn: {e}", exc_info=True)
            if self._current_turn_ctx:
                try:
                    self._current_turn_ctx.__exit__(None, None, None)
                except Exception:
                    pass
            self._current_turn_generation = None
            self._current_turn_ctx = None

    async def finalize(self) -> None:
        """Finalize and flush observability data."""
        if not self.langfuse_client:
            return

        try:
            # Close any open turn (if SDK didn't send ResultMessage)
            if self._current_turn_generation:
                try:
                    # Exit the turn context to properly close the trace
                    if self._current_turn_ctx:
                        self._current_turn_ctx.__exit__(None, None, None)
                    logging.debug("Langfuse: Closed turn during finalize")
                except Exception as e:
                    logging.warning(f"Failed to close turn: {e}")
                finally:
                    self._current_turn_generation = None
                    self._current_turn_ctx = None

            # Close any open tool spans
            for tool_id, tool_span in list(self._tool_spans.items()):
                try:
                    tool_span.end()
                    logging.debug(f"Langfuse: Closed tool span {tool_id}")
                except Exception as e:
                    logging.warning(f"Failed to close tool span {tool_id}: {e}")
            self._tool_spans.clear()

            # Emit session-level summary metrics before closing context
            self._emit_session_summary()

            # Exit propagate_attributes context.
            # The context uses OpenTelemetry contextvars internally.  When
            # initialize() and finalize() run in different async tasks (e.g.
            # lifespan startup vs shutdown), the contextvar token belongs to
            # a different Context and detach raises ValueError.  This is
            # harmless — all trace data has already been flushed — so we
            # suppress the error rather than letting it propagate.
            if self._propagate_ctx:
                try:
                    self._propagate_ctx.__exit__(None, None, None)
                except (ValueError, RuntimeError):
                    logging.debug(
                        "Langfuse: propagate_attributes context detach failed "
                        "(cross-task contextvar — safe to ignore)"
                    )
                logging.info("Langfuse: Session context closed")

            # Flush data
            # Timeout is configurable via LANGFUSE_FLUSH_TIMEOUT (default: 30s)
            # Increase for large traces or constrained networks to prevent data loss
            flush_timeout = float(os.getenv("LANGFUSE_FLUSH_TIMEOUT", "30.0"))
            success, _ = await with_sync_timeout(
                self.langfuse_client.flush, flush_timeout, "Langfuse flush"
            )
            if success:
                logging.info("Langfuse: Flush completed")
            else:
                logging.error(f"Langfuse: Flush timed out after {flush_timeout}s")

        except Exception as e:
            logging.error(f"Langfuse: Failed to finalize: {e}", exc_info=True)

    async def cleanup_on_error(self, error: Exception) -> None:
        """Cleanup on error.

        Args:
            error: Exception that caused failure
        """
        if not self.langfuse_client:
            return

        try:
            # Close any open turn
            if self._current_turn_generation:
                try:
                    # Mark as error but don't add fake output
                    self._current_turn_generation.update(level="ERROR")
                    # Exit the turn context to properly close the trace
                    if self._current_turn_ctx:
                        self._current_turn_ctx.__exit__(None, None, None)
                    logging.debug("Langfuse: Closed turn during error cleanup")
                except Exception as e:
                    logging.warning(f"Failed to close turn during error: {e}")
                finally:
                    self._current_turn_generation = None
                    self._current_turn_ctx = None

            # Close any open tool spans
            for tool_id, tool_span in list(self._tool_spans.items()):
                try:
                    tool_span.update(level="ERROR")
                    tool_span.end()
                    logging.debug(
                        f"Langfuse: Closed tool span {tool_id} during error cleanup"
                    )
                except Exception as e:
                    logging.warning(
                        f"Failed to close tool span {tool_id} during error: {e}"
                    )
            self._tool_spans.clear()

            # Close propagate context (see finalize() for cross-task note)
            if self._propagate_ctx:
                try:
                    self._propagate_ctx.__exit__(None, None, None)
                except (ValueError, RuntimeError):
                    pass

            # Timeout is configurable via LANGFUSE_FLUSH_TIMEOUT (default: 30s)
            flush_timeout = float(os.getenv("LANGFUSE_FLUSH_TIMEOUT", "30.0"))
            success, _ = await with_sync_timeout(
                self.langfuse_client.flush, flush_timeout, "Langfuse error flush"
            )
            if not success:
                logging.error(f"Langfuse: Error flush timed out after {flush_timeout}s")

        except Exception as cleanup_err:
            logging.error(f"Langfuse: Failed to cleanup: {cleanup_err}", exc_info=True)
