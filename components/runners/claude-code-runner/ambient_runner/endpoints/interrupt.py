"""POST /interrupt — interrupt the current run."""

import logging

from fastapi import APIRouter, HTTPException, Request

logger = logging.getLogger(__name__)

router = APIRouter()


@router.post("/interrupt")
async def interrupt_run(request: Request):
    """Interrupt the current agent execution."""
    bridge = request.app.state.bridge

    # Try to get thread_id from request body
    thread_id = None
    try:
        body = await request.json()
        thread_id = body.get("thread_id")
    except Exception:
        pass

    logger.info(f"Interrupt request received (thread_id={thread_id})")
    try:
        await bridge.interrupt(thread_id)
        return {"message": "Interrupt signal sent"}
    except Exception as e:
        logger.error(f"Interrupt failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))
