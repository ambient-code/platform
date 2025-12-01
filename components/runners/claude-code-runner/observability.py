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

import os
import logging
from typing import Any
from urllib.parse import urlparse

from security_utils import (
    sanitize_exception_message,
    sanitize_model_name,
    with_sync_timeout,
)


class ObservabilityManager:
    """Manages Langfuse observability for Claude sessions.
    """

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
        self.langfuse_client = None
        self._propagate_ctx = None
        self._tool_spans: dict[str, Any] = {}  # Stores span objects directly
        self._current_turn_generation = None  # Track active turn for tool span parenting
        self._current_turn_ctx = None  # Track turn context manager for proper cleanup
        self._pending_initial_prompt = None  # Store initial prompt for turn 1

    async def initialize(self, prompt: str, namespace: str, model: str = None) -> bool:
        """Initialize Langfuse observability.

        Args:
            prompt: Initial prompt for the session
            namespace: Kubernetes namespace
            model: Model name to track in metadata (e.g., 'claude-3-5-sonnet-20241022')

        Returns:
            True if Langfuse initialized successfully
        """
        langfuse_enabled = os.getenv("LANGFUSE_ENABLED", "").strip().lower() in (
            "1",
            "true",
            "yes",
        )
        if not langfuse_enabled:
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
            logging.warning("LANGFUSE_HOST is missing. Add to secret (e.g., http://langfuse:3000).")
            return False

        # Validate host format
        try:
            parsed = urlparse(host)
            if not parsed.scheme or not parsed.netloc or parsed.scheme not in ("http", "https"):
                logging.warning(f"LANGFUSE_HOST invalid format: {host}")
                return False
        except Exception as e:
            logging.warning(f"Failed to parse LANGFUSE_HOST: {e}")
            return False

        try:
            # Initialize client
            self.langfuse_client = Langfuse(
                public_key=public_key, secret_key=secret_key, host=host
            )

            # Build metadata with model information
            metadata = {
                "namespace": namespace,
                "user_name": self.user_name,
                "initial_prompt": prompt[:200] if len(prompt) > 200 else prompt
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
                    # Add model as a tag for easy filtering in Langfuse UI
                    tags.append(f"model:{sanitized_model}")
                    logging.info(f"Langfuse: Model '{sanitized_model}' added to session metadata and tags")
                else:
                    logging.warning(f"Langfuse: Model name '{model}' failed sanitization - omitting from metadata")

            # Enter propagate_attributes context - all traces share session_id/user_id/tags/metadata
            # Each turn will be a separate trace, automatically grouped by session_id
            # Wrap context creation and __enter__ to ensure proper cleanup on failure
            try:
                self._propagate_ctx = propagate_attributes(
                    user_id=self.user_id,
                    session_id=self.session_id,
                    tags=tags,
                    metadata=metadata
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

            logging.info(f"Langfuse: Session tracking enabled (session_id={self.session_id}, user_id={self.user_id}, model={model})")
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
            logging.debug("Langfuse: Trace already active for current turn, skipping duplicate start_turn")
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
                logging.info(f"Langfuse: Starting turn trace with model={model} and actual user input")
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
            logging.info(f"Langfuse: Created new trace (model={model})")

        except Exception as e:
            logging.error(f"Langfuse: Failed to start turn: {e}", exc_info=True)

    def end_turn(self, turn_count: int, message: Any, usage: dict | None = None) -> None:
        """Complete turn tracking with output and usage data (called when ResultMessage arrives).

        Updates the turn generation with the assistant's output, usage metrics, and SDK's
        authoritative turn number in metadata, then closes it.

        Args:
            turn_count: Current turn number (from SDK's authoritative num_turns in ResultMessage)
            message: AssistantMessage from Claude SDK
            usage: Usage dict from ResultMessage with input_tokens, output_tokens, cache tokens, etc.
        """
        if not self._current_turn_generation:
            logging.warning(f"Langfuse: end_turn called but no active turn for turn {turn_count}")
            return

        try:
            from claude_agent_sdk import TextBlock

            # Extract text content
            text_content = []
            message_content = getattr(message, "content", []) or []
            for blk in message_content:
                if isinstance(blk, TextBlock):
                    text_content.append(getattr(blk, "text", ""))

            output_text = "\n".join(text_content) if text_content else "(no text output)"

            # Calculate usage_details if we have usage data
            usage_details_dict = None
            if usage and isinstance(usage, dict):
                input_tokens = usage.get("input_tokens", 0)
                output_tokens = usage.get("output_tokens", 0)
                cache_creation = usage.get("cache_creation_input_tokens", 0)
                cache_read = usage.get("cache_read_input_tokens", 0)

                # Langfuse canonical format with separate cache tokens for accurate cost calculation
                # Each token type has different pricing in Anthropic Claude:
                # - input: $3.00 per 1M tokens
                # - cache_creation_input_tokens: $3.75 per 1M (25% premium)
                # - cache_read_input_tokens: $0.30 per 1M (90% discount)
                usage_details_dict = {
                    "input": input_tokens,  # Regular input tokens only
                    "output": output_tokens,
                }

                # Add cache tokens separately if present for accurate cost calculation
                if cache_read > 0:
                    usage_details_dict["cache_read_input_tokens"] = cache_read
                if cache_creation > 0:
                    usage_details_dict["cache_creation_input_tokens"] = cache_creation

            # Update with output, usage_details, and turn number in metadata
            # SDK v3 requires 'usage_details' parameter for usage tracking
            update_params = {
                "output": output_text,
                "metadata": {"turn": turn_count}  # Add SDK's authoritative turn number
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
                    logging.warning(f"Langfuse: Flush failed after turn {turn_count}: {e}")

            if usage_details_dict:
                input_count = usage_details_dict.get('input', 0)
                output_count = usage_details_dict.get('output', 0)
                cache_read_count = usage_details_dict.get('cache_read_input_tokens', 0)
                cache_creation_count = usage_details_dict.get('cache_creation_input_tokens', 0)
                total_tokens = input_count + output_count + cache_read_count + cache_creation_count

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
                    logging.warning(f"Langfuse: Cleanup during error failed: {cleanup_error}")
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
                    metadata={"tool_id": tool_id, "tool_name": tool_name}
                )
                self._tool_spans[tool_id] = span
                logging.debug(f"Langfuse: Started tool span for {tool_name} (id={tool_id}) under turn")
            else:
                # Fallback: create orphaned span if no active turn (shouldn't happen)
                logging.warning(f"No active turn for tool {tool_name}, creating orphaned span")
                span = self.langfuse_client.start_observation(
                    as_type="span",
                    name=f"tool_{tool_name}",
                    input=tool_input,
                    metadata={"tool_id": tool_id, "tool_name": tool_name}
                )
                self._tool_spans[tool_id] = span
                logging.debug(f"Langfuse: Started orphaned tool span for {tool_name} (id={tool_id})")
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
                metadata={"is_error": is_error or False}
            )

            # End the span to close it properly
            tool_span.end()

            del self._tool_spans[tool_use_id]
            logging.debug(f"Langfuse: Completed tool span for {tool_use_id}")

        except Exception as e:
            logging.debug(f"Langfuse: Failed to track tool result: {e}")

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

            # Exit propagate_attributes context
            if self._propagate_ctx:
                self._propagate_ctx.__exit__(None, None, None)
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
                    logging.debug(f"Langfuse: Closed tool span {tool_id} during error cleanup")
                except Exception as e:
                    logging.warning(f"Failed to close tool span {tool_id} during error: {e}")
            self._tool_spans.clear()

            # Close propagate context
            if self._propagate_ctx:
                self._propagate_ctx.__exit__(None, None, None)

            # Timeout is configurable via LANGFUSE_FLUSH_TIMEOUT (default: 30s)
            flush_timeout = float(os.getenv("LANGFUSE_FLUSH_TIMEOUT", "30.0"))
            success, _ = await with_sync_timeout(
                self.langfuse_client.flush, flush_timeout, "Langfuse error flush"
            )
            if not success:
                logging.error(f"Langfuse: Error flush timed out after {flush_timeout}s")

        except Exception as cleanup_err:
            logging.error(f"Langfuse: Failed to cleanup: {cleanup_err}", exc_info=True)
