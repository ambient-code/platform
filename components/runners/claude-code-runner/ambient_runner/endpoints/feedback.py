"""POST /feedback — Langfuse thumbs-up/down scoring (SDK-provided)."""

# Re-export the existing endpoint router for now.
# In a future extraction to a standalone package, this will be self-contained.
from endpoints.feedback import router

__all__ = ["router"]
