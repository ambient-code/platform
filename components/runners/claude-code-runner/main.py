"""
AG-UI Server entry point for Claude Code runner.
Implements the official AG-UI server pattern.
"""
import asyncio
import os
import json
import logging
from contextlib import asynccontextmanager
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
    
    # Check if this is a resume session via IS_RESUME env var
    # This is set by the operator when restarting a stopped/completed/failed session
    is_resume = os.getenv("IS_RESUME", "").strip().lower() == "true"
    if is_resume:
        logger.info("IS_RESUME=true - this is a resumed session")
    
    # INITIAL_PROMPT is no longer auto-executed on startup
    # User must explicitly send the first message to start the conversation
    # Workflow greetings are still triggered when a workflow is activated
    initial_prompt = os.getenv("INITIAL_PROMPT", "").strip()
    if initial_prompt:
        logger.info(f"INITIAL_PROMPT detected ({len(initial_prompt)} chars) but not auto-executing (user will send first message)")
    
    logger.info(f"AG-UI server ready for session {session_id}")
    
    yield
    
    # Cleanup
    logger.info("Shutting down AG-UI server...")


async def auto_execute_initial_prompt(prompt: str, session_id: str):
    """Auto-execute INITIAL_PROMPT by POSTing to backend after short delay.
    
    The delay gives the runner service time to register in DNS. Backend has retry
    logic to handle if Service DNS isn't ready yet, so this can be short.
    
    Only called for fresh sessions (no hydrated state in .claude/).
    """
    import uuid
    import aiohttp
    
    # Configurable delay (default 1s, was 3s)
    # Backend has retry logic, so we don't need to wait long
    delay_seconds = float(os.getenv("INITIAL_PROMPT_DELAY_SECONDS", "1"))
    logger.info(f"Waiting {delay_seconds}s before auto-executing INITIAL_PROMPT (allow Service DNS to propagate)...")
    await asyncio.sleep(delay_seconds)
    
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
# Prevent duplicate workflow updates/greetings from concurrent calls
_workflow_change_lock = asyncio.Lock()


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
            
            # Initialize adapter on first run
            if not _adapter_initialized:
                logger.info("First run - initializing adapter with workspace preparation")
                await adapter.initialize(context)
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


def _check_mcp_authentication(server_name: str) -> tuple[bool | None, str | None]:
    """
    Check if credentials are available for known MCP servers.
    
    Returns:
        Tuple of (is_authenticated, auth_message)
        Returns (None, None) for servers we don't know how to check
    """
    from pathlib import Path
    
    # Google Workspace MCP - we know how to check this
    if server_name == "google-workspace":
        # Check mounted secret location first, then workspace copy
        secret_path = Path("/app/.google_workspace_mcp/credentials/credentials.json")
        workspace_path = Path("/workspace/.google_workspace_mcp/credentials/credentials.json")
        
        for cred_path in [workspace_path, secret_path]:
            if cred_path.exists():
                try:
                    if cred_path.stat().st_size > 0:
                        return True, "Google OAuth credentials available"
                except OSError:
                    pass
        return False, "Google OAuth not configured - authenticate via Integrations page"
    
    # Jira/Atlassian MCP - we know how to check this
    if server_name in ("mcp-atlassian", "jira"):
        jira_url = os.getenv("JIRA_URL", "").strip()
        jira_token = os.getenv("JIRA_API_TOKEN", "").strip()
        
        if jira_url and jira_token:
            return True, "Jira credentials configured"
        elif jira_url:
            return False, "Jira URL set but API token missing"
        else:
            return False, "Jira not configured - set credentials in Workspace Settings"
    
    # For all other servers (webfetch, unknown) - don't claim to know auth status
    return None, None


@app.get("/mcp/status")
async def get_mcp_status():
    """
    Returns MCP servers configured for this session with authentication status.
    Goes straight to the source - uses adapter's _load_mcp_config() method.
    
    For known integrations (Google, Jira), also checks if credentials are present.
    """
    try:
        global adapter
        
        if not adapter:
            return {
                "servers": [],
                "totalCount": 0,
                "message": "Adapter not initialized yet"
            }
        
        mcp_servers_list = []
        
        # Get the working directory (same logic as adapter uses)
        workspace_path = adapter.context.workspace_path if adapter.context else "/workspace"
        
        active_workflow_url = os.getenv('ACTIVE_WORKFLOW_GIT_URL', '').strip()
        cwd_path = workspace_path
        
        if active_workflow_url:
            workflow_name = active_workflow_url.split("/")[-1].removesuffix(".git")
            workflow_path = os.path.join(workspace_path, "workflows", workflow_name)
            if os.path.exists(workflow_path):
                cwd_path = workflow_path
        
        # Use adapter's method to load MCP config (same as it does during runs)
        mcp_config = adapter._load_mcp_config(cwd_path)
        logger.info(f"MCP config: {mcp_config}")
        
        if mcp_config:
            for server_name, server_config in mcp_config.items():
                # Check authentication status for known servers (Google, Jira)
                is_authenticated, auth_message = _check_mcp_authentication(server_name)
                
                # Platform servers are built-in (webfetch), workflow servers come from config
                is_platform = server_name == "webfetch"
                
                server_info = {
                    "name": server_name,
                    "displayName": server_name.replace('-', ' ').replace('_', ' ').title(),
                    "status": "configured",
                    "command": server_config.get("command", ""),
                    "source": "platform" if is_platform else "workflow"
                }
                
                # Only include auth fields for servers we know how to check
                if is_authenticated is not None:
                    server_info["authenticated"] = is_authenticated
                    server_info["authMessage"] = auth_message
                
                mcp_servers_list.append(server_info)
        
        return {
            "servers": mcp_servers_list,
            "totalCount": len(mcp_servers_list),
            "note": "Status shows 'configured' - check 'authenticated' field for credential status"
        }
        
    except Exception as e:
        logger.error(f"Failed to get MCP status: {e}", exc_info=True)
        return {
            "servers": [],
            "totalCount": 0,
            "error": str(e)
        }


async def clone_workflow_at_runtime(git_url: str, branch: str, subpath: str) -> tuple[bool, str]:
    """
    Clone a workflow repository at runtime.
    
    This mirrors the logic in hydrate.sh but runs when workflows are changed
    after the pod has started.
    
    Returns:
        (success, workflow_dir_path) tuple
    """
    import tempfile
    import shutil
    from pathlib import Path
    
    if not git_url:
        return False, ""
    
    # Derive workflow name from URL
    workflow_name = git_url.split("/")[-1].removesuffix(".git")
    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    workflow_final = Path(workspace_path) / "workflows" / workflow_name
    
    logger.info(f"Cloning workflow '{workflow_name}' from {git_url}@{branch}")
    if subpath:
        logger.info(f"  Subpath: {subpath}")
    
    # Create temp directory for clone
    temp_dir = Path(tempfile.mkdtemp(prefix="workflow-clone-"))
    
    try:
        # Build git clone command with optional auth token
        github_token = os.getenv("GITHUB_TOKEN", "").strip()
        gitlab_token = os.getenv("GITLAB_TOKEN", "").strip()
        
        # Determine which token to use based on URL
        clone_url = git_url
        if github_token and "github" in git_url.lower():
            clone_url = git_url.replace("https://", f"https://x-access-token:{github_token}@")
            logger.info("Using GITHUB_TOKEN for workflow authentication")
        elif gitlab_token and "gitlab" in git_url.lower():
            clone_url = git_url.replace("https://", f"https://oauth2:{gitlab_token}@")
            logger.info("Using GITLAB_TOKEN for workflow authentication")
        
        # Clone the repository
        process = await asyncio.create_subprocess_exec(
            "git", "clone", "--branch", branch, "--single-branch", "--depth", "1",
            clone_url, str(temp_dir),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await process.communicate()
        
        if process.returncode != 0:
            # Redact tokens from error message
            error_msg = stderr.decode()
            if github_token:
                error_msg = error_msg.replace(github_token, "***REDACTED***")
            if gitlab_token:
                error_msg = error_msg.replace(gitlab_token, "***REDACTED***")
            logger.error(f"Failed to clone workflow: {error_msg}")
            return False, ""
        
        logger.info("Clone successful, processing...")
        
        # Handle subpath extraction
        if subpath:
            subpath_full = temp_dir / subpath
            if subpath_full.exists() and subpath_full.is_dir():
                logger.info(f"Extracting subpath: {subpath}")
                # Remove existing workflow dir if exists
                if workflow_final.exists():
                    shutil.rmtree(workflow_final)
                # Create parent dirs and copy subpath
                workflow_final.parent.mkdir(parents=True, exist_ok=True)
                shutil.copytree(subpath_full, workflow_final)
                logger.info(f"Workflow extracted to {workflow_final}")
            else:
                logger.warning(f"Subpath '{subpath}' not found, using entire repo")
                if workflow_final.exists():
                    shutil.rmtree(workflow_final)
                shutil.move(str(temp_dir), str(workflow_final))
        else:
            # No subpath - use entire repo
            if workflow_final.exists():
                shutil.rmtree(workflow_final)
            shutil.move(str(temp_dir), str(workflow_final))
        
        logger.info(f"Workflow '{workflow_name}' ready at {workflow_final}")
        return True, str(workflow_final)
        
    except Exception as e:
        logger.error(f"Error cloning workflow: {e}")
        return False, ""
    finally:
        # Cleanup temp directory if it still exists
        if temp_dir.exists():
            shutil.rmtree(temp_dir, ignore_errors=True)


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
    git_url = (body.get("gitUrl") or "").strip()
    branch = (body.get("branch") or "main").strip() or "main"
    path = (body.get("path") or "").strip()
    
    logger.info(f"Workflow change request: {git_url}@{branch} (path: {path})")

    async with _workflow_change_lock:
        current_git_url = os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip()
        current_branch = os.getenv("ACTIVE_WORKFLOW_BRANCH", "main").strip() or "main"
        current_path = os.getenv("ACTIVE_WORKFLOW_PATH", "").strip()

        if (
            current_git_url == git_url
            and current_branch == branch
            and current_path == path
        ):
            logger.info("Workflow unchanged; skipping reinit and greeting")
            return {"message": "Workflow already active", "gitUrl": git_url, "branch": branch, "path": path}

        # Clone the workflow repository at runtime
        # This is needed because the init container only runs once at pod startup
        if git_url:
            success, workflow_path = await clone_workflow_at_runtime(git_url, branch, path)
            if not success:
                logger.warning("Failed to clone workflow, will use default workflow directory")

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
        asyncio.create_task(trigger_workflow_greeting(git_url, branch, path))

        return {"message": "Workflow updated", "gitUrl": git_url, "branch": branch, "path": path}


async def get_default_branch(repo_path: str) -> str:
    """
    Get the default branch of a repository with robust fallback.

    Tries multiple methods in order:
    1. symbolic-ref on origin/HEAD
    2. git remote show origin (more reliable but slower)
    3. Fallback to common defaults: main, master, develop

    Args:
        repo_path: Path to the git repository

    Returns:
        The default branch name
    """
    # Method 1: symbolic-ref (fast but may not be set)
    process = await asyncio.create_subprocess_exec(
        "git", "-C", str(repo_path), "symbolic-ref", "refs/remotes/origin/HEAD",
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE
    )
    stdout, stderr = await process.communicate()
    if process.returncode == 0:
        # Output is like "refs/remotes/origin/main"
        default_branch = stdout.decode().strip().split("/")[-1]
        if default_branch:
            logger.info(f"Default branch from symbolic-ref: {default_branch}")
            return default_branch

    # Method 2: remote show origin (more reliable)
    process = await asyncio.create_subprocess_exec(
        "git", "-C", str(repo_path), "remote", "show", "origin",
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE
    )
    stdout, stderr = await process.communicate()
    if process.returncode == 0:
        # Look for line like "  HEAD branch: main"
        for line in stdout.decode().split("\n"):
            if "HEAD branch:" in line:
                default_branch = line.split(":")[-1].strip()
                if default_branch and default_branch != "(unknown)":
                    logger.info(f"Default branch from remote show: {default_branch}")
                    return default_branch

    # Method 3: Try common default branch names
    for candidate in ["main", "master", "develop"]:
        process = await asyncio.create_subprocess_exec(
            "git", "-C", str(repo_path), "rev-parse", "--verify", f"origin/{candidate}",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        await process.communicate()
        if process.returncode == 0:
            logger.info(f"Default branch found by trying common names: {candidate}")
            return candidate

    # Final fallback
    logger.warning("Could not determine default branch, falling back to 'main'")
    return "main"


async def clone_repo_at_runtime(git_url: str, branch: str, name: str) -> tuple[bool, str, bool]:
    """
    Clone a repository at runtime or add a new branch to existing repo.

    Behavior:
    - If repo doesn't exist: clone it (no --single-branch to support multi-branch)
    - If repo exists: fetch and checkout the new branch (idempotent)
    - If branch is empty/None: auto-generate unique ambient/<session-id> branch
    - If branch doesn't exist remotely: create it from default branch


    Args:
        git_url: Git repository URL
        branch: Branch to checkout (or empty/None to auto-generate)
        name: Name for the cloned directory (derived from URL if empty)

    Returns:
        (success, repo_dir_path, was_newly_cloned) tuple
        - success: True if repo is available (either newly cloned or already existed)
        - repo_dir_path: Path to the repo directory
        - was_newly_cloned: True only if the repo was actually cloned this time
    """
    import tempfile
    import shutil
    from pathlib import Path

    if not git_url:
        return False, "", False

    # Derive repo name from URL if not provided
    if not name:
        name = git_url.split("/")[-1].removesuffix(".git")

    # Generate unique branch name if not specified (only if user didn't provide one)
    # IMPORTANT: Keep in sync with backend (sessions.go) and frontend (add-context-modal.tsx)
    if not branch or branch.strip() == "":
        session_id = os.getenv("AGENTIC_SESSION_NAME", "").strip() or os.getenv("SESSION_ID", "unknown")
        branch = f"ambient/{session_id}"
        logger.info(f"No branch specified, auto-generated: {branch}")

    # Repos are stored in /workspace/repos/{name} (matching hydrate.sh)
    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    repos_dir = Path(workspace_path) / "repos"
    repos_dir.mkdir(parents=True, exist_ok=True)
    repo_final = repos_dir / name

    # Build clone URL with auth token
    github_token = os.getenv("GITHUB_TOKEN", "").strip()
    gitlab_token = os.getenv("GITLAB_TOKEN", "").strip()
    clone_url = git_url
    if github_token and "github" in git_url.lower():
        clone_url = git_url.replace("https://", f"https://x-access-token:{github_token}@")
        logger.info("Using GITHUB_TOKEN for authentication")
    elif gitlab_token and "gitlab" in git_url.lower():
        clone_url = git_url.replace("https://", f"https://oauth2:{gitlab_token}@")
        logger.info("Using GITLAB_TOKEN for authentication")

    # Case 1: Repo already exists - add new branch
    if repo_final.exists():
        logger.info(f"Repo '{name}' already exists at {repo_final}, adding branch '{branch}'")
        try:
            # Fetch latest refs
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_final), "fetch", "origin",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            await process.communicate()

            # Try to checkout the branch
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_final), "checkout", branch,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await process.communicate()

            if process.returncode == 0:
                logger.info(f"Checked out existing branch '{branch}'")
                return True, str(repo_final), False  # Already existed, not newly cloned

            # Branch doesn't exist locally, try to checkout from remote
            logger.info(f"Branch '{branch}' not found locally, trying origin/{branch}")
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_final), "checkout", "-b", branch, f"origin/{branch}",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await process.communicate()

            if process.returncode == 0:
                logger.info(f"Checked out branch '{branch}' from origin")
                return True, str(repo_final), False  # Already existed, not newly cloned

            # Branch doesn't exist remotely, create from default branch
            logger.info(f"Branch '{branch}' not found on remote, creating from default branch")

            # Get default branch using robust detection
            default_branch = await get_default_branch(str(repo_final))

            # Checkout default branch first
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_final), "checkout", default_branch,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            await process.communicate()

            # Create new branch from default
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_final), "checkout", "-b", branch,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await process.communicate()

            if process.returncode == 0:
                logger.info(f"Created new branch '{branch}' from '{default_branch}'")
                return True, str(repo_final), False  # Already existed, not newly cloned
            else:
                logger.error(f"Failed to create branch: {stderr.decode()}")
                return False, "", False

        except Exception as e:
            logger.error(f"Error adding branch to existing repo: {e}")
            return False, "", False

    # Case 2: Repo doesn't exist - clone it
    logger.info(f"Cloning repo '{name}' from {git_url}@{branch}")

    # Create temp directory for clone
    temp_dir = Path(tempfile.mkdtemp(prefix="repo-clone-"))

    try:
        # Clone without --single-branch to support multi-branch workflows
        # No --depth=1 to allow full branch operations
        process = await asyncio.create_subprocess_exec(
            "git", "clone",
            clone_url, str(temp_dir),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await process.communicate()

        if process.returncode != 0:
            error_msg = stderr.decode()
            if github_token:
                error_msg = error_msg.replace(github_token, "***REDACTED***")
            if gitlab_token:
                error_msg = error_msg.replace(gitlab_token, "***REDACTED***")
            logger.error(f"Failed to clone repo: {error_msg}")
            return False, "", False

        logger.info("Clone successful, checking out requested branch...")

        # Try to checkout requested/auto-generated branch
        process = await asyncio.create_subprocess_exec(
            "git", "-C", str(temp_dir), "checkout", branch,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE
        )
        stdout, stderr = await process.communicate()

        if process.returncode != 0:
            # Branch doesn't exist, create it from default branch
            logger.info(f"Branch '{branch}' not found, creating from default branch")
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(temp_dir), "checkout", "-b", branch,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            await process.communicate()


        # Move to final location
        logger.info("Moving to final location...")
        repo_final.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(str(temp_dir), str(repo_final))

        logger.info(f"Repo '{name}' ready at {repo_final} on branch '{branch}'")
        return True, str(repo_final), True  # Newly cloned

    except Exception as e:
        logger.error(f"Error cloning repo: {e}")
        return False, "", False
    finally:
        if temp_dir.exists():
            shutil.rmtree(temp_dir, ignore_errors=True)


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
    Add repository - clones repo and triggers Claude SDK client restart.
    
    Accepts: {"url": "...", "branch": "...", "name": "..."}
    """
    global _adapter_initialized
    
    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")
    
    body = await request.json()
    url = body.get("url", "")
    branch = body.get("branch", "main")
    name = body.get("name", "")
    
    logger.info(f"Add repo request: url={url}, branch={branch}, name={name}")
    
    if not url:
        raise HTTPException(status_code=400, detail="Repository URL is required")
    
    # Derive name from URL if not provided
    if not name:
        name = url.split("/")[-1].removesuffix(".git")
    
    # Clone the repository at runtime
    success, repo_path, was_newly_cloned = await clone_repo_at_runtime(url, branch, name)
    if not success:
        raise HTTPException(status_code=500, detail=f"Failed to clone repository: {url}")

    # Only update state and trigger notification if repo was newly cloned
    # This prevents duplicate notifications when both backend and operator call this endpoint
    if was_newly_cloned:
        # Update REPOS_JSON env var
        repos_json = os.getenv("REPOS_JSON", "[]")
        try:
            repos = json.loads(repos_json) if repos_json else []
        except:
            repos = []

        # Add new repo
        repos.append({
            "name": name,
            "input": {
                "url": url,
                "branch": branch
            }
        })

        os.environ["REPOS_JSON"] = json.dumps(repos)

        # Reset adapter state to force reinitialization on next run
        _adapter_initialized = False
        adapter._first_run = True

        logger.info(f"Repo '{name}' added and cloned, adapter will reinitialize on next run")

        # Trigger a notification to Claude about the new repository
        asyncio.create_task(trigger_repo_added_notification(name, url))
    else:
        logger.info(f"Repo '{name}' already existed, skipping notification (idempotent call)")

    return {"message": "Repository added", "name": name, "path": repo_path, "newly_cloned": was_newly_cloned}


async def trigger_repo_added_notification(repo_name: str, repo_url: str):
    """Notify Claude that a repository has been added."""
    import uuid
    import aiohttp
    
    # Wait a moment for repo to be fully ready
    await asyncio.sleep(1)
    
    logger.info(f"Triggering repo added notification for: {repo_name}")
    
    try:
        backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
        project_name = os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
        session_id = context.session_id if context else "unknown"
        
        if not backend_url or not project_name:
            logger.error("Cannot trigger repo notification: BACKEND_API_URL or PROJECT_NAME not set")
            return
        
        url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"
        
        notification = f"The repository '{repo_name}' has been added to your workspace. You can now access it at the path 'repos/{repo_name}/'. Please acknowledge this to the user and let them know you can now read and work with files in this repository."
        
        payload = {
            "threadId": session_id,
            "runId": str(uuid.uuid4()),
            "messages": [{
                "id": str(uuid.uuid4()),
                "role": "user",
                "content": notification,
                "metadata": {
                    "hidden": True,
                    "autoSent": True,
                    "source": "repo_added"
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
                    logger.info(f"Repo notification sent: {result}")
                else:
                    error_text = await resp.text()
                    logger.error(f"Repo notification failed: {resp.status} - {error_text}")
    
    except Exception as e:
        logger.error(f"Failed to trigger repo notification: {e}")


@app.post("/repos/remove")
async def remove_repo(request: Request):
    """
    Remove repository - triggers Claude SDK client restart.

    Accepts: {"name": "..."}
    """
    import shutil
    from pathlib import Path

    global _adapter_initialized

    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")

    body = await request.json()
    repo_name = body.get("name", "")
    logger.info(f"Remove repo request: {repo_name}")

    # Delete repository from filesystem
    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    repo_path = Path(workspace_path) / "repos" / repo_name

    if repo_path.exists():
        try:
            shutil.rmtree(repo_path)
            logger.info(f"Deleted repository directory: {repo_path}")
        except Exception as e:
            logger.error(f"Failed to delete repository directory {repo_path}: {e}")
            raise HTTPException(status_code=500, detail=f"Failed to delete repository: {e}")
    else:
        logger.warning(f"Repository directory not found: {repo_path}")

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


@app.get("/repos/status")
async def get_repos_status():
    """
    Get current status of all repositories in the workspace.

    Returns for each repo:
    - url: Repository URL
    - name: Directory name
    - branches: All local branches
    - currentActiveBranch: Currently checked out branch
    - defaultBranch: Default branch of remote
    """
    if not adapter:
        raise HTTPException(status_code=503, detail="Adapter not initialized")

    import re
    from pathlib import Path

    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    repos_dir = Path(workspace_path) / "repos"

    if not repos_dir.exists():
        return {"repos": []}

    repos_status = []

    # Iterate through all directories in repos/
    for repo_path in repos_dir.iterdir():
        if not repo_path.is_dir() or not (repo_path / ".git").exists():
            continue

        try:
            repo_name = repo_path.name

            # Get remote URL
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_path), "config", "--get", "remote.origin.url",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await process.communicate()
            repo_url = stdout.decode().strip() if process.returncode == 0 else ""

            # Strip any embedded tokens from URL before returning (security)
            # Remove patterns like: https://x-access-token:TOKEN@github.com -> https://github.com
            repo_url = re.sub(r'https://[^:]+:[^@]+@', 'https://', repo_url)

            # Get current active branch
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_path), "rev-parse", "--abbrev-ref", "HEAD",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await process.communicate()
            current_branch = stdout.decode().strip() if process.returncode == 0 else "unknown"

            # Get all local branches
            process = await asyncio.create_subprocess_exec(
                "git", "-C", str(repo_path), "branch", "--format=%(refname:short)",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE
            )
            stdout, stderr = await process.communicate()
            branches = [b.strip() for b in stdout.decode().split("\n") if b.strip()] if process.returncode == 0 else []

            # Get default branch using robust detection
            default_branch = await get_default_branch(str(repo_path))

            repos_status.append({
                "url": repo_url,
                "name": repo_name,
                "branches": branches,
                "currentActiveBranch": current_branch,
                "defaultBranch": default_branch,
            })

        except Exception as e:
            logger.error(f"Error getting status for repo {repo_path}: {e}")
            continue

    return {"repos": repos_status}


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

