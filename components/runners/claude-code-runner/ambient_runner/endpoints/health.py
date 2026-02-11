"""GET /health — health check endpoint."""

from fastapi import APIRouter

router = APIRouter()


@router.get("/health")
async def health():
    """Health check."""
    return {"status": "healthy"}
