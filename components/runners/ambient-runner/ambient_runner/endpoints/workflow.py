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

from ambient_runner.platform.auth import ensure_git_auth
from ambient_runner.platform.config import load_ambient_config
from ambient_runner.platform.utils import get_bot_token, redact_secrets

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

    github_token = request.headers.get("X-GitHub-Token", "").strip() or None
    gitlab_token = request.headers.get("X-GitLab-Token", "").strip() or None

    if github_token:
        logger.info("Using GitHub authentication from request header")
    elif gitlab_token:
        logger.info("Using GitLab authentication from request header")

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
            return {
                "message": "Workflow already active",
                "gitUrl": git_url,
                "branch": branch,
                "path": path,
            }

        if git_url:
            success, _wf_path = await clone_workflow_at_runtime(
                git_url, branch, path, github_token, gitlab_token
            )
            if not success:
                logger.warning(
                    "Failed to clone workflow, will use default workflow directory"
                )

        os.environ["ACTIVE_WORKFLOW_GIT_URL"] = git_url
        os.environ["ACTIVE_WORKFLOW_BRANCH"] = branch
        os.environ["ACTIVE_WORKFLOW_PATH"] = path

        bridge.mark_dirty()

        return {
            "message": "Workflow updated",
            "gitUrl": git_url,
            "branch": branch,
            "path": path,
        }


# ------------------------------------------------------------------
# Helpers
# ------------------------------------------------------------------


async def clone_workflow_at_runtime(
    git_url: str,
    branch: str,
    subpath: str,
    github_token_override: str | None = None,
    gitlab_token_override: str | None = None,
) -> tuple[bool, str]:
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
        ensure_git_auth(
            github_token=github_token_override, gitlab_token=gitlab_token_override
        )
        clone_url = git_url

        process = await asyncio.create_subprocess_exec(
            "git",
            "clone",
            "--branch",
            branch,
            "--single-branch",
            "--depth",
            "1",
            clone_url,
            str(temp_dir),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await process.communicate()

        if process.returncode != 0:
            logger.error(f"Failed to clone workflow: {redact_secrets(stderr.decode())}")
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


async def _trigger_workflow_greeting(workflow_dir: str, context):
    """Send the workflow's startupPrompt (from ambient.json) after a workflow change.

    If the workflow has no startupPrompt, no greeting is sent.
    """
    try:
        if not workflow_dir or not Path(workflow_dir).exists():
            logger.info(f"Workflow dir '{workflow_dir}' not found, skipping greeting")
            return

        config = (
            load_ambient_config(workflow_dir) if Path(workflow_dir).exists() else {}
        )
        startup_prompt = (config.get("startupPrompt") or "").strip()

        if not startup_prompt:
            logger.info(
                f"Workflow at '{workflow_dir}' has no startupPrompt in ambient.json, "
                f"skipping greeting"
            )
            return

        backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
        project_name = os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
        session_id = context.session_id if context else "unknown"

        if not backend_url or not project_name:
            logger.error(
                "Cannot trigger workflow greeting: BACKEND_API_URL or PROJECT_NAME not set"
            )
            return

        url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"

        payload = {
            "threadId": session_id,
            "runId": str(uuid.uuid4()),
            "messages": [
                {
                    "id": str(uuid.uuid4()),
                    "role": "user",
                    "content": startup_prompt,
                    "metadata": {
                        "hidden": True,
                        "autoSent": True,
                        "source": "workflow_startup_prompt",
                    },
                }
            ],
        }

        bot_token = get_bot_token()
        headers = {"Content-Type": "application/json"}
        if bot_token:
            headers["Authorization"] = f"Bearer {bot_token}"

        async with aiohttp.ClientSession() as session:
            async with session.post(url, json=payload, headers=headers) as resp:
                if resp.status == 200:
                    logger.info(f"Workflow startupPrompt sent for '{workflow_dir}'")
                else:
                    logger.error(
                        f"Workflow startupPrompt failed: {resp.status} - {await resp.text()}"
                    )
    except Exception as e:
        logger.error(f"Failed to send workflow startupPrompt: {e}")
