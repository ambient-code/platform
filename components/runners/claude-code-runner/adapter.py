#!/usr/bin/env python3
"""
Claude Code Adapter for AG-UI Server.

Refactored from wrapper.py to use async generators that yield AG-UI events
instead of WebSocket messaging. This is the core adapter that wraps the
Claude Code SDK and produces a stream of AG-UI protocol events.
"""

import asyncio
import json as _json
import logging
import os
import re
import shutil
import sys
import uuid
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, AsyncIterator, Optional
from urllib import error as _urllib_error
from urllib import request as _urllib_request
from urllib.parse import urlparse, urlunparse

# Set umask to make files readable by content service container
os.umask(0o022)

# AG-UI Protocol Events
from ag_ui.core import (
    BaseEvent,
    EventType,
    RawEvent,
    RunAgentInput,
    RunErrorEvent,
    RunFinishedEvent,
    RunStartedEvent,
    StateDeltaEvent,
    StateSnapshotEvent,
    StepFinishedEvent,
    StepStartedEvent,
    TextMessageContentEvent,
    TextMessageEndEvent,
    TextMessageStartEvent,
    ToolCallArgsEvent,
    ToolCallEndEvent,
    ToolCallStartEvent,
)

from context import RunnerContext

logger = logging.getLogger(__name__)


class PrerequisiteError(RuntimeError):
    """Raised when slash-command prerequisites are missing."""

    pass


class ClaudeCodeAdapter:
    """
    Adapter that wraps the Claude Code SDK for AG-UI server.

    Produces AG-UI events via async generator instead of WebSocket.
    """

    def __init__(self):
        self.context: Optional[RunnerContext] = None
        self.last_exit_code = 1
        self._restart_requested = False
        self._first_run = True
        self._skip_resume_on_restart = False
        self._turn_count = 0

        # AG-UI streaming state (per-run, not instance state)
        # NOTE: _current_message_id and _current_tool_id are now local variables
        # in _run_claude_agent_sdk to avoid race conditions with concurrent runs
        self._current_run_id: Optional[str] = None
        self._current_thread_id: Optional[str] = None

        # Active client reference for interrupt support
        self._active_client: Optional[Any] = None

    async def initialize(self, context: RunnerContext):
        """Initialize the adapter with context."""
        self.context = context
        logger.info(f"Initialized Claude Code adapter for session {context.session_id}")

        # Copy Google OAuth credentials from mounted Secret to writable workspace location
        await self._setup_google_credentials()

        # Workspace is already prepared by init container (hydrate.sh)
        # - Repos cloned to /workspace/repos/
        # - Workflows cloned to /workspace/workflows/
        # - State hydrated from S3 to .claude/, artifacts/, file-uploads/
        logger.info("Workspace prepared by init container, validating...")

        # Validate prerequisite files exist for phase-based commands
        try:
            await self._validate_prerequisites()
        except PrerequisiteError as exc:
            self.last_exit_code = 2
            logger.error(
                "Prerequisite validation failed during initialization: %s", exc
            )
            raise

    def _timestamp(self) -> str:
        """Return current UTC timestamp in ISO format."""
        return datetime.now(timezone.utc).isoformat()

    async def process_run(self, input_data: RunAgentInput) -> AsyncIterator[BaseEvent]:
        """
        Process a run and yield AG-UI events.

        This is the main entry point called by the FastAPI server.

        Args:
            input_data: RunAgentInput with thread_id, run_id, messages, tools
            app_state: Optional FastAPI app.state for persistent client storage/reuse

        Yields:
            AG-UI events (RunStartedEvent, TextMessageContentEvent, etc.)
        """
        thread_id = input_data.thread_id or self.context.session_id
        run_id = input_data.run_id or str(uuid.uuid4())

        self._current_thread_id = thread_id
        self._current_run_id = run_id

        # Check for newly available Google OAuth credentials (user may have authenticated mid-session)
        # This picks up credentials after K8s syncs the mounted secret (~60s after OAuth completes)
        await self.refresh_google_credentials()

        try:
            # Emit RUN_STARTED
            yield RunStartedEvent(
                type=EventType.RUN_STARTED,
                thread_id=thread_id,
                run_id=run_id,
            )

            # Echo user messages as events (for history/display)
            for msg in input_data.messages or []:
                msg_dict = (
                    msg
                    if isinstance(msg, dict)
                    else (msg.model_dump() if hasattr(msg, "model_dump") else {})
                )
                role = msg_dict.get("role", "")

                if role == "user":
                    msg_id = msg_dict.get("id", str(uuid.uuid4()))
                    content = msg_dict.get("content", "")
                    msg_metadata = msg_dict.get("metadata", {})

                    # Check if message should be hidden from UI
                    is_hidden = isinstance(msg_metadata, dict) and msg_metadata.get(
                        "hidden", False
                    )
                    if is_hidden:
                        logger.info(
                            f"Message {msg_id[:8]} marked as hidden (auto-sent initial/workflow prompt)"
                        )

                    # Emit user message as TEXT_MESSAGE events
                    # Include metadata in RAW event for frontend filtering
                    if is_hidden:
                        yield RawEvent(
                            type=EventType.RAW,
                            thread_id=thread_id,
                            run_id=run_id,
                            event={
                                "type": "message_metadata",
                                "messageId": msg_id,
                                "metadata": msg_metadata,
                                "hidden": True,
                            },
                        )

                    yield TextMessageStartEvent(
                        type=EventType.TEXT_MESSAGE_START,
                        thread_id=thread_id,
                        run_id=run_id,
                        message_id=msg_id,
                        role="user",
                    )

                    if content:
                        yield TextMessageContentEvent(
                            type=EventType.TEXT_MESSAGE_CONTENT,
                            thread_id=thread_id,
                            run_id=run_id,
                            message_id=msg_id,
                            delta=content,
                        )

                    yield TextMessageEndEvent(
                        type=EventType.TEXT_MESSAGE_END,
                        thread_id=thread_id,
                        run_id=run_id,
                        message_id=msg_id,
                    )

            # Extract user message from input
            logger.info(
                f"Extracting user message from {len(input_data.messages)} messages"
            )
            user_message = self._extract_user_message(input_data)
            logger.info(
                f"Extracted user message: '{user_message[:100] if user_message else '(empty)'}...'"
            )

            if not user_message:
                logger.warning("No user message found in input")
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=thread_id,
                    run_id=run_id,
                    event={"type": "system_log", "message": "No user message provided"},
                )
                yield RunFinishedEvent(
                    type=EventType.RUN_FINISHED,
                    thread_id=thread_id,
                    run_id=run_id,
                )
                return

            # Run Claude SDK and yield events
            logger.info(f"Starting Claude SDK with prompt: '{user_message[:50]}...'")
            async for event in self._run_claude_agent_sdk(
                user_message, thread_id, run_id
            ):
                yield event
            logger.info(f"Claude SDK processing completed for run {run_id}")

            # Emit RUN_FINISHED
            yield RunFinishedEvent(
                type=EventType.RUN_FINISHED,
                thread_id=thread_id,
                run_id=run_id,
            )

            self.last_exit_code = 0

        except PrerequisiteError as e:
            self.last_exit_code = 2
            logger.error(f"Prerequisite validation failed: {e}")
            yield RunErrorEvent(
                type=EventType.RUN_ERROR,
                thread_id=thread_id,
                run_id=run_id,
                message=str(e),
            )
        except Exception as e:
            self.last_exit_code = 1
            logger.error(f"Error in process_run: {e}")
            yield RunErrorEvent(
                type=EventType.RUN_ERROR,
                thread_id=thread_id,
                run_id=run_id,
                message=str(e),
            )

    def _extract_user_message(self, input_data: RunAgentInput) -> str:
        """Extract user message text from RunAgentInput."""
        messages = input_data.messages or []
        logger.info(
            f"Extracting from {len(messages)} messages, types: {[type(m).__name__ for m in messages]}"
        )

        # Find the last user message
        for msg in reversed(messages):
            logger.debug(
                f"Checking message: type={type(msg).__name__}, hasattr(role)={hasattr(msg, 'role')}"
            )

            if hasattr(msg, "role") and msg.role == "user":
                # Handle different content formats
                content = getattr(msg, "content", "")
                if isinstance(content, str):
                    logger.info(
                        f"Found user message (object format): '{content[:50]}...'"
                    )
                    return content
                elif isinstance(content, list):
                    # Content blocks format
                    for block in content:
                        if hasattr(block, "text"):
                            return block.text
                        elif isinstance(block, dict) and "text" in block:
                            return block["text"]
            elif isinstance(msg, dict):
                logger.debug(
                    f"Dict message: role={msg.get('role')}, content={msg.get('content', '')[:30]}..."
                )
                if msg.get("role") == "user":
                    content = msg.get("content", "")
                    if isinstance(content, str):
                        logger.info(
                            f"Found user message (dict format): '{content[:50]}...'"
                        )
                        return content

        logger.warning("No user message found!")
        return ""

    async def _run_claude_agent_sdk(
        self, prompt: str, thread_id: str, run_id: str
    ) -> AsyncIterator[BaseEvent]:
        """Execute the Claude Code SDK with the given prompt and yield AG-UI events.

        Creates a fresh client for each run - simpler and more reliable than client reuse.

        Args:
            prompt: The user prompt to send to Claude
            thread_id: AG-UI thread identifier
            run_id: AG-UI run identifier
        """
        # Per-run state - NOT instance variables to avoid race conditions with concurrent runs
        current_message_id: Optional[str] = None

        logger.info(
            f"_run_claude_agent_sdk called with prompt length={len(prompt)}, will create fresh client"
        )
        try:
            # Refresh Google credentials before each run (picks up newly authenticated creds)
            await self.refresh_google_credentials()

            # Check for authentication method
            logger.info("Checking authentication configuration...")
            api_key = self.context.get_env("ANTHROPIC_API_KEY", "")
            use_vertex = (
                self.context.get_env("CLAUDE_CODE_USE_VERTEX", "").strip() == "1"
            )

            logger.info(
                f"Auth config: api_key={'set' if api_key else 'not set'}, use_vertex={use_vertex}"
            )

            if not api_key and not use_vertex:
                raise RuntimeError(
                    "Either ANTHROPIC_API_KEY or CLAUDE_CODE_USE_VERTEX=1 must be set"
                )

            # Set environment variables BEFORE importing SDK
            if api_key:
                os.environ["ANTHROPIC_API_KEY"] = api_key
                logger.info("Using Anthropic API key authentication")

            # Configure Vertex AI if requested
            if use_vertex:
                vertex_credentials = await self._setup_vertex_credentials()
                if "ANTHROPIC_API_KEY" in os.environ:
                    logger.info("Clearing ANTHROPIC_API_KEY to force Vertex AI mode")
                    del os.environ["ANTHROPIC_API_KEY"]

                os.environ["CLAUDE_CODE_USE_VERTEX"] = "1"
                os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = vertex_credentials.get(
                    "credentials_path", ""
                )
                os.environ["ANTHROPIC_VERTEX_PROJECT_ID"] = vertex_credentials.get(
                    "project_id", ""
                )
                os.environ["CLOUD_ML_REGION"] = vertex_credentials.get("region", "")

            # NOW we can safely import the SDK
            from claude_agent_sdk import (
                AssistantMessage,
                ClaudeAgentOptions,
                ClaudeSDKClient,
                ResultMessage,
                SystemMessage,
                TextBlock,
                ThinkingBlock,
                ToolResultBlock,
                ToolUseBlock,
                UserMessage,
                create_sdk_mcp_server,
            )
            from claude_agent_sdk import tool as sdk_tool
            from claude_agent_sdk.types import StreamEvent

            from observability import ObservabilityManager

            # Extract and sanitize user context for observability
            raw_user_id = os.getenv("USER_ID", "").strip()
            raw_user_name = os.getenv("USER_NAME", "").strip()
            user_id, user_name = self._sanitize_user_context(raw_user_id, raw_user_name)

            # Get model configuration
            model = self.context.get_env("LLM_MODEL")
            configured_model = model or "claude-sonnet-4-5@20250929"

            if use_vertex and model:
                configured_model = self._map_to_vertex_model(model)

            # Initialize observability
            obs = ObservabilityManager(
                session_id=self.context.session_id, user_id=user_id, user_name=user_name
            )
            await obs.initialize(
                prompt=prompt,
                namespace=self.context.get_env("AGENTIC_SESSION_NAMESPACE", "unknown"),
                model=configured_model,
            )
            obs._pending_initial_prompt = prompt

            # Check if this is a resume session via IS_RESUME env var
            # This is set by the operator when restarting a stopped/completed/failed session
            is_continuation = (
                self.context.get_env("IS_RESUME", "").strip().lower() == "true"
            )
            if is_continuation:
                logger.info("IS_RESUME=true - treating as continuation")

            # Determine cwd and additional dirs
            repos_cfg = self._get_repos_config()
            cwd_path = self.context.workspace_path
            add_dirs = []
            derived_name = None

            # Check for active workflow first
            active_workflow_url = (os.getenv("ACTIVE_WORKFLOW_GIT_URL") or "").strip()
            if active_workflow_url:
                cwd_path, add_dirs, derived_name = self._setup_workflow_paths(
                    active_workflow_url, repos_cfg
                )
            elif repos_cfg:
                cwd_path, add_dirs = self._setup_multi_repo_paths(repos_cfg)
            else:
                cwd_path = str(Path(self.context.workspace_path) / "artifacts")

            # Load ambient.json configuration
            ambient_config = (
                self._load_ambient_config(cwd_path) if active_workflow_url else {}
            )

            # Ensure working directory exists
            cwd_path_obj = Path(cwd_path)
            if not cwd_path_obj.exists():
                logger.warning(
                    f"Working directory does not exist, creating: {cwd_path}"
                )
                try:
                    cwd_path_obj.mkdir(parents=True, exist_ok=True)
                except Exception as e:
                    logger.error(f"Failed to create working directory: {e}")
                    cwd_path = self.context.workspace_path

            logger.info(f"Claude SDK CWD: {cwd_path}")
            logger.info(f"Claude SDK additional directories: {add_dirs}")

            # Load MCP server configuration (webfetch is included in static .mcp.json)
            mcp_servers = self._load_mcp_config(cwd_path) or {}

            # Pre-flight check: Validate MCP server authentication status
            # Import here to avoid circular dependency
            from main import _check_mcp_authentication

            mcp_auth_warnings = []
            if mcp_servers:
                for server_name in mcp_servers.keys():
                    is_auth, msg = _check_mcp_authentication(server_name)

                    if is_auth is False:
                        # Authentication definitely failed
                        mcp_auth_warnings.append(f"⚠️  {server_name}: {msg}")
                    elif is_auth is None:
                        # Authentication needs refresh or uncertain
                        mcp_auth_warnings.append(f"ℹ️  {server_name}: {msg}")

            if mcp_auth_warnings:
                warning_msg = "**MCP Server Authentication Issues:**\n\n" + "\n".join(mcp_auth_warnings)
                warning_msg += "\n\nThese servers may not work correctly until re-authenticated."
                logger.warning(warning_msg)

                # Yield a user-visible message about auth issues
                # Generate IDs for this warning message
                warning_message_id = str(uuid.uuid4())

                yield TextMessageStartEvent(
                    type=EventType.TEXT_MESSAGE_START,
                    thread_id=thread_id,
                    run_id=run_id,
                    message_id=warning_message_id,
                    role="assistant"
                )
                yield TextMessageContentEvent(
                    type=EventType.TEXT_MESSAGE_CONTENT,
                    thread_id=thread_id,
                    run_id=run_id,
                    message_id=warning_message_id,
                    delta=warning_msg
                )
                yield TextMessageEndEvent(
                    type=EventType.TEXT_MESSAGE_END,
                    thread_id=thread_id,
                    run_id=run_id,
                    message_id=warning_message_id
                )

            # Create custom session control tools
            # Capture self reference for the restart tool closure
            adapter_ref = self

            @sdk_tool(
                "restart_session",
                "Restart the Claude session to recover from issues, clear state, or get a fresh connection. Use this if you detect you're in a broken state or need to reset.",
                {},
            )
            async def restart_session_tool(args: dict) -> dict:
                """Tool that allows Claude to request a session restart."""
                adapter_ref._restart_requested = True
                logger.info("🔄 Session restart requested by Claude via MCP tool")
                return {
                    "content": [
                        {
                            "type": "text",
                            "text": "Session restart has been requested. The current run will complete and a fresh session will be established. Your conversation context will be preserved on disk.",
                        }
                    ]
                }

            # Create SDK MCP server for session tools
            session_tools_server = create_sdk_mcp_server(
                name="session", version="1.0.0", tools=[restart_session_tool]
            )
            mcp_servers["session"] = session_tools_server
            logger.info("Added custom session control MCP tools (restart_session)")

            # Disable built-in WebFetch in favor of WebFetch.MCP from config
            allowed_tools = [
                "Read",
                "Write",
                "Bash",
                "Glob",
                "Grep",
                "Edit",
                "MultiEdit",
                "WebSearch",
            ]
            if mcp_servers:
                for server_name in mcp_servers.keys():
                    allowed_tools.append(f"mcp__{server_name}")
                logger.info(
                    f"MCP tool permissions granted for servers: {list(mcp_servers.keys())}"
                )

            # Build workspace context system prompt
            workspace_prompt = self._build_workspace_context_prompt(
                repos_cfg=repos_cfg,
                workflow_name=derived_name if active_workflow_url else None,
                artifacts_path="artifacts",
                ambient_config=ambient_config,
            )
            # SystemPromptPreset format: uses claude_code preset with appended workspace context
            system_prompt_config = {
                "type": "preset",
                "preset": "claude_code",
                "append": workspace_prompt,
            }

            # Capture stderr from the SDK to diagnose MCP server failures
            def sdk_stderr_handler(line: str):
                logger.warning(f"[SDK stderr] {line.rstrip()}")

            # Configure SDK options
            options = ClaudeAgentOptions(
                cwd=cwd_path,
                permission_mode="acceptEdits",
                allowed_tools=allowed_tools,
                mcp_servers=mcp_servers,
                setting_sources=["project"],
                system_prompt=system_prompt_config,
                include_partial_messages=True,
                stderr=sdk_stderr_handler,
            )

            if self._skip_resume_on_restart:
                self._skip_resume_on_restart = False

            # Set additional options
            try:
                if add_dirs:
                    options.add_dirs = add_dirs
            except Exception:
                pass

            if model:
                try:
                    options.model = configured_model
                except Exception:
                    pass

            max_tokens_env = self.context.get_env(
                "LLM_MAX_TOKENS"
            ) or self.context.get_env("MAX_TOKENS")
            if max_tokens_env:
                try:
                    options.max_tokens = int(max_tokens_env)
                except Exception:
                    pass

            temperature_env = self.context.get_env(
                "LLM_TEMPERATURE"
            ) or self.context.get_env("TEMPERATURE")
            if temperature_env:
                try:
                    options.temperature = float(temperature_env)
                except Exception:
                    pass

            result_payload = None
            current_message = None
            sdk_session_id = None

            def create_sdk_client(opts, disable_continue=False):
                if disable_continue and hasattr(opts, "continue_conversation"):
                    opts.continue_conversation = False
                return ClaudeSDKClient(options=opts)

            # Create fresh client for each run
            # (Python SDK has issues with client reuse despite docs suggesting it should work)
            logger.info("Creating new ClaudeSDKClient for this run...")

            # Enable continue_conversation to resume from disk state
            if not self._first_run or is_continuation:
                try:
                    options.continue_conversation = True
                    logger.info(
                        "Enabled continue_conversation (will resume from disk state)"
                    )
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=thread_id,
                        run_id=run_id,
                        event={
                            "type": "system_log",
                            "message": "🔄 Resuming conversation from disk state",
                        },
                    )
                except Exception as e:
                    logger.warning(f"Failed to set continue_conversation: {e}")

            try:
                logger.info("Creating ClaudeSDKClient...")
                client = create_sdk_client(options)
                logger.info("Connecting ClaudeSDKClient (initializing subprocess)...")
                await client.connect()
                logger.info("ClaudeSDKClient connected successfully!")
            except Exception as resume_error:
                error_str = str(resume_error).lower()
                if "no conversation found" in error_str or "session" in error_str:
                    logger.warning(f"Conversation continuation failed: {resume_error}")
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=thread_id,
                        run_id=run_id,
                        event={
                            "type": "system_log",
                            "message": "⚠️ Could not continue conversation, starting fresh...",
                        },
                    )
                    client = create_sdk_client(options, disable_continue=True)
                    await client.connect()
                else:
                    raise

            try:
                # Store client reference for interrupt support
                self._active_client = client

                # Process the prompt
                step_id = str(uuid.uuid4())
                yield StepStartedEvent(
                    type=EventType.STEP_STARTED,
                    thread_id=thread_id,
                    run_id=run_id,
                    step_id=step_id,
                    step_name="processing_prompt",
                )

                logger.info(f"Sending query to Claude SDK: '{prompt[:100]}...'")
                await client.query(prompt)
                logger.info("Query sent, waiting for response stream...")

                # Process response stream
                logger.info("Starting to consume receive_response() iterator...")
                message_count = 0

                async for message in client.receive_response():
                    message_count += 1
                    logger.info(
                        f"[ClaudeSDKClient Message #{message_count}]: {message}"
                    )

                    # Handle StreamEvent for real-time streaming chunks
                    if isinstance(message, StreamEvent):
                        event_data = message.event
                        event_type = event_data.get("type")

                        if event_type == "message_start":
                            current_message_id = str(uuid.uuid4())
                            yield TextMessageStartEvent(
                                type=EventType.TEXT_MESSAGE_START,
                                thread_id=thread_id,
                                run_id=run_id,
                                message_id=current_message_id,
                                role="assistant",
                            )

                        elif event_type == "content_block_delta":
                            delta_data = event_data.get("delta", {})
                            if delta_data.get("type") == "text_delta":
                                text_chunk = delta_data.get("text", "")
                                if text_chunk and current_message_id:
                                    yield TextMessageContentEvent(
                                        type=EventType.TEXT_MESSAGE_CONTENT,
                                        thread_id=thread_id,
                                        run_id=run_id,
                                        message_id=current_message_id,
                                        delta=text_chunk,
                                    )
                        continue

                    # Capture SDK session ID from init message
                    if isinstance(message, SystemMessage):
                        if message.subtype == "init" and message.data.get("session_id"):
                            sdk_session_id = message.data.get("session_id")
                            logger.info(f"Captured SDK session ID: {sdk_session_id}")

                    if isinstance(message, (AssistantMessage, UserMessage)):
                        if isinstance(message, AssistantMessage):
                            current_message = message
                            obs.start_turn(configured_model, user_input=prompt)

                            # Emit trace_id for feedback association
                            # Frontend can use this to link feedback to specific Langfuse traces
                            trace_id = obs.get_current_trace_id()
                            if trace_id:
                                yield RawEvent(
                                    type=EventType.RAW,
                                    thread_id=thread_id,
                                    run_id=run_id,
                                    event={
                                        "type": "langfuse_trace",
                                        "traceId": trace_id,
                                    },
                                )

                        # Process all blocks in the message
                        for block in getattr(message, "content", []) or []:
                            if isinstance(block, TextBlock):
                                text_piece = getattr(block, "text", None)
                                if text_piece:
                                    logger.info(
                                        f"TextBlock received (complete), text length={len(text_piece)}"
                                    )

                            elif isinstance(block, ToolUseBlock):
                                tool_name = getattr(block, "name", "") or "unknown"
                                tool_input = getattr(block, "input", {}) or {}
                                tool_id = getattr(block, "id", None) or str(
                                    uuid.uuid4()
                                )
                                parent_tool_use_id = getattr(
                                    message, "parent_tool_use_id", None
                                )

                                logger.info(
                                    f"ToolUseBlock detected: {tool_name} (id={tool_id[:12]})"
                                )

                                yield ToolCallStartEvent(
                                    type=EventType.TOOL_CALL_START,
                                    thread_id=thread_id,
                                    run_id=run_id,
                                    tool_call_id=tool_id,
                                    tool_call_name=tool_name,
                                    parent_tool_call_id=parent_tool_use_id,
                                )

                                if tool_input:
                                    args_json = _json.dumps(tool_input)
                                    yield ToolCallArgsEvent(
                                        type=EventType.TOOL_CALL_ARGS,
                                        thread_id=thread_id,
                                        run_id=run_id,
                                        tool_call_id=tool_id,
                                        delta=args_json,
                                    )

                                obs.track_tool_use(tool_name, tool_id, tool_input)

                            elif isinstance(block, ToolResultBlock):
                                tool_use_id = getattr(block, "tool_use_id", None)
                                content = getattr(block, "content", None)
                                is_error = getattr(block, "is_error", None)
                                result_text = getattr(block, "text", None)
                                result_content = (
                                    content if content is not None else result_text
                                )

                                if result_content is not None:
                                    try:
                                        result_str = _json.dumps(result_content)
                                    except (TypeError, ValueError):
                                        result_str = str(result_content)
                                else:
                                    result_str = ""

                                if tool_use_id:
                                    yield ToolCallEndEvent(
                                        type=EventType.TOOL_CALL_END,
                                        thread_id=thread_id,
                                        run_id=run_id,
                                        tool_call_id=tool_use_id,
                                        result=result_str if not is_error else None,
                                        error=result_str if is_error else None,
                                    )

                                obs.track_tool_result(
                                    tool_use_id, result_content, is_error or False
                                )

                            elif isinstance(block, ThinkingBlock):
                                thinking_text = getattr(block, "thinking", "")
                                signature = getattr(block, "signature", "")
                                yield RawEvent(
                                    type=EventType.RAW,
                                    thread_id=thread_id,
                                    run_id=run_id,
                                    event={
                                        "type": "thinking_block",
                                        "thinking": thinking_text,
                                        "signature": signature,
                                    },
                                )

                        # End text message after processing all blocks
                        if getattr(message, "content", []) and current_message_id:
                            yield TextMessageEndEvent(
                                type=EventType.TEXT_MESSAGE_END,
                                thread_id=thread_id,
                                run_id=run_id,
                                message_id=current_message_id,
                            )
                            current_message_id = None

                    elif isinstance(message, SystemMessage):
                        text = getattr(message, "text", None)
                        if text:
                            yield RawEvent(
                                type=EventType.RAW,
                                thread_id=thread_id,
                                run_id=run_id,
                                event={
                                    "type": "system_log",
                                    "level": "debug",
                                    "message": str(text),
                                },
                            )

                    elif isinstance(message, ResultMessage):
                        usage_raw = getattr(message, "usage", None)
                        sdk_num_turns = getattr(message, "num_turns", None)

                        logger.info(
                            f"ResultMessage: num_turns={sdk_num_turns}, usage={usage_raw}"
                        )

                        # Convert usage object to dict if needed
                        if usage_raw is not None and not isinstance(usage_raw, dict):
                            try:
                                if hasattr(usage_raw, "__dict__"):
                                    usage_raw = usage_raw.__dict__
                                elif hasattr(usage_raw, "model_dump"):
                                    usage_raw = usage_raw.model_dump()
                            except Exception as e:
                                logger.warning(
                                    f"Could not convert usage object to dict: {e}"
                                )

                        # Update turn count
                        if (
                            sdk_num_turns is not None
                            and sdk_num_turns > self._turn_count
                        ):
                            self._turn_count = sdk_num_turns

                        # Complete turn tracking
                        if current_message:
                            obs.end_turn(
                                self._turn_count,
                                current_message,
                                usage_raw if isinstance(usage_raw, dict) else None,
                            )
                            current_message = None

                        result_payload = {
                            "subtype": getattr(message, "subtype", None),
                            "duration_ms": getattr(message, "duration_ms", None),
                            "is_error": getattr(message, "is_error", None),
                            "num_turns": getattr(message, "num_turns", None),
                            "total_cost_usd": getattr(message, "total_cost_usd", None),
                            "usage": usage_raw,
                            "result": getattr(message, "result", None),
                        }

                        # Emit state delta with result
                        yield StateDeltaEvent(
                            type=EventType.STATE_DELTA,
                            thread_id=thread_id,
                            run_id=run_id,
                            delta=[
                                {
                                    "op": "replace",
                                    "path": "/lastResult",
                                    "value": result_payload,
                                }
                            ],
                        )

                # End step
                yield StepFinishedEvent(
                    type=EventType.STEP_FINISHED,
                    thread_id=thread_id,
                    run_id=run_id,
                    step_id=step_id,
                    step_name="processing_prompt",
                )

                logger.info(
                    f"Response iterator fully consumed ({message_count} messages total)"
                )

                # Mark first run complete
                self._first_run = False

                # Check if restart was requested by Claude
                if self._restart_requested:
                    logger.info("🔄 Restart was requested, emitting restart event")
                    self._restart_requested = False  # Reset flag
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=thread_id,
                        run_id=run_id,
                        event={
                            "type": "session_restart_requested",
                            "message": "Claude requested a session restart. Reconnecting...",
                        },
                    )

            finally:
                # Clear active client reference
                self._active_client = None

                # Always disconnect client at end of run
                if client is not None:
                    logger.info("Disconnecting client (end of run)")
                    await client.disconnect()

            # Finalize observability
            await obs.finalize()

        except Exception as e:
            logger.error(f"Failed to run Claude Code SDK: {e}")
            if "obs" in locals():
                await obs.cleanup_on_error(e)
            raise

    async def interrupt(self) -> None:
        """
        Interrupt the active Claude SDK execution.
        """
        if self._active_client is None:
            logger.warning("Interrupt requested but no active client")
            return

        try:
            logger.info("Sending interrupt signal to Claude SDK client...")
            await self._active_client.interrupt()
            logger.info("Interrupt signal sent successfully")
        except Exception as e:
            logger.error(f"Failed to interrupt Claude SDK: {e}")

    def _setup_workflow_paths(
        self, active_workflow_url: str, repos_cfg: list
    ) -> tuple[str, list, str]:
        """Setup paths for workflow mode."""
        add_dirs = []
        derived_name = None
        cwd_path = self.context.workspace_path

        try:
            owner, repo, _ = self._parse_owner_repo(active_workflow_url)
            derived_name = repo or ""
            if not derived_name:
                p = urlparse(active_workflow_url)
                parts = [pt for pt in (p.path or "").split("/") if pt]
                if parts:
                    derived_name = parts[-1]
            derived_name = (derived_name or "").removesuffix(".git").strip()

            if derived_name:
                workflow_path = str(
                    Path(self.context.workspace_path) / "workflows" / derived_name
                )
                if Path(workflow_path).exists():
                    cwd_path = workflow_path
                    logger.info(f"Using workflow as CWD: {derived_name}")
                else:
                    logger.warning(
                        f"Workflow directory not found: {workflow_path}, using default"
                    )
                    cwd_path = str(
                        Path(self.context.workspace_path) / "workflows" / "default"
                    )
            else:
                cwd_path = str(
                    Path(self.context.workspace_path) / "workflows" / "default"
                )
        except Exception as e:
            logger.warning(f"Failed to derive workflow name: {e}, using default")
            cwd_path = str(Path(self.context.workspace_path) / "workflows" / "default")

        # Add all repos as additional directories (repos are in /workspace/repos/{name})
        repos_base = Path(self.context.workspace_path) / "repos"
        for r in repos_cfg:
            name = (r.get("name") or "").strip()
            if name:
                repo_path = str(repos_base / name)
                if repo_path not in add_dirs:
                    add_dirs.append(repo_path)

        # Add artifacts and file-uploads directories
        artifacts_path = str(Path(self.context.workspace_path) / "artifacts")
        if artifacts_path not in add_dirs:
            add_dirs.append(artifacts_path)

        file_uploads_path = str(Path(self.context.workspace_path) / "file-uploads")
        if file_uploads_path not in add_dirs:
            add_dirs.append(file_uploads_path)

        return cwd_path, add_dirs, derived_name

    def _setup_multi_repo_paths(self, repos_cfg: list) -> tuple[str, list]:
        """Setup paths for multi-repo mode.

        Repos are cloned to /workspace/repos/{name} by both:
        - hydrate.sh (init container)
        - clone_repo_at_runtime() (runtime addition)
        """
        add_dirs = []
        repos_base = Path(self.context.workspace_path) / "repos"

        main_name = (os.getenv("MAIN_REPO_NAME") or "").strip()
        if not main_name:
            idx_raw = (os.getenv("MAIN_REPO_INDEX") or "").strip()
            try:
                idx_val = int(idx_raw) if idx_raw else 0
            except Exception:
                idx_val = 0
            if idx_val < 0 or idx_val >= len(repos_cfg):
                idx_val = 0
            main_name = (repos_cfg[idx_val].get("name") or "").strip()

        # Main repo path is /workspace/repos/{name}
        cwd_path = (
            str(repos_base / main_name) if main_name else self.context.workspace_path
        )

        for r in repos_cfg:
            name = (r.get("name") or "").strip()
            if not name:
                continue
            # All repos are in /workspace/repos/{name}
            p = str(repos_base / name)
            if p != cwd_path:
                add_dirs.append(p)

        # Add artifacts and file-uploads directories
        artifacts_path = str(Path(self.context.workspace_path) / "artifacts")
        if artifacts_path not in add_dirs:
            add_dirs.append(artifacts_path)

        file_uploads_path = str(Path(self.context.workspace_path) / "file-uploads")
        if file_uploads_path not in add_dirs:
            add_dirs.append(file_uploads_path)

        return cwd_path, add_dirs

    @staticmethod
    def _sanitize_user_context(user_id: str, user_name: str) -> tuple[str, str]:
        """Validate and sanitize user context fields to prevent injection attacks."""
        if user_id:
            user_id = str(user_id).strip()
            if len(user_id) > 255:
                user_id = user_id[:255]
            sanitized_id = re.sub(r"[^a-zA-Z0-9@._-]", "", user_id)
            user_id = sanitized_id

        if user_name:
            user_name = str(user_name).strip()
            if len(user_name) > 255:
                user_name = user_name[:255]
            sanitized_name = re.sub(r"[\x00-\x1f\x7f-\x9f]", "", user_name)
            user_name = sanitized_name

        return user_id, user_name

    def _map_to_vertex_model(self, model: str) -> str:
        """Map Anthropic API model names to Vertex AI model names."""
        model_map = {
            "claude-opus-4-5": "claude-opus-4-5@20251101",
            "claude-opus-4-1": "claude-opus-4-1@20250805",
            "claude-sonnet-4-5": "claude-sonnet-4-5@20250929",
            "claude-haiku-4-5": "claude-haiku-4-5@20251001",
        }
        return model_map.get(model, model)

    async def _setup_vertex_credentials(self) -> dict:
        """Set up Google Cloud Vertex AI credentials from service account."""
        service_account_path = self.context.get_env(
            "GOOGLE_APPLICATION_CREDENTIALS", ""
        ).strip()
        project_id = self.context.get_env("ANTHROPIC_VERTEX_PROJECT_ID", "").strip()
        region = self.context.get_env("CLOUD_ML_REGION", "").strip()

        if not service_account_path:
            raise RuntimeError(
                "GOOGLE_APPLICATION_CREDENTIALS must be set when CLAUDE_CODE_USE_VERTEX=1"
            )
        if not project_id:
            raise RuntimeError(
                "ANTHROPIC_VERTEX_PROJECT_ID must be set when CLAUDE_CODE_USE_VERTEX=1"
            )
        if not region:
            raise RuntimeError(
                "CLOUD_ML_REGION must be set when CLAUDE_CODE_USE_VERTEX=1"
            )

        if not Path(service_account_path).exists():
            raise RuntimeError(
                f"Service account key file not found at {service_account_path}"
            )

        logger.info(f"Vertex AI configured: project={project_id}, region={region}")
        return {
            "credentials_path": service_account_path,
            "project_id": project_id,
            "region": region,
        }

    async def _prepare_workspace(self) -> AsyncIterator[BaseEvent]:
        """Validate workspace prepared by init container.

        The init-hydrate container now handles:
        - Downloading state from S3 (.claude/, artifacts/, file-uploads/)
        - Cloning repos to /workspace/repos/
        - Cloning workflows to /workspace/workflows/

        Runner just validates and logs what's ready.
        """
        workspace = Path(self.context.workspace_path)
        logger.info(f"Validating workspace at {workspace}")

        # Check what was hydrated
        hydrated_paths = []
        for path_name in [".claude", "artifacts", "file-uploads"]:
            path_dir = workspace / path_name
            if path_dir.exists():
                file_count = len([f for f in path_dir.rglob("*") if f.is_file()])
                if file_count > 0:
                    hydrated_paths.append(f"{path_name} ({file_count} files)")

        if hydrated_paths:
            logger.info(f"Hydrated from S3: {', '.join(hydrated_paths)}")
        else:
            logger.info("No state hydrated (fresh session)")

        # No further preparation needed - init container did the work

    async def _validate_prerequisites(self):
        """Validate prerequisite files exist for phase-based slash commands."""
        prompt = self.context.get_env("INITIAL_PROMPT", "")
        if not prompt:
            return

        prompt_lower = prompt.strip().lower()

        prerequisites = {
            "/speckit.plan": (
                "spec.md",
                "Specification file (spec.md) not found. Please run /speckit.specify first.",
            ),
            "/speckit.tasks": (
                "plan.md",
                "Planning file (plan.md) not found. Please run /speckit.plan first.",
            ),
            "/speckit.implement": (
                "tasks.md",
                "Tasks file (tasks.md) not found. Please run /speckit.tasks first.",
            ),
        }

        for cmd, (required_file, error_msg) in prerequisites.items():
            if prompt_lower.startswith(cmd):
                workspace = Path(self.context.workspace_path)
                found = False

                if (workspace / required_file).exists():
                    found = True
                    break

                for subdir in workspace.rglob("specs/*/"):
                    if (subdir / required_file).exists():
                        found = True
                        break

                if not found:
                    raise PrerequisiteError(error_msg)
                break

    async def _initialize_workflow_if_set(self) -> AsyncIterator[BaseEvent]:
        """Validate workflow was cloned by init container."""
        active_workflow_url = (os.getenv("ACTIVE_WORKFLOW_GIT_URL") or "").strip()
        if not active_workflow_url:
            return

        try:
            owner, repo, _ = self._parse_owner_repo(active_workflow_url)
            derived_name = repo or ""
            if not derived_name:
                p = urlparse(active_workflow_url)
                parts = [pt for pt in (p.path or "").split("/") if pt]
                if parts:
                    derived_name = parts[-1]
            derived_name = (derived_name or "").removesuffix(".git").strip()

            if not derived_name:
                logger.warning("Could not derive workflow name from URL")
                return

            # Check for cloned workflow (init container uses -clone-temp suffix)
            workspace = Path(self.context.workspace_path)
            workflow_temp_dir = workspace / "workflows" / f"{derived_name}-clone-temp"
            workflow_dir = workspace / "workflows" / derived_name

            if workflow_temp_dir.exists():
                logger.info(
                    f"Workflow {derived_name} cloned by init container at {workflow_temp_dir.name}"
                )
            elif workflow_dir.exists():
                logger.info(f"Workflow {derived_name} available at {workflow_dir.name}")
            else:
                logger.warning(
                    f"Workflow {derived_name} not found (init container may have failed to clone)"
                )

        except Exception as e:
            logger.error(f"Failed to validate workflow: {e}")

    async def _run_cmd(self, cmd, cwd=None, capture_stdout=False, ignore_errors=False):
        """Run a subprocess command asynchronously."""
        cmd_safe = [self._redact_secrets(str(arg)) for arg in cmd]
        logger.info(f"Running command: {' '.join(cmd_safe)}")

        proc = await asyncio.create_subprocess_exec(
            *cmd,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
            cwd=cwd or self.context.workspace_path,
        )
        stdout_data, stderr_data = await proc.communicate()
        stdout_text = stdout_data.decode("utf-8", errors="replace")
        stderr_text = stderr_data.decode("utf-8", errors="replace")

        if stdout_text.strip():
            logger.info(f"Command stdout: {self._redact_secrets(stdout_text.strip())}")
        if stderr_text.strip():
            logger.info(f"Command stderr: {self._redact_secrets(stderr_text.strip())}")

        if proc.returncode != 0 and not ignore_errors:
            raise RuntimeError(stderr_text or f"Command failed: {' '.join(cmd_safe)}")

        if capture_stdout:
            return stdout_text
        return ""

    def _url_with_token(self, url: str, token: str) -> str:
        """Add authentication token to URL."""
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

    def _redact_secrets(self, text: str) -> str:
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

    async def _fetch_token_for_url(self, url: str) -> str:
        """Fetch appropriate token based on repository URL."""
        try:
            parsed = urlparse(url)
            hostname = parsed.hostname or ""

            if "gitlab" in hostname.lower():
                token = os.getenv("GITLAB_TOKEN", "").strip()
                if token:
                    logger.info(f"Using GITLAB_TOKEN for {hostname}")
                    return token
                else:
                    logger.warning(f"No GITLAB_TOKEN found for GitLab URL: {url}")
                    return ""

            token = os.getenv("GITHUB_TOKEN") or await self._fetch_github_token()
            if token:
                logger.info(f"Using GitHub token for {hostname}")
            return token

        except Exception as e:
            logger.warning(
                f"Failed to parse URL {url}: {e}, falling back to GitHub token"
            )
            return os.getenv("GITHUB_TOKEN") or await self._fetch_github_token()

    async def _fetch_github_token(self) -> str:
        """Fetch GitHub token from backend API or environment."""
        cached = os.getenv("GITHUB_TOKEN", "").strip()
        if cached:
            logger.info("Using GITHUB_TOKEN from environment")
            return cached

        # Build mint URL from environment
        base = os.getenv("BACKEND_API_URL", "").rstrip("/")
        project = os.getenv("PROJECT_NAME", "").strip()
        session_id = self.context.session_id

        if not base or not project or not session_id:
            logger.warning("Cannot fetch GitHub token: missing environment variables")
            return ""

        url = f"{base}/projects/{project}/agentic-sessions/{session_id}/github/token"
        logger.info(f"Fetching GitHub token from: {url}")

        req = _urllib_request.Request(
            url, data=b"{}", headers={"Content-Type": "application/json"}, method="POST"
        )
        bot = (os.getenv("BOT_TOKEN") or "").strip()
        if bot:
            req.add_header("Authorization", f"Bearer {bot}")

        loop = asyncio.get_event_loop()

        def _do_req():
            try:
                with _urllib_request.urlopen(req, timeout=10) as resp:
                    return resp.read().decode("utf-8", errors="replace")
            except Exception as e:
                logger.warning(f"GitHub token fetch failed: {e}")
                return ""

        resp_text = await loop.run_in_executor(None, _do_req)
        if not resp_text:
            return ""

        try:
            data = _json.loads(resp_text)
            token = str(data.get("token") or "")
            if token:
                logger.info("Successfully fetched GitHub token from backend")
            return token
        except Exception as e:
            logger.error(f"Failed to parse token response: {e}")
            return ""

    def _parse_owner_repo(self, url: str) -> tuple[str, str, str]:
        """Return (owner, name, host) from various URL formats."""
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

    def _get_repos_config(self) -> list[dict]:
        """Read repos mapping from REPOS_JSON env if present.

        Expected format: [{"url": "...", "branch": "main", "autoPush": true}, ...]
        Returns: [{"name": "repo-name", "url": "...", "branch": "...", "autoPush": bool}, ...]
        """
        try:
            raw = os.getenv("REPOS_JSON", "").strip()
            if not raw:
                return []
            data = _json.loads(raw)
            if isinstance(data, list):
                out = []
                for it in data:
                    if not isinstance(it, dict):
                        continue

                    # Extract simple format fields
                    url = str(it.get("url") or "").strip()
                    # Auto-generate branch from session name if not provided
                    branch_from_json = it.get("branch")
                    if branch_from_json and str(branch_from_json).strip():
                        branch = str(branch_from_json).strip()
                    else:
                        # Fallback: use AGENTIC_SESSION_NAME to match backend logic
                        session_id = os.getenv("AGENTIC_SESSION_NAME", "").strip()
                        branch = f"ambient/{session_id}" if session_id else "main"
                    # Parse autoPush as boolean, defaulting to False for invalid types
                    auto_push_raw = it.get("autoPush", False)
                    auto_push = (
                        auto_push_raw if isinstance(auto_push_raw, bool) else False
                    )

                    if not url:
                        continue

                    # Derive repo name from URL if not provided
                    name = str(it.get("name") or "").strip()
                    if not name:
                        try:
                            owner, repo, _ = self._parse_owner_repo(url)
                            derived = repo or ""
                            if not derived:
                                p = urlparse(url)
                                parts = [pt for pt in (p.path or "").split("/") if pt]
                                if parts:
                                    derived = parts[-1]
                            name = (derived or "").removesuffix(".git").strip()
                        except Exception:
                            name = ""

                    if name and url:
                        out.append(
                            {
                                "name": name,
                                "url": url,
                                "branch": branch,
                                "autoPush": auto_push,
                            }
                        )
                return out
        except Exception:
            return []
        return []

    def _expand_env_vars(self, value: Any) -> Any:
        """Recursively expand ${VAR} and ${VAR:-default} patterns in config values."""
        if isinstance(value, str):
            # Pattern: ${VAR} or ${VAR:-default}
            pattern = r'\$\{([^}:]+)(?::-([^}]*))?\}'

            def replace_var(match):
                var_name = match.group(1)
                default_val = match.group(2) if match.group(2) is not None else ""
                return os.environ.get(var_name, default_val)

            return re.sub(pattern, replace_var, value)
        elif isinstance(value, dict):
            return {k: self._expand_env_vars(v) for k, v in value.items()}
        elif isinstance(value, list):
            return [self._expand_env_vars(item) for item in value]
        return value

    def _load_mcp_config(self, cwd_path: str) -> Optional[dict]:
        """Load MCP server configuration from the ambient runner's .mcp.json file."""
        try:
            # Allow override via MCP_CONFIG_FILE env var (useful for e2e with minimal MCPs)
            mcp_config_file = self.context.get_env(
                "MCP_CONFIG_FILE", "/app/claude-runner/.mcp.json"
            )
            runner_mcp_file = Path(mcp_config_file)

            if runner_mcp_file.exists() and runner_mcp_file.is_file():
                logger.info(f"Loading MCP config from: {runner_mcp_file}")
                with open(runner_mcp_file, "r") as f:
                    config = _json.load(f)
                    mcp_servers = config.get("mcpServers", {})
                    # Expand environment variables in the config
                    expanded = self._expand_env_vars(mcp_servers)
                    logger.info(f"Expanded MCP config env vars for {len(expanded)} servers")
                    return expanded
            else:
                logger.info(f"No MCP config file found at: {runner_mcp_file}")
                return None

        except _json.JSONDecodeError as e:
            logger.error(f"Failed to parse MCP config: {e}")
            return None
        except Exception as e:
            logger.error(f"Error loading MCP config: {e}")
            return None

    def _load_ambient_config(self, cwd_path: str) -> dict:
        """Load ambient.json configuration from workflow directory."""
        try:
            config_path = Path(cwd_path) / ".ambient" / "ambient.json"

            if not config_path.exists():
                logger.info(f"No ambient.json found at {config_path}, using defaults")
                return {}

            with open(config_path, "r") as f:
                config = _json.load(f)
                logger.info(f"Loaded ambient.json: name={config.get('name')}")
                return config

        except _json.JSONDecodeError as e:
            logger.error(f"Failed to parse ambient.json: {e}")
            return {}
        except Exception as e:
            logger.error(f"Error loading ambient.json: {e}")
            return {}

    def _build_workspace_context_prompt(
        self, repos_cfg, workflow_name, artifacts_path, ambient_config
    ):
        """Generate concise system prompt describing workspace layout."""
        prompt = "# Workspace Structure\n\n"

        # Workflow directory (if active)
        if workflow_name:
            prompt += f"**Working Directory**: workflows/{workflow_name}/ (workflow logic - do not create files here)\n\n"

        # Artifacts
        prompt += f"**Artifacts**: {artifacts_path} (create all output files here)\n\n"

        # Uploaded files
        file_uploads_path = Path(self.context.workspace_path) / "file-uploads"
        if file_uploads_path.exists() and file_uploads_path.is_dir():
            try:
                files = sorted(
                    [f.name for f in file_uploads_path.iterdir() if f.is_file()]
                )
                if files:
                    max_display = 10
                    if len(files) <= max_display:
                        prompt += f"**Uploaded Files**: {', '.join(files)}\n\n"
                    else:
                        prompt += f"**Uploaded Files** ({len(files)} total): {', '.join(files[:max_display])}, and {len(files) - max_display} more\n\n"
            except Exception:
                pass
        else:
            prompt += "**Uploaded Files**: None\n\n"

        # Repositories
        if repos_cfg:
            session_id = os.getenv("AGENTIC_SESSION_NAME", "").strip()
            feature_branch = f"ambient/{session_id}" if session_id else None

            repo_names = [
                repo.get("name", f"repo-{i}") for i, repo in enumerate(repos_cfg)
            ]
            if len(repo_names) <= 5:
                prompt += f"**Repositories**: {', '.join([f'repos/{name}/' for name in repo_names])}\n"
            else:
                prompt += f"**Repositories** ({len(repo_names)} total): {', '.join([f'repos/{name}/' for name in repo_names[:5]])}, and {len(repo_names) - 5} more\n"

            if feature_branch:
                prompt += f"**Working Branch**: `{feature_branch}` (all repos are on this feature branch)\n\n"
            else:
                prompt += "\n"

            # Add git push instructions for repos with autoPush enabled
            auto_push_repos = [
                repo for repo in repos_cfg if repo.get("autoPush", False)
            ]
            if auto_push_repos:
                push_branch = feature_branch or "ambient/<session-id>"

                prompt += "## Git Push Instructions\n\n"
                prompt += "The following repositories have auto-push enabled. When you make changes to these repositories, you MUST commit and push your changes:\n\n"
                for repo in auto_push_repos:
                    repo_name = repo.get("name", "unknown")
                    prompt += f"- **repos/{repo_name}/**\n"
                prompt += "\nAfter making changes to any auto-push repository:\n"
                prompt += "1. Use `git add` to stage your changes\n"
                prompt += '2. Use `git commit -m "description"` to commit with a descriptive message\n'
                prompt += f"3. Use `git push origin {push_branch}` to push to the remote repository\n\n"

        # MCP Integration Setup Instructions
        prompt += "## MCP Integrations\n"
        prompt += "If you need Google Drive access: Ask user to go to Integrations page in Ambient and authenticate with Google Drive.\n"
        prompt += "If you need Jira access: Ask user to go to Workspace Settings in Ambient and configure Jira credentials there.\n\n"

        # Workflow instructions (if any)
        if ambient_config.get("systemPrompt"):
            prompt += f"## Workflow Instructions\n{ambient_config['systemPrompt']}\n\n"

        return prompt

    async def _setup_google_credentials(self):
        """Copy Google OAuth credentials from mounted Secret to writable workspace location.

        The secret is always mounted (as placeholder if user hasn't authenticated).
        This method checks if credentials.json exists and has content.
        Call refresh_google_credentials() periodically to pick up new credentials after OAuth.
        """
        success = await self._try_copy_google_credentials()

        if success:
            # Extract and set USER_GOOGLE_EMAIL from credentials
            await self._set_google_user_email()

    async def _try_copy_google_credentials(self) -> bool:
        """Attempt to copy Google credentials from mounted secret.

        Returns:
            True if credentials were successfully copied, False otherwise.
        """
        secret_path = Path("/app/.google_workspace_mcp/credentials/credentials.json")

        # Check if secret file exists
        if not secret_path.exists():
            logging.debug(
                "Google OAuth credentials not found at %s (placeholder secret or not mounted)",
                secret_path,
            )
            return False

        # Check if file has content (not empty placeholder)
        try:
            if secret_path.stat().st_size == 0:
                logging.debug(
                    "Google OAuth credentials file is empty (user hasn't authenticated yet)"
                )
                return False
        except OSError as e:
            logging.debug("Could not stat Google OAuth credentials file: %s", e)
            return False

        # Create writable credentials directory in workspace
        workspace_creds_dir = Path("/workspace/.google_workspace_mcp/credentials")
        workspace_creds_dir.mkdir(parents=True, exist_ok=True)

        # Copy credentials from read-only Secret mount to writable workspace
        dest_path = workspace_creds_dir / "credentials.json"
        try:
            shutil.copy2(secret_path, dest_path)
            # Make it writable so workspace-mcp can update tokens
            dest_path.chmod(0o644)
            logging.info(
                "✓ Copied Google OAuth credentials from Secret to writable workspace at %s",
                dest_path,
            )
            return True
        except Exception as e:
            logging.error("Failed to copy Google OAuth credentials: %s", e)
            return False

    async def refresh_google_credentials(self) -> bool:
        """Check for and copy new Google OAuth credentials.

        Call this method periodically (e.g., before processing a message) to detect
        when a user completes the OAuth flow and credentials become available.

        Kubernetes automatically updates the mounted secret volume when the secret
        changes (typically within ~60 seconds), so this will pick up new credentials
        without requiring a pod restart.

        Also updates USER_GOOGLE_EMAIL environment variable if credentials changed.

        Returns:
            True if new credentials were found and copied, False otherwise.
        """
        dest_path = Path(
            "/workspace/.google_workspace_mcp/credentials/credentials.json"
        )

        # If we already have credentials in workspace, check if source is newer
        if dest_path.exists():
            secret_path = Path(
                "/app/.google_workspace_mcp/credentials/credentials.json"
            )
            if secret_path.exists():
                try:
                    # Compare modification times - secret mount updates when K8s syncs
                    if secret_path.stat().st_mtime > dest_path.stat().st_mtime:
                        logging.info(
                            "Detected updated Google OAuth credentials, refreshing..."
                        )
                        success = await self._try_copy_google_credentials()
                        if success:
                            # Update USER_GOOGLE_EMAIL when credentials refresh
                            await self._set_google_user_email()
                            logging.info("Google credentials refreshed successfully")
                        return success
                except OSError:
                    pass
            return False

        # No credentials yet, try to copy
        if await self._try_copy_google_credentials():
            logging.info(
                "✓ Google OAuth credentials now available (user completed authentication)"
            )
            # Set USER_GOOGLE_EMAIL for new credentials
            await self._set_google_user_email()
            return True
        return False

    async def _set_google_user_email(self):
        """Ensure USER_GOOGLE_EMAIL is set for the MCP server.

        The operator now sets USER_GOOGLE_EMAIL directly from cluster credentials.
        This function only logs the current value for debugging.
        """
        current_email = os.environ.get("USER_GOOGLE_EMAIL", "")
        if current_email and current_email != "user@example.com":
            logging.info(f"USER_GOOGLE_EMAIL already set by operator: {current_email}")
        else:
            logging.warning("USER_GOOGLE_EMAIL not set or is placeholder - Google MCP may not work")
