"""POST /model — Switch the LLM model at runtime."""

import asyncio
import logging
import os

from fastapi import APIRouter, HTTPException, Request

logger = logging.getLogger(__name__)

router = APIRouter()

# Serialise model changes to prevent concurrent switches
_model_change_lock = asyncio.Lock()


@router.post("/model")
async def switch_model(request: Request):
    """Switch the LLM model used by this session.

    The agent must be idle (not mid-generation). If a run is in
    progress the endpoint returns 422.
    """
    bridge = request.app.state.bridge
    context = bridge.context
    if not context:
        raise HTTPException(status_code=503, detail="Context not initialized")

    body = await request.json()
    new_model = (body.get("model") or "").strip()

    if not new_model:
        raise HTTPException(status_code=400, detail="model is required")

    previous_model = os.getenv("LLM_MODEL", "")

    if new_model == previous_model:
        return {
            "message": "Model unchanged",
            "model": new_model,
        }

    # Check if agent is mid-generation.
    # The session manager holds a per-thread asyncio.Lock during runs.
    session_manager = getattr(bridge, "_session_manager", None)
    if session_manager:
        thread_id = context.session_id if context else ""
        lock = session_manager.get_lock(thread_id) if thread_id else None
        if lock and lock.locked():
            raise HTTPException(
                status_code=422,
                detail="Cannot switch model while agent is generating a response. Wait for the current turn to complete.",
            )

    # Fast-reject if another switch is already in progress.
    # asyncio is single-threaded, so no yield between locked() and acquire().
    if _model_change_lock.locked():
        raise HTTPException(
            status_code=409,
            detail="A model switch is already in progress",
        )
    async with _model_change_lock:
        return await _perform_model_switch(bridge, context, new_model, previous_model)


async def _perform_model_switch(
    bridge, context, new_model: str, previous_model: str
) -> dict:
    """Execute the model switch: update env, rebuild adapter, emit event."""
    logger.info(f"Switching model from '{previous_model}' to '{new_model}'")

    # Update environment variable (read by setup_sdk_authentication on next init)
    os.environ["LLM_MODEL"] = new_model

    # Also update the Vertex ID mapping if applicable
    use_vertex = os.getenv("USE_VERTEX", "").strip().lower() in ("1", "true", "yes")
    if use_vertex:
        # Clear the manifest override so auth.py re-derives from the new LLM_MODEL
        os.environ.pop("LLM_MODEL_VERTEX_ID", None)

    # Emit confirmation event BEFORE mark_dirty destroys the session manager
    _emit_model_switched_event(bridge, context, new_model, previous_model)

    # Signal adapter rebuild — stops current workers, preserves session IDs
    bridge.mark_dirty()

    logger.info(f"Model switch complete: {previous_model} -> {new_model}")

    return {
        "message": "Model switched",
        "model": new_model,
        "previousModel": previous_model,
    }


def _emit_model_switched_event(bridge, context, new_model: str, previous_model: str):
    """Push a custom AG-UI event to notify the frontend of the model switch."""
    try:
        from ag_ui.core import CustomEvent, EventType

        event = CustomEvent(
            type=EventType.CUSTOM,
            name="ambient:model_switched",
            value={
                "previousModel": previous_model,
                "newModel": new_model,
            },
        )

        # Route to the between-run event queue so the frontend picks it up
        session_manager = getattr(bridge, "_session_manager", None)
        if session_manager:
            thread_id = context.session_id if context else ""
            worker = session_manager.get_existing(thread_id)
            if worker:
                worker._between_run_queue.put_nowait(event)
                logger.info("Model switch event emitted to between-run queue")
                return

        logger.warning("No active worker to emit model switch event")
    except Exception as e:
        logger.warning(f"Failed to emit model switch event: {e}")
