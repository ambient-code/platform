"""Content service endpoints — file CRUD, git status, workflow metadata.

These endpoints replace the Go-based ambient-content sidecar container,
providing workspace file operations directly from the runner process.
All file paths are validated to stay within WORKSPACE_PATH.
"""

import asyncio
import json
import logging
import os
import base64
from datetime import datetime, timezone
from pathlib import Path

from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import Response

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/content")


# ------------------------------------------------------------------
# Path safety
# ------------------------------------------------------------------


def _get_workspace_path() -> Path:
    """Return the workspace root as a resolved Path."""
    return Path(os.getenv("WORKSPACE_PATH", "/workspace")).resolve()


def _safe_resolve(relative: str) -> Path:
    """Resolve *relative* under WORKSPACE_PATH; raise 400 on traversal."""
    workspace = _get_workspace_path()
    # Normalise: strip leading slashes so it's always relative
    cleaned = relative.lstrip("/")
    target = (workspace / cleaned).resolve()
    if not (target == workspace or str(target).startswith(str(workspace) + os.sep)):
        raise HTTPException(status_code=400, detail="invalid path")
    return target


# ------------------------------------------------------------------
# File CRUD
# ------------------------------------------------------------------


@router.get("/list")
async def content_list(path: str = ""):
    """List directory contents (mirrors Go ContentList)."""
    abs_path = _safe_resolve(path)

    if not abs_path.exists():
        raise HTTPException(status_code=404, detail="not found")

    if not abs_path.is_dir():
        # Single file metadata
        stat = abs_path.stat()
        rel = "/" + str(abs_path.relative_to(_get_workspace_path()))
        return {
            "items": [
                {
                    "name": abs_path.name,
                    "path": rel,
                    "isDir": False,
                    "size": stat.st_size,
                    "modifiedAt": datetime.fromtimestamp(
                        stat.st_mtime, tz=timezone.utc
                    ).strftime("%Y-%m-%dT%H:%M:%SZ"),
                }
            ]
        }

    items = []
    for entry in sorted(abs_path.iterdir(), key=lambda e: e.name):
        try:
            stat = entry.stat()
        except OSError:
            continue
        rel = "/" + str(entry.relative_to(_get_workspace_path()))
        items.append(
            {
                "name": entry.name,
                "path": rel,
                "isDir": entry.is_dir(),
                "size": stat.st_size,
                "modifiedAt": datetime.fromtimestamp(
                    stat.st_mtime, tz=timezone.utc
                ).strftime("%Y-%m-%dT%H:%M:%SZ"),
            }
        )

    return {"items": items}


@router.get("/file")
async def content_read(path: str = ""):
    """Read a file (mirrors Go ContentRead)."""
    abs_path = _safe_resolve(path)

    if not abs_path.exists():
        raise HTTPException(status_code=404, detail="not found")

    try:
        data = abs_path.read_bytes()
    except OSError as exc:
        logger.error("ContentRead: read failed for %s: %s", abs_path, exc)
        raise HTTPException(status_code=500, detail="read failed")

    return Response(content=data, media_type="application/octet-stream")


@router.post("/write")
async def content_write(request: Request):
    """Write a file (mirrors Go ContentWrite).

    Body: { path, content, encoding? }  — encoding "base64" decodes first.
    """
    body = await request.json()
    file_path = body.get("path", "")
    content = body.get("content", "")
    encoding = body.get("encoding", "")

    abs_path = _safe_resolve(file_path)

    # Create parent directories
    abs_path.parent.mkdir(parents=True, exist_ok=True)

    if encoding and encoding.lower() == "base64":
        try:
            data = base64.b64decode(content)
        except Exception:
            raise HTTPException(status_code=400, detail="invalid base64 content")
    else:
        data = content.encode("utf-8")

    try:
        abs_path.write_bytes(data)
    except OSError as exc:
        logger.error("ContentWrite: write failed for %s: %s", abs_path, exc)
        raise HTTPException(status_code=500, detail="failed to write file")

    return {"message": "ok"}


@router.delete("/delete")
async def content_delete(request: Request):
    """Delete a file (mirrors Go ContentDelete).

    Body: { path }
    """
    body = await request.json()
    file_path = body.get("path", "")

    abs_path = _safe_resolve(file_path)

    if not abs_path.exists():
        raise HTTPException(status_code=404, detail="file not found")

    try:
        abs_path.unlink()
    except OSError as exc:
        logger.error("ContentDelete: delete failed for %s: %s", abs_path, exc)
        raise HTTPException(status_code=500, detail="failed to delete file")

    return {"message": "file deleted successfully"}


# ------------------------------------------------------------------
# Git helpers
# ------------------------------------------------------------------


async def _git(*args: str, cwd: str) -> tuple[int, str, str]:
    """Run a git command asynchronously and return (returncode, stdout, stderr)."""
    proc = await asyncio.create_subprocess_exec(
        "git",
        *args,
        cwd=cwd,
        stdout=asyncio.subprocess.PIPE,
        stderr=asyncio.subprocess.PIPE,
    )
    stdout, stderr = await proc.communicate()
    return proc.returncode, stdout.decode().strip(), stderr.decode().strip()


# ------------------------------------------------------------------
# Git status
# ------------------------------------------------------------------


@router.get("/git-status")
async def content_git_status(path: str = ""):
    """Git status for a repo path (mirrors Go ContentGitStatus)."""
    abs_path = _safe_resolve(path)

    # Check directory exists
    if not abs_path.exists() or not abs_path.is_dir():
        return {"initialized": False, "hasChanges": False}

    # Check .git exists
    if not (abs_path / ".git").exists():
        return {"initialized": False, "hasChanges": False}

    cwd = str(abs_path)

    # Get current branch
    rc, branch_out, _ = await _git("rev-parse", "--abbrev-ref", "HEAD", cwd=cwd)
    current_branch = branch_out if rc == 0 else ""

    # Check remote
    rc, remote_out, _ = await _git("remote", "get-url", "origin", cwd=cwd)
    remote_url = remote_out if rc == 0 else ""
    has_remote = rc == 0 and remote_url != ""

    # Get diff stats (files changed)
    rc, diff_stat, _ = await _git("diff", "--stat", "HEAD", cwd=cwd)

    # Also check for untracked and staged files
    rc2, status_out, _ = await _git("status", "--porcelain", cwd=cwd)
    status_lines = [l for l in status_out.split("\n") if l.strip()] if status_out else []

    files_added = 0
    files_removed = 0
    total_added = 0
    total_removed = 0

    if status_lines:
        # Use numstat for accurate line counts
        rc3, numstat_out, _ = await _git(
            "diff", "--numstat", "HEAD", cwd=cwd
        )
        if rc3 == 0 and numstat_out:
            for line in numstat_out.split("\n"):
                parts = line.split("\t")
                if len(parts) >= 3:
                    added = int(parts[0]) if parts[0] != "-" else 0
                    removed = int(parts[1]) if parts[1] != "-" else 0
                    total_added += added
                    total_removed += removed
                    if added > 0:
                        files_added += 1
                    if removed > 0:
                        files_removed += 1

        # If numstat didn't pick up changes, count status lines
        if files_added == 0 and files_removed == 0:
            files_added = len(status_lines)

    has_changes = files_added > 0 or files_removed > 0 or total_added > 0 or total_removed > 0

    return {
        "initialized": True,
        "hasChanges": has_changes,
        "branch": current_branch,
        "remoteUrl": remote_url,
        "hasRemote": has_remote,
        "filesAdded": files_added,
        "filesRemoved": files_removed,
        "uncommittedFiles": files_added + files_removed,
        "totalAdded": total_added,
        "totalRemoved": total_removed,
    }


# ------------------------------------------------------------------
# Git configure remote
# ------------------------------------------------------------------


@router.post("/git-configure-remote")
async def content_git_configure_remote(request: Request):
    """Configure git remote (mirrors Go ContentGitConfigureRemote).

    Body: { path, remoteUrl, branch? }
    """
    body = await request.json()
    rel_path = body.get("path", "")
    remote_url = body.get("remoteUrl", "")
    branch = body.get("branch", "") or "main"

    abs_path = _safe_resolve(rel_path)

    if not abs_path.exists() or not abs_path.is_dir():
        raise HTTPException(status_code=400, detail="directory not found")

    cwd = str(abs_path)

    # Initialize git if not already
    if not (abs_path / ".git").exists():
        rc, _, stderr = await _git("init", cwd=cwd)
        if rc != 0:
            logger.error("git init failed: %s", stderr)
            raise HTTPException(status_code=500, detail="failed to initialize git")
        logger.info("Initialized git repository at %s", abs_path)

    # Inject authentication token into URL
    auth_url = remote_url
    github_token = (
        request.headers.get("X-GitHub-Token", "").strip()
        or os.getenv("GITHUB_TOKEN", "").strip()
    )
    gitlab_token = (
        request.headers.get("X-GitLab-Token", "").strip()
        or os.getenv("GITLAB_TOKEN", "").strip()
    )

    if github_token and "github" in remote_url.lower():
        auth_url = remote_url.replace(
            "https://", f"https://x-access-token:{github_token}@"
        )
    elif gitlab_token and "gitlab" in remote_url.lower():
        auth_url = remote_url.replace(
            "https://", f"https://oauth2:{gitlab_token}@"
        )

    # Check if remote exists
    rc, _, _ = await _git("remote", "get-url", "origin", cwd=cwd)
    if rc == 0:
        await _git("remote", "set-url", "origin", auth_url, cwd=cwd)
    else:
        await _git("remote", "add", "origin", auth_url, cwd=cwd)

    logger.info("Configured remote for %s: %s", abs_path, remote_url)

    # Fetch from remote (best-effort)
    rc, _, fetch_err = await _git("fetch", "origin", branch, cwd=cwd)
    if rc != 0:
        logger.warning(
            "Initial fetch after configure remote failed (non-fatal): %s", fetch_err
        )
    else:
        logger.info("Fetched origin/%s after configuring remote", branch)

    return {"message": "remote configured", "remote": remote_url, "branch": branch}


# ------------------------------------------------------------------
# Git list branches
# ------------------------------------------------------------------


@router.get("/git-list-branches")
async def content_git_list_branches(path: str = ""):
    """List remote branches (mirrors Go ContentGitListBranches)."""
    abs_path = _safe_resolve(path)
    cwd = str(abs_path)

    rc, stdout, stderr = await _git("branch", "-r", "--format=%(refname:short)", cwd=cwd)
    if rc != 0:
        logger.error("git list branches failed: %s", stderr)
        raise HTTPException(status_code=500, detail="Internal server error")

    branches = [b.strip() for b in stdout.split("\n") if b.strip()]
    # Strip 'origin/' prefix
    branches = [
        b.removeprefix("origin/") for b in branches if not b.endswith("/HEAD")
    ]

    return {"branches": branches}


# ------------------------------------------------------------------
# Workflow metadata
# ------------------------------------------------------------------


def _parse_frontmatter(file_path: Path) -> dict[str, str]:
    """Extract YAML frontmatter key: value pairs from a markdown file."""
    try:
        content = file_path.read_text(encoding="utf-8")
    except OSError:
        return {}

    if not content.startswith("---\n"):
        return {}

    end_idx = content.find("\n---", 4)
    if end_idx == -1:
        return {}

    frontmatter = content[4:end_idx]
    result: dict[str, str] = {}
    for line in frontmatter.split("\n"):
        if not line.strip():
            continue
        parts = line.split(":", 1)
        if len(parts) == 2:
            key = parts[0].strip()
            value = parts[1].strip().strip("\"'")
            result[key] = value

    return result


def _find_active_workflow_dir() -> str | None:
    """Find the active workflow directory for the current session."""
    workspace = _get_workspace_path()
    workflows_base = workspace / "workflows"

    if not workflows_base.exists():
        return None

    for entry in workflows_base.iterdir():
        if (
            entry.is_dir()
            and entry.name != "default"
            and not entry.name.endswith("-clone-temp")
        ):
            claude_dir = entry / ".claude"
            if claude_dir.exists() and claude_dir.is_dir():
                return str(entry)

    return None


def _parse_ambient_config(workflow_dir: str) -> dict:
    """Read and parse ambient.json from workflow directory."""
    config_path = Path(workflow_dir) / ".ambient" / "ambient.json"

    if not config_path.exists():
        return {"artifactsDir": ""}

    try:
        data = json.loads(config_path.read_text(encoding="utf-8"))
        return data
    except (OSError, json.JSONDecodeError) as exc:
        logger.warning("Failed to parse ambient.json: %s", exc)
        return {"artifactsDir": ""}


@router.get("/workflow-metadata")
async def content_workflow_metadata(session: str = ""):
    """Read workflow commands/agents from .claude/ and .ambient/ directories.

    Mirrors Go ContentWorkflowMetadata.
    """
    if not session:
        raise HTTPException(status_code=400, detail="missing session parameter")

    workflow_dir = _find_active_workflow_dir()
    if not workflow_dir:
        return {
            "commands": [],
            "agents": [],
            "config": {"artifactsDir": "artifacts"},
        }

    wf_path = Path(workflow_dir)
    ambient_config = _parse_ambient_config(workflow_dir)

    # Parse commands from .claude/commands/*.md
    commands_dir = wf_path / ".claude" / "commands"
    commands = []

    if commands_dir.exists():
        for md_file in sorted(commands_dir.iterdir()):
            if md_file.is_dir() or not md_file.name.endswith(".md"):
                continue

            metadata = _parse_frontmatter(md_file)
            command_name = md_file.stem

            display_name = metadata.get("displayName") or command_name

            order = 2**31 - 1  # default large value
            if "order" in metadata:
                try:
                    order = int(metadata["order"])
                except ValueError:
                    pass

            commands.append(
                {
                    "id": command_name,
                    "name": display_name,
                    "description": metadata.get("description", ""),
                    "slashCommand": f"/{command_name}",
                    "icon": metadata.get("icon", ""),
                    "order": order,
                }
            )

        # Sort by order, then alphabetically by id
        commands.sort(key=lambda c: (c["order"], c["id"]))

    # Parse agents from .claude/agents/*.md
    agents_dir = wf_path / ".claude" / "agents"
    agents = []

    if agents_dir.exists():
        for md_file in sorted(agents_dir.iterdir()):
            if md_file.is_dir() or not md_file.name.endswith(".md"):
                continue

            metadata = _parse_frontmatter(md_file)
            agent_id = md_file.stem

            agents.append(
                {
                    "id": agent_id,
                    "name": metadata.get("name", ""),
                    "description": metadata.get("description", ""),
                    "tools": metadata.get("tools", ""),
                }
            )

    config_response = {
        "name": ambient_config.get("name", ""),
        "description": ambient_config.get("description", ""),
        "systemPrompt": ambient_config.get("systemPrompt", ""),
        "artifactsDir": ambient_config.get("artifactsDir", ""),
    }
    if "rubric" in ambient_config and ambient_config["rubric"]:
        config_response["rubric"] = ambient_config["rubric"]

    return {"commands": commands, "agents": agents, "config": config_response}
