"""
AG-UI Server entry point for Claude Code runner.
Implements the official AG-UI server pattern.
"""

import asyncio
import logging
import os
from contextlib import asynccontextmanager
from typing import Any, Dict, List, Optional, Union

import uvicorn
from ag_ui.core import RunAgentInput
from ag_ui.encoder import EventEncoder
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import StreamingResponse
from pydantic import BaseModel

from context import RunnerContext
from endpoints import state

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# ------------------------------------------------------------------
# Input model
# ------------------------------------------------------------------


class RunnerInput(BaseModel):
    """Input model for runner with optional AG-UI fields."""
    threadId: Optional[str] = None
    thread_id: Optional[str] = None
    runId: Optional[str] = None
    run_id: Optional[str] = None
    parentRunId: Optional[str] = None
    parent_run_id: Optional[str] = None
    messages: List[Dict[str, Any]]
    state: Optional[Dict[str, Any]] = None
    tools: Optional[List[Any]] = None
    context: Optional[Union[List[Any], Dict[str, Any]]] = None
    forwardedProps: Optional[Dict[str, Any]] = None
    environment: Optional[Dict[str, str]] = None
    metadata: Optional[Dict[str, Any]] = None

    def to_run_agent_input(self) -> RunAgentInput:
        """Convert to official RunAgentInput model."""
        import uuid

        thread_id = self.threadId or self.thread_id
        run_id = self.runId or self.run_id
        parent_run_id = self.parentRunId or self.parent_run_id

        if not run_id:
            run_id = str(uuid.uuid4())
            logger.info(f"Generated run_id: {run_id}")

        context_list = self.context if isinstance(self.context, list) else []

        return RunAgentInput(
            thread_id=thread_id,
            run_id=run_id,
            parent_run_id=parent_run_id,
            messages=self.messages,
            state=self.state or {},
            tools=self.tools or [],
            context=context_list,
            forwarded_props=self.forwardedProps or {},
        )


# ------------------------------------------------------------------
# Lifespan
# ------------------------------------------------------------------


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize and cleanup application resources."""
    session_id = os.getenv("SESSION_ID", "unknown")
    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")

    logger.info(f"Initializing AG-UI server for session {session_id}")

    state.context = RunnerContext(
        session_id=session_id,
        workspace_path=workspace_path,
    )

    is_resume = os.getenv("IS_RESUME", "").strip().lower() == "true"
    if is_resume:
        logger.info("IS_RESUME=true - this is a resumed session")

    is_interactive = os.getenv("INTERACTIVE", "true").strip().lower() == "true"

    initial_prompt = os.getenv("INITIAL_PROMPT", "").strip()
    if initial_prompt:
        if not is_interactive and not is_resume:
            logger.info(f"INITIAL_PROMPT detected ({len(initial_prompt)} chars) - auto-executing for non-interactive session")
            asyncio.create_task(auto_execute_initial_prompt(initial_prompt, session_id))
        else:
            mode = "resumed" if is_resume else "interactive"
            logger.info(f"INITIAL_PROMPT detected ({len(initial_prompt)} chars) but not auto-executing ({mode} session)")

    logger.info(f"AG-UI server ready for session {session_id}")

    yield

    if state._obs:
        await state._obs.finalize()
    logger.info("Shutting down AG-UI server...")


async def auto_execute_initial_prompt(prompt: str, session_id: str):
    """Auto-execute INITIAL_PROMPT for non-interactive sessions."""
    import uuid
    import aiohttp

    delay_seconds = float(os.getenv("INITIAL_PROMPT_DELAY_SECONDS", "1"))
    logger.info(f"Waiting {delay_seconds}s before auto-executing INITIAL_PROMPT...")
    await asyncio.sleep(delay_seconds)

    backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
    project_name = (
        os.getenv("PROJECT_NAME", "").strip()
        or os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
    )

    if not backend_url or not project_name:
        logger.error("Cannot auto-execute INITIAL_PROMPT: BACKEND_API_URL or PROJECT_NAME not set")
        return

    url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"

    payload = {
        "threadId": session_id,
        "runId": str(uuid.uuid4()),
        "messages": [{
            "id": str(uuid.uuid4()),
            "role": "user",
            "content": prompt,
            "metadata": {"hidden": True, "autoSent": True, "source": "runner_initial_prompt"},
        }],
    }

    bot_token = os.getenv("BOT_TOKEN", "").strip()
    headers = {"Content-Type": "application/json"}
    if bot_token:
        headers["Authorization"] = f"Bearer {bot_token}"

    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(url, json=payload, headers=headers, timeout=aiohttp.ClientTimeout(total=30)) as resp:
                if resp.status == 200:
                    logger.info(f"INITIAL_PROMPT auto-execution started: {await resp.json()}")
                else:
                    logger.warning(f"INITIAL_PROMPT failed with status {resp.status}: {(await resp.text())[:200]}")
    except Exception as e:
        logger.warning(f"INITIAL_PROMPT auto-execution error (backend will retry): {e}")


# ------------------------------------------------------------------
# App + routers
# ------------------------------------------------------------------


app = FastAPI(title="Claude Code AG-UI Server", version="0.2.0", lifespan=lifespan)

# Include endpoint routers
from endpoints.capabilities import router as capabilities_router
from endpoints.feedback import router as feedback_router
from endpoints.mcp_status import router as mcp_status_router
from endpoints.repos import router as repos_router
from endpoints.workflow import router as workflow_router

app.include_router(capabilities_router)
app.include_router(feedback_router)
app.include_router(mcp_status_router)
app.include_router(repos_router)
app.include_router(workflow_router)


# ------------------------------------------------------------------
# Platform setup helper
# ------------------------------------------------------------------


async def _ensure_platform_ready():
    """Run one-time platform setup (auth, workspace, prerequisites)."""
    if state._platform_ready:
        return

    import adapter as adapter_mod
    import workspace

    await workspace.validate_prerequisites(state.context)
    state._configured_model, state._platform_info = await adapter_mod.setup_platform(state.context)

    try:
        from observability import ObservabilityManager
        import auth

        raw_user_id = os.getenv("USER_ID", "").strip()
        raw_user_name = os.getenv("USER_NAME", "").strip()
        user_id, user_name = auth.sanitize_user_context(raw_user_id, raw_user_name)

        obs = ObservabilityManager(
            session_id=state.context.session_id,
            user_id=user_id,
            user_name=user_name,
        )
        await obs.initialize(
            prompt="(pending)",
            namespace=state.context.get_env("AGENTIC_SESSION_NAMESPACE", "unknown"),
            model=state._configured_model,
        )
        state._obs = obs
    except Exception as e:
        logger.warning(f"Failed to initialize observability: {e}")

    state._platform_ready = True
    logger.info("Platform setup complete")


def _ensure_adapter():
    """Build or rebuild the adapter if the config has changed."""
    if state.adapter is not None and not state._adapter_dirty:
        return

    import adapter as adapter_mod

    state.adapter = adapter_mod.build_adapter(
        state.context,
        configured_model=state._configured_model,
        cwd_path=state._platform_info["cwd_path"],
        add_dirs=state._platform_info["add_dirs"],
        first_run=state._first_run,
        obs=state._obs,
    )
    state._adapter_dirty = False
    logger.info("Adapter built (persistent, will be reused across runs)")


# ------------------------------------------------------------------
# Core endpoints
# ------------------------------------------------------------------


@app.post("/")
async def run_agent(input_data: RunnerInput, request: Request):
    """AG-UI run endpoint — standard pattern: adapter.run() → stream events."""
    if not state.context:
        raise HTTPException(status_code=503, detail="Context not initialized")

    run_agent_input = input_data.to_run_agent_input()
    accept_header = request.headers.get("accept", "text/event-stream")
    encoder = EventEncoder(accept=accept_header)

    logger.info(f"Processing run: thread_id={run_agent_input.thread_id}, run_id={run_agent_input.run_id}")

    async def event_stream():
        try:
            from middleware import tracing_middleware, emit_developer_message

            # --- One-time platform setup ---
            if not state._platform_ready:
                await _ensure_platform_ready()

                # Emit developer events for setup progress
                async for evt in emit_developer_message(
                    f"Platform ready — model: {state._configured_model}, "
                    f"cwd: {state._platform_info.get('cwd_path', 'N/A')}"
                ):
                    yield encoder.encode(evt)

            # --- Build or reuse adapter ---
            _ensure_adapter()

            # Extract user prompt for observability
            user_msg = ""
            for msg in reversed(run_agent_input.messages or []):
                role = getattr(msg, "role", msg.get("role") if isinstance(msg, dict) else "")
                if role == "user":
                    content = getattr(msg, "content", msg.get("content", "") if isinstance(msg, dict) else "")
                    if isinstance(content, str):
                        user_msg = content
                        break

            # Wrap adapter stream with tracing middleware
            wrapped_stream = tracing_middleware(
                state.adapter.run(run_agent_input),
                obs=state._obs,
                model=state._configured_model,
                prompt=user_msg,
            )

            async for event in wrapped_stream:
                yield encoder.encode(event)

            state._first_run = False

        except Exception as e:
            logger.error(f"Error in event stream: {e}")
            from ag_ui.core import EventType, RunErrorEvent
            yield encoder.encode(RunErrorEvent(
                type=EventType.RUN_ERROR,
                thread_id=run_agent_input.thread_id or (state.context.session_id if state.context else ""),
                run_id=run_agent_input.run_id or "unknown",
                message=str(e),
            ))

    return StreamingResponse(
        event_stream(),
        media_type=encoder.get_content_type(),
        headers={"Cache-Control": "no-cache", "X-Accel-Buffering": "no"},
    )


@app.post("/interrupt")
async def interrupt_run():
    """Interrupt the current Claude SDK execution."""
    if not state.adapter:
        raise HTTPException(status_code=503, detail="No active adapter")

    logger.info("Interrupt request received")
    try:
        await state.adapter.interrupt()
        return {"message": "Interrupt signal sent to Claude SDK"}
    except Exception as e:
        logger.error(f"Interrupt failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/health")
async def health():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "session_id": state.context.session_id if state.context else None,
    }


# ------------------------------------------------------------------
# Entry point
# ------------------------------------------------------------------


def main():
    """Start the AG-UI server."""
    port = int(os.getenv("AGUI_PORT", "8000"))
    host = os.getenv("AGUI_HOST", "0.0.0.0")
    logger.info(f"Starting Claude Code AG-UI server on {host}:{port}")
    uvicorn.run(app, host=host, port=port, log_level="info")


if __name__ == "__main__":
    main()
