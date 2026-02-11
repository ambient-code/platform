"""POST /interrupt — interrupt the current run."""

import logging

from fastapi import APIRouter, HTTPException, Request

logger = logging.getLogger(__name__)

router = APIRouter()


@router.post("/interrupt")
async def interrupt_run(request: Request):
    """Interrupt the current agent execution."""
    bridge = request.app.state.bridge

    logger.info("Interrupt request received")
    try:
        await bridge.interrupt()
        return {"message": "Interrupt signal sent"}
    except Exception as e:
        logger.error(f"Interrupt failed: {e}")
        raise HTTPException(status_code=500, detail=str(e))
