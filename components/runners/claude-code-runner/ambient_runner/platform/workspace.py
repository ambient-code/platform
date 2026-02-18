"""
Workspace and path management for the Ambient Runner SDK.

Handles workflow/repo directory setup, workspace validation,
and prerequisite checking for phase-based commands.
"""

import logging
import os
from pathlib import Path
from urllib.parse import urlparse

from ambient_runner.platform.context import RunnerContext
from ambient_runner.platform.utils import parse_owner_repo

logger = logging.getLogger(__name__)


class PrerequisiteError(RuntimeError):
    """Raised when slash-command prerequisites are missing."""

    pass


def setup_workflow_paths(
    context: RunnerContext, active_workflow_url: str, repos_cfg: list
) -> tuple[str, list, str]:
    """Setup CWD and additional directories for workflow mode.

    Returns:
        (cwd_path, additional_dirs, derived_workflow_name)
    """
    add_dirs: list[str] = []
    derived_name = None
    cwd_path = context.workspace_path

    try:
        _owner, repo, _ = parse_owner_repo(active_workflow_url)
        derived_name = repo or ""
        if not derived_name:
            p = urlparse(active_workflow_url)
            parts = [pt for pt in (p.path or "").split("/") if pt]
            if parts:
                derived_name = parts[-1]
        derived_name = (derived_name or "").removesuffix(".git").strip()

        if derived_name:
            workflow_path = str(
                Path(context.workspace_path) / "workflows" / derived_name
            )
            if Path(workflow_path).exists():
                cwd_path = workflow_path
                logger.info(f"Using workflow as CWD: {derived_name}")
            else:
                logger.warning(
                    f"Workflow directory not found: {workflow_path}, using default"
                )
                cwd_path = str(
                    Path(context.workspace_path) / "workflows" / "default"
                )
        else:
            cwd_path = str(
                Path(context.workspace_path) / "workflows" / "default"
            )
    except Exception as e:
        logger.warning(f"Failed to derive workflow name: {e}, using default")
        cwd_path = str(
            Path(context.workspace_path) / "workflows" / "default"
        )

    # Add all repos as additional directories
    repos_base = Path(context.workspace_path) / "repos"
    for r in repos_cfg:
        name = (r.get("name") or "").strip()
        if name:
            repo_path = str(repos_base / name)
            if repo_path not in add_dirs:
                add_dirs.append(repo_path)

    # Add artifacts and file-uploads directories
    artifacts_path = str(Path(context.workspace_path) / "artifacts")
    if artifacts_path not in add_dirs:
        add_dirs.append(artifacts_path)

    file_uploads_path = str(Path(context.workspace_path) / "file-uploads")
    if file_uploads_path not in add_dirs:
        add_dirs.append(file_uploads_path)

    return cwd_path, add_dirs, derived_name


def setup_multi_repo_paths(
    context: RunnerContext, repos_cfg: list
) -> tuple[str, list]:
    """Setup CWD and additional directories for multi-repo mode.

    Repos are cloned to /workspace/repos/{name} by both
    hydrate.sh (init container) and clone_repo_at_runtime().

    Returns:
        (cwd_path, additional_dirs)
    """
    add_dirs: list[str] = []
    repos_base = Path(context.workspace_path) / "repos"

    main_name = (os.getenv("MAIN_REPO_NAME") or "").strip()
    if not main_name:
        idx_raw = (os.getenv("MAIN_REPO_INDEX") or "").strip()
        try:
            idx_val = int(idx_raw) if idx_raw else 0
        except Exception:
            idx_val = 0
        if idx_val < 0 or idx_val >= len(repos_cfg):
            idx_val = 0
        main_name = (repos_cfg[idx_val].get("name") or "").strip()

    cwd_path = (
        str(repos_base / main_name) if main_name else context.workspace_path
    )

    for r in repos_cfg:
        name = (r.get("name") or "").strip()
        if not name:
            continue
        p = str(repos_base / name)
        if p != cwd_path:
            add_dirs.append(p)

    # Add artifacts and file-uploads directories
    artifacts_path = str(Path(context.workspace_path) / "artifacts")
    if artifacts_path not in add_dirs:
        add_dirs.append(artifacts_path)

    file_uploads_path = str(Path(context.workspace_path) / "file-uploads")
    if file_uploads_path not in add_dirs:
        add_dirs.append(file_uploads_path)

    return cwd_path, add_dirs


def resolve_workspace_paths(context: RunnerContext) -> tuple[str, list[str]]:
    """Resolve the working directory and additional directories.

    Determines the working directory based on active workflow, repos config,
    or falls back to the artifacts directory.

    Returns:
        (cwd_path, additional_dirs)
    """
    import ambient_runner.platform.config as runner_config

    repos_cfg = runner_config.get_repos_config()
    cwd_path = context.workspace_path
    add_dirs: list[str] = []

    active_workflow_url = (os.getenv("ACTIVE_WORKFLOW_GIT_URL") or "").strip()

    if active_workflow_url:
        cwd_path, add_dirs, _ = setup_workflow_paths(
            context, active_workflow_url, repos_cfg
        )
    elif repos_cfg:
        cwd_path, add_dirs = setup_multi_repo_paths(context, repos_cfg)
    else:
        cwd_path = str(Path(context.workspace_path) / "artifacts")

    cwd_path_obj = Path(cwd_path)
    if not cwd_path_obj.exists():
        logger.warning(f"Working directory missing, creating: {cwd_path}")
        try:
            cwd_path_obj.mkdir(parents=True, exist_ok=True)
        except Exception as e:
            logger.error(f"Failed to create working directory: {e}")
            cwd_path = context.workspace_path

    logger.info(f"Claude SDK CWD: {cwd_path}")
    if add_dirs:
        logger.info(f"Claude SDK additional directories: {add_dirs}")

    return cwd_path, add_dirs


async def prepare_workspace(context: RunnerContext) -> None:
    """Validate workspace prepared by init container.

    The init-hydrate container handles downloading state from S3,
    cloning repos, and cloning workflows. This just validates and logs.
    """
    workspace = Path(context.workspace_path)
    logger.info(f"Validating workspace at {workspace}")

    hydrated_paths = []
    for path_name in [".claude", "artifacts", "file-uploads"]:
        path_dir = workspace / path_name
        if path_dir.exists():
            file_count = len([f for f in path_dir.rglob("*") if f.is_file()])
            if file_count > 0:
                hydrated_paths.append(f"{path_name} ({file_count} files)")

    if hydrated_paths:
        logger.info(f"Hydrated from S3: {', '.join(hydrated_paths)}")
    else:
        logger.info("No state hydrated (fresh session)")


async def validate_prerequisites(context: RunnerContext) -> None:
    """Validate prerequisite files exist for phase-based slash commands.

    Raises:
        PrerequisiteError: If a required file is missing.
    """
    prompt = context.get_env("INITIAL_PROMPT", "")
    if not prompt:
        return

    prompt_lower = prompt.strip().lower()

    prerequisites = {
        "/speckit.plan": (
            "spec.md",
            "Specification file (spec.md) not found. Please run /speckit.specify first.",
        ),
        "/speckit.tasks": (
            "plan.md",
            "Planning file (plan.md) not found. Please run /speckit.plan first.",
        ),
        "/speckit.implement": (
            "tasks.md",
            "Tasks file (tasks.md) not found. Please run /speckit.tasks first.",
        ),
    }

    for cmd, (required_file, error_msg) in prerequisites.items():
        if prompt_lower.startswith(cmd):
            workspace = Path(context.workspace_path)
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


async def initialize_workflow_if_set(context: RunnerContext) -> None:
    """Validate workflow was cloned by init container."""
    active_workflow_url = (os.getenv("ACTIVE_WORKFLOW_GIT_URL") or "").strip()
    if not active_workflow_url:
        return

    try:
        _owner, repo, _ = parse_owner_repo(active_workflow_url)
        derived_name = repo or ""
        if not derived_name:
            p = urlparse(active_workflow_url)
            parts = [pt for pt in (p.path or "").split("/") if pt]
            if parts:
                derived_name = parts[-1]
        derived_name = (derived_name or "").removesuffix(".git").strip()

        if not derived_name:
            logger.warning("Could not derive workflow name from URL")
            return

        workspace = Path(context.workspace_path)
        workflow_temp_dir = workspace / "workflows" / f"{derived_name}-clone-temp"
        workflow_dir = workspace / "workflows" / derived_name

        if workflow_temp_dir.exists():
            logger.info(
                f"Workflow {derived_name} cloned by init container "
                f"at {workflow_temp_dir.name}"
            )
        elif workflow_dir.exists():
            logger.info(
                f"Workflow {derived_name} available at {workflow_dir.name}"
            )
        else:
            logger.warning(
                f"Workflow {derived_name} not found "
                "(init container may have failed to clone)"
            )

    except Exception as e:
        logger.error(f"Failed to validate workflow: {e}")
