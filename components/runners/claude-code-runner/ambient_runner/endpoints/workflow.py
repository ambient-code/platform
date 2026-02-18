"""POST /workflow — Change active workflow at runtime."""

import asyncio
import logging
import os
import shutil
import tempfile
import uuid
from pathlib import Path

import aiohttp
from fastapi import APIRouter, HTTPException, Request

logger = logging.getLogger(__name__)

router = APIRouter()

# Serialise workflow changes to prevent concurrent reinit
_workflow_change_lock = asyncio.Lock()


@router.post("/workflow")
async def change_workflow(request: Request):
    """Change active workflow — triggers adapter reinit and greeting."""
    bridge = request.app.state.bridge
    context = bridge.context
    if not context:
        raise HTTPException(status_code=503, detail="Context not initialized")

    body = await request.json()
    git_url = (body.get("gitUrl") or "").strip()
    branch = (body.get("branch") or "main").strip() or "main"
    path = (body.get("path") or "").strip()

    logger.info(f"Workflow change request: {git_url}@{branch} (path: {path})")

    async with _workflow_change_lock:
        current_git_url = os.getenv("ACTIVE_WORKFLOW_GIT_URL", "").strip()
        current_branch = os.getenv("ACTIVE_WORKFLOW_BRANCH", "main").strip() or "main"
        current_path = os.getenv("ACTIVE_WORKFLOW_PATH", "").strip()

        if current_git_url == git_url and current_branch == branch and current_path == path:
            logger.info("Workflow unchanged; skipping reinit and greeting")
            return {"message": "Workflow already active", "gitUrl": git_url, "branch": branch, "path": path}

        if git_url:
            success, _wf_path = await clone_workflow_at_runtime(git_url, branch, path)
            if not success:
                logger.warning("Failed to clone workflow, will use default workflow directory")

        os.environ["ACTIVE_WORKFLOW_GIT_URL"] = git_url
        os.environ["ACTIVE_WORKFLOW_BRANCH"] = branch
        os.environ["ACTIVE_WORKFLOW_PATH"] = path

        bridge.mark_dirty()

        logger.info("Workflow updated, adapter will reinitialize on next run")
        asyncio.create_task(_trigger_workflow_greeting(git_url, branch, path, context))

        return {"message": "Workflow updated", "gitUrl": git_url, "branch": branch, "path": path}


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------


async def clone_workflow_at_runtime(git_url: str, branch: str, subpath: str) -> tuple[bool, str]:
    """Clone a workflow repository at runtime."""
    if not git_url:
        return False, ""

    workflow_name = git_url.split("/")[-1].removesuffix(".git")
    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    workflow_final = Path(workspace_path) / "workflows" / workflow_name

    logger.info(f"Cloning workflow '{workflow_name}' from {git_url}@{branch}")
    if subpath:
        logger.info(f"  Subpath: {subpath}")

    temp_dir = Path(tempfile.mkdtemp(prefix="workflow-clone-"))

    try:
        github_token = os.getenv("GITHUB_TOKEN", "").strip()
        gitlab_token = os.getenv("GITLAB_TOKEN", "").strip()

        clone_url = git_url
        if github_token and "github" in git_url.lower():
            clone_url = git_url.replace("https://", f"https://x-access-token:{github_token}@")
        elif gitlab_token and "gitlab" in git_url.lower():
            clone_url = git_url.replace("https://", f"https://oauth2:{gitlab_token}@")

        process = await asyncio.create_subprocess_exec(
            "git", "clone", "--branch", branch, "--single-branch", "--depth", "1",
            clone_url, str(temp_dir),
            stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await process.communicate()

        if process.returncode != 0:
            error_msg = stderr.decode()
            for tok in (github_token, gitlab_token):
                if tok:
                    error_msg = error_msg.replace(tok, "***REDACTED***")
            logger.error(f"Failed to clone workflow: {error_msg}")
            return False, ""

        if subpath:
            subpath_full = temp_dir / subpath
            if subpath_full.exists() and subpath_full.is_dir():
                if workflow_final.exists():
                    shutil.rmtree(workflow_final)
                workflow_final.parent.mkdir(parents=True, exist_ok=True)
                shutil.copytree(subpath_full, workflow_final)
            else:
                logger.warning(f"Subpath '{subpath}' not found, using entire repo")
                if workflow_final.exists():
                    shutil.rmtree(workflow_final)
                shutil.move(str(temp_dir), str(workflow_final))
        else:
            if workflow_final.exists():
                shutil.rmtree(workflow_final)
            shutil.move(str(temp_dir), str(workflow_final))

        logger.info(f"Workflow '{workflow_name}' ready at {workflow_final}")
        return True, str(workflow_final)

    except Exception as e:
        logger.error(f"Error cloning workflow: {e}")
        return False, ""
    finally:
        if temp_dir.exists():
            shutil.rmtree(temp_dir, ignore_errors=True)


async def _trigger_workflow_greeting(git_url: str, branch: str, path: str, context):
    """POST a greeting prompt to the backend after a workflow change."""
    try:
        backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
        project_name = os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
        session_id = context.session_id if context else "unknown"

        if not backend_url or not project_name:
            logger.error("Cannot trigger workflow greeting: BACKEND_API_URL or PROJECT_NAME not set")
            return

        url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"
        workflow_name = git_url.split("/")[-1].removesuffix(".git")
        if path:
            workflow_name = path.split("/")[-1]

        payload = {
            "threadId": session_id,
            "runId": str(uuid.uuid4()),
            "messages": [{
                "id": str(uuid.uuid4()),
                "role": "user",
                "content": f"Greet the user and explain that the {workflow_name} workflow is now active. Briefly describe what this workflow helps with. Keep it concise and friendly.",
                "metadata": {"hidden": True, "autoSent": True, "source": "workflow_activation"},
            }],
        }

        bot_token = os.getenv("BOT_TOKEN", "").strip()
        headers = {"Content-Type": "application/json"}
        if bot_token:
            headers["Authorization"] = f"Bearer {bot_token}"

        async with aiohttp.ClientSession() as session:
            async with session.post(url, json=payload, headers=headers) as resp:
                if resp.status == 200:
                    logger.info(f"Workflow greeting started: {await resp.json()}")
                else:
                    logger.error(f"Workflow greeting failed: {resp.status} - {await resp.text()}")
    except Exception as e:
        logger.error(f"Failed to trigger workflow greeting: {e}")
