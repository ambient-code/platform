"""Background task endpoints — stop, list, output."""

import json
import logging
from pathlib import Path

from fastapi import APIRouter, HTTPException, Request

logger = logging.getLogger(__name__)

router = APIRouter(prefix="/tasks", tags=["tasks"])

# Max transcript size to read into memory (10 MB).
_MAX_OUTPUT_BYTES = 10 * 1024 * 1024


@router.post("/{task_id}/stop")
async def stop_task(task_id: str, request: Request):
    """Stop a running background task by ID."""
    bridge = request.app.state.bridge

    thread_id = None
    try:
        body = await request.json()
        thread_id = body.get("thread_id")
    except Exception:
        pass

    logger.info(f"Stop task request: task_id={task_id}")
    try:
        await bridge.stop_task(task_id, thread_id)
        return {"message": "stop signal sent"}
    except Exception as e:
        logger.error(f"stop_task({task_id}) failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("/{task_id}/output")
async def get_task_output(task_id: str, request: Request):
    """Get the transcript/output of a background task (running or completed)."""
    bridge = request.app.state.bridge
    output_path = bridge.task_outputs.get(task_id)

    # Fallback: search known directories where the SDK writes output files.
    # During execution: /tmp/claude-*/.../{session_id}/tasks/{task_id}.output
    # After completion: ~/.claude/projects/.../{session_id}/subagents/agent-{task_id}.jsonl
    if not output_path:
        search_dirs = [
            Path("/tmp"),
            Path.home() / ".claude" / "projects",
        ]
        for search_dir in search_dirs:
            if not search_dir.exists():
                continue
            for match in search_dir.rglob(f"*{task_id}*"):
                output_path = str(match)
                break
            if output_path:
                break

    if not output_path:
        raise HTTPException(
            status_code=404, detail=f"No output found for task {task_id}"
        )

    # Allow paths under ~/.claude/ or /tmp/ (SDK writes to both)
    resolved = Path(output_path).resolve()
    allowed_roots = [
        (Path.home() / ".claude").resolve(),
        Path("/tmp").resolve(),
    ]
    if not any(resolved.is_relative_to(root) for root in allowed_roots):
        raise HTTPException(status_code=403, detail="Access denied")

    if not resolved.exists():
        raise HTTPException(
            status_code=404, detail=f"Output file not found: {output_path}"
        )

    if resolved.stat().st_size > _MAX_OUTPUT_BYTES:
        raise HTTPException(
            status_code=413, detail="Transcript too large"
        )

    try:
        entries = []
        with open(resolved) as f:
            for line in f:
                line = line.strip()
                if line:
                    try:
                        entries.append(json.loads(line))
                    except json.JSONDecodeError:
                        entries.append({"raw": line})
        return {"task_id": task_id, "output": entries}
    except Exception as e:
        logger.error(f"Failed to read output for task {task_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))


@router.get("")
async def list_tasks(request: Request):
    """List all tracked background tasks."""
    bridge = request.app.state.bridge
    return {"tasks": list(bridge.task_registry.values())}
