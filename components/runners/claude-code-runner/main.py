"""
AG-UI Server entry point for Claude Code runner.
Implements the official AG-UI server pattern.
"""
import asyncio
import os
import json
import logging
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Optional, List, Dict, Any, Union

from fastapi import FastAPI, Request, HTTPException
from fastapi.responses import StreamingResponse
from pydantic import BaseModel
import uvicorn

from ag_ui.core import RunAgentInput
from ag_ui.encoder import EventEncoder

from context import RunnerContext

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


# Flexible input model that matches what our frontend actually sends
class RunnerInput(BaseModel):
    """Input model for runner with optional AG-UI fields."""
    threadId: Optional[str] = None
    thread_id: Optional[str] = None  # Support both camelCase and snake_case
    runId: Optional[str] = None
    run_id: Optional[str] = None
    parentRunId: Optional[str] = None
    parent_run_id: Optional[str] = None
    messages: List[Dict[str, Any]]
    state: Optional[Dict[str, Any]] = None
    tools: Optional[List[Any]] = None
    context: Optional[Union[List[Any], Dict[str, Any]]] = None  # Accept both list and dict, convert to list
    forwardedProps: Optional[Dict[str, Any]] = None
    environment: Optional[Dict[str, str]] = None
    metadata: Optional[Dict[str, Any]] = None
    
    def to_run_agent_input(self) -> RunAgentInput:
        """Convert to official RunAgentInput model."""
        import uuid
        
        # Normalize field names (prefer camelCase for AG-UI)
        thread_id = self.threadId or self.thread_id
        run_id = self.runId or self.run_id
        parent_run_id = self.parentRunId or self.parent_run_id
        
        # Generate runId if not provided
        if not run_id:
            run_id = str(uuid.uuid4())
            logger.info(f"Generated run_id: {run_id}")
        
        # Context should be a list, not a dict
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

# Global context and adapter
context: Optional[RunnerContext] = None
adapter = None  # Will be ClaudeCodeAdapter after initialization


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Initialize and cleanup application resources."""
    global context, adapter
    
    # Import adapter here to avoid circular imports
    from adapter import ClaudeCodeAdapter
    
    # Initialize context from environment
    session_id = os.getenv("SESSION_ID", "unknown")
    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    
    logger.info(f"Initializing AG-UI server for session {session_id}")
    
    context = RunnerContext(
        session_id=session_id,
        workspace_path=workspace_path,
    )
    
    adapter = ClaudeCodeAdapter()
    adapter.context = context
    
    logger.info("Adapter initialized - fresh client will be created for each run")
    
    # Check if this is a restart/resume (initial prompt was already sent)
    # We use our own marker file since Claude SDK's internal state location varies
    initial_prompt_marker = Path(workspace_path) / ".initial_prompt_sent"
    
    # Check for INITIAL_PROMPT and auto-execute (only if this is first run)
    initial_prompt = os.getenv("INITIAL_PROMPT", "").strip()
    if initial_prompt and not initial_prompt_marker.exists():
        logger.info(f"INITIAL_PROMPT detected ({len(initial_prompt)} chars), will auto-execute after 3s delay")
        asyncio.create_task(auto_execute_initial_prompt(initial_prompt, session_id, initial_prompt_marker))
    elif initial_prompt:
        logger.info(f"INITIAL_PROMPT detected but marker exists - skipping auto-execution (session resume)")
    
    logger.info(f"AG-UI server ready for session {session_id}")
    
    yield
    
    # Cleanup
    logger.info("Shutting down AG-UI server...")


async def auto_execute_initial_prompt(prompt: str, session_id: str, marker_path: Path):
    """Auto-execute INITIAL_PROMPT by POSTing to backend after short delay.
    
    The 3-second delay gives the runner time to fully start. Backend has retry
    logic to handle if Service DNS isn't ready yet.
    
    Creates a marker file on success to prevent re-sending on session resume.
    """
    import uuid
    import aiohttp
    
    # Give runner time to fully start before backend tries to reach us
    logger.info("Waiting 3s before auto-executing INITIAL_PROMPT (allow Service DNS to propagate)...")
    await asyncio.sleep(3)
    
    logger.info("Auto-executing INITIAL_PROMPT via backend POST...")
    
    # Get backend URL from environment
    backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
    project_name = os.getenv("PROJECT_NAME", "").strip() or os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
    
    if not backend_url or not project_name:
        logger.error("Cannot auto-execute INITIAL_PROMPT: BACKEND_API_URL or PROJECT_NAME not set")
        return
    
    # BACKEND_API_URL already includes /api suffix from operator
    url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"
    logger.info(f"Auto-execution URL: {url}")
    
    payload = {
        "threadId": session_id,
        "runId": str(uuid.uuid4()),
        "messages": [{
            "id": str(uuid.uuid4()),
            "role": "user",
            "content": prompt,
            "metadata": {
                "hidden": True,
                "autoSent": True,
                "source": "runner_initial_prompt"
            }
        }]
    }
    
    # Get BOT_TOKEN for auth
    bot_token = os.getenv("BOT_TOKEN", "").strip()
    headers = {"Content-Type": "application/json"}
    if bot_token:
        headers["Authorization"] = f"Bearer {bot_token}"
    
    try:
        async with aiohttp.ClientSession() as session:
            async with session.post(url, json=payload, headers=headers, timeout=aiohttp.ClientTimeout(total=30)) as resp:
                if resp.status == 200:
                    result = await resp.json()
                    logger.info(f"INITIAL_PROMPT auto-execution started: {result}")
                    # Create marker file to prevent re-sending on resume
                    try:
                        marker_path.parent.mkdir(parents=True, exist_ok=True)
                        marker_path.write_text(f"sent_at={asyncio.get_event_loop().time()}\n")
                        logger.info(f"Created initial prompt marker: {marker_path}")
                    except Exception as marker_err:
                        logger.warning(f"Failed to create initial prompt marker: {marker_err}")
                else:
                    error_text = await resp.text()
                    logger.warning(f"INITIAL_PROMPT failed with status {resp.status}: {error_text[:200]}")
    except Exception as e:
        logger.warning(f"INITIAL_PROMPT auto-execution error (backend will retry): {e}")



app = FastAPI(
    title="Claude Code AG-UI Server",
    version="0.2.0",
    lifespan=lifespan
)


# Track if adapter has been initialized
_adapter_initialized = False


@app.post("/")
async def run_agent(input_data: RunnerInput, request: Request):
    """
    AG-UI compatible run endpoint.
    
    Accepts flexible input with thread_id, run_id, messages.
    Optional fields: state, tools, context, forwardedProps.
    Returns SSE stream of AG-UI events.
    """
    global _adapter_initialized
    
    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")
    
    # Convert to official RunAgentInput
    run_agent_input = input_data.to_run_agent_input()
    
    # Get Accept header for encoder
    accept_header = request.headers.get("accept", "text/event-stream")
    encoder = EventEncoder(accept=accept_header)
    
    logger.info(f"Processing run: thread_id={run_agent_input.thread_id}, run_id={run_agent_input.run_id}")
    
    async def event_generator():
        """Generate AG-UI events from adapter."""
        global _adapter_initialized
        
        try:
            logger.info("Event generator started")
            
            # Initialize adapter on first run (yields setup events)
            if not _adapter_initialized:
                logger.info("First run - initializing adapter with workspace preparation")
                async for event in adapter.initialize(context):
                    logger.debug(f"Yielding initialization event: {event.type}")
                    yield encoder.encode(event)
                logger.info("Adapter initialization complete")
                _adapter_initialized = True
            
            logger.info("Starting adapter.process_run()...")
            
            # Process the run (creates fresh client each time)
            async for event in adapter.process_run(run_agent_input):
                logger.debug(f"Yielding run event: {event.type}")
                yield encoder.encode(event)
            logger.info("adapter.process_run() completed")
        except Exception as e:
            logger.error(f"Error in event generator: {e}")
            # Yield error event
            from ag_ui.core import RunErrorEvent, EventType
            error_event = RunErrorEvent(
                type=EventType.RUN_ERROR,
                thread_id=run_agent_input.thread_id or context.session_id,
                run_id=run_agent_input.run_id or "unknown",
                message=str(e)
            )
            yield encoder.encode(error_event)
    
    return StreamingResponse(
        event_generator(),
        media_type=encoder.get_content_type(),
        headers={
            "Cache-Control": "no-cache",
            "X-Accel-Buffering": "no",
        }
    )


@app.post("/interrupt")
async def interrupt_run():
    """
    Interrupt the current Claude SDK execution.
    
    Sends interrupt signal to Claude subprocess to stop mid-execution.
    See: https://platform.claude.com/docs/en/agent-sdk/python#methods
    """
    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")
    
    logger.info("Interrupt request received")
    
    try:
        # Call adapter's interrupt method which signals the active Claude SDK client
        await adapter.interrupt()
        
        return {"message": "Interrupt signal sent to Claude SDK"}
    except Exception as e:
        logger.error(f"Interrupt failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@app.post("/workflow")
async def change_workflow(request: Request):
    """
    Change active workflow - triggers Claude SDK client restart and new greeting.
    
    Accepts: {"gitUrl": "...", "branch": "...", "path": "..."}
    """
    global _adapter_initialized
    
    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")
    
    body = await request.json()
    git_url = body.get("gitUrl", "")
    branch = body.get("branch", "main")
    path = body.get("path", "")
    
    logger.info(f"Workflow change request: {git_url}@{branch} (path: {path})")
    
    # Update environment variables
    os.environ["ACTIVE_WORKFLOW_GIT_URL"] = git_url
    os.environ["ACTIVE_WORKFLOW_BRANCH"] = branch
    os.environ["ACTIVE_WORKFLOW_PATH"] = path
    
    # Reset adapter state to force reinitialization on next run
    _adapter_initialized = False
    adapter._first_run = True
    
    logger.info("Workflow updated, adapter will reinitialize on next run")
    
    # Trigger a new run to greet user with workflow context
    # This runs in background via backend POST
    import asyncio
    asyncio.create_task(trigger_workflow_greeting(git_url, branch, path))
    
    return {"message": "Workflow updated", "gitUrl": git_url, "branch": branch, "path": path}


async def trigger_workflow_greeting(git_url: str, branch: str, path: str):
    """Trigger workflow greeting after workflow change."""
    import uuid
    import aiohttp
    
    # Wait a moment for workflow to be cloned/initialized
    await asyncio.sleep(3)
    
    logger.info("Triggering workflow greeting...")
    
    try:
        backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
        project_name = os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
        session_id = context.session_id if context else "unknown"
        
        if not backend_url or not project_name:
            logger.error("Cannot trigger workflow greeting: BACKEND_API_URL or PROJECT_NAME not set")
            return
        
        url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"
        
        # Extract workflow name for greeting
        workflow_name = git_url.split("/")[-1].removesuffix(".git")
        if path:
            workflow_name = path.split("/")[-1]
        
        greeting = f"Greet the user and explain that the {workflow_name} workflow is now active. Briefly describe what this workflow helps with based on the systemPrompt in ambient.json. Keep it concise and friendly."
        
        payload = {
            "threadId": session_id,
            "runId": str(uuid.uuid4()),
            "messages": [{
                "id": str(uuid.uuid4()),
                "role": "user",
                "content": greeting,
                "metadata": {
                    "hidden": True,
                    "autoSent": True,
                    "source": "workflow_activation"
                }
            }]
        }
        
        bot_token = os.getenv("BOT_TOKEN", "").strip()
        headers = {"Content-Type": "application/json"}
        if bot_token:
            headers["Authorization"] = f"Bearer {bot_token}"
        
        async with aiohttp.ClientSession() as session:
            async with session.post(url, json=payload, headers=headers) as resp:
                if resp.status == 200:
                    result = await resp.json()
                    logger.info(f"Workflow greeting started: {result}")
                else:
                    error_text = await resp.text()
                    logger.error(f"Workflow greeting failed: {resp.status} - {error_text}")
    
    except Exception as e:
        logger.error(f"Failed to trigger workflow greeting: {e}")


@app.post("/repos/add")
async def add_repo(request: Request):
    """
    Add repository - triggers Claude SDK client restart.
    
    Accepts: {"url": "...", "branch": "...", "name": "..."}
    """
    global _adapter_initialized
    
    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")
    
    body = await request.json()
    logger.info(f"Add repo request: {body}")
    
    # Update REPOS_JSON env var
    repos_json = os.getenv("REPOS_JSON", "[]")
    try:
        repos = json.loads(repos_json) if repos_json else []
    except:
        repos = []
    
    # Add new repo
    repos.append({
        "name": body.get("name", ""),
        "input": {
            "url": body.get("url", ""),
            "branch": body.get("branch", "main")
        }
    })
    
    os.environ["REPOS_JSON"] = json.dumps(repos)
    
    # Reset adapter state
    _adapter_initialized = False
    adapter._first_run = True
    
    logger.info(f"Repo added, adapter will reinitialize on next run")
    
    return {"message": "Repository added"}


@app.post("/repos/remove")
async def remove_repo(request: Request):
    """
    Remove repository - triggers Claude SDK client restart.
    
    Accepts: {"name": "..."}
    """
    global _adapter_initialized
    
    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")
    
    body = await request.json()
    repo_name = body.get("name", "")
    logger.info(f"Remove repo request: {repo_name}")
    
    # Update REPOS_JSON env var
    repos_json = os.getenv("REPOS_JSON", "[]")
    try:
        repos = json.loads(repos_json) if repos_json else []
    except:
        repos = []
    
    # Remove repo by name
    repos = [r for r in repos if r.get("name") != repo_name]
    
    os.environ["REPOS_JSON"] = json.dumps(repos)
    
    # Reset adapter state
    _adapter_initialized = False
    adapter._first_run = True
    
    logger.info(f"Repo removed, adapter will reinitialize on next run")
    
    return {"message": "Repository removed"}


@app.get("/health")
async def health():
    """Health check endpoint."""
    return {
        "status": "healthy",
        "session_id": context.session_id if context else None,
    }


def main():
    """Start the AG-UI server."""
    port = int(os.getenv("AGUI_PORT", "8000"))
    host = os.getenv("AGUI_HOST", "0.0.0.0")
    
    logger.info(f"Starting Claude Code AG-UI server on {host}:{port}")
    
    uvicorn.run(
        app,
        host=host,
        port=port,
        log_level="info",
    )


if __name__ == "__main__":
    main()

