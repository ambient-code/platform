"""GET /health — health check endpoint."""

from fastapi import APIRouter, Request

router = APIRouter()


@router.get("/health")
async def health(request: Request):
    """Health check."""
    bridge = request.app.state.bridge
    context = bridge.context
    return {
        "status": "healthy",
        "session_id": context.session_id if context else None,
    }
