"""Unit tests for the /workflow endpoint — clone_workflow_at_runtime.

Focuses on the bug where shutil.move fails with FileNotFoundError when the
parent /workspace/workflows directory does not yet exist (Issue #1493).
"""

import asyncio
import shutil
from pathlib import Path
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from ambient_runner.endpoints.workflow import clone_workflow_at_runtime


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _make_successful_process(returncode: int = 0) -> AsyncMock:
    """Return a mock asyncio subprocess that exits successfully."""
    proc = AsyncMock()
    proc.returncode = returncode
    proc.communicate = AsyncMock(return_value=(b"", b""))
    return proc


def _make_failing_process() -> AsyncMock:
    proc = AsyncMock()
    proc.returncode = 1
    proc.communicate = AsyncMock(return_value=(b"", b"error: not found"))
    return proc


# ---------------------------------------------------------------------------
# Tests: no-subpath path
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_no_subpath_creates_parent_directory(tmp_path: Path):
    """clone_workflow_at_runtime must create /workspace/workflows when it does
    not exist before moving the temp clone (fixes the FileNotFoundError bug)."""
    workspace = tmp_path / "workspace"
    workspace.mkdir()
    # Intentionally do NOT create `workspace/workflows`

    git_url = "https://github.com/example/my-workflow.git"
    branch = "main"

    # Simulate git clone by writing a file into the temp dir
    async def fake_subprocess(*args, **kwargs):
        # Extract the destination dir (last positional arg to git clone)
        dest = args[-1] if args else kwargs.get("args", [None])[-1]
        Path(dest).mkdir(parents=True, exist_ok=True)
        (Path(dest) / "ambient.json").write_text("{}")
        return _make_successful_process()

    with patch(
        "asyncio.create_subprocess_exec", side_effect=fake_subprocess
    ), patch(
        "ambient_runner.endpoints.workflow.ensure_git_auth"
    ), patch(
        "os.getenv", side_effect=lambda k, d="": str(workspace) if k == "WORKSPACE_PATH" else d
    ):
        success, path = await clone_workflow_at_runtime(git_url, branch, "")

    assert success, "Expected clone to succeed"
    assert path != "", "Expected a non-empty path"
    expected = workspace / "workflows" / "my-workflow"
    assert expected.exists(), (
        f"Parent directory was not created; {expected} does not exist"
    )


@pytest.mark.asyncio
async def test_no_subpath_succeeds_when_parent_already_exists(tmp_path: Path):
    """When /workspace/workflows already exists the operation should still work."""
    workspace = tmp_path / "workspace"
    (workspace / "workflows").mkdir(parents=True)

    git_url = "https://github.com/example/repo.git"

    async def fake_subprocess(*args, **kwargs):
        dest = args[-1] if args else kwargs.get("args", [None])[-1]
        Path(dest).mkdir(parents=True, exist_ok=True)
        return _make_successful_process()

    with patch(
        "asyncio.create_subprocess_exec", side_effect=fake_subprocess
    ), patch(
        "ambient_runner.endpoints.workflow.ensure_git_auth"
    ), patch(
        "os.getenv", side_effect=lambda k, d="": str(workspace) if k == "WORKSPACE_PATH" else d
    ):
        success, path = await clone_workflow_at_runtime(git_url, "main", "")

    assert success


@pytest.mark.asyncio
async def test_git_clone_failure_returns_false(tmp_path: Path):
    """A failed git clone should return (False, '') without raising."""
    workspace = tmp_path / "workspace"
    workspace.mkdir()

    git_url = "https://github.com/example/private-repo.git"

    with patch(
        "asyncio.create_subprocess_exec", return_value=_make_failing_process()
    ), patch(
        "ambient_runner.endpoints.workflow.ensure_git_auth"
    ), patch(
        "os.getenv", side_effect=lambda k, d="": str(workspace) if k == "WORKSPACE_PATH" else d
    ):
        success, path = await clone_workflow_at_runtime(git_url, "main", "")

    assert not success
    assert path == ""


# ---------------------------------------------------------------------------
# Tests: subpath-not-found fallback
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_subpath_not_found_creates_parent_directory(tmp_path: Path):
    """When the specified subpath does not exist in the cloned repo, the entire
    repo should be used — and the parent directory must still be created."""
    workspace = tmp_path / "workspace"
    workspace.mkdir()
    # No workflows/ dir

    git_url = "https://github.com/example/monorepo.git"

    async def fake_subprocess(*args, **kwargs):
        dest = args[-1] if args else kwargs.get("args", [None])[-1]
        Path(dest).mkdir(parents=True, exist_ok=True)
        # Do NOT create the requested subpath
        return _make_successful_process()

    with patch(
        "asyncio.create_subprocess_exec", side_effect=fake_subprocess
    ), patch(
        "ambient_runner.endpoints.workflow.ensure_git_auth"
    ), patch(
        "os.getenv", side_effect=lambda k, d="": str(workspace) if k == "WORKSPACE_PATH" else d
    ):
        success, path = await clone_workflow_at_runtime(
            git_url, "main", "does-not-exist"
        )

    assert success, "Expected fallback to entire repo to succeed"
    expected = workspace / "workflows" / "monorepo"
    assert expected.exists(), (
        f"Parent directory was not created in subpath-fallback path; {expected} missing"
    )


# ---------------------------------------------------------------------------
# Tests: subpath found (existing behaviour, should remain unchanged)
# ---------------------------------------------------------------------------


@pytest.mark.asyncio
async def test_subpath_found_extracts_correctly(tmp_path: Path):
    """When the subpath exists, only that subdirectory is extracted."""
    workspace = tmp_path / "workspace"
    workspace.mkdir()

    git_url = "https://github.com/example/monorepo.git"

    async def fake_subprocess(*args, **kwargs):
        dest = args[-1] if args else kwargs.get("args", [None])[-1]
        dest_path = Path(dest)
        dest_path.mkdir(parents=True, exist_ok=True)
        # Create the subpath content
        subdir = dest_path / "workflows" / "my-workflow"
        subdir.mkdir(parents=True)
        (subdir / "ambient.json").write_text('{"name":"my-workflow"}')
        return _make_successful_process()

    with patch(
        "asyncio.create_subprocess_exec", side_effect=fake_subprocess
    ), patch(
        "ambient_runner.endpoints.workflow.ensure_git_auth"
    ), patch(
        "os.getenv", side_effect=lambda k, d="": str(workspace) if k == "WORKSPACE_PATH" else d
    ):
        success, path = await clone_workflow_at_runtime(
            git_url, "main", "workflows/my-workflow"
        )

    assert success
    extracted = workspace / "workflows" / "monorepo"
    assert extracted.exists()
    assert (extracted / "ambient.json").exists()
