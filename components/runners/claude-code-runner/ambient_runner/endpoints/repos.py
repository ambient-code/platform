"""Repository management endpoints (SDK-provided).

Re-exports the router from the top-level endpoints package.
When the SDK is extracted to a standalone package, this will be self-contained.
"""

# Import from the runner's top-level endpoints (not a relative import within ambient_runner)
import importlib
_mod = importlib.import_module("endpoints.repos")
router = _mod.router

__all__ = ["router"]
