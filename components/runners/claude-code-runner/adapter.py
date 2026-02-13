#!/usr/bin/env python3
"""
Claude Code Adapter for AG-UI Server.

Core adapter that wraps the Claude Code SDK and produces a stream of
AG-UI protocol events.  Business logic is delegated to focused modules:

- ``auth``      — credential fetching and authentication setup
- ``config``    — ambient.json, MCP, and repos configuration
- ``workspace`` — path setup, validation, prerequisites
- ``prompts``   — system prompt construction and constants
- ``tools``     — MCP tool definitions (session, rubric)
- ``utils``     — general utilities (redaction, URL parsing, subprocesses)
"""

import json as _json
import logging
import os
import uuid
from pathlib import Path
from typing import Any, AsyncIterator, Optional

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
    StepFinishedEvent,
    StepStartedEvent,
    TextMessageContentEvent,
    TextMessageEndEvent,
    TextMessageStartEvent,
    ToolCallArgsEvent,
    ToolCallEndEvent,
    ToolCallStartEvent,
)

import auth
import config as runner_config
import prompts
import workspace
from context import RunnerContext
from tools import create_restart_session_tool, create_rubric_mcp_tool, load_rubric_content
from utils import redact_secrets, run_cmd, url_with_token, parse_owner_repo
from workspace import PrerequisiteError

logger = logging.getLogger(__name__)


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
        self._current_run_id: Optional[str] = None
        self._current_thread_id: Optional[str] = None

        # Active client reference for interrupt support
        self._active_client: Optional[Any] = None

    async def initialize(self, context: RunnerContext):
        """Initialize the adapter with context."""
        self.context = context
        logger.info(
            f"Initialized Claude Code adapter for session {context.session_id}"
        )

        # Credentials are fetched on-demand from backend API
        logger.info("Credentials will be fetched on-demand from backend API")

        # Workspace is already prepared by init container (hydrate.sh)
        logger.info("Workspace prepared by init container, validating...")

        # Validate prerequisite files for phase-based commands
        try:
            await workspace.validate_prerequisites(self.context)
        except PrerequisiteError as exc:
            self.last_exit_code = 2
            logger.error(
                "Prerequisite validation failed during initialization: %s", exc
            )
            raise

    async def process_run(
        self, input_data: RunAgentInput
    ) -> AsyncIterator[BaseEvent]:
        """Process a run and yield AG-UI events.

        This is the main entry point called by the FastAPI server.
        """
        thread_id = input_data.thread_id or self.context.session_id
        run_id = input_data.run_id or str(uuid.uuid4())

        self._current_thread_id = thread_id
        self._current_run_id = run_id

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
                    else (
                        msg.model_dump() if hasattr(msg, "model_dump") else {}
                    )
                )
                role = msg_dict.get("role", "")

                if role == "user":
                    msg_id = msg_dict.get("id", str(uuid.uuid4()))
                    content = msg_dict.get("content", "")
                    msg_metadata = msg_dict.get("metadata", {})

                    is_hidden = isinstance(
                        msg_metadata, dict
                    ) and msg_metadata.get("hidden", False)
                    if is_hidden:
                        logger.info(
                            f"Message {msg_id[:8]} marked as hidden "
                            "(auto-sent initial/workflow prompt)"
                        )
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
                f"Extracting user message from "
                f"{len(input_data.messages)} messages"
            )
            user_message = self._extract_user_message(input_data)
            logger.info(
                f"Extracted user message: "
                f"'{user_message[:100] if user_message else '(empty)'}...'"
            )

            if not user_message:
                logger.warning("No user message found in input")
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=thread_id,
                    run_id=run_id,
                    event={
                        "type": "system_log",
                        "message": "No user message provided",
                    },
                )
                yield RunFinishedEvent(
                    type=EventType.RUN_FINISHED,
                    thread_id=thread_id,
                    run_id=run_id,
                )
                return

            # Run Claude SDK and yield events
            logger.info(
                f"Starting Claude SDK with prompt: '{user_message[:50]}...'"
            )
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
            f"Extracting from {len(messages)} messages, "
            f"types: {[type(m).__name__ for m in messages]}"
        )

        for msg in reversed(messages):
            if hasattr(msg, "role") and msg.role == "user":
                content = getattr(msg, "content", "")
                if isinstance(content, str):
                    return content
                elif isinstance(content, list):
                    for block in content:
                        if hasattr(block, "text"):
                            return block.text
                        elif isinstance(block, dict) and "text" in block:
                            return block["text"]
            elif isinstance(msg, dict):
                if msg.get("role") == "user":
                    content = msg.get("content", "")
                    if isinstance(content, str):
                        return content

        logger.warning("No user message found!")
        return ""

    # ------------------------------------------------------------------
    # SDK orchestration
    # ------------------------------------------------------------------

    async def _run_claude_agent_sdk(
        self, prompt: str, thread_id: str, run_id: str
    ) -> AsyncIterator[BaseEvent]:
        """Execute the Claude Code SDK with the given prompt and yield AG-UI events."""
        current_message_id: Optional[str] = None

        logger.info(
            f"_run_claude_agent_sdk called with prompt length={len(prompt)}, "
            "will create fresh client"
        )
        try:
            # --- Authentication ---
            logger.info("Checking authentication configuration...")
            api_key = self.context.get_env("ANTHROPIC_API_KEY", "")
            use_vertex = (
                self.context.get_env("CLAUDE_CODE_USE_VERTEX", "").strip()
                == "1"
            )

            logger.info(
                f"Auth config: api_key={'set' if api_key else 'not set'}, "
                f"use_vertex={use_vertex}"
            )

            if not api_key and not use_vertex:
                raise RuntimeError(
                    "Either ANTHROPIC_API_KEY or CLAUDE_CODE_USE_VERTEX=1 "
                    "must be set"
                )

            if api_key:
                os.environ["ANTHROPIC_API_KEY"] = api_key
                logger.info("Using Anthropic API key authentication")

            if use_vertex:
                vertex_credentials = await auth.setup_vertex_credentials(
                    self.context
                )
                if "ANTHROPIC_API_KEY" in os.environ:
                    logger.info(
                        "Clearing ANTHROPIC_API_KEY to force Vertex AI mode"
                    )
                    del os.environ["ANTHROPIC_API_KEY"]

                os.environ["CLAUDE_CODE_USE_VERTEX"] = "1"
                os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = (
                    vertex_credentials.get("credentials_path", "")
                )
                os.environ["ANTHROPIC_VERTEX_PROJECT_ID"] = (
                    vertex_credentials.get("project_id", "")
                )
                os.environ["CLOUD_ML_REGION"] = vertex_credentials.get(
                    "region", ""
                )

            # --- SDK imports (after env vars are set) ---
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

            # --- Observability ---
            raw_user_id = os.getenv("USER_ID", "").strip()
            raw_user_name = os.getenv("USER_NAME", "").strip()
            user_id, user_name = auth.sanitize_user_context(
                raw_user_id, raw_user_name
            )

            model = self.context.get_env("LLM_MODEL")
            configured_model = model or "claude-sonnet-4-5@20250929"

            if use_vertex and model:
                configured_model = auth.map_to_vertex_model(model)

            obs = ObservabilityManager(
                session_id=self.context.session_id,
                user_id=user_id,
                user_name=user_name,
            )
            await obs.initialize(
                prompt=prompt,
                namespace=self.context.get_env(
                    "AGENTIC_SESSION_NAMESPACE", "unknown"
                ),
                model=configured_model,
            )
            obs._pending_initial_prompt = prompt

            # --- Workspace paths ---
            is_continuation = (
                self.context.get_env("IS_RESUME", "").strip().lower()
                == "true"
            )
            if is_continuation:
                logger.info("IS_RESUME=true - treating as continuation")

            repos_cfg = runner_config.get_repos_config()
            cwd_path = self.context.workspace_path
            add_dirs = []
            derived_name = None

            active_workflow_url = (
                os.getenv("ACTIVE_WORKFLOW_GIT_URL") or ""
            ).strip()
            if active_workflow_url:
                cwd_path, add_dirs, derived_name = (
                    workspace.setup_workflow_paths(
                        self.context, active_workflow_url, repos_cfg
                    )
                )
            elif repos_cfg:
                cwd_path, add_dirs = workspace.setup_multi_repo_paths(
                    self.context, repos_cfg
                )
            else:
                cwd_path = str(
                    Path(self.context.workspace_path) / "artifacts"
                )

            # --- Config ---
            ambient_config = (
                runner_config.load_ambient_config(cwd_path)
                if active_workflow_url
                else {}
            )

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

            # --- Credentials ---
            await auth.populate_runtime_credentials(self.context)

            # --- MCP servers ---
            mcp_servers = (
                runner_config.load_mcp_config(self.context, cwd_path) or {}
            )

            # Pre-flight check: Validate MCP server authentication
            from main import _check_mcp_authentication

            mcp_auth_warnings = []
            if mcp_servers:
                for server_name in mcp_servers.keys():
                    is_auth, msg = _check_mcp_authentication(server_name)
                    if is_auth is False:
                        mcp_auth_warnings.append(
                            f"⚠️  {server_name}: {msg}"
                        )
                    elif is_auth is None:
                        mcp_auth_warnings.append(
                            f"ℹ️  {server_name}: {msg}"
                        )

            if mcp_auth_warnings:
                warning_msg = (
                    "**MCP Server Authentication Issues:**\n\n"
                    + "\n".join(mcp_auth_warnings)
                    + "\n\nThese servers may not work correctly "
                    "until re-authenticated."
                )
                logger.warning(warning_msg)
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=thread_id,
                    run_id=run_id,
                    event={
                        "type": "mcp_authentication_warning",
                        "message": warning_msg,
                        "servers": [
                            s.split(": ")[1] if ": " in s else s
                            for s in mcp_auth_warnings
                        ],
                    },
                )

            # --- MCP tools ---
            # Session control tool
            restart_tool = create_restart_session_tool(self, sdk_tool)
            session_tools_server = create_sdk_mcp_server(
                name="session", version="1.0.0", tools=[restart_tool]
            )
            mcp_servers["session"] = session_tools_server
            logger.info(
                "Added custom session control MCP tools (restart_session)"
            )

            # Dynamic rubric evaluation tool
            rubric_content, rubric_config = load_rubric_content(cwd_path)
            if rubric_content or rubric_config:
                rubric_tool = create_rubric_mcp_tool(
                    rubric_content=rubric_content or "",
                    rubric_config=rubric_config,
                    obs=obs,
                    session_id=self.context.session_id,
                    sdk_tool_decorator=sdk_tool,
                )
                if rubric_tool:
                    rubric_server = create_sdk_mcp_server(
                        name="rubric",
                        version="1.0.0",
                        tools=[rubric_tool],
                    )
                    mcp_servers["rubric"] = rubric_server
                    logger.info(
                        "Added dynamic rubric evaluation MCP tool "
                        f"(categories: "
                        f"{list(rubric_config.get('schema', {}).keys())})"
                    )

            # Tool permissions
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
                    f"MCP tool permissions granted for servers: "
                    f"{list(mcp_servers.keys())}"
                )

            # --- System prompt ---
            workspace_prompt = prompts.build_workspace_context_prompt(
                repos_cfg=repos_cfg,
                workflow_name=(
                    derived_name if active_workflow_url else None
                ),
                artifacts_path="artifacts",
                ambient_config=ambient_config,
                workspace_path=self.context.workspace_path,
            )
            system_prompt_config = {
                "type": "preset",
                "preset": "claude_code",
                "append": workspace_prompt,
            }

            # Capture stderr from the SDK
            def sdk_stderr_handler(line: str):
                logger.warning(f"[SDK stderr] {line.rstrip()}")

            # --- SDK options ---
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

            # --- Client creation ---
            result_payload = None
            current_message = None
            sdk_session_id = None

            def create_sdk_client(opts, disable_continue=False):
                if disable_continue and hasattr(opts, "continue_conversation"):
                    opts.continue_conversation = False
                return ClaudeSDKClient(options=opts)

            logger.info("Creating new ClaudeSDKClient for this run...")

            if not self._first_run or is_continuation:
                try:
                    options.continue_conversation = True
                    logger.info(
                        "Enabled continue_conversation "
                        "(will resume from disk state)"
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
                    logger.warning(
                        f"Failed to set continue_conversation: {e}"
                    )

            try:
                logger.info("Creating ClaudeSDKClient...")
                client = create_sdk_client(options)
                logger.info(
                    "Connecting ClaudeSDKClient (initializing subprocess)..."
                )
                await client.connect()
                logger.info("ClaudeSDKClient connected successfully!")
            except Exception as resume_error:
                error_str = str(resume_error).lower()
                if (
                    "no conversation found" in error_str
                    or "session" in error_str
                ):
                    logger.warning(
                        f"Conversation continuation failed: {resume_error}"
                    )
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

                logger.info(
                    f"Sending query to Claude SDK: '{prompt[:100]}...'"
                )
                await client.query(prompt)
                logger.info("Query sent, waiting for response stream...")

                # --- Process response stream ---
                logger.info(
                    "Starting to consume receive_response() iterator..."
                )
                message_count = 0

                async for message in client.receive_response():
                    message_count += 1
                    logger.info(
                        f"[ClaudeSDKClient Message #{message_count}]: "
                        f"{message}"
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

                    # Capture SDK session ID
                    if isinstance(message, SystemMessage):
                        if message.subtype == "init" and message.data.get(
                            "session_id"
                        ):
                            sdk_session_id = message.data.get("session_id")
                            logger.info(
                                f"Captured SDK session ID: {sdk_session_id}"
                            )

                    if isinstance(message, (AssistantMessage, UserMessage)):
                        if isinstance(message, AssistantMessage):
                            current_message = message
                            obs.start_turn(
                                configured_model, user_input=prompt
                            )

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
                        for block in (
                            getattr(message, "content", []) or []
                        ):
                            if isinstance(block, TextBlock):
                                text_piece = getattr(block, "text", None)
                                if text_piece:
                                    logger.info(
                                        f"TextBlock received (complete), "
                                        f"text length={len(text_piece)}"
                                    )

                            elif isinstance(block, ToolUseBlock):
                                tool_name = (
                                    getattr(block, "name", "") or "unknown"
                                )
                                tool_input = (
                                    getattr(block, "input", {}) or {}
                                )
                                tool_id = getattr(
                                    block, "id", None
                                ) or str(uuid.uuid4())
                                parent_tool_use_id = getattr(
                                    message, "parent_tool_use_id", None
                                )

                                logger.info(
                                    f"ToolUseBlock detected: {tool_name} "
                                    f"(id={tool_id[:12]})"
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

                                obs.track_tool_use(
                                    tool_name, tool_id, tool_input
                                )

                            elif isinstance(block, ToolResultBlock):
                                tool_use_id = getattr(
                                    block, "tool_use_id", None
                                )
                                content = getattr(block, "content", None)
                                is_error = getattr(block, "is_error", None)
                                result_text = getattr(block, "text", None)
                                result_content = (
                                    content
                                    if content is not None
                                    else result_text
                                )

                                if result_content is not None:
                                    try:
                                        result_str = _json.dumps(
                                            result_content
                                        )
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
                                        result=(
                                            result_str
                                            if not is_error
                                            else None
                                        ),
                                        error=(
                                            result_str
                                            if is_error
                                            else None
                                        ),
                                    )

                                obs.track_tool_result(
                                    tool_use_id,
                                    result_content,
                                    is_error or False,
                                )

                            elif isinstance(block, ThinkingBlock):
                                thinking_text = getattr(
                                    block, "thinking", ""
                                )
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
                        if (
                            getattr(message, "content", [])
                            and current_message_id
                        ):
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
                            f"ResultMessage: num_turns={sdk_num_turns}, "
                            f"usage={usage_raw}"
                        )

                        # Convert usage object to dict if needed
                        if usage_raw is not None and not isinstance(
                            usage_raw, dict
                        ):
                            try:
                                if hasattr(usage_raw, "__dict__"):
                                    usage_raw = usage_raw.__dict__
                                elif hasattr(usage_raw, "model_dump"):
                                    usage_raw = usage_raw.model_dump()
                            except Exception as e:
                                logger.warning(
                                    "Could not convert usage object "
                                    f"to dict: {e}"
                                )

                        if (
                            sdk_num_turns is not None
                            and sdk_num_turns > self._turn_count
                        ):
                            self._turn_count = sdk_num_turns

                        if current_message:
                            obs.end_turn(
                                self._turn_count,
                                current_message,
                                (
                                    usage_raw
                                    if isinstance(usage_raw, dict)
                                    else None
                                ),
                            )
                            current_message = None

                        result_payload = {
                            "subtype": getattr(message, "subtype", None),
                            "duration_ms": getattr(
                                message, "duration_ms", None
                            ),
                            "is_error": getattr(message, "is_error", None),
                            "num_turns": getattr(message, "num_turns", None),
                            "total_cost_usd": getattr(
                                message, "total_cost_usd", None
                            ),
                            "usage": usage_raw,
                            "result": getattr(message, "result", None),
                        }

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
                    f"Response iterator fully consumed "
                    f"({message_count} messages total)"
                )

                self._first_run = False

                # Check if restart was requested
                if self._restart_requested:
                    logger.info(
                        "🔄 Restart was requested, emitting restart event"
                    )
                    self._restart_requested = False
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=thread_id,
                        run_id=run_id,
                        event={
                            "type": "session_restart_requested",
                            "message": "Claude requested a session restart. "
                            "Reconnecting...",
                        },
                    )

            finally:
                self._active_client = None
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
        """Interrupt the active Claude SDK execution."""
        if self._active_client is None:
            logger.warning("Interrupt requested but no active client")
            return

        try:
            logger.info("Sending interrupt signal to Claude SDK client...")
            await self._active_client.interrupt()
            logger.info("Interrupt signal sent successfully")
        except Exception as e:
            logger.error(f"Failed to interrupt Claude SDK: {e}")
