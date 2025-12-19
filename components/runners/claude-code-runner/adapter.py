#!/usr/bin/env python3
"""
Claude Code Adapter for AG-UI Server.

Refactored from wrapper.py to use async generators that yield AG-UI events
instead of WebSocket messaging. This is the core adapter that wraps the
Claude Code SDK and produces a stream of AG-UI protocol events.
"""

import asyncio
import os
import sys
import logging
import json as _json
import re
import shutil
import uuid
from pathlib import Path
from typing import AsyncIterator, Optional, Any
from urllib.parse import urlparse, urlunparse
from urllib import request as _urllib_request, error as _urllib_error
from datetime import datetime, timezone

# Set umask to make files readable by content service container
os.umask(0o022)

# AG-UI Protocol Events
from ag_ui.core import (
    EventType,
    RunAgentInput,
    BaseEvent,
    RunStartedEvent,
    RunFinishedEvent,
    RunErrorEvent,
    TextMessageStartEvent,
    TextMessageContentEvent,
    TextMessageEndEvent,
    ToolCallStartEvent,
    ToolCallArgsEvent,
    ToolCallEndEvent,
    StepStartedEvent,
    StepFinishedEvent,
    StateSnapshotEvent,
    StateDeltaEvent,
    RawEvent,
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

        # AG-UI streaming state
        self._current_message_id: Optional[str] = None
        self._current_tool_id: Optional[str] = None
        self._current_run_id: Optional[str] = None
        self._current_thread_id: Optional[str] = None
        
        # Active client reference for interrupt support
        self._active_client: Optional[Any] = None

    async def initialize(self, context: RunnerContext):
        """Initialize the adapter with context."""
        self.context = context
        logger.info(f"Initialized Claude Code adapter for session {context.session_id}")

        # Set default USER_GOOGLE_EMAIL if not provided
        if not os.getenv("USER_GOOGLE_EMAIL"):
            os.environ["USER_GOOGLE_EMAIL"] = "user@example.com"
            logger.info("USER_GOOGLE_EMAIL not set, using default: user@example.com")
        else:
            logger.info(f"USER_GOOGLE_EMAIL set to: {os.getenv('USER_GOOGLE_EMAIL')}")

        # Copy Google OAuth credentials from mounted Secret to writable workspace location
        await self._setup_google_credentials()
        
        # Prepare workspace from input repo if provided
        async for event in self._prepare_workspace():
            yield event
            
        # Initialize workflow if ACTIVE_WORKFLOW env vars are set
        async for event in self._initialize_workflow_if_set():
            yield event
            
        # Validate prerequisite files exist for phase-based commands
        try:
            await self._validate_prerequisites()
        except PrerequisiteError as exc:
            self.last_exit_code = 2
            logger.error("Prerequisite validation failed during initialization: %s", exc)
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
                msg_dict = msg if isinstance(msg, dict) else (msg.model_dump() if hasattr(msg, 'model_dump') else {})
                role = msg_dict.get('role', '')
                
                if role == 'user':
                    msg_id = msg_dict.get('id', str(uuid.uuid4()))
                    content = msg_dict.get('content', '')
                    msg_metadata = msg_dict.get('metadata', {})
                    
                    # Check if message should be hidden from UI
                    is_hidden = isinstance(msg_metadata, dict) and msg_metadata.get('hidden', False)
                    if is_hidden:
                        logger.info(f"Message {msg_id[:8]} marked as hidden (auto-sent initial/workflow prompt)")
                    
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
                            }
                        )
                    
                    yield TextMessageStartEvent(
                        type=EventType.TEXT_MESSAGE_START,
                        thread_id=thread_id,
                        run_id=run_id,
                        message_id=msg_id,
                        role='user',
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
            logger.info(f"Extracting user message from {len(input_data.messages)} messages")
            user_message = self._extract_user_message(input_data)
            logger.info(f"Extracted user message: '{user_message[:100] if user_message else '(empty)'}...'")
            
            if not user_message:
                logger.warning("No user message found in input")
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=thread_id,
                    run_id=run_id,
                    event={"type": "system_log", "message": "No user message provided"}
                )
                yield RunFinishedEvent(
                    type=EventType.RUN_FINISHED,
                    thread_id=thread_id,
                    run_id=run_id,
                )
                return
            
            # Run Claude SDK and yield events
            logger.info(f"Starting Claude SDK with prompt: '{user_message[:50]}...'")
            async for event in self._run_claude_agent_sdk(user_message, thread_id, run_id):
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
        logger.info(f"Extracting from {len(messages)} messages, types: {[type(m).__name__ for m in messages]}")
        
        # Find the last user message
        for msg in reversed(messages):
            logger.debug(f"Checking message: type={type(msg).__name__}, hasattr(role)={hasattr(msg, 'role')}")
            
            if hasattr(msg, 'role') and msg.role == 'user':
                # Handle different content formats
                content = getattr(msg, 'content', '')
                if isinstance(content, str):
                    logger.info(f"Found user message (object format): '{content[:50]}...'")
                    return content
                elif isinstance(content, list):
                    # Content blocks format
                    for block in content:
                        if hasattr(block, 'text'):
                            return block.text
                        elif isinstance(block, dict) and 'text' in block:
                            return block['text']
            elif isinstance(msg, dict):
                logger.debug(f"Dict message: role={msg.get('role')}, content={msg.get('content', '')[:30]}...")
                if msg.get('role') == 'user':
                    content = msg.get('content', '')
                    if isinstance(content, str):
                        logger.info(f"Found user message (dict format): '{content[:50]}...'")
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
        logger.info(f"_run_claude_agent_sdk called with prompt length={len(prompt)}, will create fresh client")
        try:
            # Check for authentication method
            logger.info("Checking authentication configuration...")
            api_key = self.context.get_env('ANTHROPIC_API_KEY', '')
            use_vertex = self.context.get_env('CLAUDE_CODE_USE_VERTEX', '').strip() == '1'
            
            logger.info(f"Auth config: api_key={'set' if api_key else 'not set'}, use_vertex={use_vertex}")

            if not api_key and not use_vertex:
                raise RuntimeError("Either ANTHROPIC_API_KEY or CLAUDE_CODE_USE_VERTEX=1 must be set")

            # Set environment variables BEFORE importing SDK
            if api_key:
                os.environ['ANTHROPIC_API_KEY'] = api_key
                logger.info("Using Anthropic API key authentication")

            # Configure Vertex AI if requested
            if use_vertex:
                vertex_credentials = await self._setup_vertex_credentials()
                if 'ANTHROPIC_API_KEY' in os.environ:
                    logger.info("Clearing ANTHROPIC_API_KEY to force Vertex AI mode")
                    del os.environ['ANTHROPIC_API_KEY']

                os.environ['CLAUDE_CODE_USE_VERTEX'] = '1'
                os.environ['GOOGLE_APPLICATION_CREDENTIALS'] = vertex_credentials.get('credentials_path', '')
                os.environ['ANTHROPIC_VERTEX_PROJECT_ID'] = vertex_credentials.get('project_id', '')
                os.environ['CLOUD_ML_REGION'] = vertex_credentials.get('region', '')

            # NOW we can safely import the SDK
            from claude_agent_sdk import ClaudeSDKClient, ClaudeAgentOptions
            from claude_agent_sdk import (
                AssistantMessage,
                UserMessage,
                SystemMessage,
                ResultMessage,
                TextBlock,
                ThinkingBlock,
                ToolUseBlock,
                ToolResultBlock,
            )
            from claude_agent_sdk.types import StreamEvent

            from observability import ObservabilityManager

            # Extract and sanitize user context for observability
            raw_user_id = os.getenv('USER_ID', '').strip()
            raw_user_name = os.getenv('USER_NAME', '').strip()
            user_id, user_name = self._sanitize_user_context(raw_user_id, raw_user_name)

            # Get model configuration
            model = self.context.get_env('LLM_MODEL')
            configured_model = model or 'claude-sonnet-4-5@20250929'

            if use_vertex and model:
                configured_model = self._map_to_vertex_model(model)

            # Initialize observability
            obs = ObservabilityManager(
                session_id=self.context.session_id,
                user_id=user_id,
                user_name=user_name
            )
            await obs.initialize(
                prompt=prompt,
                namespace=self.context.get_env('AGENTIC_SESSION_NAMESPACE', 'unknown'),
                model=configured_model
            )
            obs._pending_initial_prompt = prompt

            # Check if continuing from previous session
            parent_session_id = self.context.get_env('PARENT_SESSION_ID', '').strip()
            is_continuation = bool(parent_session_id)

            # Determine cwd and additional dirs
            repos_cfg = self._get_repos_config()
            cwd_path = self.context.workspace_path
            add_dirs = []
            derived_name = None

            # Check for active workflow first
            active_workflow_url = (os.getenv('ACTIVE_WORKFLOW_GIT_URL') or '').strip()
            if active_workflow_url:
                cwd_path, add_dirs, derived_name = self._setup_workflow_paths(
                    active_workflow_url, repos_cfg
                )
            elif repos_cfg:
                cwd_path, add_dirs = self._setup_multi_repo_paths(repos_cfg)
            else:
                cwd_path = str(Path(self.context.workspace_path) / "artifacts")

            # Load ambient.json configuration
            ambient_config = self._load_ambient_config(cwd_path) if active_workflow_url else {}

            # Ensure working directory exists
            cwd_path_obj = Path(cwd_path)
            if not cwd_path_obj.exists():
                logger.warning(f"Working directory does not exist, creating: {cwd_path}")
                try:
                    cwd_path_obj.mkdir(parents=True, exist_ok=True)
                except Exception as e:
                    logger.error(f"Failed to create working directory: {e}")
                    cwd_path = self.context.workspace_path

            logger.info(f"Claude SDK CWD: {cwd_path}")
            logger.info(f"Claude SDK additional directories: {add_dirs}")

            # Load MCP server configuration
            mcp_servers = self._load_mcp_config(cwd_path)
            allowed_tools = ["Read", "Write", "Bash", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch", "WebFetch"]
            if mcp_servers:
                for server_name in mcp_servers.keys():
                    allowed_tools.append(f"mcp__{server_name}")
                logger.info(f"MCP tool permissions granted for servers: {list(mcp_servers.keys())}")

            # Build workspace context system prompt
            workspace_prompt = self._build_workspace_context_prompt(
                repos_cfg=repos_cfg,
                workflow_name=derived_name if active_workflow_url else None,
                artifacts_path="artifacts",
                ambient_config=ambient_config
            )
            system_prompt_config = {"type": "text", "text": workspace_prompt}

            # Configure SDK options
            options = ClaudeAgentOptions(
                cwd=cwd_path,
                permission_mode="acceptEdits",
                allowed_tools=allowed_tools,
                mcp_servers=mcp_servers,
                setting_sources=["project"],
                system_prompt=system_prompt_config,
                include_partial_messages=True,
            )

            # Enable continue_conversation for session resumption
            if not self._first_run or is_continuation:
                try:
                    options.continue_conversation = True
                    logger.info("Enabled continue_conversation for session resumption")
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=thread_id,
                        run_id=run_id,
                        event={"type": "system_log", "message": "🔄 Continuing conversation from previous state"}
                    )
                except Exception as e:
                    logger.warning(f"Failed to set continue_conversation: {e}")

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

            max_tokens_env = self.context.get_env('LLM_MAX_TOKENS') or self.context.get_env('MAX_TOKENS')
            if max_tokens_env:
                try:
                    options.max_tokens = int(max_tokens_env)
                except Exception:
                    pass

            temperature_env = self.context.get_env('LLM_TEMPERATURE') or self.context.get_env('TEMPERATURE')
            if temperature_env:
                try:
                    options.temperature = float(temperature_env)
                except Exception:
                    pass

            result_payload = None
            current_message = None
            sdk_session_id = None

            def create_sdk_client(opts, disable_continue=False):
                if disable_continue and hasattr(opts, 'continue_conversation'):
                    opts.continue_conversation = False
                return ClaudeSDKClient(options=opts)

            # Always create a fresh client for each run (simple and reliable)
            logger.info("Creating new ClaudeSDKClient for this run...")
            
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
                        event={"type": "system_log", "message": "⚠️ Could not continue conversation, starting fresh..."}
                    )
                    client = create_sdk_client(options, disable_continue=True)
                    await client.connect()
                else:
                    raise

            try:
                # Store client reference for interrupt support
                self._active_client = client
                
                if not self._first_run:
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=thread_id,
                        run_id=run_id,
                        event={"type": "system_log", "message": "✅ Continuing conversation"}
                    )
                    logger.info("SDK continuing conversation from local state")

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
                async for message in client.receive_response():
                    logger.info(f"[ClaudeSDKClient]: {message}")

                    # Handle StreamEvent for real-time streaming chunks
                    if isinstance(message, StreamEvent):
                        event_data = message.event
                        event_type = event_data.get('type')

                        if event_type == 'message_start':
                            self._current_message_id = str(uuid.uuid4())
                            yield TextMessageStartEvent(
                                type=EventType.TEXT_MESSAGE_START,
                                thread_id=thread_id,
                                run_id=run_id,
                                message_id=self._current_message_id,
                                role="assistant",
                            )

                        elif event_type == 'content_block_delta':
                            delta_data = event_data.get('delta', {})
                            if delta_data.get('type') == 'text_delta':
                                text_chunk = delta_data.get('text', '')
                                if text_chunk:
                                    yield TextMessageContentEvent(
                                        type=EventType.TEXT_MESSAGE_CONTENT,
                                        thread_id=thread_id,
                                        run_id=run_id,
                                        message_id=self._current_message_id,
                                        delta=text_chunk,
                                    )
                        continue

                    # Capture SDK session ID from init message
                    if isinstance(message, SystemMessage):
                        if message.subtype == 'init' and message.data.get('session_id'):
                            sdk_session_id = message.data.get('session_id')
                            logger.info(f"Captured SDK session ID: {sdk_session_id}")

                    if isinstance(message, (AssistantMessage, UserMessage)):
                        if isinstance(message, AssistantMessage):
                            current_message = message
                            obs.start_turn(configured_model, user_input=prompt)

                        # Process all blocks in the message
                        for block in getattr(message, 'content', []) or []:
                            if isinstance(block, TextBlock):
                                text_piece = getattr(block, 'text', None)
                                if text_piece:
                                    logger.info(f"TextBlock received (complete), text length={len(text_piece)}")

                            elif isinstance(block, ToolUseBlock):
                                tool_name = getattr(block, 'name', '') or 'unknown'
                                tool_input = getattr(block, 'input', {}) or {}
                                tool_id = getattr(block, 'id', None) or str(uuid.uuid4())
                                parent_tool_use_id = getattr(message, 'parent_tool_use_id', None)

                                logger.info(f"ToolUseBlock detected: {tool_name} (id={tool_id[:12]})")

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
                                tool_use_id = getattr(block, 'tool_use_id', None)
                                content = getattr(block, 'content', None)
                                is_error = getattr(block, 'is_error', None)
                                result_text = getattr(block, 'text', None)
                                result_content = content if content is not None else result_text

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

                                obs.track_tool_result(tool_use_id, result_content, is_error or False)

                            elif isinstance(block, ThinkingBlock):
                                thinking_text = getattr(block, 'thinking', '')
                                signature = getattr(block, 'signature', '')
                                yield RawEvent(
                                    type=EventType.RAW,
                                    thread_id=thread_id,
                                    run_id=run_id,
                                    event={
                                        "type": "thinking_block",
                                        "thinking": thinking_text,
                                        "signature": signature,
                                    }
                                )

                        # End text message after processing all blocks
                        if getattr(message, 'content', []) and self._current_message_id:
                            yield TextMessageEndEvent(
                                type=EventType.TEXT_MESSAGE_END,
                                thread_id=thread_id,
                                run_id=run_id,
                                message_id=self._current_message_id,
                            )
                            self._current_message_id = None

                    elif isinstance(message, SystemMessage):
                        text = getattr(message, 'text', None)
                        if text:
                            yield RawEvent(
                                type=EventType.RAW,
                                thread_id=thread_id,
                                run_id=run_id,
                                event={"type": "system_log", "level": "debug", "message": str(text)}
                            )

                    elif isinstance(message, ResultMessage):
                        usage_raw = getattr(message, 'usage', None)
                        sdk_num_turns = getattr(message, 'num_turns', None)

                        logger.info(f"ResultMessage: num_turns={sdk_num_turns}, usage={usage_raw}")

                        # Convert usage object to dict if needed
                        if usage_raw is not None and not isinstance(usage_raw, dict):
                            try:
                                if hasattr(usage_raw, '__dict__'):
                                    usage_raw = usage_raw.__dict__
                                elif hasattr(usage_raw, 'model_dump'):
                                    usage_raw = usage_raw.model_dump()
                            except Exception as e:
                                logger.warning(f"Could not convert usage object to dict: {e}")

                        # Update turn count
                        if sdk_num_turns is not None and sdk_num_turns > self._turn_count:
                            self._turn_count = sdk_num_turns

                        # Complete turn tracking
                        if current_message:
                            obs.end_turn(self._turn_count, current_message, usage_raw if isinstance(usage_raw, dict) else None)
                            current_message = None

                        result_payload = {
                            "subtype": getattr(message, 'subtype', None),
                            "duration_ms": getattr(message, 'duration_ms', None),
                            "is_error": getattr(message, 'is_error', None),
                            "num_turns": getattr(message, 'num_turns', None),
                            "total_cost_usd": getattr(message, 'total_cost_usd', None),
                            "usage": usage_raw,
                            "result": getattr(message, 'result', None),
                        }

                        # Emit state delta with result
                        yield StateDeltaEvent(
                            type=EventType.STATE_DELTA,
                            thread_id=thread_id,
                            run_id=run_id,
                            delta=[{"op": "replace", "path": "/lastResult", "value": result_payload}],
                        )

                # End step
                yield StepFinishedEvent(
                    type=EventType.STEP_FINISHED,
                    thread_id=thread_id,
                    run_id=run_id,
                    step_id=step_id,
                    step_name="processing_prompt",
                )

                # Mark first run complete
                self._first_run = False

            finally:
                # Clear active client reference (interrupt no longer valid for this run)
                self._active_client = None
                
                # Always disconnect client at end of run (no persistence)
                if client is not None:
                    logger.info("Disconnecting client (end of run)")
                    await client.disconnect()
            
            # Finalize observability
            await obs.finalize()

        except Exception as e:
            logger.error(f"Failed to run Claude Code SDK: {e}")
            if 'obs' in locals():
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


    def _setup_workflow_paths(self, active_workflow_url: str, repos_cfg: list) -> tuple[str, list, str]:
        """Setup paths for workflow mode."""
        add_dirs = []
        derived_name = None
        cwd_path = self.context.workspace_path

        try:
            owner, repo, _ = self._parse_owner_repo(active_workflow_url)
            derived_name = repo or ''
            if not derived_name:
                p = urlparse(active_workflow_url)
                parts = [pt for pt in (p.path or '').split('/') if pt]
                if parts:
                    derived_name = parts[-1]
            derived_name = (derived_name or '').removesuffix('.git').strip()

            if derived_name:
                workflow_path = str(Path(self.context.workspace_path) / "workflows" / derived_name)
                if Path(workflow_path).exists():
                    cwd_path = workflow_path
                    logger.info(f"Using workflow as CWD: {derived_name}")
                else:
                    logger.warning(f"Workflow directory not found: {workflow_path}, using default")
                    cwd_path = str(Path(self.context.workspace_path) / "workflows" / "default")
            else:
                cwd_path = str(Path(self.context.workspace_path) / "workflows" / "default")
        except Exception as e:
            logger.warning(f"Failed to derive workflow name: {e}, using default")
            cwd_path = str(Path(self.context.workspace_path) / "workflows" / "default")

        # Add all repos as additional directories
        for r in repos_cfg:
            name = (r.get('name') or '').strip()
            if name:
                repo_path = str(Path(self.context.workspace_path) / name)
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
        """Setup paths for multi-repo mode."""
        add_dirs = []
        
        main_name = (os.getenv('MAIN_REPO_NAME') or '').strip()
        if not main_name:
            idx_raw = (os.getenv('MAIN_REPO_INDEX') or '').strip()
            try:
                idx_val = int(idx_raw) if idx_raw else 0
            except Exception:
                idx_val = 0
            if idx_val < 0 or idx_val >= len(repos_cfg):
                idx_val = 0
            main_name = (repos_cfg[idx_val].get('name') or '').strip()

        cwd_path = str(Path(self.context.workspace_path) / main_name) if main_name else self.context.workspace_path

        for r in repos_cfg:
            name = (r.get('name') or '').strip()
            if not name:
                continue
            p = str(Path(self.context.workspace_path) / name)
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
            sanitized_id = re.sub(r'[^a-zA-Z0-9@._-]', '', user_id)
            user_id = sanitized_id

        if user_name:
            user_name = str(user_name).strip()
            if len(user_name) > 255:
                user_name = user_name[:255]
            sanitized_name = re.sub(r'[\x00-\x1f\x7f-\x9f]', '', user_name)
            user_name = sanitized_name

        return user_id, user_name

    def _map_to_vertex_model(self, model: str) -> str:
        """Map Anthropic API model names to Vertex AI model names."""
        model_map = {
            'claude-opus-4-5': 'claude-opus-4-5@20251101',
            'claude-opus-4-1': 'claude-opus-4-1@20250805',
            'claude-sonnet-4-5': 'claude-sonnet-4-5@20250929',
            'claude-haiku-4-5': 'claude-haiku-4-5@20251001',
        }
        return model_map.get(model, model)

    async def _setup_vertex_credentials(self) -> dict:
        """Set up Google Cloud Vertex AI credentials from service account."""
        service_account_path = self.context.get_env('GOOGLE_APPLICATION_CREDENTIALS', '').strip()
        project_id = self.context.get_env('ANTHROPIC_VERTEX_PROJECT_ID', '').strip()
        region = self.context.get_env('CLOUD_ML_REGION', '').strip()

        if not service_account_path:
            raise RuntimeError("GOOGLE_APPLICATION_CREDENTIALS must be set when CLAUDE_CODE_USE_VERTEX=1")
        if not project_id:
            raise RuntimeError("ANTHROPIC_VERTEX_PROJECT_ID must be set when CLAUDE_CODE_USE_VERTEX=1")
        if not region:
            raise RuntimeError("CLOUD_ML_REGION must be set when CLAUDE_CODE_USE_VERTEX=1")

        if not Path(service_account_path).exists():
            raise RuntimeError(f"Service account key file not found at {service_account_path}")

        logger.info(f"Vertex AI configured: project={project_id}, region={region}")
        return {
            'credentials_path': service_account_path,
            'project_id': project_id,
            'region': region,
        }

    async def _prepare_workspace(self) -> AsyncIterator[BaseEvent]:
        """Clone input repo/branch into workspace and configure git remotes."""
        workspace = Path(self.context.workspace_path)
        workspace.mkdir(parents=True, exist_ok=True)

        parent_session_id = self.context.get_env('PARENT_SESSION_ID', '').strip()
        reusing_workspace = bool(parent_session_id)

        logger.info(f"Workspace preparation: parent_session_id={parent_session_id[:8] if parent_session_id else 'None'}, reusing={reusing_workspace}")

        repos_cfg = self._get_repos_config()
        if repos_cfg:
            async for event in self._prepare_multi_repo_workspace(workspace, repos_cfg, reusing_workspace):
                yield event
            return

        # Single-repo legacy flow
        input_repo = os.getenv("INPUT_REPO_URL", "").strip()
        if not input_repo:
            logger.info("No INPUT_REPO_URL configured, skipping single-repo setup")
            return

        input_branch = os.getenv("INPUT_BRANCH", "").strip() or "main"
        output_repo = os.getenv("OUTPUT_REPO_URL", "").strip()

        token = await self._fetch_token_for_url(input_repo)
        workspace_has_git = (workspace / ".git").exists()

        try:
            if not workspace_has_git:
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=self._current_thread_id or self.context.session_id,
                    run_id=self._current_run_id or "init",
                    event={"type": "system_log", "message": "📥 Cloning input repository..."}
                )
                clone_url = self._url_with_token(input_repo, token) if token else input_repo
                await self._run_cmd(["git", "clone", "--branch", input_branch, "--single-branch", clone_url, str(workspace)], cwd=str(workspace.parent))
                await self._run_cmd(["git", "remote", "set-url", "origin", clone_url], cwd=str(workspace), ignore_errors=True)
            elif reusing_workspace:
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=self._current_thread_id or self.context.session_id,
                    run_id=self._current_run_id or "init",
                    event={"type": "system_log", "message": "✓ Preserving workspace (continuation)"}
                )
                await self._run_cmd(["git", "remote", "set-url", "origin", self._url_with_token(input_repo, token) if token else input_repo], cwd=str(workspace), ignore_errors=True)
            else:
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=self._current_thread_id or self.context.session_id,
                    run_id=self._current_run_id or "init",
                    event={"type": "system_log", "message": "🔄 Resetting workspace to clean state"}
                )
                await self._run_cmd(["git", "remote", "set-url", "origin", self._url_with_token(input_repo, token) if token else input_repo], cwd=str(workspace))
                await self._run_cmd(["git", "fetch", "origin", input_branch], cwd=str(workspace))
                await self._run_cmd(["git", "checkout", input_branch], cwd=str(workspace))
                await self._run_cmd(["git", "reset", "--hard", f"origin/{input_branch}"], cwd=str(workspace))

            # Git identity
            user_name = os.getenv("GIT_USER_NAME", "").strip() or "Ambient Code Bot"
            user_email = os.getenv("GIT_USER_EMAIL", "").strip() or "bot@ambient-code.local"
            await self._run_cmd(["git", "config", "user.name", user_name], cwd=str(workspace))
            await self._run_cmd(["git", "config", "user.email", user_email], cwd=str(workspace))

            if output_repo:
                out_url = self._url_with_token(output_repo, token) if token else output_repo
                await self._run_cmd(["git", "remote", "remove", "output"], cwd=str(workspace), ignore_errors=True)
                await self._run_cmd(["git", "remote", "add", "output", out_url], cwd=str(workspace))

        except Exception as e:
            logger.error(f"Failed to prepare workspace: {e}")
            yield RawEvent(
                type=EventType.RAW,
                thread_id=self._current_thread_id or self.context.session_id,
                run_id=self._current_run_id or "init",
                event={"type": "system_log", "message": f"Workspace preparation failed: {e}"}
            )

        # Create artifacts directory
        try:
            artifacts_dir = workspace / "artifacts"
            artifacts_dir.mkdir(parents=True, exist_ok=True)
        except Exception as e:
            logger.warning(f"Failed to create artifacts directory: {e}")

    async def _prepare_multi_repo_workspace(
        self, workspace: Path, repos_cfg: list, reusing_workspace: bool
    ) -> AsyncIterator[BaseEvent]:
        """Prepare workspace for multi-repo mode."""
        try:
            for r in repos_cfg:
                name = (r.get('name') or '').strip()
                inp = r.get('input') or {}
                url = (inp.get('url') or '').strip()
                branch = (inp.get('branch') or '').strip() or 'main'
                if not name or not url:
                    continue

                repo_dir = workspace / name
                token = await self._fetch_token_for_url(url)
                repo_exists = repo_dir.exists() and (repo_dir / ".git").exists()

                if not repo_exists:
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=self._current_thread_id or self.context.session_id,
                        run_id=self._current_run_id or "init",
                        event={"type": "system_log", "message": f"📥 Cloning {name}..."}
                    )
                    clone_url = self._url_with_token(url, token) if token else url
                    await self._run_cmd(["git", "clone", "--branch", branch, "--single-branch", clone_url, str(repo_dir)], cwd=str(workspace))
                    await self._run_cmd(["git", "remote", "set-url", "origin", clone_url], cwd=str(repo_dir), ignore_errors=True)
                elif reusing_workspace:
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=self._current_thread_id or self.context.session_id,
                        run_id=self._current_run_id or "init",
                        event={"type": "system_log", "message": f"✓ Preserving {name} (continuation)"}
                    )
                    await self._run_cmd(["git", "remote", "set-url", "origin", self._url_with_token(url, token) if token else url], cwd=str(repo_dir), ignore_errors=True)
                else:
                    yield RawEvent(
                        type=EventType.RAW,
                        thread_id=self._current_thread_id or self.context.session_id,
                        run_id=self._current_run_id or "init",
                        event={"type": "system_log", "message": f"🔄 Resetting {name} to clean state"}
                    )
                    await self._run_cmd(["git", "remote", "set-url", "origin", self._url_with_token(url, token) if token else url], cwd=str(repo_dir), ignore_errors=True)
                    await self._run_cmd(["git", "fetch", "origin", branch], cwd=str(repo_dir))
                    await self._run_cmd(["git", "checkout", branch], cwd=str(repo_dir))
                    await self._run_cmd(["git", "reset", "--hard", f"origin/{branch}"], cwd=str(repo_dir))

                # Git identity
                user_name = os.getenv("GIT_USER_NAME", "").strip() or "Ambient Code Bot"
                user_email = os.getenv("GIT_USER_EMAIL", "").strip() or "bot@ambient-code.local"
                await self._run_cmd(["git", "config", "user.name", user_name], cwd=str(repo_dir))
                await self._run_cmd(["git", "config", "user.email", user_email], cwd=str(repo_dir))

                # Configure output remote
                out = r.get('output') or {}
                out_url_raw = (out.get('url') or '').strip()
                if out_url_raw:
                    out_url = self._url_with_token(out_url_raw, token) if token else out_url_raw
                    await self._run_cmd(["git", "remote", "remove", "output"], cwd=str(repo_dir), ignore_errors=True)
                    await self._run_cmd(["git", "remote", "add", "output", out_url], cwd=str(repo_dir))

        except Exception as e:
            logger.error(f"Failed to prepare multi-repo workspace: {e}")
            yield RawEvent(
                type=EventType.RAW,
                thread_id=self._current_thread_id or self.context.session_id,
                run_id=self._current_run_id or "init",
                event={"type": "system_log", "message": f"Workspace preparation failed: {e}"}
            )

    async def _validate_prerequisites(self):
        """Validate prerequisite files exist for phase-based slash commands."""
        prompt = self.context.get_env("INITIAL_PROMPT", "")
        if not prompt:
            return

        prompt_lower = prompt.strip().lower()

        prerequisites = {
            "/speckit.plan": ("spec.md", "Specification file (spec.md) not found. Please run /speckit.specify first."),
            "/speckit.tasks": ("plan.md", "Planning file (plan.md) not found. Please run /speckit.plan first."),
            "/speckit.implement": ("tasks.md", "Tasks file (tasks.md) not found. Please run /speckit.tasks first.")
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
        """Initialize workflow on startup if ACTIVE_WORKFLOW env vars are set."""
        active_workflow_url = (os.getenv('ACTIVE_WORKFLOW_GIT_URL') or '').strip()
        if not active_workflow_url:
            return

        active_workflow_branch = (os.getenv('ACTIVE_WORKFLOW_BRANCH') or 'main').strip()
        active_workflow_path = (os.getenv('ACTIVE_WORKFLOW_PATH') or '').strip()

        try:
            owner, repo, _ = self._parse_owner_repo(active_workflow_url)
            derived_name = repo or ''
            if not derived_name:
                p = urlparse(active_workflow_url)
                parts = [pt for pt in (p.path or '').split('/') if pt]
                if parts:
                    derived_name = parts[-1]
            derived_name = (derived_name or '').removesuffix('.git').strip()

            if not derived_name:
                logger.warning("Could not derive workflow name from URL, skipping initialization")
                return

            workflow_dir = Path(self.context.workspace_path) / "workflows" / derived_name

            if workflow_dir.exists():
                logger.info(f"Workflow {derived_name} already exists, skipping initialization")
                return

            logger.info(f"Initializing workflow {derived_name} from CR spec on startup")
            async for event in self._clone_workflow_repository(active_workflow_url, active_workflow_branch, active_workflow_path, derived_name):
                yield event

        except Exception as e:
            logger.error(f"Failed to initialize workflow on startup: {e}")

    async def _clone_workflow_repository(
        self, git_url: str, branch: str, path: str, workflow_name: str
    ) -> AsyncIterator[BaseEvent]:
        """Clone workflow repository."""
        workspace = Path(self.context.workspace_path)
        workflow_dir = workspace / "workflows" / workflow_name
        temp_clone_dir = workspace / "workflows" / f"{workflow_name}-clone-temp"

        if workflow_dir.exists():
            yield RawEvent(
                type=EventType.RAW,
                thread_id=self._current_thread_id or self.context.session_id,
                run_id=self._current_run_id or "init",
                event={"type": "system_log", "message": f"✓ Workflow {workflow_name} already loaded"}
            )
            return

        token = await self._fetch_token_for_url(git_url)

        yield RawEvent(
            type=EventType.RAW,
            thread_id=self._current_thread_id or self.context.session_id,
            run_id=self._current_run_id or "init",
            event={"type": "system_log", "message": f"📥 Cloning workflow {workflow_name}..."}
        )

        clone_url = self._url_with_token(git_url, token) if token else git_url
        await self._run_cmd(["git", "clone", "--branch", branch, "--single-branch", clone_url, str(temp_clone_dir)], cwd=str(workspace))

        if path and path.strip():
            subdir_path = temp_clone_dir / path.strip()
            if subdir_path.exists() and subdir_path.is_dir():
                shutil.copytree(subdir_path, workflow_dir)
                shutil.rmtree(temp_clone_dir)
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=self._current_thread_id or self.context.session_id,
                    run_id=self._current_run_id or "init",
                    event={"type": "system_log", "message": f"✓ Extracted workflow from: {path}"}
                )
            else:
                temp_clone_dir.rename(workflow_dir)
                yield RawEvent(
                    type=EventType.RAW,
                    thread_id=self._current_thread_id or self.context.session_id,
                    run_id=self._current_run_id or "init",
                    event={"type": "system_log", "message": f"⚠️ Path '{path}' not found, using full repository"}
                )
        else:
            temp_clone_dir.rename(workflow_dir)

        yield RawEvent(
            type=EventType.RAW,
            thread_id=self._current_thread_id or self.context.session_id,
            run_id=self._current_run_id or "init",
            event={"type": "system_log", "message": f"✅ Workflow {workflow_name} ready"}
        )

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
            if 'gitlab' in hostname.lower():
                auth = f"oauth2:{token}@"
            else:
                auth = f"x-access-token:{token}@"

            new_netloc = auth + netloc
            return urlunparse((parsed.scheme, new_netloc, parsed.path,
                               parsed.params, parsed.query, parsed.fragment))
        except Exception:
            return url

    def _redact_secrets(self, text: str) -> str:
        """Redact tokens and secrets from text for safe logging."""
        if not text:
            return text

        text = re.sub(r'gh[pousr]_[a-zA-Z0-9]{36,255}', 'gh*_***REDACTED***', text)
        text = re.sub(r'sk-ant-[a-zA-Z0-9\-_]{30,200}', 'sk-ant-***REDACTED***', text)
        text = re.sub(r'pk-lf-[a-zA-Z0-9\-_]{10,100}', 'pk-lf-***REDACTED***', text)
        text = re.sub(r'sk-lf-[a-zA-Z0-9\-_]{10,100}', 'sk-lf-***REDACTED***', text)
        text = re.sub(r'x-access-token:[^@\s]+@', 'x-access-token:***REDACTED***@', text)
        text = re.sub(r'oauth2:[^@\s]+@', 'oauth2:***REDACTED***@', text)
        text = re.sub(r'://[^:@\s]+:[^@\s]+@', '://***REDACTED***@', text)
        text = re.sub(
            r'(ANTHROPIC_API_KEY|LANGFUSE_SECRET_KEY|LANGFUSE_PUBLIC_KEY|BOT_TOKEN|GIT_TOKEN)\s*=\s*[^\s\'"]+',
            r'\1=***REDACTED***',
            text
        )
        return text

    async def _fetch_token_for_url(self, url: str) -> str:
        """Fetch appropriate token based on repository URL."""
        try:
            parsed = urlparse(url)
            hostname = parsed.hostname or ""

            if 'gitlab' in hostname.lower():
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
            logger.warning(f"Failed to parse URL {url}: {e}, falling back to GitHub token")
            return os.getenv("GITHUB_TOKEN") or await self._fetch_github_token()

    async def _fetch_github_token(self) -> str:
        """Fetch GitHub token from backend API or environment."""
        cached = os.getenv("GITHUB_TOKEN", "").strip()
        if cached:
            logger.info("Using GITHUB_TOKEN from environment")
            return cached

        # Build mint URL from environment
        base = os.getenv('BACKEND_API_URL', '').rstrip('/')
        project = os.getenv('PROJECT_NAME', '').strip()
        session_id = self.context.session_id

        if not base or not project or not session_id:
            logger.warning("Cannot fetch GitHub token: missing environment variables")
            return ""

        url = f"{base}/projects/{project}/agentic-sessions/{session_id}/github/token"
        logger.info(f"Fetching GitHub token from: {url}")

        req = _urllib_request.Request(url, data=b"{}", headers={'Content-Type': 'application/json'}, method='POST')
        bot = (os.getenv('BOT_TOKEN') or '').strip()
        if bot:
            req.add_header('Authorization', f'Bearer {bot}')

        loop = asyncio.get_event_loop()

        def _do_req():
            try:
                with _urllib_request.urlopen(req, timeout=10) as resp:
                    return resp.read().decode('utf-8', errors='replace')
            except Exception as e:
                logger.warning(f"GitHub token fetch failed: {e}")
                return ''

        resp_text = await loop.run_in_executor(None, _do_req)
        if not resp_text:
            return ""

        try:
            data = _json.loads(resp_text)
            token = str(data.get('token') or '')
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
        """Read repos mapping from REPOS_JSON env if present."""
        try:
            raw = os.getenv('REPOS_JSON', '').strip()
            if not raw:
                return []
            data = _json.loads(raw)
            if isinstance(data, list):
                out = []
                for it in data:
                    if not isinstance(it, dict):
                        continue
                    name = str(it.get('name') or '').strip()
                    input_obj = it.get('input') or {}
                    output_obj = it.get('output') or None
                    url = str((input_obj or {}).get('url') or '').strip()
                    if not name and url:
                        try:
                            owner, repo, _ = self._parse_owner_repo(url)
                            derived = repo or ''
                            if not derived:
                                p = urlparse(url)
                                parts = [pt for pt in (p.path or '').split('/') if pt]
                                if parts:
                                    derived = parts[-1]
                            name = (derived or '').removesuffix('.git').strip()
                        except Exception:
                            name = ''
                    if name and isinstance(input_obj, dict) and url:
                        out.append({'name': name, 'input': input_obj, 'output': output_obj})
                return out
        except Exception:
            return []
        return []

    def _load_mcp_config(self, cwd_path: str) -> Optional[dict]:
        """Load MCP server configuration from the ambient runner's .mcp.json file."""
        try:
            runner_mcp_file = Path("/app/claude-runner/.mcp.json")

            if runner_mcp_file.exists() and runner_mcp_file.is_file():
                logger.info(f"Loading MCP config from runner directory: {runner_mcp_file}")
                with open(runner_mcp_file, 'r') as f:
                    config = _json.load(f)
                    return config.get('mcpServers', {})
            else:
                logger.info("No .mcp.json file found in runner directory")
                return None

        except _json.JSONDecodeError as e:
            logger.error(f"Failed to parse .mcp.json: {e}")
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

            with open(config_path, 'r') as f:
                config = _json.load(f)
                logger.info(f"Loaded ambient.json: name={config.get('name')}")
                return config

        except _json.JSONDecodeError as e:
            logger.error(f"Failed to parse ambient.json: {e}")
            return {}
        except Exception as e:
            logger.error(f"Error loading ambient.json: {e}")
            return {}

    def _build_workspace_context_prompt(self, repos_cfg, workflow_name, artifacts_path, ambient_config):
        """Generate comprehensive system prompt describing workspace layout."""
        prompt = "You are Claude Code working in a structured development workspace.\n\n"

        if workflow_name:
            prompt += "## Current Workflow\n"
            prompt += f"Working directory: workflows/{workflow_name}/\n"
            prompt += "This directory contains workflow logic and automation scripts.\n\n"

        prompt += "## User-Uploaded Files (IMPORTANT)\n"
        prompt += "Location: file-uploads/\n"
        prompt += "Purpose: User-uploaded context files (screenshots, documents, images, PDFs, specs, designs).\n"
        prompt += "ALWAYS check this directory when starting a new task - it often contains critical context.\n\n"

        file_uploads_path = Path(self.context.workspace_path) / "file-uploads"
        if file_uploads_path.exists() and file_uploads_path.is_dir():
            try:
                files = sorted([f.name for f in file_uploads_path.iterdir() if f.is_file()])
                if files:
                    prompt += f"Currently uploaded files ({len(files)}):\n"
                    for filename in files:
                        prompt += f"  - {filename}\n"
                    prompt += "READ THESE FILES if they're relevant to the user's task!\n"
            except Exception:
                pass

        prompt += "\n## Shared Artifacts Directory\n"
        prompt += f"Location: {artifacts_path}\n"
        prompt += "Purpose: Create all output artifacts (documents, specs, reports) here.\n\n"

        if repos_cfg:
            prompt += "## Available Code Repositories\n"
            for i, repo in enumerate(repos_cfg):
                name = repo.get('name', f'repo-{i}')
                prompt += f"- {name}/\n"
            prompt += "\nThese repositories contain source code you can read or modify.\n\n"

        if ambient_config.get("systemPrompt"):
            prompt += f"## Workflow Instructions\n{ambient_config['systemPrompt']}\n\n"

        prompt += "## Navigation\n"
        prompt += "All directories are accessible via relative or absolute paths.\n"

        return prompt


    async def _setup_google_credentials(self):
        """Copy Google OAuth credentials from mounted Secret to writable workspace location.
        
        The secret is always mounted (as placeholder if user hasn't authenticated).
        This method checks if credentials.json exists and has content.
        Call refresh_google_credentials() periodically to pick up new credentials after OAuth.
        """
        await self._try_copy_google_credentials()

    async def _try_copy_google_credentials(self) -> bool:
        """Attempt to copy Google credentials from mounted secret.
        
        Returns:
            True if credentials were successfully copied, False otherwise.
        """
        secret_path = Path("/app/.google_workspace_mcp/credentials/credentials.json")
        
        # Check if secret file exists
        if not secret_path.exists():
            logging.debug("Google OAuth credentials not found at %s (placeholder secret or not mounted)", secret_path)
            return False
        
        # Check if file has content (not empty placeholder)
        try:
            if secret_path.stat().st_size == 0:
                logging.debug("Google OAuth credentials file is empty (user hasn't authenticated yet)")
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
            logging.info("✓ Copied Google OAuth credentials from Secret to writable workspace at %s", dest_path)
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

        Also updates USER_GOOGLE_EMAIL environment variable from credentials if available.

        Returns:
            True if new credentials were found and copied, False otherwise.
        """
        dest_path = Path("/workspace/.google_workspace_mcp/credentials/credentials.json")

        # If we already have credentials in workspace, check if source is newer
        if dest_path.exists():
            secret_path = Path("/app/.google_workspace_mcp/credentials/credentials.json")
            if secret_path.exists():
                try:
                    # Compare modification times - secret mount updates when K8s syncs
                    if secret_path.stat().st_mtime > dest_path.stat().st_mtime:
                        logging.info("Detected updated Google OAuth credentials, refreshing...")
                        if await self._try_copy_google_credentials():
                            # Update USER_GOOGLE_EMAIL from the new credentials
                            self._update_email_from_credentials(dest_path)
                            return True
                except OSError:
                    pass
            # Always try to update email even if file didn't change (in case env var is still default)
            self._update_email_from_credentials(dest_path)
            return False

        # No credentials yet, try to copy
        if await self._try_copy_google_credentials():
            logging.info("✓ Google OAuth credentials now available (user completed authentication)")
            # Update USER_GOOGLE_EMAIL from the new credentials
            self._update_email_from_credentials(dest_path)
            return True
        return False

    def _update_email_from_credentials(self, creds_path: Path):
        """Read user email from credentials file and update USER_GOOGLE_EMAIL env var.

        The email is extracted from the OAuth credentials stored by the backend.
        """
        try:
            # First check if there's a user_email file in the secret mount
            secret_email_path = Path("/app/.google_workspace_mcp/credentials/user_email")
            if secret_email_path.exists():
                email = secret_email_path.read_text().strip()
                if email and email != "user@example.com":
                    current_email = os.getenv("USER_GOOGLE_EMAIL", "")
                    if current_email != email:
                        os.environ["USER_GOOGLE_EMAIL"] = email
                        logging.info(f"Updated USER_GOOGLE_EMAIL from secret: {email}")
                    return

            # Fallback: try to parse email from credentials.json (some OAuth providers include it)
            if creds_path.exists() and creds_path.stat().st_size > 0:
                try:
                    with open(creds_path, 'r') as f:
                        creds = _json.load(f)
                        # Some OAuth responses include email in the credentials
                        email = creds.get('email') or creds.get('user_email')
                        if email and email != "user@example.com":
                            current_email = os.getenv("USER_GOOGLE_EMAIL", "")
                            if current_email != email:
                                os.environ["USER_GOOGLE_EMAIL"] = email
                                logging.info(f"Updated USER_GOOGLE_EMAIL from credentials.json: {email}")
                except (_json.JSONDecodeError, KeyError):
                    pass
        except Exception as e:
            logging.debug(f"Could not update USER_GOOGLE_EMAIL from credentials: {e}")