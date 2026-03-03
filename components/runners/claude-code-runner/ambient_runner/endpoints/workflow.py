"""POST /workflow — Change active workflow at runtime."""

import asyncio
import logging
import os
import shutil
import tempfile
from pathlib import Path

from fastapi import APIRouter, HTTPException, Request

logger = logging.getLogger(__name__)

router = APIRouter()

# Serialise workflow changes to prevent concurrent reinit
_workflow_change_lock = asyncio.Lock()


@router.post("/workflow")
async def change_workflow(request: Request):
    """Change active workflow — triggers adapter reinit."""
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
            logger.info("Workflow unchanged; skipping reinit")
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
