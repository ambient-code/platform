"""Repository management endpoints: /repos/add, /repos/remove, /repos/status."""

import asyncio
import json
import logging
import os
import re
import shutil
import tempfile
import uuid
from pathlib import Path

import aiohttp
from fastapi import APIRouter, HTTPException, Request

from ambient_runner.platform.auth import ensure_git_auth
from ambient_runner.platform.utils import get_bot_token, redact_secrets

logger = logging.getLogger(__name__)

router = APIRouter()


# ------------------------------------------------------------------
# Endpoints
# ------------------------------------------------------------------


@router.post("/repos/add")
async def add_repo(request: Request):
    """Clone a repo into the workspace and trigger adapter reinit."""
    bridge = request.app.state.bridge
    context = bridge.context
    if not context:
        raise HTTPException(status_code=503, detail="Context not initialized")

    body = await request.json()
    url = body.get("url", "")
    branch = body.get("branch", "main")
    name = body.get("name", "")

    github_token = request.headers.get("X-GitHub-Token", "").strip() or None
    gitlab_token = request.headers.get("X-GitLab-Token", "").strip() or None

    if github_token:
        logger.info("Using GitHub authentication from request header")
    elif gitlab_token:
        logger.info("Using GitLab authentication from request header")

    logger.info(f"Add repo request: url={url}, branch={branch}, name={name}")

    if not url:
        raise HTTPException(status_code=400, detail="Repository URL is required")

    if not name:
        name = url.split("/")[-1].removesuffix(".git")

    success, repo_path, was_newly_cloned = await clone_repo_at_runtime(
        url, branch, name, github_token, gitlab_token
    )
    if not success:
        raise HTTPException(
            status_code=500, detail=f"Failed to clone repository: {url}"
        )

    if was_newly_cloned:
        repos_json = os.getenv("REPOS_JSON", "[]")
        try:
            repos = json.loads(repos_json) if repos_json else []
        except Exception:
            repos = []
        repos.append({"name": name, "input": {"url": url, "branch": branch}})
        os.environ["REPOS_JSON"] = json.dumps(repos)

        bridge.mark_dirty()
        logger.info(
            f"Repo '{name}' added and cloned, adapter will reinitialize on next run"
        )
        asyncio.create_task(_trigger_repo_added_notification(name, url, context))
    else:
        logger.info(
            f"Repo '{name}' already existed, skipping notification (idempotent call)"
        )

    return {
        "message": "Repository added",
        "name": name,
        "path": repo_path,
        "newly_cloned": was_newly_cloned,
    }


@router.post("/repos/remove")
async def remove_repo(request: Request):
    """Remove a repo from the workspace."""
    bridge = request.app.state.bridge
    context = bridge.context
    if not context:
        raise HTTPException(status_code=503, detail="Context not initialized")

    body = await request.json()
    repo_name = body.get("name", "")
    logger.info(f"Remove repo request: {repo_name}")

    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    repo_path = Path(workspace_path) / "repos" / repo_name

    if repo_path.exists():
        try:
            shutil.rmtree(repo_path)
            logger.info(f"Deleted repository directory: {repo_path}")
        except Exception as e:
            logger.error(f"Failed to delete repository directory {repo_path}: {e}")
            raise HTTPException(
                status_code=500, detail=f"Failed to delete repository: {e}"
            )
    else:
        logger.warning(f"Repository directory not found: {repo_path}")

    repos_json = os.getenv("REPOS_JSON", "[]")
    try:
        repos = json.loads(repos_json) if repos_json else []
    except Exception:
        repos = []
    repos = [r for r in repos if r.get("name") != repo_name]
    os.environ["REPOS_JSON"] = json.dumps(repos)

    bridge.mark_dirty()
    logger.info("Repo removed, adapter will reinitialize on next run")

    return {"message": "Repository removed"}


@router.get("/repos/status")
async def get_repos_status():
    """Get current status of all repositories in the workspace."""
    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    repos_dir = Path(workspace_path) / "repos"

    if not repos_dir.exists():
        return {"repos": []}

    repos_status = []
    for repo_path in repos_dir.iterdir():
        if not repo_path.is_dir() or not (repo_path / ".git").exists():
            continue

        try:
            repo_name = repo_path.name

            process = await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(repo_path),
                "config",
                "--get",
                "remote.origin.url",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, stderr = await process.communicate()
            repo_url = stdout.decode().strip() if process.returncode == 0 else ""
            repo_url = re.sub(r"https://[^:]+:[^@]+@", "https://", repo_url)

            process = await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(repo_path),
                "rev-parse",
                "--abbrev-ref",
                "HEAD",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, stderr = await process.communicate()
            current_branch = (
                stdout.decode().strip() if process.returncode == 0 else "unknown"
            )

            process = await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(repo_path),
                "branch",
                "--format=%(refname:short)",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            stdout, stderr = await process.communicate()
            branches = (
                [b.strip() for b in stdout.decode().split("\n") if b.strip()]
                if process.returncode == 0
                else []
            )

            default_branch = await get_default_branch(str(repo_path))

            repos_status.append(
                {
                    "url": repo_url,
                    "name": repo_name,
                    "branches": branches,
                    "currentActiveBranch": current_branch,
                    "defaultBranch": default_branch,
                }
            )
        except Exception as e:
            logger.error(f"Error getting status for repo {repo_path}: {e}")
            continue

    return {"repos": repos_status}


# ------------------------------------------------------------------
# Git helpers
# ------------------------------------------------------------------


async def get_default_branch(repo_path: str) -> str:
    """Get the default branch with robust fallback."""
    process = await asyncio.create_subprocess_exec(
        "git",
        "-C",
        str(repo_path),
        "symbolic-ref",
        "refs/remotes/origin/HEAD",
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    stdout, _ = await process.communicate()
    if process.returncode == 0:
        branch = stdout.decode().strip().split("/")[-1]
        if branch:
            return branch

    process = await asyncio.create_subprocess_exec(
        "git",
        "-C",
        str(repo_path),
        "remote",
        "show",
        "origin",
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    stdout, _ = await process.communicate()
    if process.returncode == 0:
        for line in stdout.decode().split("\n"):
            if "HEAD branch:" in line:
                branch = line.split(":")[-1].strip()
                if branch and branch != "(unknown)":
                    return branch

    for candidate in ["main", "master", "develop"]:
        process = await asyncio.create_subprocess_exec(
            "git",
            "-C",
            str(repo_path),
            "rev-parse",
            "--verify",
            f"origin/{candidate}",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        await process.communicate()
        if process.returncode == 0:
            return candidate

    return "main"


async def _sanitize_remote_url(repo_path: Path) -> None:
    """Strip embedded credentials from the origin remote URL.

    Repos cloned by older code may have tokens baked into the URL like
    https://x-access-token:TOKEN@github.com/... — replace with the clean URL
    so the credential helper is used instead.
    """
    try:
        process = await asyncio.create_subprocess_exec(
            "git",
            "-C",
            str(repo_path),
            "config",
            "--get",
            "remote.origin.url",
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, _ = await process.communicate()
        if process.returncode != 0:
            return
        current_url = stdout.decode().strip()
        clean_url = re.sub(r"https://[^:]+:[^@]+@", "https://", current_url)
        if clean_url != current_url:
            await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(repo_path),
                "remote",
                "set-url",
                "origin",
                clean_url,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            logger.info(f"Sanitized remote URL for {repo_path.name}")
    except Exception as e:
        logger.warning(f"Failed to sanitize remote URL for {repo_path}: {e}")


async def clone_repo_at_runtime(
    git_url: str,
    branch: str,
    name: str,
    github_token_override: str | None = None,
    gitlab_token_override: str | None = None,
) -> tuple[bool, str, bool]:
    """Clone a repository at runtime or add a new branch to existing repo."""
    if not git_url:
        return False, "", False

    if not name:
        name = git_url.split("/")[-1].removesuffix(".git")

    if not branch or branch.strip() == "":
        session_id = os.getenv("AGENTIC_SESSION_NAME", "").strip() or os.getenv(
            "SESSION_ID", "unknown"
        )
        branch = f"ambient/{session_id}"
        logger.info(f"No branch specified, auto-generated: {branch}")

    workspace_path = os.getenv("WORKSPACE_PATH", "/workspace")
    repos_dir = Path(workspace_path) / "repos"
    repos_dir.mkdir(parents=True, exist_ok=True)
    repo_final = repos_dir / name

    ensure_git_auth(github_token_override, gitlab_token_override)
    clone_url = git_url

    # Case 1: Repo already exists
    if repo_final.exists():
        logger.info(f"Repo '{name}' already exists, adding branch '{branch}'")
        try:
            # Clean any previously-embedded tokens from the remote URL
            await _sanitize_remote_url(repo_final)

            await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(repo_final),
                "fetch",
                "origin",
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )

            for checkout_args in (
                ["git", "-C", str(repo_final), "checkout", branch],
                [
                    "git",
                    "-C",
                    str(repo_final),
                    "checkout",
                    "-b",
                    branch,
                    f"origin/{branch}",
                ],
            ):
                p = await asyncio.create_subprocess_exec(
                    *checkout_args,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE,
                )
                await p.communicate()
                if p.returncode == 0:
                    return True, str(repo_final), False

            default_branch = await get_default_branch(str(repo_final))
            await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(repo_final),
                "checkout",
                default_branch,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            p = await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(repo_final),
                "checkout",
                "-b",
                branch,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )
            await p.communicate()
            if p.returncode == 0:
                return True, str(repo_final), False
            return False, "", False
        except Exception as e:
            logger.error(f"Error adding branch to existing repo: {e}")
            return False, "", False

    # Case 2: Clone fresh
    logger.info(f"Cloning repo '{name}' from {git_url}@{branch}")
    temp_dir = Path(tempfile.mkdtemp(prefix="repo-clone-"))

    try:
        process = await asyncio.create_subprocess_exec(
            "git",
            "clone",
            clone_url,
            str(temp_dir),
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        stdout, stderr = await process.communicate()
        if process.returncode != 0:
            logger.error(f"Failed to clone repo: {redact_secrets(stderr.decode())}")
            return False, "", False

        p = await asyncio.create_subprocess_exec(
            "git",
            "-C",
            str(temp_dir),
            "checkout",
            branch,
            stdout=asyncio.subprocess.PIPE,
            stderr=asyncio.subprocess.PIPE,
        )
        await p.communicate()
        if p.returncode != 0:
            await asyncio.create_subprocess_exec(
                "git",
                "-C",
                str(temp_dir),
                "checkout",
                "-b",
                branch,
                stdout=asyncio.subprocess.PIPE,
                stderr=asyncio.subprocess.PIPE,
            )

        repo_final.parent.mkdir(parents=True, exist_ok=True)
        shutil.move(str(temp_dir), str(repo_final))
        # Strip any embedded credentials from the remote URL (e.g. if the
        # caller passed an already-authenticated URL).
        await _sanitize_remote_url(repo_final)
        logger.info(f"Repo '{name}' ready at {repo_final} on branch '{branch}'")
        return True, str(repo_final), True

    except Exception as e:
        logger.error(f"Error cloning repo: {e}")
        return False, "", False
    finally:
        if temp_dir.exists():
            shutil.rmtree(temp_dir, ignore_errors=True)


async def _trigger_repo_added_notification(repo_name: str, repo_url: str, context):
    """Notify Claude that a repository has been added."""
    await asyncio.sleep(1)

    try:
        backend_url = os.getenv("BACKEND_API_URL", "").rstrip("/")
        project_name = os.getenv("AGENTIC_SESSION_NAMESPACE", "").strip()
        session_id = context.session_id if context else "unknown"

        if not backend_url or not project_name:
            return

        url = f"{backend_url}/projects/{project_name}/agentic-sessions/{session_id}/agui/run"
        payload = {
            "threadId": session_id,
            "runId": str(uuid.uuid4()),
            "messages": [
                {
                    "id": str(uuid.uuid4()),
                    "role": "user",
                    "content": f"The repository '{repo_name}' has been added to your workspace. You can now access it at the path 'repos/{repo_name}/'. Please acknowledge this to the user.",
                    "metadata": {
                        "hidden": True,
                        "autoSent": True,
                        "source": "repo_added",
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
                    logger.info(f"Repo notification sent: {await resp.json()}")
                else:
                    logger.error(
                        f"Repo notification failed: {resp.status} - {await resp.text()}"
                    )
    except Exception as e:
        logger.error(f"Failed to trigger repo notification: {e}")
